package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
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

func main() {
	cfg := loadConfig()
	setupLogging(cfg)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	pub := newSockudoPublisher(cfg)
	rooms := newRoomRegistry(cfg, pub)

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

	// Control: bắt đầu/dừng track; events được publish lên Sockudo channel
	// "user_<username>" (client subscribe qua @sockudo/client).
	r.POST("/api/connect", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"connected": false, "error": "thiếu username"})
			return
		}
		username := normalizeUsername(req.Username)
		data, err := rooms.connect(username)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"connected": false, "error": friendlyError(err)})
			return
		}
		c.JSON(http.StatusOK, data)
	})

	r.POST("/api/disconnect", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "thiếu username"})
			return
		}
		rooms.disconnect(normalizeUsername(req.Username))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	serveFrontend(r)

	addr := ":" + strconv.Itoa(cfg.Port)
	logf("[tiktok-bar] server listening on http://localhost%s", addr)
	logf("[tiktok-bar] signing: self-hosted (QuickJS) — no third-party sign server")
	logf("[tiktok-bar] connection mode: %s (poll %dms)", cfg.ConnectionMode, cfg.PollIntervalMs)
	logf("[tiktok-bar] realtime: Sockudo %s (app %s)", cfg.SockudoURL, cfg.SockudoAppID)

	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// serveFrontend phục vụ bản build frontend (Vite/Vue) qua gin: assets tĩnh +
// SPA fallback (mọi route không khớp API → index.html).
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
