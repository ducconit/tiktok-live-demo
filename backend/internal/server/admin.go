package server

import (
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/auth"
	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/ctxkey"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/internal/user"
	"github.com/gin-gonic/gin"
)

// AdminHandler — API quản trị riêng: CRUD admin, xoá cache, remote config.
type AdminHandler struct {
	admins *user.AdminService
	cm     *cache.Manager
	cfg    *config.Manager
}

func NewAdminHandler(admins *user.AdminService, cm *cache.Manager, cfg *config.Manager) *AdminHandler {
	return &AdminHandler{admins: admins, cm: cm, cfg: cfg}
}

// RegisterRoutes — group /api/v1/admin (đã RequireAuth; permission per-route).
func (h *AdminHandler) RegisterRoutes(g *gin.RouterGroup) {
	// CRUD admin
	g.GET("/admins", auth.RequirePermission("admins.read"), h.listAdmins)
	g.POST("/admins", auth.RequirePermission("admins.write"), h.createAdmin)
	g.PUT("/admins/:id", auth.RequirePermission("admins.write"), h.updateAdmin)
	g.DELETE("/admins/:id", auth.RequirePermission("admins.write"), h.deleteAdmin)

	// Xoá cache
	g.DELETE("/cache", auth.RequirePermission("cache.delete"), h.clearCache)
	g.GET("/cache", auth.RequirePermission("cache.read"), h.cacheInfo)

	// Remote config — GET /config/dynamic (list keys đổi được), PUT /config (set 1 key)
	g.GET("/config/dynamic", auth.RequirePermission("config.read"), h.listConfig)
	g.PUT("/config", auth.RequirePermission("config.write"), h.setConfig)
}

// cacheInfo — thông tin cache: store đang dùng, danh sách store, prefix.
func (h *AdminHandler) cacheInfo(c *gin.Context) {
	response.OK(c, gin.H{
		"stores":  h.cm.StoreNames(),
		"default": h.cm.DefaultStore(),
		"prefix":  h.cm.Prefix(),
	})
}

// listConfig — toàn bộ dynamic config keys đang hiệu lực (dashboard remote config).
func (h *AdminHandler) listConfig(c *gin.Context) {
	response.OK(c, h.cfg.AllDynamic())
}

func (h *AdminHandler) listAdmins(c *gin.Context) {
	p := response.ParsePageParams(c)
	items, total, err := h.admins.List(c, p.Page, p.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	dto := make([]user.UserDTO, 0, len(items))
	for _, u := range items {
		dto = append(dto, user.ToPublicDTO(u))
	}
	response.OKWithMeta(c, dto, response.BuildMeta(p, int(total)))
}

type createAdminBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func (h *AdminHandler) createAdmin(c *gin.Context) {
	var body createAdminBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" || len(body.Password) < 8 || strings.TrimSpace(body.FullName) == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.register_required"))
		return
	}
	u, err := h.admins.Create(c, body.Email, body.Password, body.FullName)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, user.ToPublicDTO(u))
}

type updateAdminBody struct {
	FullName string `json:"full_name"`
	IsActive *bool  `json:"is_active"`
}

func (h *AdminHandler) updateAdmin(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_id", "error.invalid_id"))
		return
	}
	var body updateAdminBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.invalid_body"))
		return
	}
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}
	u, err := h.admins.Update(c, id, body.FullName, isActive)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user.ToPublicDTO(u))
}

func (h *AdminHandler) deleteAdmin(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_id", "error.invalid_id"))
		return
	}
	if err := user.CannotDeleteSelf(ctxkey.UserID(c), id); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.admins.Delete(c, id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Đã xoá admin"})
}

// clearCache — xoá toàn bộ cache (mọi store).
func (h *AdminHandler) clearCache(c *gin.Context) {
	if err := cache.Clear(h.cm, c); err != nil {
		response.Error(c, apperr.WrapInternal(err))
		return
	}
	response.OK(c, gin.H{"message": "Cache đã được xoá"})
}

// ---- remote config ----

// blocklistedConfigKeys — prefix key KHÔNG được đổi runtime (static/nguy hiểm).
var blocklistedConfigKeys = []string{"database.", "jwt.", "redis.", "minio.", "mail.", "cache."}

type setConfigBody struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func (h *AdminHandler) setConfig(c *gin.Context) {
	var body setConfigBody
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Key) == "" {
		response.Error(c, apperr.New(apperr.KindInvalid, "invalid_body", "error.key_value_required"))
		return
	}
	key := strings.TrimSpace(body.Key)
	for _, prefix := range blocklistedConfigKeys {
		if strings.HasPrefix(key, prefix) {
			response.Error(c, apperr.New(apperr.KindForbidden, "protected_key", "error.protected_key").WithData(map[string]any{"Key": prefix}))
			return
		}
	}
	if key == "server.host" || key == "server.port" {
		response.Error(c, apperr.New(apperr.KindForbidden, "protected_key", "error.protected_server_keys"))
		return
	}
	if err := h.cfg.SetDynamic(key, body.Value); err != nil {
		response.Error(c, apperr.WrapInternal(err))
		return
	}
	response.OK(c, gin.H{"message": "Config đã cập nhật (đồng bộ mọi instance)", "key": key, "value": body.Value})
}
