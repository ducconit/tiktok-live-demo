package auth

import (
	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/i18n"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/core/validation"
	"github.com/ducconit/tiktok-live-platform/backend/internal/user"
	"github.com/gin-gonic/gin"
)

// Handler — HTTP layer auth.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterPublicRoutes — group /api/v1/auth (không cần token).
func (h *Handler) RegisterPublicRoutes(g *gin.RouterGroup) {
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)
}

// RegisterAuthedRoutes — group /api/v1/auth (đã RequireAuth).
func (h *Handler) RegisterAuthedRoutes(g *gin.RouterGroup) {
	g.POST("/logout", h.logout)
	g.GET("/me", h.me)
}

type loginBody struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

func (h *Handler) login(c *gin.Context) {
	var body loginBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	if fields := validation.ValidateStruct(&body); fields != nil {
		response.Error(c, apperr.Validation(validation.FieldsMap(fields, i18n.Lang(c))))
		return
	}
	tokens, err := h.svc.Login(c, body.Email, body.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tokens)
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *Handler) refresh(c *gin.Context) {
	var body refreshBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	tokens, err := h.svc.Refresh(c, body.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tokens)
}

func (h *Handler) logout(c *gin.Context) {
	var body refreshBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	if err := h.svc.Logout(c, body.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) me(c *gin.Context) {
	u, err := h.svc.Me(c, CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user.ToDTO(u))
}
