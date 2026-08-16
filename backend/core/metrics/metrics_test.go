package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// scrape — GET handler /metrics, trả text body.
func scrape(t *testing.T, h http.Handler, req *http.Request) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Code, w.Body.String()
}

func TestHandler_NoAuth_ExposesMetrics(t *testing.T) {
	h := Handler("")
	code, body := scrape(t, h, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "go_goroutines", "go collector sẵn")
	assert.Contains(t, body, "process_resident_memory_bytes", "process collector sẵn")
	// http_requests_total chỉ xuất hiện khi đã có request (label rỗng không export) —
	// verify ở TestGinMiddleware_IncrementsCounters
}

func TestHandler_AuthToken_Required(t *testing.T) {
	h := Handler("secret-token")

	// Không có token → 401
	code, body := scrape(t, h, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Contains(t, body, "unauthorized")

	// Token sai → 401
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	code, _ = scrape(t, h, req)
	assert.Equal(t, http.StatusUnauthorized, code)

	// Token đúng → 200
	reqOK := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	reqOK.Header.Set("Authorization", "Bearer secret-token")
	code, body = scrape(t, h, reqOK)
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "go_goroutines")
}

func TestGinMiddleware_IncrementsCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Registry riêng để không đụng global (test độc lập)
	reg := prometheus.NewRegistry()
	total := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_http_total"}, []string{"method", "path", "status"})
	_ = reg.Register(total)

	// Dùng middleware thật nhưng theo dõi qua collector chung của package
	r := gin.New()
	r.Use(GinMiddleware())
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/users/:id", func(c *gin.Context) { c.Status(http.StatusNotFound) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/users/42", nil))
	assert.Equal(t, http.StatusNotFound, w2.Code)

	// Verify qua handler (metric đã tăng)
	_, body := scrape(t, Handler(""), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, body, `http_requests_total{method="GET",path="/healthz",status="200"} 1`)
	assert.Contains(t, body, `http_requests_total{method="GET",path="/users/:id",status="404"} 1`, "path = route (không phải id cụ thể)")
	assert.Contains(t, body, `http_request_duration_seconds_count{method="GET",path="/healthz"}`)
}

func TestGinMiddleware_CountsRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Request bị abort (vd rate-limit) vẫn phải đếm
	r := gin.New()
	r.Use(GinMiddleware())
	r.Use(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusTooManyRequests)
	})
	r.GET("/x", func(c *gin.Context) {})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	_, body := scrape(t, Handler(""), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, body, `http_requests_total{method="GET",path="/x",status="429"} 1`, "request bị chặn vẫn đếm")
}

func TestHandler_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	Handler("").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestSetMaintenance_GaugeReflectsState(t *testing.T) {
	// Mặc định gauge phải tồn tại và = 0
	SetMaintenance(false)
	_, body := scrape(t, Handler(""), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, body, "app_maintenance_mode 0", "bình thường → 0")

	// Bật maintenance → gauge = 1 (đồng bộ khi admin PUT /config đổi)
	SetMaintenance(true)
	_, body = scrape(t, Handler(""), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, body, "app_maintenance_mode 1", "maintenance → 1")

	// Tắt lại
	SetMaintenance(false)
	_, body = scrape(t, Handler(""), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, body, "app_maintenance_mode 0")
}

func TestStartHealthChecks_SetsServiceUp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	StartHealthChecks(ctx, []HealthCheck{
		{Name: "fake-up", Ping: func(context.Context) error { return nil }},
		{Name: "fake-down", Ping: func(context.Context) error { return errors.New("boom") }},
	})

	_, body := scrape(t, Handler(""), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, body, `service_up{service="fake-up"} 1`, "ping OK → up")
	assert.Contains(t, body, `service_up{service="fake-down"} 0`, "ping lỗi → down")

	// Cancel ctx → goroutine dừng (không panic, không leak — chỉ verify chạy được)
	cancel()
}
