package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/auth"
	"github.com/ducconit/tiktok-live-platform/backend/core/build"
	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/core/i18n"
	"github.com/ducconit/tiktok-live-platform/backend/core/mailer"
	"github.com/ducconit/tiktok-live-platform/backend/core/metrics"
	"github.com/ducconit/tiktok-live-platform/backend/core/otelx"
	"github.com/ducconit/tiktok-live-platform/backend/core/otp"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/core/storage"
	"github.com/ducconit/tiktok-live-platform/backend/internal/apikey"
	"github.com/ducconit/tiktok-live-platform/backend/internal/live"
	"github.com/ducconit/tiktok-live-platform/backend/internal/rbac"
	"github.com/ducconit/tiktok-live-platform/backend/internal/stats"
	"github.com/ducconit/tiktok-live-platform/backend/internal/user"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// readHeaderTimeout — giới hạn đọc header HTTP request (chống slowloris).
const readHeaderTimeout = 10 * time.Second

// Deps — ngoại vi tuỳ chọn (nil = tính năng tương ứng báo lỗi rõ, app vẫn chạy).
type Deps struct {
	OTP     *otp.Service
	Mailer  *mailer.Mailer
	Storage *storage.Manager
}

// Server — gin engine + http.Server, DI toàn bộ dependency ở New.
type Server struct {
	*gin.Engine
	cfg            *config.Manager
	pool           *database.Pool
	rdb            *redis.Client
	cache          *cache.Manager
	build          build.Info
	httpServer     *http.Server
	rateCounter    Counter // dùng chung global + auth group (store theo config)
	authHandler    *auth.Handler
	userHandler    *user.Handler
	rbacHandler    *rbac.Handler
	statsHandler   *stats.Handler
	configHandler  *configHandler
	accountHandler *user.AccountHandler
	adminHandler   *AdminHandler
	apiKeyHandler  *apikey.Handler
	apiKeySvc      *apikey.Service
	liveSvc        *live.Service
	liveHandler    *live.Handler
	openapi        *openapiHandler
	deps           Deps
}

func New(mgr *config.Manager, pool *database.Pool, rdb *redis.Client, cm *cache.Manager, info build.Info, deps Deps) *Server {
	if mgr.Cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{Engine: gin.New(), cfg: mgr, pool: pool, rdb: rdb, cache: cm, build: info, deps: deps}

	// Đồng bộ metric maintenance với dynamic config (đổi runtime qua admin PUT /config
	// → gauge cập nhật ngay — Prometheus/alert nhìn thấy trạng thái thật)
	metrics.SetMaintenance(maintenanceOn(s.cfg.GetDynamic("app.maintenance_mode")))
	s.cfg.OnChange("app.maintenance_mode", func(v any) {
		metrics.SetMaintenance(maintenanceOn(v))
	})

	// Rate limit đọc CONFIG ĐỘNG (server.rate_limit) — đổi runtime qua
	// SetDynamic("server.rate_limit", N) → tự đồng bộ mọi instance qua Redis pub/sub.
	// Store theo server.rate_limit_store: "" → theo cache store (memory|redis);
	// cache=redis → rate limit tự multi-instance (RedisCounter INCR+EXPIRE).
	rateCounter, err := buildRateLimit(mgr.Cfg.Server.RateLimitStore, cm.DefaultStore(), rdb)
	if err != nil {
		// Store không hợp lệ → panic lúc start (fail-fast, không chạy app hỏng)
		panic(fmt.Sprintf("server: %v", err))
	}
	s.rateCounter = rateCounter
	s.Use(
		requestID(),
		loggerMiddleware(),
		gin.Recovery(),
		// Ngôn ngữ request (Accept-Language) — trước mọi middleware dùng message
		i18n.Middleware(),
		// OpenTelemetry trace span per request (noop khi tắt — zero-config)
		otelx.GinMiddleware(),
		// Prometheus metrics — đếm MỌI request (kể cả bị rate-limit/maintenance chặn)
		metrics.GinMiddleware(),
		cors.New(cors.Config{
			AllowOrigins:     []string{"*"}, // dev; tighten ở prod qua env
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"X-Request-ID"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		rateLimit(
			rateCounter,
			"",
			func() int { return s.cfg.Int("server.rate_limit", defaultRateLimit) },
		),
		gzip.Gzip(gzip.DefaultCompression),
		// Maintenance mode (503) — dynamic config app.maintenance_mode;
		// ngoại lệ: /config (client cần đọc trạng thái maintain)
		s.maintenance(),
	)

	// OpenAPI spec tự động — tạo TRƯỚC setupRoutes (route đăng ký method value,
	// nếu tạo sau sẽ capture nil → panic khi request đến).
	// Bật/tắt qua config: OPENAPI_ENABLED=false (hoặc openapi.enabled: false) —
	// không cần sửa code (production có thể tắt để không lộ spec).
	if mgr.Cfg.OpenAPI.Enabled {
		s.openapi = newOpenAPIHandler(mgr.Cfg.App.Name, mgr.Cfg.App.Title, info.Version)
	}

	s.setupRoutes()

	// Snapshot routes SAU setupRoutes — spec luôn khớp routes thật
	if s.openapi != nil {
		s.openapi.snapshotRoutes(s.Engine)
	}

	s.httpServer = &http.Server{
		Addr:              net.JoinHostPort(mgr.Cfg.Server.Host, mgr.Cfg.Server.Port), // mặc định 127.0.0.1:3300
		Handler:           s.Engine,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return s
}

func (s *Server) ListenAndServe() error { return s.httpServer.ListenAndServe() }
func (s *Server) Shutdown(ctx context.Context) error {
	if s.liveSvc != nil {
		s.liveSvc.Close()
	}
	return s.httpServer.Shutdown(ctx)
}

// ---- Middlewares nhỏ, tự viết (tránh dep thừa) ----

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

// loggerMiddleware — log mọi request theo quy tắc log:
//
//	>= 500 → ERROR (kèm lỗi chi tiết — lỗi hệ thống)
//	400-499 → WARN (lỗi nghiệp vụ/client — có code)
//	còn lại  → INFO
func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"request_id", c.GetString("request_id"),
		}

		switch {
		case status >= http.StatusInternalServerError:
			// Lỗi hệ thống — bắt buộc ghi ERROR (quy tắc: mọi lỗi đều phải log)
			if ae := response.ErrorFromContext(c); ae != nil {
				attrs = append(attrs, "code", ae.Code, "error", ae.Message, "details", ae.Details)
			}
			slog.Error("http", attrs...)
		case status >= http.StatusBadRequest:
			if ae := response.ErrorFromContext(c); ae != nil {
				attrs = append(attrs, "code", ae.Code, "error", ae.Message)
			}
			slog.Warn("http", attrs...)
		default:
			slog.Info("http", attrs...)
		}
	}
}
