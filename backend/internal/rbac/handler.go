package rbac

import (
	"net/http"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/i18n"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/core/validation"
	"github.com/gin-gonic/gin"
)

// Handler — HTTP layer RBAC.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes — gắn routes vào group đã RequireAuth; permission check per-route.
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/roles", h.listRoles)                           // roles.read
	g.POST("/roles", h.createRole)                         // roles.write
	g.PUT("/roles/:id", h.updateRole)                      // roles.write
	g.DELETE("/roles/:id", h.deleteRole)                   // roles.write
	g.GET("/roles/:id/permissions", h.rolePermissions)     // roles.read
	g.PUT("/roles/:id/permissions", h.setRolePermissions)  // roles.write
	g.GET("/permissions", h.listPermissions)               // roles.read
	g.POST("/users/:id/roles", h.assignUserRole)           // users.write
	g.DELETE("/users/:id/roles/:roleId", h.removeUserRole) // users.write
}

type roleBody struct {
	ID          string `json:"id" validate:"required,min=2,max=64"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

func (h *Handler) listRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, roles)
}

func (h *Handler) createRole(c *gin.Context) {
	var body roleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	if fields := validation.ValidateStruct(&body); fields != nil {
		response.Error(c, apperr.Validation(validation.FieldsMap(fields, i18n.Lang(c))))
		return
	}
	role, err := h.svc.CreateRole(c, RoleInput(body))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, role)
}

func (h *Handler) updateRole(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	var body roleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	role, err := h.svc.UpdateRole(c, id, RoleInput(body))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, role)
}

func (h *Handler) deleteRole(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	if err := h.svc.DeleteRole(c, id); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) rolePermissions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	perms, err := h.svc.repo.ListRolePermissions(c, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, perms)
}

func (h *Handler) setRolePermissions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	var body struct {
		PermissionIDs []string `json:"permission_ids" validate:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	if err := h.svc.SetRolePermissions(c, id, body.PermissionIDs); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) listPermissions(c *gin.Context) {
	perms, err := h.svc.repo.ListPermissions(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, perms)
}

func (h *Handler) assignUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	var body struct {
		RoleID string `json:"role_id" validate:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperr.BadRequest("invalid_json", "error.invalid_body"))
		return
	}
	if err := h.svc.AssignUserRole(c, userID, body.RoleID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) removeUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	roleID := c.Param("roleId") // varchar — không parse uuid
	if roleID == "" {
		response.Error(c, apperr.BadRequest("invalid_id", "error.invalid_id"))
		return
	}
	if err := h.svc.RemoveUserRole(c, userID, roleID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

var _ = http.StatusOK // giữ import net/http nếu cần sau
