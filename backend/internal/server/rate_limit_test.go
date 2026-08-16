package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCounter_Incr(t *testing.T) {
	mc := newMemoryCounter()
	ctx := context.Background()

	n1, err := mc.Incr(ctx, "k", time.Second)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n1)

	n2, _ := mc.Incr(ctx, "k", time.Second)
	assert.Equal(t, int64(2), n2)

	// Key khác đếm riêng
	n3, _ := mc.Incr(ctx, "k2", time.Second)
	assert.Equal(t, int64(1), n3)

	// Reset xoá bộ đếm
	mc.Reset(ctx, "k")
	n4, _ := mc.Incr(ctx, "k", time.Second)
	assert.Equal(t, int64(1), n4)
}

func TestMemoryCounter_Expiry(t *testing.T) {
	mc := newMemoryCounter()
	ctx := context.Background()

	_, _ = mc.Incr(ctx, "k", 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	n, _ := mc.Incr(ctx, "k", time.Second)
	assert.Equal(t, int64(1), n, "hết hạn TTL → đếm lại từ 1")
}

func TestRateLimit_BlockOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mc := newMemoryCounter()

	// Limit 3 request / giây
	h := rateLimit(mc, "test", func() int { return 3 })

	var lastStatus int
	for i := 0; i < 4; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
		h(c)
		lastStatus = w.Code
	}
	assert.Equal(t, http.StatusTooManyRequests, lastStatus, "request thứ 4 vượt limit 3 → 429")
}

func TestRateLimit_UnderLimitOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mc := newMemoryCounter()
	h := rateLimit(mc, "test", func() int { return 5 })

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
		h(c)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestClientIP_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// CF-Connecting-IP ưu tiên
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c.Request.RemoteAddr = "10.0.0.5:1234"
	c.Request.Header.Set("CF-Connecting-IP", "203.0.113.7")
	c.Request.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.1")
	assert.Equal(t, "203.0.113.7", clientIP(c))

	// X-Forwarded-For → lấy IP đầu tiên
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c2.Request.RemoteAddr = "10.0.0.5:1234"
	c2.Request.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.1")
	assert.Equal(t, "198.51.100.1", clientIP(c2))

	// Không có header → RemoteAddr (bỏ port)
	c3, _ := gin.CreateTestContext(httptest.NewRecorder())
	c3.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	c3.Request.RemoteAddr = "203.0.113.9:5678"
	assert.Equal(t, "203.0.113.9", clientIP(c3))
}

func TestBuildRateLimit_StoreSelection(t *testing.T) {
	// "" → theo cache store
	mem, err := buildRateLimit("", "memory", nil)
	require.NoError(t, err)
	assert.IsType(t, &memoryCounter{}, mem)

	// "memory" ép
	mem2, err := buildRateLimit("memory", "redis", nil)
	require.NoError(t, err)
	assert.IsType(t, &memoryCounter{}, mem2)

	// "redis" nhưng thiếu client → lỗi
	_, err = buildRateLimit("redis", "memory", nil)
	require.Error(t, err)

	// store không hợp lệ → lỗi
	_, err = buildRateLimit("bogus", "memory", nil)
	require.Error(t, err)
}
