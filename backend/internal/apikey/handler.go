package apikey

import (
	"strings"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/auth"
	"github.com/ducconit/tiktok-live-platform/backend/core/ctxkey"
	"github.com/ducconit/tiktok-live-platform/backend/core/i18n"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
)

// Handler — HTTP layer quản trị API key (namespace admin).
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes — group /api/v1/admin (đã RequireAuth); permission per-route.
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/api-keys", auth.RequirePermission("api_keys.read"), h.list)
	g.GET("/api-keys/:id", auth.RequirePermission("api_keys.read"), h.get)
	g.POST("/api-keys", auth.RequirePermission("api_keys.write"), h.create)
	g.PUT("/api-keys/:id", auth.RequirePermission("api_keys.write"), h.update)
	g.DELETE("/api-keys/:id", auth.RequirePermission("api_keys.write"), h.revoke)
	g.POST("/api-keys/:id/rotate", auth.RequirePermission("api_keys.write"), h.rotate)
}

func (h *Handler) list(c *gin.Context) {
	p := response.ParsePageParams(c)
	items, total, err := h.svc.List(c, c.Query("q"), p.Page, p.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKWithMeta(c, ToDTOs(items), response.BuildMeta(p, int(total)))
}

func (h *Handler) get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	rec, err := h.svc.Get(c, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ToDTO(rec))
}

type createBody struct {
	Name      string     `json:"name" validate:"required,min=3,max=64"`
	Scopes    []string   `json:"scopes" validate:"max=10"`
	ExpiresAt *time.Time `json:"expires_at"` // RFC3339; NULL = không hết hạn
}

func (h *Handler) create(c *gin.Context) {
	var body createBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || len(body.Name) < 3 {
		response.Error(c, apperr.Validation(map[string]string{"name": i18n.T(i18n.Lang(c), "error.apikey_name_min", nil)}))
		return
	}
	created, err := h.svc.Create(c, CreateParams{
		Name:      body.Name,
		Scopes:    body.Scopes,
		ExpiresAt: body.ExpiresAt,
		CreatedBy: ctxkey.UserID(c),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, CreatedDTO{Key: created.Key, DTO: ToDTO(created.Record)})
}

type updateBody struct {
	Name      *string    `json:"name"`
	Scopes    *[]string  `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"` // NULL = bỏ hết hạn
	IsActive  *bool      `json:"is_active"`
}

func (h *Handler) update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	var body updateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}

	// Lấy bản ghi hiện tại → merge field (tránh ghi đè nhầm)
	rec, err := h.svc.Get(c, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	name := rec.Name
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
		if name == "" {
			response.Error(c, apperr.Validation(map[string]string{"name": i18n.T(i18n.Lang(c), "error.apikey_name_required", nil)}))
			return
		}
	}
	scopes := rec.Scopes
	if body.Scopes != nil {
		scopes = *body.Scopes
		if scopes == nil {
			scopes = []string{} // tránh NULL vào column NOT NULL
		}
	}
	expiresAt := rec.ExpiresAt
	if body.ExpiresAt != nil {
		expiresAt = body.ExpiresAt
	}
	isActive := rec.IsActive
	if body.IsActive != nil {
		isActive = *body.IsActive
	}

	updated, err := h.svc.Update(c, id, UpdateParams{
		Name:      name,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		IsActive:  isActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ToDTO(updated))
}

func (h *Handler) revoke(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	if err := h.svc.Revoke(c, id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "error.api_key_revoked"})
}

func (h *Handler) rotate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	created, err := h.svc.Rotate(c, id, ctxkey.UserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, CreatedDTO{Key: created.Key, DTO: ToDTO(created.Record)})
}
