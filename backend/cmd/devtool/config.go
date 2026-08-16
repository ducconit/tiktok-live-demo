package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

// configImportCmd — đổ file YAML (mặc định config.yml) vào bảng app_config,
// dùng cho CONFIG_DSN=postgres://... — devtool config:import [file.yml]
func configImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config:import [file.yml]",
		Short: "Đổ config YAML vào bảng app_config (cho CONFIG_DSN=postgres://)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := "config.yml"
			if len(args) == 1 {
				file = args[0]
			}
			mgr, err := loadCfg()
			if err != nil {
				return err
			}

			ctx := context.Background()
			pool, err := database.NewPool(ctx, mgr.Cfg.Database)
			if err != nil {
				return err
			}
			defer pool.Close()

			// Đọc YAML → map phẳng key.path = value
			kv, err := readConfigYAML(file)
			if err != nil {
				return err
			}
			if len(kv) == 0 {
				return fmt.Errorf("file %s rỗng hoặc không có key nào", file)
			}

			tx, err := pool.Write().Begin(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = tx.Rollback(ctx) }()

			for key, value := range kv {
				if _, err := tx.Exec(ctx,
					`INSERT INTO app_config (key, value, updated_at) VALUES ($1, $2, now())
					 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
					key, value); err != nil {
					return fmt.Errorf("upsert %s: %w", key, err)
				}
			}
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			slog.Info("config imported", "file", file, "keys", len(kv))
			return nil
		},
	}
}

// readConfigYAML — đọc YAML, flatten thành map phẳng key.path = string value.
func readConfigYAML(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("đọc %s: %w", path, err)
	}
	k := koanf.New(".")
	if err := k.Load(rawbytes.Provider(raw), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return config.FlattenConfig(k.All()), nil
}

func configCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config:set <key> <value>",
		Short: "Đổi dynamic config (sync mọi instance qua Redis pub/sub)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := loadCfg()
			if err != nil {
				return err
			}
			key, value := args[0], args[1]

			// Kết nối Redis để bật dynamic sync (client từ DSN)
			ctx := context.Background()
			rdb, err := mgr.Cfg.Redis.Client()
			if err != nil {
				return fmt.Errorf("redis DSN sai: %w", err)
			}
			defer func() { _ = rdb.Close() }()
			if err := rdb.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("redis không kết nối được (%s) — dynamic sync bắt buộc cần redis", mgr.Cfg.Redis.URL)
			}

			if err := mgr.InitDynamic(ctx, rdb); err != nil {
				return err
			}
			defer mgr.Close()

			if err := mgr.SetDynamic(key, value); err != nil {
				return err
			}
			slog.Info("config đã cập nhật + broadcast", "key", key, "value", value, "channel", mgr.Channel())
			return nil
		},
	}
}
