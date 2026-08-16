package live

import (
	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
)

// Handler — HTTP layer live tracker (/api/v1/public/live — không cần token).
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes — group /api/v1/public/live (public — end-user frontend).
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/:username", h.preview)
	g.POST("/:username/connect", h.connect)
	g.POST("/:username/disconnect", h.disconnect)
}

// preview — GET /live/:username → { live, title, userCount, owner }.
func (h *Handler) preview(c *gin.Context) {
	username := normalizeUsername(c.Param("username"))
	if username == "" {
		response.Error(c, apperr.BadRequest("invalid_username", "error.live_username_required"))
		return
	}
	data, err := h.svc.Preview(username)
	if err != nil {
		response.Error(c, apperr.New(apperr.KindInternal, "500", "error.live_preview"))
		return
	}
	response.OK(c, data)
}

// connect — POST /live/:username/connect → { connected, roomId, roomInfo }.
func (h *Handler) connect(c *gin.Context) {
	username := normalizeUsername(c.Param("username"))
	if username == "" {
		response.Error(c, apperr.BadRequest("invalid_username", "error.live_username_required"))
		return
	}
	data, err := h.svc.Connect(username)
	if err != nil {
		response.Error(c, classifyLiveError(err))
		return
	}
	response.OK(c, data)
}

// disconnect — POST /live/:username/disconnect → { ok }.
func (h *Handler) disconnect(c *gin.Context) {
	username := normalizeUsername(c.Param("username"))
	if username == "" {
		response.Error(c, apperr.BadRequest("invalid_username", "error.live_username_required"))
		return
	}
	h.svc.Disconnect(username)
	response.OK(c, gin.H{"ok": true})
}
