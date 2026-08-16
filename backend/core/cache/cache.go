// Package cache — multi-store cache trên eko/gocache v4.
//
//	store memory (mặc định): Ristretto (dgraph-io) — zero-config, 1 instance
//	store redis:             go-redis — dùng chung giữa nhiều instance (sau LB)
//
// Type-safe qua marshaler (msgpack): Get[T](m, ctx, key), Set(m, ctx, key, value).
// Go không cho method generic → API là free functions.
package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	libstore "github.com/eko/gocache/lib/v4/store"
	redisstore "github.com/eko/gocache/store/redis/v4"
	ristrettostore "github.com/eko/gocache/store/ristretto/v4"
	"github.com/redis/go-redis/v9"
)

// ErrNotFound — cache miss (caller check bằng errors.Is).
var ErrNotFound = errors.New("cache: key không tồn tại")

// Cấu hình Ristretto (memory store) — không magic number.
const (
	ristrettoNumCounters = 1e7     // số key theo dõi tần suất
	ristrettoMaxCost     = 1 << 30 // dung lượng tối đa (bytes)
	ristrettoBufferItems = 64      // buffer ghi bất đồng bộ
	defaultTTL           = 5 * time.Minute
)

// Config — cấu hình cache (từ config.yml / env).
type Config struct {
	Store      string        // memory (Ristretto, mặc định) | redis
	Prefix     string        // tiền tố key (vd tên app khi dùng chung redis)
	DefaultTTL time.Duration // mặc định 5m (Set không truyền ttl)
}

// Manager — nhiều store, chọn store mặc định qua config.
type Manager struct {
	defaultName string
	prefix      string
	stores      map[string]*marshaler.Marshaler
	closeFns    []func()
}

// New — khởi tạo manager theo config. Store redis mà thiếu client → lỗi.
func New(cfg Config, rdb *redis.Client) (*Manager, error) {
	if cfg.Store == "" {
		cfg.Store = "memory"
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = defaultTTL
	}
	m := &Manager{prefix: cfg.Prefix, stores: map[string]*marshaler.Marshaler{}}
	opts := []libstore.Option{
		libstore.WithExpiration(cfg.DefaultTTL),
		libstore.WithSynchronousSet(), // ristretto commit đồng bộ — Set xong Get được ngay
	}

	build := func(name string, st libstore.StoreInterface) {
		m.stores[name] = marshaler.New(cache.New[any](st))
	}

	switch cfg.Store {
	case "memory":
		rc, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: ristrettoNumCounters,
			MaxCost:     ristrettoMaxCost,
			BufferItems: ristrettoBufferItems,
		})
		if err != nil {
			return nil, fmt.Errorf("cache: init ristretto: %w", err)
		}
		m.closeFns = append(m.closeFns, rc.Close)
		build("memory", ristrettostore.NewRistretto(rc, opts...))
		m.defaultName = "memory"

	case "redis":
		if rdb == nil {
			return nil, errors.New("cache: store redis cần redis client (rdb nil)")
		}
		build("redis", redisstore.NewRedis(rdb, opts...))
		m.defaultName = "redis"

	default:
		return nil, fmt.Errorf("cache: store %q không hỗ trợ (memory | redis)", cfg.Store)
	}
	return m, nil
}

// Close — giải phóng tài nguyên store (ristretto...).
func (m *Manager) Close() {
	for _, f := range m.closeFns {
		f()
	}
}

// StoreNames — danh sách store đã khởi tạo (debug).
func (m *Manager) StoreNames() []string {
	names := make([]string, 0, len(m.stores))
	for name := range m.stores {
		names = append(names, name)
	}
	return names
}

// DefaultStore — tên store mặc định (memory | redis).
func (m *Manager) DefaultStore() string {
	return m.defaultName
}

// Prefix — tiền tố key đang dùng (tránh trùng khi dùng chung redis).
func (m *Manager) Prefix() string {
	return m.prefix
}

// ---- API (free functions — Go không cho method generic) ----

// Get — lấy giá trị từ store mặc định. Miss → ErrNotFound.
func Get[T any](m *Manager, ctx context.Context, key string) (T, error) {
	return get[T](m, ctx, m.defaultName, key)
}

// GetFrom — lấy giá trị từ store theo tên (vd "redis").
func GetFrom[T any](m *Manager, storeName string, ctx context.Context, key string) (T, error) {
	return get[T](m, ctx, storeName, key)
}

// Set — lưu giá trị (store mặc định, TTL mặc định của store).
func Set[T any](m *Manager, ctx context.Context, key string, value T) error {
	return set(m, ctx, m.defaultName, key, value, nil)
}

// SetWithTTL — lưu với TTL riêng cho key này.
func SetWithTTL[T any](m *Manager, ctx context.Context, key string, value T, ttl time.Duration) error {
	return set(m, ctx, m.defaultName, key, value, &ttl)
}

// Delete — xoá key (store mặc định).
func Delete(m *Manager, ctx context.Context, key string) error {
	return del(m, ctx, m.defaultName, key)
}

// Clear — xoá toàn bộ dữ liệu trong store (admin: DELETE /admin/cache).
func Clear(m *Manager, ctx context.Context) error {
	for name, store := range m.stores {
		if err := store.Clear(ctx); err != nil {
			return fmt.Errorf("cache: clear %s: %w", name, err)
		}
	}
	return nil
}

// ---- internal ----

func get[T any](m *Manager, ctx context.Context, storeName, key string) (T, error) {
	var zero T
	store := m.stores[storeName]
	if store == nil {
		return zero, fmt.Errorf("cache: store %q không tồn tại", storeName)
	}
	if _, err := store.Get(ctx, m.prefix+key, &zero); err != nil {
		if errors.Is(err, libstore.NotFound{}) {
			return zero, ErrNotFound
		}
		return zero, fmt.Errorf("cache: get %s: %w", key, err)
	}
	return zero, nil
}

// itemCost — Ristretto yêu cầu cost > 0 để lưu item (cost 0 → bị loại, cache luôn miss).
const itemCost = 1

func set[T any](m *Manager, ctx context.Context, storeName, key string, value T, ttl *time.Duration) error {
	store := m.stores[storeName]
	if store == nil {
		return fmt.Errorf("cache: store %q không tồn tại", storeName)
	}
	if ttl != nil {
		return store.Set(ctx, m.prefix+key, value, libstore.WithExpiration(*ttl), libstore.WithCost(itemCost))
	}
	return store.Set(ctx, m.prefix+key, value, libstore.WithCost(itemCost))
}

func del(m *Manager, ctx context.Context, storeName, key string) error {
	store := m.stores[storeName]
	if store == nil {
		return fmt.Errorf("cache: store %q không tồn tại", storeName)
	}
	return store.Delete(ctx, m.prefix+key)
}
