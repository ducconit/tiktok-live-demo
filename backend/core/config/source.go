package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/retry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ============================================================
// CONFIG_DSN — nguồn cấu hình chính (kiểu "data source name"):
//
//	file://config.yml       file YAML ở pwd (MẶC ĐỊNH, zero-config)
//	postgres://user:pass@host:5432/db   config lưu trong bảng app_config
//
// Mọi nguồn khác (defaults trong code, .env, OS env) là override/fallback.
// ============================================================

// DefaultDSN — file config.yml ở thư mục chạy lệnh.
const (
	DefaultDSN          = "file://config.yml"
	configSourceTimeout = 10 * time.Second // giới hạn đọc config từ postgres
)

// AppConfigTable — bảng lưu config khi dùng postgres DSN.
const AppConfigTable = "app_config"

// loadFromDSN — nạp cấu hình từ CONFIG_DSN vào koanf instance.
// File không tồn tại → bỏ qua (zero-config); DSN lỗi → trả lỗi (không nuốt).
func loadFromDSN(k *koanf.Koanf, dsn string) error {
	if dsn == "" {
		dsn = DefaultDSN
	}
	switch {
	case strings.HasPrefix(dsn, "file://"):
		path := strings.TrimPrefix(dsn, "file://")
		if path == "" {
			path = "config.yml"
		}
		if _, err := os.Stat(path); err != nil {
			slog.Info("config: file DSN không tồn tại — dùng defaults", "path", path)
			return nil
		}
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return fmt.Errorf("config: load %s: %w", path, err)
		}
		slog.Info("config: loaded from file", "path", path)
		return nil

	case strings.HasPrefix(dsn, "postgres://"), strings.HasPrefix(dsn, "postgresql://"):
		if err := loadFromPostgres(k, dsn); err != nil {
			return fmt.Errorf("config: load postgres: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("CONFIG_DSN không hợp lệ %q — hỗ trợ file://path hoặc postgres://", dsn)
	}
}

// loadFromPostgres — đọc config từ bảng app_config(key, value) → merge vào koanf.
// Retry khi postgres chưa sẵn sàng (khởi động cùng compose) — không chết ngay.
func loadFromPostgres(k *koanf.Koanf, dsn string) error {
	return retry.Do(context.Background(), retry.Config{Attempts: 10, InitialWait: 2 * time.Second}, "config postgres", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), configSourceTimeout)
		defer cancel()

		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer pool.Close()
		if err := pool.Ping(ctx); err != nil {
			return fmt.Errorf("ping: %w", err)
		}

		rows, err := pool.Query(ctx, "SELECT key, value FROM "+AppConfigTable)
		if err != nil {
			return fmt.Errorf("query %s: %w", AppConfigTable, err)
		}
		defer rows.Close()

		m := map[string]any{}
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			m[key] = value
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("rows: %w", err)
		}
		if len(m) == 0 {
			slog.Warn("config: bảng "+AppConfigTable+" rỗng — dùng defaults", "dsn", dsn)
			return nil
		}

		if err := k.Load(confmap.Provider(m, "."), nil); err != nil {
			return fmt.Errorf("merge: %w", err)
		}
		slog.Info("config: loaded from postgres", "keys", len(m))
		return nil
	})
}

// FlattenConfig — ép map nested thành map phẳng key.path = string value.
// Dùng cho config:import (YAML → bảng app_config).
func FlattenConfig(m map[string]any) map[string]string {
	out := map[string]string{}
	flattenConfig("", m, out)
	return out
}

func flattenConfig(prefix string, m map[string]any, out map[string]string) {
	for key, val := range m {
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		if nested, ok := val.(map[string]any); ok {
			flattenConfig(full, nested, out)
			continue
		}
		out[full] = fmt.Sprintf("%v", val)
	}
}
