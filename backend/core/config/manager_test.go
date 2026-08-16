package config

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// Integration test: sync qua Redis pub/sub — 2 instance, 1 đổi config → instance kia nhận.
// Cần Redis/Valkey thật (docker compose) tại REDIS_ADDR test (mặc định localhost:6380).
func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := "localhost:6380" // compose FORWARD_VALKEY_PORT
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis không khả dụng tại %s (%v) — skip integration", addr, err)
	}
	return rdb
}

func TestDynamicSync_TwoInstances(t *testing.T) {
	rdb := newTestRedis(t)
	defer func() { _ = rdb.Close() }()
	ctx := context.Background()

	// Dọn state cũ (test lặp)
	mgrA, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	_ = rdb.Del(ctx, mgrA.stateKey)
	_ = rdb.Del(ctx, mgrA.channel)

	mgrB, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	// Khác instance ID
	if mgrA.instanceID == mgrB.instanceID {
		t.Fatal("2 instance trùng ID — không hợp lệ")
	}

	if err := mgrA.InitDynamic(ctx, rdb); err != nil {
		t.Fatalf("InitDynamic A: %v", err)
	}
	defer mgrA.Close()
	if err := mgrB.InitDynamic(ctx, rdb); err != nil {
		t.Fatalf("InitDynamic B: %v", err)
	}
	defer mgrB.Close()

	// B lắng nghe thay đổi
	got := make(chan any, 1)
	mgrB.OnChange("server.rate_limit", func(v any) { got <- v })

	// A đổi config → B phải nhận
	if err := mgrA.SetDynamic("server.rate_limit", 250); err != nil {
		t.Fatalf("SetDynamic A: %v", err)
	}

	select {
	case v := <-got:
		// JSON qua pub/sub → number decode thành float64
		var n int
		switch tv := v.(type) {
		case int:
			n = tv
		case float64:
			n = int(tv)
		default:
			t.Fatalf("B nhận value kiểu %T = %v", v, v)
		}
		if n != 250 {
			t.Errorf("B nhận value = %v, want 250", v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("B không nhận update từ A (timeout)")
	}

	// Verify local của B đã đổi
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if mgrB.Int("server.rate_limit", 0) == 250 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if mgrB.Int("server.rate_limit", 0) != 250 {
		t.Errorf("B.rate_limit = %d, want 250 (sync)", mgrB.Int("server.rate_limit", 0))
	}

	// State persist — instance C join phải tự đồng bộ
	mgrC, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if err := mgrC.InitDynamic(ctx, rdb); err != nil {
		t.Fatalf("InitDynamic C: %v", err)
	}
	defer mgrC.Close()

	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if mgrC.Int("server.rate_limit", 0) == 250 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := mgrC.Int("server.rate_limit", 0); got != 250 {
		t.Errorf("C (instance join) rate_limit = %d, want 250 (state sync)", got)
	}
}

func TestDynamicSync_ConcurrentSafe(t *testing.T) {
	rdb := newTestRedis(t)
	defer func() { _ = rdb.Close() }()
	ctx := context.Background()

	mgr, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	_ = rdb.Del(ctx, mgr.stateKey)
	if err := mgr.InitDynamic(ctx, rdb); err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	// Đọc/ghi đồng thời — race detector sẽ bắt nếu thiếu lock
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = mgr.SetDynamic("key.concurrent", i)
			_ = mgr.Int("server.rate_limit", 100)
			_ = mgr.GetDynamic("key.concurrent")
		}(i)
	}
	wg.Wait()
}
