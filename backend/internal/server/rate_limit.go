package server

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Cấu hình mặc định middleware (không magic number).
const (
	rateWindow           = time.Second     // cửa sổ đếm: 1 giây
	rateLimitKeyTTL      = 2 * time.Second // TTL key redis (window + chùng)
	defaultRateLimit     = 100             // mặc định khi config 0/âm
	defaultAuthRateLimit = 10              // auth group: chống brute force login/OTP
)

// Counter — đếm số request trong cửa sổ (atomic, dùng chung giữa instance).
type Counter interface {
	// Incr — tăng bộ đếm cho key, trả giá trị sau khi tăng.
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	// Reset — xoá key (khi đổi limit runtime).
	Reset(ctx context.Context, key string)
}

// ---- MemoryCounter: fixed-window in-memory (1 instance, zero-config) ----

type memEntry struct {
	count int64
	exp   time.Time
}

type memoryCounter struct {
	mu sync.Mutex
	m  map[string]*memEntry
}

func newMemoryCounter() *memoryCounter {
	return &memoryCounter{m: make(map[string]*memEntry)}
}

func (mc *memoryCounter) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	now := time.Now()
	e, ok := mc.m[key]
	if !ok || now.After(e.exp) {
		e = &memEntry{count: 0, exp: now.Add(ttl)}
		mc.m[key] = e
	}
	e.count++
	return e.count, nil
}

func (mc *memoryCounter) Reset(_ context.Context, key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	delete(mc.m, key)
}

// ---- RedisCounter: fixed-window trên Redis (INCR+EXPIRE — atomic, multi-instance) ----
// Dùng CHUNG go-redis client với cache → khi cache.store=redis, rate limit
// tự multi-instance mà không cần config riêng.

type redisCounter struct {
	rdb *redis.Client
}

func newRedisCounter(rdb *redis.Client) *redisCounter {
	return &redisCounter{rdb: rdb}
}

func (rc *redisCounter) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := rc.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("ratelimit: redis incr %s: %w", key, err)
	}
	if n == 1 {
		rc.rdb.Expire(ctx, key, ttl)
	}
	return n, nil
}

func (rc *redisCounter) Reset(ctx context.Context, key string) {
	rc.rdb.Del(ctx, key)
}

// ---- Limiter ----

type rateLimiter struct {
	counter Counter
	prefix  string
}

// rateLimit — fixed-window per client; limit đọc từ getter MỖI REQUEST
// (dynamic config — đổi runtime qua SetDynamic → áp dụng ngay).
func rateLimit(counter Counter, prefix string, getLimit func() int) gin.HandlerFunc {
	rl := &rateLimiter{counter: counter, prefix: prefix}
	return func(c *gin.Context) {
		limit := getLimit()
		if limit <= 0 {
			limit = defaultRateLimit
		}

		key := rl.key(c)
		n, err := rl.counter.Incr(c.Request.Context(), key, rateLimitKeyTTL)
		if err != nil {
			// Counter lỗi (redis chết...) → KHÔNG chặn request, chỉ log —
			// fail-open an toàn hơn fail-closed cho rate limit.
			c.Next()
			return
		}
		if n > int64(limit) {
			response.Error(c, apperr.New(apperr.KindTooManyRequests, "429", "error.too_many_requests"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// key — client IP theo chuẩn sau proxy + scope prefix.
func (rl *rateLimiter) key(c *gin.Context) string {
	return fmt.Sprintf("rate:%s:%s:%d", rl.prefix, clientIP(c), time.Now().Unix())
}

// clientIP — IP thật sau proxy/LB/CDN:
//
//  1. CF-Connecting-IP (Cloudflare)   2. X-Forwarded-For (first)
//  3. RemoteAddr (trực tiếp)
//
// KHÔNG tin header nếu request từ mạng nội bộ không qua proxy — chuẩn chung
// là đặt sau reverse proxy và chặn truy cập thẳng ở firewall.
func clientIP(c *gin.Context) string {
	if ip := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(xff)
	}
	return c.ClientIP()
}

// buildRateLimit — tạo Counter theo config server.rate_limit_store:
//
//	"" (mặc định) → theo cache store (memory | redis) — cache=redis thì rate
//	limit tự dùng redis (multi-instance sẵn sàng)
//	"memory" / "redis" → ép store cụ thể
func buildRateLimit(storeCfg string, cacheStore string, rdb *redis.Client) (Counter, error) {
	store := storeCfg
	if store == "" {
		store = cacheStore
	}
	switch store {
	case "redis":
		if rdb == nil {
			return nil, fmt.Errorf("ratelimit: store redis cần redis client (rdb nil)")
		}
		return newRedisCounter(rdb), nil
	case "memory", "":
		return newMemoryCounter(), nil
	default:
		return nil, fmt.Errorf("ratelimit: store %q không hỗ trợ (memory | redis)", store)
	}
}
