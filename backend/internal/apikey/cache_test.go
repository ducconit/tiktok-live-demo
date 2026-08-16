package apikey

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/google/uuid"
)

// TestCacheInfo — cache.Set/Get với type Info (chứa uuid.UUID + []string) — mô phỏng Lookup.
func TestCacheInfo(t *testing.T) {
	cm, err := cache.New(cache.Config{Store: "memory", DefaultTTL: time.Minute}, nil)
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	defer cm.Close()

	info := Info{ID: uuid.NewString(), Name: "test-key", Scopes: []string{"a", "b"}}
	key := "apikey:hash:test123"

	if err := cache.Set[Info](cm, context.Background(), key, info); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Ristretto commit ASYNC — Get ngay sau Set có thể miss → chờ eventual (như core/cache).
	ctx := context.Background()
	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err := cache.Get[Info](cm, ctx, key)
		if err == nil && reflect.DeepEqual(got, info) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout chờ cache key — last err=%v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
