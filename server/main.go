package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/websocket"
)

func logf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func setupLogging(cfg config) {
	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		log.Printf("[tiktok-bar] failed to create log dir: %v", err)
		return
	}

	serverLog, err := os.OpenFile(filepath.Join(cfg.LogDir, "server.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stdout, serverLog))
	}

	if l, err := newWebcastLogger(filepath.Join(cfg.LogDir, "webcast.log")); err == nil {
		rawLogger = l
		logf("[tiktok-bar] raw events -> %s", filepath.Join(cfg.LogDir, "webcast.log"))
	} else {
		logf("[tiktok-bar] failed to open raw log: %v", err)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func main() {
	cfg := loadConfig()

	setupLogging(cfg)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logf("ws upgrade: %v", err)
			return
		}
		c := newClient(conn, cfg)
		go c.writePump()
		go c.readPump()
	})

	serveFrontend(mux)

	addr := ":" + strconv.Itoa(cfg.Port)
	logf("[tiktok-bar] server listening on http://localhost%s", addr)
	if cfg.SignAPIKey == "" && cfg.SignServerURL == "" {
		logf("[tiktok-bar] signing: self-hosted (QuickJS) — no third-party sign server")
	} else {
		logf("[tiktok-bar] signing: external sign server (%s)", cfg.SignServerURL)
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func serveFrontend(mux *http.ServeMux) {
	var dist string
	for _, candidate := range []string{"frontend/dist", "../frontend/dist"} {
		if _, err := os.Stat(candidate); err == nil {
			if abs, err := filepath.Abs(candidate); err == nil {
				dist = abs
				break
			}
		}
	}
	if dist == "" {
		logf("[tiktok-bar] no frontend build found (looked for frontend/dist, ../frontend/dist) — serving API only")
		return
	}

	fs := http.FileServer(http.Dir(dist))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || !fileExists(filepath.Join(dist, r.URL.Path)) {
			http.ServeFile(w, r, filepath.Join(dist, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	}))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
