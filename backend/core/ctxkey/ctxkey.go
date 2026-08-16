// Package ctxkey — context keys + helpers dùng chung cho HTTP middleware/handler
// (tránh import cycle: core/auth và internal/* cùng đọc user id từ context).
package ctxkey

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Key names trong gin context (cùng chuỗi ở middleware set và handler get).
const (
	AuthUserID = "auth_user_id"
	AuthRoles  = "auth_roles"
	AuthPerms  = "auth_perms"

	AuthAPIKeyID   = "auth_api_key_id"
	AuthAPIKeyName = "auth_api_key_name"
	AuthAPIScopes  = "auth_api_scopes"

	LangKey = "lang" // ngôn ngữ request (i18n.Middleware set)
)

// DefaultUserRole — role gán cho tài khoản mới đăng ký (xem seed_refs).
const DefaultUserRole = "user"

// SetUserID — ghi user id (gọi từ auth middleware).
func SetUserID(c *gin.Context, id string) { c.Set(AuthUserID, id) }

// SetClaims — ghi roles + permissions đã verify.
func SetClaims(c *gin.Context, roles, perms []string) {
	c.Set(AuthRoles, roles)
	c.Set(AuthPerms, perms)
}

// SetAPIKey — ghi thông tin API key đã verify (middleware apikey).
func SetAPIKey(c *gin.Context, id string, name string, scopes []string) {
	c.Set(AuthAPIKeyID, id)
	c.Set(AuthAPIKeyName, name)
	c.Set(AuthAPIScopes, scopes)
}

// UserID — user id hiện tại (sau RequireAuth). Rỗng nếu thiếu.
func UserID(c *gin.Context) string {
	id, _ := c.Get(AuthUserID)
	uid, _ := id.(string)
	return uid
}

// UserIDFrom — đọc user id từ context.Context bất kỳ (gin.Context truyền thẳng vào
// service/repo — đọc qua Value). Zero UUID nếu thiếu.
func UserIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(AuthUserID).(string)
	return id
}

// APIScopesFrom — scopes của API key từ context.Context (dùng trong service).
func APIScopesFrom(ctx context.Context) []string {
	scopes, _ := ctx.Value(AuthAPIScopes).([]string)
	return scopes
}

// APIKeyID — api key id hiện tại (sau RequireAPIKey). Rỗng nếu thiếu.
func APIKeyID(c *gin.Context) string {
	id, _ := c.Get(AuthAPIKeyID)
	s, _ := id.(string)
	return s
}

// APIScopes — scopes của API key đang gọi (sau RequireAPIKey).
func APIScopes(c *gin.Context) []string {
	scopes, _ := c.Get(AuthAPIScopes)
	s, _ := scopes.([]string)
	return s
}

// Roles — roles đã verify (sau RequireAuth/RequirePermission).
func Roles(c *gin.Context) []string {
	v, _ := c.Get(AuthRoles)
	r, _ := v.([]string)
	return r
}

// Permissions — permissions đã verify (sau RequireAuth/RequirePermission).
func Permissions(c *gin.Context) []string {
	v, _ := c.Get(AuthPerms)
	p, _ := v.([]string)
	return p
}

// APIKeyName — tên API key đang gọi (sau RequireAPIKey).
func APIKeyName(c *gin.Context) string {
	v, _ := c.Get(AuthAPIKeyName)
	s, _ := v.(string)
	return s
}
