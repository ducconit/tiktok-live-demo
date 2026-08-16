package stats

import (
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/gin-gonic/gin"
)

// DayCount — 1 điểm dữ liệu signups/ngày.
type DayCount struct {
	Day   time.Time `json:"day"`
	Count int64     `json:"count"`
}

// RoleCount — phân bố user theo role.
type RoleCount struct {
	Role  string `json:"role"` // role id (varchar — admin tự điền)
	Count int64  `json:"count"`
}

// Stats — payload /api/v1/stats.
type Stats struct {
	TotalUsers       int64       `json:"total_users"`
	ActiveUsers      int64       `json:"active_users"`
	RecentUsers      int64       `json:"recent_users"`
	SignupsByDay     []DayCount  `json:"signups_by_day"`
	RoleDistribution []RoleCount `json:"role_distribution"`
}

// Handler — HTTP layer stats (chỉ đọc → dùng replica pool; cache 30s qua core/cache).
type Handler struct {
	q     *db.Queries
	cache *cache.Manager
}

func NewHandler(p *database.Pool, cm *cache.Manager) *Handler {
	return &Handler{q: db.New(p.Read()), cache: cm}
}

// RegisterRoutes — gắn route vào group đã RequireAuth.
func (h *Handler) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/stats", h.get)
}

func (h *Handler) get(c *gin.Context) {
	ctx := c

	// Cache 30s (ví dụ dùng core/cache — store theo CACHE_STORE config)
	const (
		cacheKey = "stats:overview"
		cacheTTL = 30 * time.Second
	)
	if h.cache != nil {
		if v, err := cache.Get[Stats](h.cache, ctx, cacheKey); err == nil {
			response.OK(c, v)
			return
		}
	}

	total, err := h.q.CountUsersTotal(ctx)
	if err != nil {
		response.Error(c, apperr.WrapInternal(err))
		return
	}
	active, err := h.q.CountUsersActive(ctx)
	if err != nil {
		response.Error(c, apperr.WrapInternal(err))
		return
	}
	recentRows, err := h.q.CountUsersRecent(ctx)
	if err != nil {
		response.Error(c, apperr.WrapInternal(err))
		return
	}
	var recent int64
	for _, r := range recentRows {
		recent += r.Count
	}

	days := make([]DayCount, 0, len(recentRows))
	for _, s := range recentRows {
		days = append(days, DayCount{Day: s.Day, Count: s.Count})
	}

	distribution, err := h.q.RoleDistribution(ctx)
	if err != nil {
		response.Error(c, apperr.WrapInternal(err))
		return
	}
	roles := make([]RoleCount, 0, len(distribution))
	for _, r := range distribution {
		roles = append(roles, RoleCount{Role: r.ID, Count: r.Count})
	}

	out := Stats{
		TotalUsers:       total,
		ActiveUsers:      active,
		RecentUsers:      recent,
		SignupsByDay:     days,
		RoleDistribution: roles,
	}
	if h.cache != nil {
		_ = cache.SetWithTTL(h.cache, ctx, cacheKey, out, cacheTTL)
	}
	response.OK(c, out)
}
