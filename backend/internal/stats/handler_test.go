package stats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/core/retry"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGet_NeedsDB — SKIP: cần Postgres thật + dữ liệu seed (compose dev).
// Case được biết (aggregate stats: total/active/recent + phân bố role, cache 30s)
// nhưng chưa kiểm chứng trong CI.
func TestGet_NeedsDB(t *testing.T) {
	if retry.SkipRetry() {
		t.Skip("CI/test không có postgres — skip (case: /stats aggregate + cache)")
	}
	dsn := "postgres://app:app_password_dev@localhost:5433/tiktok_live_platform?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := database.NewPool(ctx, config.DatabaseConfig{URL: dsn})
	if err != nil {
		t.Skipf("postgres không khả dụng (%v) — skip (case: /stats)", err)
	}
	defer pool.Close()

	cm, err := cache.New(cache.Config{Store: "memory", DefaultTTL: time.Minute}, nil)
	require.NoError(t, err)
	defer cm.Close()

	h := NewHandler(pool, cm)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/stats", nil)

	h.get(c)
	assert.Equal(t, http.StatusOK, w.Code, "có dữ liệu seed → /stats trả 200")
	assert.Contains(t, w.Body.String(), "total_users")

	// Lần 2 phải trúng cache (không lỗi, cùng shape)
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/stats", nil)
	h.get(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, w.Body.String(), w2.Body.String(), "cache 30s → response giống hệt")
}
