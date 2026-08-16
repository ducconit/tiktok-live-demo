package main

import (
	"sync"
)

// roomRegistry quản lý các tracker đang chạy — **1 tracker / username**.
// Nhiều tab/browser cùng connect 1 username sẽ DÙNG CHUNG 1 TikTok connection
// (events publish lên channel "user_<username>", mọi subscriber đều nhận).
// Dùng reference counting: tracker dừng khi tab cuối cùng disconnect.
type roomRegistry struct {
	mu     sync.Mutex
	locks  map[string]*sync.Mutex // per-username lock để serialize connect/disconnect cùng user
	tracks map[string]*trackEntry
	pub    *sockudoPublisher
	cfg    config
}

type trackEntry struct {
	controller controller
	refs       int
	result     map[string]interface{} // {connected:true, roomId, roomInfo}
}

func newRoomRegistry(cfg config, pub *sockudoPublisher) *roomRegistry {
	return &roomRegistry{
		locks:  map[string]*sync.Mutex{},
		tracks: map[string]*trackEntry{},
		pub:    pub,
		cfg:    cfg,
	}
}

func (r *roomRegistry) lockFor(username string) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.locks[username]
	if !ok {
		l = &sync.Mutex{}
		r.locks[username] = l
	}
	return l
}

// connect: nếu chưa track → start tracker; nếu đã track (tab khác) → tăng refs
// và trả về connected status hiện có (không tạo thêm TikTok connection).
func (r *roomRegistry) connect(username string) (map[string]interface{}, error) {
	l := r.lockFor(username)
	l.Lock()
	defer l.Unlock()

	if e, ok := r.tracks[username]; ok {
		e.refs++
		return e.result, nil
	}

	c, connected, err := startLive(username, r.cfg, r.pub)
	if err != nil {
		return nil, err
	}
	data, _ := connected.Data.(map[string]interface{})
	result := map[string]interface{}{
		"connected": true,
		"roomId":    data["roomId"],
		"roomInfo":  data["roomInfo"],
	}

	r.mu.Lock()
	r.tracks[username] = &trackEntry{controller: c, refs: 1, result: result}
	r.mu.Unlock()
	return result, nil
}

// disconnect: giảm refs; chỉ dừng tracker + publish "idle" khi refs về 0
// (tab cuối cùng rời).
func (r *roomRegistry) disconnect(username string) {
	l := r.lockFor(username)
	l.Lock()
	defer l.Unlock()

	e, ok := r.tracks[username]
	if !ok {
		return
	}
	e.refs--
	if e.refs > 0 {
		return
	}

	r.mu.Lock()
	delete(r.tracks, username)
	r.mu.Unlock()
	e.controller.Stop()
	_ = r.pub.publish("user_"+username, "event", statusEvent("idle"))
}

// connectedUserCount trả số username đang được track (debug/hữu ích).
func (r *roomRegistry) connectedUserCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.tracks)
}
