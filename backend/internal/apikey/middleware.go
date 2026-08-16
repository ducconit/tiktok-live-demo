package apikey

import (
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/ctxkey"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
)

// RequireAPIKey — verify API key cho namespace /integrations (server-server).
//
// Nhận key từ header (ưu tiên):
//
//	Authorization: Bearer gvs_live_...
//	X-API-Key: gvs_live_...
//
// Sau khi verify: set ctxkey (api key id, name, scopes) cho handler dùng.
func RequireAPIKey(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractKey(c)
		if key == "" {
			response.Error(c, apperr.Unauthorized("missing_api_key", "Thiếu API key (Authorization: Bearer hoặc X-API-Key)"))
			c.Abort()
			return
		}

		info, err := svc.Lookup(c, key)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		ctxkey.SetAPIKey(c, info.ID, info.Name, info.Scopes)
		c.Next()
	}
}

// RequireAPIScope — check scope của API key (dùng sau RequireAPIKey).
// Scope rỗng ("") không được phép — phải khai báo tường minh khi tạo key.
func RequireAPIScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, s := range ctxkey.APIScopes(c) {
			if s == scope {
				c.Next()
				return
			}
		}
		response.Error(c, apperr.Forbidden("api_key_scope", "error.api_key_scope").WithData(map[string]any{"Scope": scope}))
		c.Abort()
	}
}

// extractKey — đọc key từ Authorization: Bearer hoặc X-API-Key.
func extractKey(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); h != "" {
		if key, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(key)
		}
	}
	return strings.TrimSpace(c.GetHeader("X-API-Key"))
}
