package ctxkey

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSetAndGet_UserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	id := uuid.NewString()
	SetUserID(c, id)
	assert.Equal(t, id, UserID(c))
	assert.Equal(t, id, UserIDFrom(c)) // gin.Context implement context.Context
}

func TestSetAndGet_Claims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	SetClaims(c, []string{"admin"}, []string{"users.read", "users.write"})
	assert.Equal(t, []string{"admin"}, Roles(c))
	assert.Equal(t, []string{"users.read", "users.write"}, Permissions(c))
}

func TestSetAndGet_APIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	id := uuid.NewString()
	SetAPIKey(c, id, "my-key", []string{"config.read"})
	assert.Equal(t, id, APIKeyID(c))
	assert.Equal(t, "my-key", APIKeyName(c))
	assert.Equal(t, []string{"config.read"}, APIScopesFrom(c))
}

func TestUserIDFrom_PlainContext(t *testing.T) {
	// context.Context thường (không phải gin) — context.WithValue
	// (string key giống production: gin c.Set dùng string key — SA1029 cố ý)
	ctx := context.WithValue(context.Background(), AuthUserID, uuid.NewString()) //nolint:staticcheck // string key như gin
	got := UserIDFrom(ctx)
	assert.NotEqual(t, uuid.Nil, got)
}

func TestHelpers_ZeroWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	assert.Empty(t, UserID(c), "chưa set → rỗng")
	assert.Empty(t, UserIDFrom(c))
	assert.Nil(t, Roles(c))
	assert.Nil(t, Permissions(c))
	assert.Empty(t, APIKeyID(c))
	assert.Empty(t, APIScopesFrom(c))
	assert.Empty(t, APIKeyName(c))
}

func TestUserIDFrom_TypeMismatch_ReturnsEmpty(t *testing.T) {
	ctx := context.WithValue(context.Background(), AuthUserID, 12345) //nolint:staticcheck // string key như gin
	assert.Empty(t, UserIDFrom(ctx))
}

func TestAPIScopesFrom_TypeMismatch_ReturnsNil(t *testing.T) {
	ctx := context.WithValue(context.Background(), AuthAPIScopes, "scopes") //nolint:staticcheck // string key như gin
	assert.Nil(t, APIScopesFrom(ctx))
}

func TestDefaultUserRole(t *testing.T) {
	assert.Equal(t, "user", DefaultUserRole)
}
