package config

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration test: CONFIG_DSN=postgres://... — config lưu trong bảng app_config.
// Cần Postgres thật (docker compose, forward 5433).
func testPGDSN(t *testing.T) string {
	t.Helper()
	dsn := "postgres://app:app_password_dev@localhost:5433/tiktok_live_platform?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres không khả dụng (%v) — skip", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("postgres không ping được (%v) — skip", err)
	}
	return dsn
}

func TestLoadFromPostgres(t *testing.T) {
	clearEnv()
	dsn := testPGDSN(t)
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	// Chuẩn bị dữ liệu test (xoá sạch trước)
	_, _ = pool.Exec(ctx, "DELETE FROM app_config")
	_, _ = pool.Exec(ctx, `INSERT INTO app_config (key, value) VALUES ('server.port', '6600'), ('app.env', 'staging'), ('jwt.access_ttl', '90m')`)

	t.Setenv("CONFIG_DSN", dsn)
	mgr, err := Load("")
	if err != nil {
		t.Fatalf("Load(postgres DSN) error = %v", err)
	}

	if mgr.Cfg.Server.Port != "6600" {
		t.Errorf("Server.Port = %q, want 6600 (từ postgres)", mgr.Cfg.Server.Port)
	}
	if mgr.Cfg.App.Env != "staging" {
		t.Errorf("App.Env = %q, want staging", mgr.Cfg.App.Env)
	}
	if got := mgr.Cfg.JWT.AccessTTL.String(); got != "1h30m0s" {
		t.Errorf("JWT.AccessTTL = %s, want 1h30m0s", got)
	}
	// Key không trong DB → giữ default
	if mgr.Cfg.Database.URL == "" {
		t.Error("Database.URL rỗng — phải giữ default")
	}

	// Dọn dữ liệu test
	_, _ = pool.Exec(ctx, "DELETE FROM app_config")
}

func TestFlattenConfig(t *testing.T) {
	in := map[string]any{
		"server": map[string]any{"port": "1234", "host": "0.0.0.0"},
		"app":    map[string]any{"env": "dev"},
		"top":    "value",
	}
	out := map[string]string{}
	flattenConfig("", in, out)

	want := map[string]string{
		"server.port": "1234",
		"server.host": "0.0.0.0",
		"app.env":     "dev",
		"top":         "value",
	}
	if fmt.Sprintf("%v", out) != fmt.Sprintf("%v", want) {
		t.Errorf("flattenConfig = %v, want %v", out, want)
	}
}
