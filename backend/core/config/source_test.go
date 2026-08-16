package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDSN_FileDefaultMissing(t *testing.T) {
	clearEnv()
	// Không có config.yml ở pwd test → bỏ qua, dùng defaults
	mgr, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mgr.Cfg.Server.Port != "3330" {
		t.Errorf("Server.Port = %q, want 3330 (default, không có file)", mgr.Cfg.Server.Port)
	}
}

func TestLoadFromDSN_FileOverride(t *testing.T) {
	clearEnv()
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yml")
	content := "server:\n  port: \"7700\"\n  host: 0.0.0.0\napp:\n  env: production\njwt:\n  access_ttl: 45m\n"
	if err := os.WriteFile(cfgFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	dsn := "file://" + cfgFile
	t.Setenv("CONFIG_DSN", dsn)
	mgr, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mgr.Cfg.Server.Port != "7700" {
		t.Errorf("Server.Port = %q, want 7700 (từ file DSN)", mgr.Cfg.Server.Port)
	}
	if mgr.Cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want 0.0.0.0", mgr.Cfg.Server.Host)
	}
	if mgr.Cfg.App.Env != "production" {
		t.Errorf("App.Env = %q, want production", mgr.Cfg.App.Env)
	}
	// Duration từ YAML
	if got := mgr.Cfg.JWT.AccessTTL.String(); got != "45m0s" {
		t.Errorf("JWT.AccessTTL = %s, want 45m0s", got)
	}
	// Key không có trong file → giữ default
	if mgr.Cfg.Database.URL == "" {
		t.Error("Database.URL rỗng — phải giữ default")
	}
}

func TestLoadFromDSN_Invalid(t *testing.T) {
	clearEnv()
	t.Setenv("CONFIG_DSN", "s3://bucket/config")
	if _, err := Load(""); err == nil {
		t.Fatal("DSN không hợp lệ phải trả lỗi")
	}
}

func TestLoadFromDSN_FilePrecedence(t *testing.T) {
	clearEnv()
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgFile, []byte("server:\n  port: \"8800\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// OS env phải thắng file DSN
	t.Setenv("CONFIG_DSN", "file://"+cfgFile)
	t.Setenv("SERVER_PORT", "9900")

	mgr, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mgr.Cfg.Server.Port != "9900" {
		t.Errorf("Server.Port = %q, want 9900 (OS env > file DSN)", mgr.Cfg.Server.Port)
	}
}
