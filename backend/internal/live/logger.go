package live

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// webcastLogger writes raw received Webcast events to a file as JSON lines.
type webcastLogger struct {
	mu sync.Mutex
	f  *os.File
}

func newWebcastLogger(path string) (*webcastLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &webcastLogger{f: f}, nil
}

// raw logs a decoded event with its concrete Go type.
func (l *webcastLogger) raw(ev interface{}) {
	b, err := json.Marshal(ev)
	if err != nil {
		b = []byte(`{"marshal_error":true}`)
	}
	out, err := json.Marshal(map[string]any{
		"type": fmt.Sprintf("%T", ev),
		"data": json.RawMessage(b),
	})
	if err != nil {
		out = b
	}
	l.write(string(out))
}

// line logs an arbitrary message line.
func (l *webcastLogger) line(s string) {
	l.write(s)
}

func (l *webcastLogger) write(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.f.WriteString(time.Now().Format(time.RFC3339Nano) + " " + s + "\n")
}

func (l *webcastLogger) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.f.Close()
}
