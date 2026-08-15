package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 54 * time.Second
	sendBuffer = 256
)

type client struct {
	conn *websocket.Conn
	cfg  config
	send chan []byte

	mu      sync.Mutex
	tracker controller
	gen     uint64
}

func newClient(conn *websocket.Conn, cfg config) *client {
	return &client{
		conn: conn,
		cfg:  cfg,
		send: make(chan []byte, sendBuffer),
	}
}

func (c *client) emit(e event) {
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
		logf("dropping event for slow client")
	}
}

type command struct {
	Action   string `json:"action"`
	Username string `json:"username"`
}

func (c *client) connectRoom(username string) {
	c.mu.Lock()
	c.gen++
	gen := c.gen
	if c.tracker != nil {
		c.tracker.Stop()
		c.tracker = nil
	}
	c.mu.Unlock()

	// emit is generation-aware: once disconnectRoom bumps c.gen, any in-flight
	// event from the previous tracker (e.g. buffered events drained on Close) is dropped.
	emit := func(e event) {
		c.mu.Lock()
		active := gen == c.gen
		c.mu.Unlock()
		if active {
			c.emit(e)
		}
	}

	emit(event{Type: "status", Data: map[string]interface{}{"state": "connecting", "username": username}, Ts: time.Now().UnixMilli()})

	go func() {
		trk, err := startTracker(username, emit, c.cfg)

		c.mu.Lock()
		defer c.mu.Unlock()
		if gen != c.gen {
			if trk != nil {
				trk.Stop()
			}
			return
		}
		if err != nil {
			emit(event{Type: "status", Data: map[string]interface{}{"state": "error", "message": err.Error()}, Ts: time.Now().UnixMilli()})
			return
		}
		c.tracker = trk
	}()
}

func (c *client) disconnectRoom() {
	c.mu.Lock()
	c.gen++
	if c.tracker != nil {
		c.tracker.Stop()
		c.tracker = nil
	}
	c.mu.Unlock()
	c.emit(statusEvent("idle"))
}

func (c *client) readPump() {
	defer func() {
		c.disconnectRoom()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var cmd command
		if err := json.Unmarshal(msg, &cmd); err != nil {
			continue
		}
		switch cmd.Action {
		case "connect":
			if cmd.Username != "" {
				c.connectRoom(normalizeUsername(cmd.Username))
			}
		case "disconnect":
			c.disconnectRoom()
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func normalizeUsername(raw string) string {
	s := strings.TrimSpace(raw)
	if u, err := url.Parse(s); err == nil && u.Path != "" && strings.Contains(s, "/") && strings.Contains(s, ".") {
		for _, part := range strings.Split(strings.Trim(u.Path, "/"), "/") {
			if strings.HasPrefix(part, "@") {
				return strings.TrimPrefix(part, "@")
			}
		}
	}
	s = strings.TrimPrefix(s, "@")
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return s
}
