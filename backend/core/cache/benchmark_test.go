package cache

import (
	"context"
	"testing"
	"time"
)

func benchManager(b *testing.B, store string) *Manager {
	b.Helper()
	m, err := New(Config{Store: store, Prefix: "bench", DefaultTTL: time.Minute}, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(m.Close)
	return m
}

// BenchmarkManager_Set_Memory — chỉ set (Ristretto commit ASYNC → set+get ngay
// sẽ miss — xem pitfall gocache; get-hit đo riêng).
func BenchmarkManager_Set_Memory(b *testing.B) {
	m := benchManager(b, "memory")
	ctx := context.Background()
	val := map[string]any{"id": 1, "name": "bench"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := SetWithTTL(m, ctx, benchKey(i), val, time.Minute); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManager_Get_Hit — get trúng key đã set sẵn (1 key lặp lại).
func BenchmarkManager_Get_Hit(b *testing.B) {
	m := benchManager(b, "memory")
	ctx := context.Background()
	key := "hit"
	if err := SetWithTTL(m, ctx, key, 42, time.Minute); err != nil {
		b.Fatal(err)
	}
	// chờ commit async (Ristretto)
	for i := 0; i < 50; i++ {
		if _, err := Get[int](m, ctx, key); err == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Get[int](m, ctx, key); err != nil {
			b.Fatal(err)
		}
	}
}

func benchKey(i int) string {
	return "key-" + itoa(i)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
