// Package metrics — Prometheus metrics cho app.
//
// PORT RIÊNG (mặc định 127.0.0.1:9090/metrics) — quyết định có chủ đích:
//   - Scrape không đi qua middleware app (rate_limit, maintenance, auth, recovery)
//     → scrape không fail khi app gồng/maintenance
//   - Không lộ qua app port (thường public qua tunnel) — bind localhost mặc định
//   - Log không nhiễu (Prometheus scrape 15s/lần), tách lifecycle
//   - Chuẩn Go ecosystem (exporter/pprof đều tách listener)
//
// Auth OPTIONAL: set METRICS_AUTH_TOKEN → yêu cầu `Authorization: Bearer <token>`
// (Prometheus scrape_configs: bearer_token). Mặc định KHÔNG auth (localhost an toàn).
package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collectors — metric chuẩn của app (register vào default registry —
// kèm go_* + process_* collectors sẵn của client_golang).
var (
	// HTTPRequestsTotal — số request theo method + route + status.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Tổng số HTTP request xử lý.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDurationSeconds — latency histogram (ms → seconds).
	HTTPRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency HTTP request.",
			Buckets: prometheus.DefBuckets, // 5ms → 10s
		},
		[]string{"method", "path"},
	)

	// HTTPRequestsInFlight — request đang xử lý (detect hang/leak).
	HTTPRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Số request đang xử lý.",
		},
		[]string{"method"},
	)

	// AppMaintenanceMode — app đang trong chế độ bảo trì?
	// 1 = maintenance (mọi API 503), 0 = bình thường — alert/grafana hiển thị trạng thái.
	// Cập nhật runtime qua SetMaintenance (nối với dynamic config app.maintenance_mode).
	AppMaintenanceMode = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "app_maintenance_mode",
		Help: "App đang trong chế độ bảo trì (1 = maintenance — mọi API trả 503, 0 = bình thường).",
	})

	// ServiceUp — trạng thái health của service phụ thuộc (postgres, redis, minio, mail...).
	// 1 = up, 0 = down — cập nhật định kỳ bởi StartHealthChecks. Alert khi 0.
	ServiceUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "service_up",
			Help: "Service phụ thuộc còn hoạt động (1 = up, 0 = down) — ping định kỳ.",
		},
		[]string{"service"},
	)
)

func init() {
	prometheus.MustRegister(HTTPRequestsTotal, HTTPRequestDurationSeconds, HTTPRequestsInFlight, AppMaintenanceMode, ServiceUp)
}

// SetMaintenance — đồng bộ gauge với trạng thái maintenance (gọi từ server khi
// dynamic config app.maintenance_mode đổi — OnChange).
func SetMaintenance(on bool) {
	if on {
		AppMaintenanceMode.Set(1)
	} else {
		AppMaintenanceMode.Set(0)
	}
}

// HealthCheck — một service cần ping định kỳ.
type HealthCheck struct {
	Name string                          // label service (vd "postgres")
	Ping func(ctx context.Context) error // lỗi → coi là down
}

// healthCheckInterval — chu kỳ ping các service (không magic number).
const healthCheckInterval = 30 * time.Second

// StartHealthChecks — ping định kỳ các service phụ thuộc, set gauge service_up.
// Chạy NGAY lần đầu (metric có giá trị từ khi start), sau đó mỗi interval;
// dừng khi ctx cancel (graceful shutdown).
func StartHealthChecks(ctx context.Context, checks []HealthCheck) {
	run := func() {
		for _, c := range checks {
			if err := c.Ping(ctx); err != nil {
				ServiceUp.WithLabelValues(c.Name).Set(0)
				slog.Warn("health check thất bại", "service", c.Name, "err", err)
			} else {
				ServiceUp.WithLabelValues(c.Name).Set(1)
			}
		}
	}

	run()
	ticker := time.NewTicker(healthCheckInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

// Handler — endpoint /metrics (promhttp + optional Bearer auth).
func Handler(authToken string) http.Handler {
	h := promhttp.Handler()
	if authToken == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+authToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized\n"))
			return
		}
		h.ServeHTTP(w, r)
	})
}

// GinMiddleware — instrument request (đếm + duration + in-flight).
// Dùng c.FullPath() làm label path (vd /admin/users/:id) — tránh cardinality
// explosion khi path chứa id/query.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		HTTPRequestsInFlight.WithLabelValues(method).Inc()
		start := time.Now()

		c.Next()

		HTTPRequestsInFlight.WithLabelValues(method).Dec()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())
		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDurationSeconds.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}
