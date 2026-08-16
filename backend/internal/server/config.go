package server

import (
	"github.com/ducconit/tiktok-live-platform/backend/core/build"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
)

// configHandler — GET <namespace>/config: thông tin app mà client cần biết
// (version, env, build hash/date, maintenance mode...). Không cần auth
// (integrations namespace vẫn bị middleware auth chặn trước).
type configHandler struct {
	cfg   *config.Manager
	build build.Info
}

func newConfigHandler(cfg *config.Manager, info build.Info) *configHandler {
	return &configHandler{cfg: cfg, build: info}
}

func (h *configHandler) get(c *gin.Context) {
	response.OK(c, gin.H{
		"app":              h.cfg.Cfg.App.Name,
		"title":            h.cfg.Cfg.App.Title,
		"environment":      h.cfg.Cfg.App.Env,
		"version":          h.build.Version,
		"build_hash":       h.build.BuildHash,
		"build_date":       h.build.BuildDate,
		"maintenance_mode": maintenanceOn(h.cfg.GetDynamic("app.maintenance_mode")),
	})
}
