package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
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

	// Gin framework (release mode; override with GIN_MODE=debug nếu cần).
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.GET("/api/room/:username", func(c *gin.Context) {
		username := normalizeUsername(c.Param("username"))
		data, err := roomPreview(username)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"live": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)
	})

	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logf("ws upgrade: %v", err)
			return
		}
		cl := newClient(conn, cfg)
		go cl.writePump()
		go cl.readPump()
	})

	serveFrontend(r)

	addr := ":" + strconv.Itoa(cfg.Port)
	logf("[tiktok-bar] server listening on http://localhost%s", addr)
	logf("[tiktok-bar] signing: self-hosted (QuickJS) — no third-party sign server")
	logf("[tiktok-bar] connection mode: %s (poll %dms)", cfg.ConnectionMode, cfg.PollIntervalMs)

	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// serveFrontend phục vụ bản build frontend (Vite/Vue) qua gin: assets tĩnh +
// SPA fallback (mọi route không khớp API/WS → index.html).
func serveFrontend(r *gin.Engine) {
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

	r.Static("/assets", filepath.Join(dist, "assets"))
	r.NoRoute(func(c *gin.Context) {
		p := filepath.Join(dist, c.Request.URL.Path)
		if fileExists(p) {
			c.File(p)
			return
		}
		c.File(filepath.Join(dist, "index.html"))
	})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
