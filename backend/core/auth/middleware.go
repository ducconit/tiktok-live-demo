package auth

import (
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/ctxkey"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
)

// RequireAuth — verify Bearer access token, nhét claims vào context.
func RequireAuth(cfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			response.Error(c, apperr.Unauthorized("unauthorized", "Thiếu token xác thực"))
			c.Abort()
			return
		}

		claims, err := VerifyAccessToken(token, cfg.Secret)
		if err != nil {
			response.Error(c, apperr.Unauthorized("invalid_token", "error.invalid_token"))
			c.Abort()
			return
		}
		ctxkey.SetUserID(c, claims.UserID)
		ctxkey.SetClaims(c, claims.Roles, claims.Perms)
		c.Next()
	}
}

// RequirePermission — check permission (dùng sau RequireAuth).
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms := c.GetStringSlice(ctxkey.AuthPerms)
		for _, p := range perms {
			if p == perm {
				c.Next()
				return
			}
		}
		response.Error(c, apperr.Forbidden("forbidden", "Không có quyền thực hiện thao tác này"))
		c.Abort()
	}
}

// CurrentUserID — user id từ context (sau RequireAuth).
func CurrentUserID(c *gin.Context) string {
	return ctxkey.UserID(c)
}
