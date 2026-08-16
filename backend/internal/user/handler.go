package user

import (
	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/i18n"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/core/validation"
	"github.com/gin-gonic/gin"
)

// Handler — HTTP layer user.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes — group /api/v1 (đã RequireAuth); permission check per-route.
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/users", h.list)                                // users.read
	g.POST("/users", h.create)                             // users.write
	g.GET("/users/:id", h.get)                             // users.read
	g.PUT("/users/:id", h.update)                          // users.write
	g.DELETE("/users/:id", h.delete)                       // users.write
	g.POST("/users/:id/change-password", h.changePassword) // users.write (hoặc chính user)
}

type createBody struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"max=128"`
}

func (h *Handler) list(c *gin.Context) {
	p := response.ParsePageParams(c)
	q := c.Query("q")
	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := v == "true" || v == "1"
		isActive = &b
	}
	users, total, err := h.svc.List(c, q, isActive, p.Page, p.PageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OKWithMeta(c, ToDTOs(users), response.BuildMeta(p, int(total)))
}

func (h *Handler) create(c *gin.Context) {
	var body createBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	if fields := validation.ValidateStruct(&body); fields != nil {
		response.Error(c, apperr.Validation(validation.FieldsMap(fields, i18n.Lang(c))))
		return
	}
	u, err := h.svc.Create(c, CreateParams(body))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, ToDTO(u))
}

func (h *Handler) get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	u, err := h.svc.GetByID(c, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ToDTO(u))
}

type updateBody struct {
	FullName string `json:"full_name" validate:"max=128"`
	IsActive *bool  `json:"is_active"`
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
	isActive := true
	if body.IsActive != nil {
		isActive = *body.IsActive
	}
	u, err := h.svc.Update(c, id, UpdateParams{
		FullName: body.FullName,
		IsActive: isActive,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, ToDTO(u))
}

func (h *Handler) delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	if err := h.svc.Delete(c, id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

type changePasswordBody struct {
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

func (h *Handler) changePassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	var body changePasswordBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	if fields := validation.ValidateStruct(&body); fields != nil {
		response.Error(c, apperr.Validation(validation.FieldsMap(fields, i18n.Lang(c))))
		return
	}
	if err := h.svc.ChangePassword(c, id, body.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}
