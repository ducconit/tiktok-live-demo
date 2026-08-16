package main

import (
	"os"

	"github.com/joho/godotenv"
)

type config struct {
	Port           int
	LogDir         string
	ConnectionMode string // "long_poll" (default) hoặc "websocket"
	PollIntervalMs int    // ms, mặc định 3000
}

func loadConfig() config {
	_ = godotenv.Load()

	cfg := config{Port: 3001, LogDir: "logs", ConnectionMode: "long_poll", PollIntervalMs: 3000}
	if v := os.Getenv("PORT"); v != "" {
		if p := parseInt(v); p > 0 {
			cfg.Port = p
		}
	}
	if v := os.Getenv("LOG_DIR"); v != "" {
		cfg.LogDir = v
	}
	if v := os.Getenv("CONNECTION_MODE"); v == "websocket" {
		cfg.ConnectionMode = "websocket"
	}
	if v := os.Getenv("POLL_INTERVAL_MS"); v != "" {
		if n := parseInt(v); n > 0 {
			cfg.PollIntervalMs = n
		}
	}
	return cfg
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
