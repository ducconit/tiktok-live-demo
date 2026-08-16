package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
)

// ============================================================
// Dynamic config — đồng bộ giữa nhiều instance (sau load balancer):
//
//   Instance A:  SetDynamic("server.rate_limit", 200)
//     1. set local (koanf) — áp dụng ngay
//     2. persist vào Redis state key (last-write-wins theo timestamp)
//     3. publish message lên channel (Redis pub/sub)
//   Instance B (subscriber): nhận message → set local + persist state
//
// Instance mới join: đọc state key hiện tại → đồng bộ ngay từ đầu.
// Không có Redis → dynamic chỉ ảnh hưởng local (zero-config vẫn chạy).
// ============================================================

const (
	stateKeyFmt = "gvs:config:state:%s" // app.name → JSON map[key]syncMsg
	channelFmt  = "gvs:config:sync:%s"  // app.name
	stateTTL    = 0                     // 0 = không hết hạn (giữ state cho instance join sau)
)

// syncMsg — payload pub/sub + 1 entry trong state.
type syncMsg struct {
	Key   string    `json:"key"`
	Value any       `json:"value"`
	TS    time.Time `json:"ts"`
	Src   string    `json:"src"`
}

// bus — abstraction Redis (mock được trong test).
type bus interface {
	Publish(ctx context.Context, channel string, payload []byte) error
	Subscribe(ctx context.Context, channel string) (<-chan []byte, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

type redisBus struct{ rdb *redis.Client }

func (b *redisBus) Publish(ctx context.Context, channel string, payload []byte) error {
	return b.rdb.Publish(ctx, channel, payload).Err()
}

func (b *redisBus) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	sub := b.rdb.Subscribe(ctx, channel)
	// Chờ Redis xác nhận subscribed — nếu không, message publish ngay sau InitDynamic sẽ bị miss
	if _, err := sub.Receive(ctx); err != nil {
		_ = sub.Close()
		return nil, fmt.Errorf("subscribe %s: %w", channel, err)
	}
	ch := make(chan []byte, 32)
	go func() {
		defer close(ch)
		for msg := range sub.Channel() {
			select {
			case ch <- []byte(msg.Payload):
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func (b *redisBus) Get(ctx context.Context, key string) (string, error) {
	return b.rdb.Get(ctx, key).Result()
}

func (b *redisBus) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return b.rdb.Set(ctx, key, value, ttl).Err()
}

// Manager — config engine (koanf) + dynamic sync qua Redis pub/sub.
type Manager struct {
	K   *koanf.Koanf
	Cfg Config // snapshot STATIC (đọc 1 lần lúc start)

	mu          sync.RWMutex
	instanceID  string
	dsnSource   string // nguồn config: database | file | defaults (detect lúc Load)
	bus         bus    // nil = dynamic disabled
	stateKey    string
	channel     string
	dynamicKeys map[string]struct{}
	listeners   map[string][]func(value any)
	ctx         context.Context
	cancel      context.CancelFunc
}

// ConfigSource — nguồn config đang dùng: "database" | "file" | "defaults".
// database → Redis BẮT BUỘC (đồng bộ dynamic config giữa các instance).
func (m *Manager) ConfigSource() string { return m.dsnSource }

func newInstanceID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("inst-%d", time.Now().UnixNano())
	}
	return "inst-" + hex.EncodeToString(buf)
}

// InstanceID — định danh instance này (in log, phân biệt nguồn pub/sub).
func (m *Manager) InstanceID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.instanceID
}

// Channel — tên channel pub/sub (debug/info).
func (m *Manager) Channel() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channel
}

// InitDynamic bật dynamic config (gọi khi có Redis). rdb nil → disabled (zero-config).
func (m *Manager) InitDynamic(ctx context.Context, rdb *redis.Client) error {
	if rdb == nil {
		slog.Info("config: dynamic sync disabled (không có redis) — SetDynamic chỉ ảnh hưởng local")
		return nil
	}
	m.mu.Lock()
	m.bus = &redisBus{rdb: rdb}
	m.stateKey = fmt.Sprintf(stateKeyFmt, m.Cfg.App.Name)
	m.channel = fmt.Sprintf(channelFmt, m.Cfg.App.Name)
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	// Instance join: đồng bộ state hiện tại trước khi lắng nghe
	if err := m.loadState(); err != nil {
		slog.Warn("config: load state thất bại — tiếp tục (state sẽ sync khi có update)", "err", err)
	}

	ch, err := m.bus.Subscribe(m.ctx, m.channel)
	if err != nil {
		return fmt.Errorf("config: subscribe %s: %w", m.channel, err)
	}
	go m.subscribeLoop(ch)
	slog.Info("config: dynamic sync enabled", "channel", m.channel, "instance", m.instanceID)
	return nil
}

// SetDynamic — đổi config runtime: local + persist state + broadcast (pub/sub).
func (m *Manager) SetDynamic(key string, value any) error {
	msg := syncMsg{Key: key, Value: value, TS: time.Now().UTC(), Src: m.instanceID}

	m.mu.RLock()
	bus := m.bus
	m.mu.RUnlock()

	// Luôn set local (kể cả không redis)
	m.setLocal(key, value)
	m.notify(key, value)

	if bus == nil {
		return nil
	}
	return m.broadcast(bus, msg)
}

// GetDynamic — đọc giá trị động theo key (nil nếu chưa set).
func (m *Manager) GetDynamic(key string) any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.K.Get(key)
}

// Exists — key có tồn tại không.
func (m *Manager) Exists(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.K.Exists(key)
}

// Int — đọc int theo key, default khi thiếu (hay dùng cho dynamic).
func (m *Manager) Int(key string, def int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.K.Exists(key) {
		return def
	}
	return m.K.Int(key)
}

// String — đọc string theo key, default khi thiếu.
func (m *Manager) String(key string, def string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.K.Exists(key) {
		return def
	}
	return m.K.String(key)
}

// OnChange — đăng ký callback khi key dynamic thay đổi (mọi instance, kể cả nguồn).
func (m *Manager) OnChange(key string, fn func(value any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listeners == nil {
		m.listeners = map[string][]func(value any){}
	}
	m.listeners[key] = append(m.listeners[key], fn)
}

// AllDynamic — toàn bộ dynamic keys đang hiệu lực (GET /admin/config — remote config).
func (m *Manager) AllDynamic() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]any, len(m.dynamicKeys))
	for key := range m.dynamicKeys {
		if m.K.Exists(key) {
			out[key] = m.K.Get(key)
		}
	}
	return out
}

// Marshal — serialize config hiện tại ra bytes (json/yaml — dùng để backup/ghi file).
func (m *Manager) Marshal(parser koanf.Parser) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.K.Marshal(parser)
}

// Close — dừng subscriber.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
}

// ---- internal ----

func (m *Manager) setLocal(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = m.K.Set(key, value)
	if m.dynamicKeys == nil {
		m.dynamicKeys = map[string]struct{}{}
	}
	m.dynamicKeys[key] = struct{}{}
}

func (m *Manager) notify(key string, value any) {
	m.mu.RLock()
	fns := m.listeners[key]
	m.mu.RUnlock()
	for _, fn := range fns {
		fn(value)
	}
}

// broadcast — persist state + publish (chỉ gọi khi có bus).
func (m *Manager) broadcast(b bus, msg syncMsg) error {
	m.mu.RLock()
	ctx, stateKey, channel := m.ctx, m.stateKey, m.channel
	m.mu.RUnlock()

	state := m.readState(ctx, b, stateKey)
	state[msg.Key] = msg
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("config: marshal state: %w", err)
	}
	if err := b.Set(ctx, stateKey, string(payload), stateTTL); err != nil {
		return fmt.Errorf("config: persist state: %w", err)
	}

	msgPayload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("config: marshal msg: %w", err)
	}
	if err := b.Publish(ctx, channel, msgPayload); err != nil {
		return fmt.Errorf("config: publish: %w", err)
	}
	return nil
}

// loadState — instance join: áp dụng toàn bộ state hiện tại (không broadcast).
func (m *Manager) loadState() error {
	m.mu.RLock()
	ctx, b, stateKey := m.ctx, m.bus, m.stateKey
	m.mu.RUnlock()
	if b == nil {
		return nil
	}
	state := m.readState(ctx, b, stateKey)
	for _, msg := range state {
		if msg.Src == m.instanceID {
			continue
		}
		m.setLocal(msg.Key, msg.Value)
		m.notify(msg.Key, msg.Value)
	}
	if len(state) > 0 {
		slog.Info("config: đã đồng bộ state hiện tại", "keys", len(state), "instance", m.instanceID)
	}
	return nil
}

// readState — đọc JSON map[key]syncMsg từ Redis (rỗng nếu chưa có/không đọc được).
func (m *Manager) readState(ctx context.Context, b bus, stateKey string) map[string]syncMsg {
	raw, err := b.Get(ctx, stateKey)
	if err != nil || raw == "" {
		return map[string]syncMsg{}
	}
	var state map[string]syncMsg
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		slog.Warn("config: parse state lỗi — dùng state rỗng", "err", err)
		return map[string]syncMsg{}
	}
	return state
}

// subscribeLoop — nhận update từ instance khác → apply local + persist.
func (m *Manager) subscribeLoop(ch <-chan []byte) {
	for payload := range ch {
		var msg syncMsg
		if err := json.Unmarshal(payload, &msg); err != nil {
			slog.Warn("config: bỏ qua message lỗi", "err", err)
			continue
		}
		m.mu.RLock()
		b, ctx, stateKey := m.bus, m.ctx, m.stateKey
		self := msg.Src == m.instanceID
		m.mu.RUnlock()
		if self {
			continue // tin mình gửi
		}

		m.setLocal(msg.Key, msg.Value)
		m.notify(msg.Key, msg.Value)

		// Persist vào state (last-write-wins theo TS của state hiện tại)
		state := m.readState(ctx, b, stateKey)
		if cur, ok := state[msg.Key]; ok && cur.TS.After(msg.TS) {
			continue // tin cũ hơn state hiện tại — bỏ qua
		}
		state[msg.Key] = msg
		payload, err := json.Marshal(state)
		if err == nil {
			_ = b.Set(ctx, stateKey, string(payload), stateTTL)
		}
	}
}
