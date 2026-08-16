package cache

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type user struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

func newMemoryManager(t *testing.T) *Manager {
	t.Helper()
	m, err := New(Config{Store: "memory"}, nil)
	if err != nil {
		t.Fatalf("New(memory) error = %v", err)
	}
	t.Cleanup(m.Close)
	return m
}

func TestMemory_SetGetRoundtrip(t *testing.T) {
	m := newMemoryManager(t)
	ctx := context.Background()

	u := user{ID: 42, Email: "a@b.com"}
	if err := Set(m, ctx, "user:42", u); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	// Ristretto commit ASYNC — Get ngay sau Set có thể miss → chờ eventual
	waitFor(t, m, "user:42", u)

	// Primitive type
	if err := Set(m, ctx, "n", 123); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "n", 123)
}

// waitFor — chờ giá trị xuất hiện (ristretto async commit, tối đa 2s).
func waitFor[T any](t *testing.T, m *Manager, key string, want T) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := Get[T](m, ctx, key)
		if err == nil && reflect.DeepEqual(got, want) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout chờ %q = %v (cuối: %v)", key, want, func() any {
		got, _ := Get[T](m, ctx, key)
		return got
	}())
}

func TestMemory_GetMiss(t *testing.T) {
	m := newMemoryManager(t)
	_, err := Get[user](m, context.Background(), "khong-co")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("miss phải trả ErrNotFound, got %v", err)
	}
}

// TestMemory_Delete — ristretto Del là async, chờ eventual miss.
func TestMemory_Delete(t *testing.T) {
	m := newMemoryManager(t)
	ctx := context.Background()

	if err := Set(m, ctx, "tmp", "value"); err != nil {
		t.Fatal(err)
	}
	if err := Delete(m, ctx, "tmp"); err != nil {
		t.Fatal(err)
	}
	waitGone[string](t, m, "tmp")
}

// waitGone — chờ key miss (eventual consistency của ristretto).
func waitGone[T any](t *testing.T, m *Manager, key string) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := Get[T](m, ctx, key); errors.Is(err, ErrNotFound) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout chờ key %s bị xoá", key)
}

func TestMemory_TTLExpired(t *testing.T) {
	m := newMemoryManager(t)
	ctx := context.Background()
	if err := SetWithTTL(m, ctx, "ttl", "voilat", 150*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := Get[string](m, ctx, "ttl"); !errors.Is(err, ErrNotFound) {
		t.Errorf("TTL hết hạn phải miss, got %v", err)
	}
}

func TestMemory_PrefixIsolate(t *testing.T) {
	// 2 manager cùng key nhưng prefix khác nhau → không đụng nhau
	a, _ := New(Config{Store: "memory", Prefix: "app-a:"}, nil)
	defer a.Close()
	b, _ := New(Config{Store: "memory", Prefix: "app-b:"}, nil)
	defer b.Close()

	ctx := context.Background()
	_ = Set(a, ctx, "k", "value-a")
	_ = Set(b, ctx, "k", "value-b")

	waitFor(t, a, "k", "value-a")
	waitFor(t, b, "k", "value-b")
}

func TestMemory_Concurrent(t *testing.T) {
	m := newMemoryManager(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = Set(m, ctx, "c", i)
			_, _ = Get[int](m, ctx, "c")
			_, _ = Get[user](m, ctx, "user:1") // miss cũng OK
		}(i)
	}
	wg.Wait()
}

func TestNew_InvalidStore(t *testing.T) {
	if _, err := New(Config{Store: "mysql"}, nil); err == nil {
		t.Fatal("store không hỗ trợ phải trả lỗi")
	}
}

func TestNew_RedisWithoutClient(t *testing.T) {
	if _, err := New(Config{Store: "redis"}, nil); err == nil {
		t.Fatal("store redis thiếu client phải trả lỗi")
	}
}

// ---- Integration: redis thật (docker compose, 6380) ----

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6380"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis không khả dụng tại localhost:6380 (%v) — skip", err)
	}
	return rdb
}

func TestRedis_SetGet(t *testing.T) {
	rdb := testRedis(t)
	defer func() { _ = rdb.Close() }()
	ctx := context.Background()

	m, err := New(Config{Store: "redis", Prefix: "test-cache:", DefaultTTL: time.Minute}, rdb)
	if err != nil {
		t.Fatalf("New(redis) error = %v", err)
	}
	t.Cleanup(m.Close)

	u := user{ID: 7, Email: "r@b.com"}
	if err := Set(m, ctx, "user:7", u); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := Get[user](m, ctx, "user:7")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != u {
		t.Errorf("Get = %+v, want %+v", got, u)
	}

	// Miss
	if _, err := Get[user](m, ctx, "user:khong"); !errors.Is(err, ErrNotFound) {
		t.Errorf("miss phải ErrNotFound, got %v", err)
	}

	// Dọn key test
	_ = Delete(m, ctx, "user:7")
}
