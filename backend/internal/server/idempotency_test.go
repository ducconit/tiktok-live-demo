package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/core/retry"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Integration test idempotency — cần Postgres thật (docker compose, forward 5433).
func testDB(t *testing.T) *database.Pool {
	t.Helper()
	dsn := "postgres://app:app_password_dev@localhost:5433/tiktok_live_platform?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := database.NewPool(ctx, config.DatabaseConfig{URL: dsn})
	if err != nil {
		t.Skipf("postgres không khả dụng (%v) — skip", err)
	}
	return pool
}

func TestIdempotency_ReplaySameResponse(t *testing.T) {
	if retry.SkipRetry() {
		// CI không có DB — skip luôn (thay vì fail nhanh)
		t.Skip("CI/test không có postgres — skip integration")
	}
	pool := testDB(t)
	defer pool.Close()
	ctx := context.Background()

	h := newIdempotencyHandler(pool)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/test", h.Middleware(), func(c *gin.Context) {
		// Handler "đếm số lần thực thi" — body chứa số lần
		var body map[string]any
		_ = json.Unmarshal([]byte(c.PostForm("x")), &body)
		_ = body
		c.JSON(http.StatusOK, gin.H{"count": 1, "echo": c.GetHeader("X-Echo")})
	})

	do := func(key, echo string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("x=1"))
		req.Header.Set("Idempotency-Key", key)
		req.Header.Set("X-Echo", echo)
		r.ServeHTTP(w, req)
		return w
	}

	// Lần 1: thực thi, lưu response (echo=AAA)
	w1 := do("key-1", "AAA")
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), `"echo":"AAA"`)

	// Lần 2: cùng key → REPLAY response đầu (echo vẫn AAA dù gửi BBB)
	w2 := do("key-1", "BBB")
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), `"echo":"AAA"`, "phải replay response cũ, không thực thi lại")

	// Key khác → thực thi mới (echo=CCC)
	w3 := do("key-2", "CCC")
	assert.Contains(t, w3.Body.String(), `"echo":"CCC"`)

	// Cleanup key test
	_, _ = pool.Write().Exec(ctx, "DELETE FROM idempotency_keys WHERE key LIKE 'test-%' OR path = '/test'")
}

func TestIdemKeyHash_ScopedByEndpoint(t *testing.T) {
	h1 := idemKeyHash("POST", "/a", "k")
	h2 := idemKeyHash("POST", "/b", "k")
	h3 := idemKeyHash("GET", "/a", "k")
	assert.NotEqual(t, h1, h2, "path khác → key khác")
	assert.NotEqual(t, h1, h3, "method khác → key khác")
	assert.Equal(t, idemKeyHash("POST", "/a", "k"), h1, "cùng method+path+key → cùng hash")
	assert.Equal(t, 64, len(h1), "sha256 hex 64 ký tự")
}
