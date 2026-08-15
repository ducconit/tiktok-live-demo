package main

import (
	"os"

	"github.com/joho/godotenv"
)

type config struct {
	Port   int
	LogDir string
}

func loadConfig() config {
	_ = godotenv.Load()

	cfg := config{Port: 3001, LogDir: "logs"}
	if v := os.Getenv("PORT"); v != "" {
		if p := parseInt(v); p > 0 {
			cfg.Port = p
		}
	}
	if v := os.Getenv("LOG_DIR"); v != "" {
		cfg.LogDir = v
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
