package server

import (
	"net/http"

	"github.com/ducconit/tiktok-live-platform/backend/core/auth"
	"github.com/ducconit/tiktok-live-platform/backend/core/storage"
	"github.com/ducconit/tiktok-live-platform/backend/internal/apikey"
	"github.com/ducconit/tiktok-live-platform/backend/internal/live"
	"github.com/ducconit/tiktok-live-platform/backend/internal/rbac"
	"github.com/ducconit/tiktok-live-platform/backend/internal/stats"
	"github.com/ducconit/tiktok-live-platform/backend/internal/user"
)

// buildServices — DI toàn bộ: repo → service → handler.
// Thêm domain mới: tạo repo/service/handler rồi wire vào đây.
func (s *Server) buildServices() {
	userRepo := user.NewRepo(s.pool)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)

	rbacRepo := rbac.NewRepo(s.pool)
	rbacSvc := rbac.NewService(rbacRepo)
	rbacHandler := rbac.NewHandler(rbacSvc)

	authSvc := auth.NewService(s.cfg.Cfg.JWT, userRepo, rbacRepo, auth.NewTokenStore(s.pool))
	authHandler := auth.NewHandler(authSvc)

	apiKeyRepo := apikey.NewRepo(s.pool)
	apiKeySvc := apikey.NewService(apiKeyRepo, s.pool, s.cache, s.cfg.Cfg.App.Env) // pool = TxRunner
	apiKeyHandler := apikey.NewHandler(apiKeySvc)

	s.userHandler = userHandler
	s.rbacHandler = rbacHandler
	s.authHandler = authHandler
	s.apiKeyHandler = apiKeyHandler
	s.apiKeySvc = apiKeySvc
	s.statsHandler = stats.NewHandler(s.pool, s.cache)
	s.configHandler = newConfigHandler(s.cfg, s.build)
	s.accountHandler = user.NewAccountHandler(user.NewAccountService(userRepo, s.pool, s.deps.OTP, s.deps.Mailer, s.deps.Storage)) // pool = TxRunner
	s.adminHandler = NewAdminHandler(user.NewAdminService(userRepo, s.pool), s.cache, s.cfg)                                       // pool = TxRunner

	// Live tracker (TikTok + Sockudo) — /api/v1/public/live
	s.liveSvc = live.NewService(s.cfg.Cfg.Live)
	s.liveHandler = live.NewHandler(s.liveSvc)
}

// setupRoutes — đăng ký toàn bộ routes.
//
// Cấu trúc namespace:
//
//	/api/v1/public/*        — API cho ứng dụng client (mobile, web, bên thứ 3) — public
//	/api/v1/admin/*         — API cho admin dashboard (đặc thù riêng)
//	/api/v1/integrations/*  — API server-server — TẤT CẢ cần auth (JWT/api-key)
//
// Mọi namespace có GET /config (version, env, build, maintenance) — không cần auth,
// TRỪ integrations (middleware auth chặn trước).
func (s *Server) setupRoutes() {
	s.buildServices()

	// Serve file public local disk (Laravel: public/storage) — route /storage/*
	// CHỈ khi disk "public" là local có URL prefix — private disk không serve.
	if s.deps.Storage != nil {
		if pub, err := s.deps.Storage.Disk("public"); err == nil {
			if ld, ok := pub.(*storage.LocalDisk); ok && ld.Root() != "" {
				s.StaticFS("/storage", http.Dir(ld.Root()))
			}
		}
	}

	api := s.Group("/api")
	v1 := api.Group("/v1")

	// ---- /api/v1/public — ứng dụng client (mobile, web, bên thứ 3) ----
	pub := v1.Group("/public")
	// Auth group: rate limit RIÊNG thấp (server.auth_rate_limit, mặc định 10/s)
	// — chống brute force login/OTP/forgot (không dùng limit chung 100/s)
	authPub := pub.Group("/auth", rateLimit(
		s.rateCounter,
		"auth",
		func() int { return s.cfg.Int("server.auth_rate_limit", defaultAuthRateLimit) },
	))
	s.authHandler.RegisterPublicRoutes(authPub)    // login, refresh
	s.accountHandler.RegisterPublicRoutes(authPub) // register, verify, forgot, reset, resend-otp
	pub.GET("/config", s.configHandler.get)

	// ---- Live tracker (TikTok realtime) — end-user frontend ----
	s.liveHandler.RegisterRoutes(pub.Group("/live"))

	// Tài khoản của chính mình (cần token)
	pubAuthed := pub.Group("")
	pubAuthed.Use(auth.RequireAuth(s.cfg.Cfg.JWT))
	s.authHandler.RegisterAuthedRoutes(pubAuthed.Group("/auth")) // logout
	s.accountHandler.RegisterProfileRoutes(pubAuthed)            // me, avatar, change-password

	// ---- /api/v1/admin — admin dashboard ----
	adm := v1.Group("/admin")
	authAdm := adm.Group("/auth", rateLimit(
		s.rateCounter,
		"auth",
		func() int { return s.cfg.Int("server.auth_rate_limit", defaultAuthRateLimit) },
	))
	s.authHandler.RegisterPublicRoutes(authAdm) // login, refresh (không token)
	adm.GET("/config", s.configHandler.get)

	// Cần token
	admAuthed := adm.Group("")
	admAuthed.Use(auth.RequireAuth(s.cfg.Cfg.JWT))
	s.authHandler.RegisterAuthedRoutes(admAuthed.Group("/auth")) // logout, me
	s.userHandler.RegisterRoutes(admAuthed)
	s.rbacHandler.RegisterRoutes(admAuthed)
	s.statsHandler.RegisterRoutes(admAuthed)
	s.adminHandler.RegisterRoutes(admAuthed)  // admins, cache, remote config
	s.apiKeyHandler.RegisterRoutes(admAuthed) // API keys (integrations)

	// ---- /api/v1/integrations — server-server, TẤT CẢ cần auth (API key) ----
	// Idempotency: header Idempotency-Key → replay response cũ (chống xử lý 2 lần khi retry)
	intg := v1.Group("/integrations")
	intg.Use(
		apikey.RequireAPIKey(s.apiKeySvc),
		newIdempotencyHandler(s.pool).Middleware(),
	)
	intg.GET("/config", s.configHandler.get)

	// ---- OpenAPI spec tự động (public — không lộ secret) ----
	// Bật/tắt qua config openapi.enabled / OPENAPI_ENABLED (mặc định BẬT).
	if s.cfg.Cfg.OpenAPI.Enabled {
		v1.GET("/openapi.json", s.openapi.get)
	}
}
