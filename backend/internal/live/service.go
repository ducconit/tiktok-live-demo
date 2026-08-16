package live

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
)

// Service — quản lý tracker TikTok + publish events qua Sockudo (Pusher-compatible).
//
// 1 tracker / username (ref counting): nhiều tab/browser connect cùng 1 username
// DÙNG CHUNG 1 TikTok connection — events publish lên channel "user_<username>",
// mọi subscriber đều nhận. Tracker dừng khi tab cuối cùng disconnect (refs = 0).
type Service struct {
	cfg    config.LiveConfig
	pub    *sockudoPublisher
	logger *webcastLogger

	mu     sync.Mutex
	locks  map[string]*sync.Mutex // per-username lock — serialize connect/disconnect cùng user
	tracks map[string]*trackEntry

	signerOnce sync.Once
	signerInst *selfSigner
	signerErr  error
}

type trackEntry struct {
	controller controller
	refs       int
	result     map[string]any // {connected, roomId, roomInfo}
}

// NewService — khởi tạo service live tracker. Webcast logger (logs/webcast.log)
// best-effort: lỗi → nil (track vẫn chạy, chỉ mất log raw).
func NewService(cfg config.LiveConfig) *Service {
	s := &Service{
		cfg:    cfg,
		pub:    newSockudoPublisher(cfg),
		locks:  map[string]*sync.Mutex{},
		tracks: map[string]*trackEntry{},
	}
	if l, err := newWebcastLogger(filepath.Join("logs", "webcast.log")); err == nil {
		s.logger = l
	} else {
		slog.Warn("webcast logger chưa sẵn sàng — raw events không được ghi", "err", err)
	}
	return s
}

// getSigner — shared self-hosted signer (QuickJS + Chrome TLS), init một lần
// (~giây đầu), tái sử dụng cho mọi tracker.
func (s *Service) getSigner() (*selfSigner, error) {
	s.signerOnce.Do(func() {
		s.signerInst, s.signerErr = newSelfSigner()
	})
	return s.signerInst, s.signerErr
}

func (s *Service) lockFor(username string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := s.locks[username]
	if !ok {
		l = &sync.Mutex{}
		s.locks[username] = l
	}
	return l
}

// Preview — live status + room info nhẹ (không track). Client gọi trước khi
// connect (TanStack Query) để hiển thị trạng thái phòng.
func (s *Service) Preview(username string) (map[string]any, error) {
	return roomPreview(username)
}

// Connect — bắt đầu track username; nếu đã track (tab khác) → tăng refs và trả
// connected status hiện có (không tạo thêm TikTok connection).
func (s *Service) Connect(username string) (map[string]any, error) {
	l := s.lockFor(username)
	l.Lock()
	defer l.Unlock()

	if e, ok := s.tracks[username]; ok {
		e.refs++
		return e.result, nil
	}

	c, connected, err := s.startLive(username)
	if err != nil {
		return nil, err
	}
	data, _ := connected.Data.(map[string]any)
	result := map[string]any{
		"connected": true,
		"roomId":    data["roomId"],
		"roomInfo":  data["roomInfo"],
	}

	s.mu.Lock()
	s.tracks[username] = &trackEntry{controller: c, refs: 1, result: result}
	s.mu.Unlock()
	return result, nil
}

// Disconnect — giảm refs; chỉ dừng tracker + publish "idle" khi refs về 0
// (tab cuối cùng rời).
func (s *Service) Disconnect(username string) {
	l := s.lockFor(username)
	l.Lock()
	defer l.Unlock()

	e, ok := s.tracks[username]
	if !ok {
		return
	}
	e.refs--
	if e.refs > 0 {
		return
	}

	s.mu.Lock()
	delete(s.tracks, username)
	s.mu.Unlock()

	// Báo "idle" cho mọi subscriber TRƯỚC khi dừng tracker (live.Close() có thể block —
	// publish sau Stop() sẽ không kịp gửi).
	if err := s.pub.publish("user_"+username, "event", statusEvent("idle")); err != nil {
		slog.Error("sockudo publish idle", "err", err, "channel", "user_"+username)
	}
	e.controller.Stop()
}

// Close — đóng webcast logger + signer (graceful shutdown).
func (s *Service) Close() {
	if s.logger != nil {
		s.logger.close()
	}
	if s.signerInst != nil {
		s.signerInst.close()
	}
}
