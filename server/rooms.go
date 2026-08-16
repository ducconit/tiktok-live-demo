package main

import (
	"fmt"
	"sync"
)

// roomRegistry quản lý các tracker đang chạy (1 tracker / username), publish
// events qua Sockudo thay vì relay qua WebSocket nội bộ.
type roomRegistry struct {
	mu     sync.Mutex
	tracks map[string]controller
	pub    *sockudoPublisher
	cfg    config
}

func newRoomRegistry(cfg config, pub *sockudoPublisher) *roomRegistry {
	return &roomRegistry{tracks: map[string]controller{}, pub: pub, cfg: cfg}
}

// connect bắt đầu track một username; trả về trạng thái connected.
func (r *roomRegistry) connect(username string) (map[string]interface{}, error) {
	r.mu.Lock()
	if _, ok := r.tracks[username]; ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("đang theo dõi %s", username)
	}
	r.mu.Unlock()

	c, connected, err := startLive(username, r.cfg, r.pub)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.tracks[username] = c
	r.mu.Unlock()

	data, _ := connected.Data.(map[string]interface{})
	return map[string]interface{}{
		"connected": true,
		"roomId":    data["roomId"],
		"roomInfo":  data["roomInfo"],
	}, nil
}

// disconnect dừng track + báo "idle" cho mọi subscriber của channel.
func (r *roomRegistry) disconnect(username string) {
	r.mu.Lock()
	c, ok := r.tracks[username]
	delete(r.tracks, username)
	r.mu.Unlock()
	if ok {
		c.Stop()
		_ = r.pub.publish("user_"+username, "event", statusEvent("idle"))
	}
}
