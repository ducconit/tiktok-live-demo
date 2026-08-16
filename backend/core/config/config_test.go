package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/knadh/koanf/parsers/json"
)

// clearEnv — xoá mọi env binding (session Hermes export .env trước đó dính vào test).
func clearEnv() {
	for _, b := range envBindings {
		_ = os.Unsetenv(b.env)
	}
}

func TestLoad_DefaultsZeroConfig(t *testing.T) {
	clearEnv()
	// Không có file, không có env — mọi thứ từ defaults (zero-config)
	mgr, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if mgr.Cfg.App.Env != "development" {
		t.Errorf("App.Env = %q, want development", mgr.Cfg.App.Env)
	}
	if mgr.Cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want 127.0.0.1", mgr.Cfg.Server.Host)
	}
	if mgr.Cfg.Server.Port != "3330" {
		t.Errorf("Server.Port = %q, want 3330", mgr.Cfg.Server.Port)
	}
	if mgr.Cfg.JWT.AccessTTL != 15*time.Minute {
		t.Errorf("JWT.AccessTTL = %v, want 15m", mgr.Cfg.JWT.AccessTTL)
	}
	if mgr.Cfg.JWT.Secret == "" {
		t.Error("JWT.Secret rỗng — phải có default cố định (dev-secret-change-me)")
	}
	if mgr.Cfg.JWT.Secret != "dev-secret-change-me" {
		t.Errorf("JWT.Secret = %q, want default cố định (dev) — production phải chạy `app key:generate`", mgr.Cfg.JWT.Secret)
	}
	if mgr.InstanceID() == "" {
		t.Error("InstanceID rỗng")
	}
	// Dynamic default có sẵn
	if got := mgr.Int("server.rate_limit", 100); got != 100 {
		t.Errorf("rate_limit = %d, want 100", got)
	}
}

func TestLoad_EnvFileOverrides(t *testing.T) {
	clearEnv()
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("SERVER_PORT=4444\nAPP_ENV=staging\nJWT_ACCESS_TTL=30m\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	mgr, err := Load(envFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mgr.Cfg.Server.Port != "4444" {
		t.Errorf("Server.Port = %q, want 4444 (từ .env)", mgr.Cfg.Server.Port)
	}
	if mgr.Cfg.App.Env != "staging" {
		t.Errorf("App.Env = %q, want staging", mgr.Cfg.App.Env)
	}
	if mgr.Cfg.JWT.AccessTTL != 30*time.Minute {
		t.Errorf("JWT.AccessTTL = %v, want 30m (từ .env)", mgr.Cfg.JWT.AccessTTL)
	}
}

func TestLoad_EnvVarOverridesFile(t *testing.T) {
	clearEnv()
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("SERVER_PORT=4444\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// OS env phải thắng .env
	t.Setenv("SERVER_PORT", "5555")

	mgr, err := Load(envFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if mgr.Cfg.Server.Port != "5555" {
		t.Errorf("Server.Port = %q, want 5555 (OS env override .env)", mgr.Cfg.Server.Port)
	}
}

func TestLoad_ReplicaURLs(t *testing.T) {
	clearEnv()
	t.Setenv("DATABASE_REPLICAS", "postgres://r1:5432/db, postgres://r2:5432/db, ")
	mgr, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	urls := mgr.Cfg.Database.ReplicaURLs()
	if len(urls) != 2 {
		t.Fatalf("ReplicaURLs() = %v, want 2 URLs (trim + bỏ rỗng)", urls)
	}
}

// TestLoad_EveryFieldHasDefault — zero-config: KHÔNG field nào được zero sau Load
// (trừ allowlist — replicas/mail password/prefix tuỳ chọn).
func TestLoad_EveryFieldHasDefault(t *testing.T) {
	clearEnv()
	mgr, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	allowZero := map[string]bool{
		"Replicas":       true, // database replicas tuỳ chọn
		"Username":       true, // mail username tuỳ chọn
		"Password":       true, // mail password tuỳ chọn
		"Prefix":         true, // cache prefix tuỳ chọn
		"AuthToken":      true, // metrics auth tuỳ chọn ("" = không auth)
		"RateLimitStore": true, // "" = theo cache store (memory|redis)
	}

	var walk func(path string, v reflect.Value)
	walk = func(path string, v reflect.Value) {
		if v.Kind() != reflect.Struct {
			return
		}
		tv := v.Type()
		for i := 0; i < tv.NumField(); i++ {
			f := tv.Field(i)
			if !f.IsExported() {
				continue
			}
			name := f.Name
			if path != "" {
				name = path + "." + f.Name
			}
			fv := v.Field(i)
			switch fv.Kind() {
			case reflect.Struct:
				walk(name, fv)
			case reflect.String:
				if fv.String() == "" && !allowZero[f.Name] {
					t.Errorf("config %s rỗng — thiếu default (zero-config vi phạm)", name)
				}
			case reflect.Int, reflect.Int64:
				if fv.Int() == 0 && !allowZero[f.Name] {
					t.Errorf("config %s = 0 — thiếu default (zero-config vi phạm)", name)
				}
			case reflect.Bool:
				// bool zero = false — hợp lệ (vd use_ssl)
			}
		}
	}
	walk("", reflect.ValueOf(mgr.Cfg))
}

func TestSetDynamic_LocalWithoutRedis(t *testing.T) {
	// Không InitDynamic (không redis) → SetDynamic chỉ ảnh hưởng local
	mgr, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	changed := ""
	mgr.OnChange("feature.x", func(v any) { changed = v.(string) })

	if err := mgr.SetDynamic("feature.x", "on"); err != nil {
		t.Fatalf("SetDynamic() error = %v", err)
	}
	if got := mgr.GetDynamic("feature.x"); got != "on" {
		t.Errorf("GetDynamic = %v, want on", got)
	}
	if changed != "on" {
		t.Errorf("OnChange không được gọi, got %q", changed)
	}
	// Static struct không đổi
	if mgr.Cfg.Server.Port != "3330" {
		t.Errorf("Cfg.Server.Port = %q — dynamic không được đụng static struct", mgr.Cfg.Server.Port)
	}
}

func TestManager_MarshalJSON(t *testing.T) {
	mgr, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	b, err := mgr.Marshal(json.Parser())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if len(b) == 0 {
		t.Fatal("Marshal() rỗng")
	}
}
