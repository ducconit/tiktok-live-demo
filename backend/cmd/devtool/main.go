package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/internal/user"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "devtool",
		Short: "CLI dev tools tiktok-live-platform (make:migration, seed, user, make:crud, config)",
	}

	root.AddCommand(makeMigrationCmd(), seedCmd(), userCmd(), makeAdminCmd(), makeCrudCmd(), configCmd(), configImportCmd(), newProjectCmd())

	if err := root.Execute(); err != nil {
		slog.Error("cli", "err", err)
		os.Exit(1)
	}
}

// loadCfg — config chung cho mọi lệnh (đọc .env nếu có + env shell).
func loadCfg() (*config.Manager, error) {
	return config.Load(".env")
}

// ---- make:migration ----

// makeMigrationCmd — tạo file migration mới bằng goose v3 (đúng chuẩn).
//
//	go run ./cmd/devtool make:migration create_orders
//	→ migrations/00009_create_orders.sql (version = max(DB) + 1, pad 5 số)
func makeMigrationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:migration <tên>",
		Short: "Tạo file migration mới (goose v3 — migrations/0000N_<tên>.sql)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			// Chuẩn tên: snake_case, lowercase
			name := strings.ToLower(strings.TrimSpace(args[0]))
			name = strings.ReplaceAll(name, " ", "_")
			if name == "" {
				return fmt.Errorf("tên migration không hợp lệ: %q", args[0])
			}
			path, err := database.CreateMigration(cmd.Context(), cfg.Cfg.Database.URL, name)
			if err != nil {
				return err
			}
			fmt.Println("Migration đã tạo:", path)
			fmt.Println("→ Mở file và viết SQL Up/Down giữa các annotation -- +goose Up / -- +goose Down")
			return nil
		},
	}
}

// ---- seed ----

func seedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Tạo tài khoản system_admin (SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			ctx := context.Background()
			pool, err := database.NewPool(ctx, cfg.Cfg.Database)
			if err != nil {
				return err
			}
			defer pool.Close()
			return database.SeedAdmin(ctx, pool, os.Getenv("SEED_ADMIN_EMAIL"), os.Getenv("SEED_ADMIN_PASSWORD"))
		},
	}
}

// ---- user:create ----

func userCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "user:create", Short: "Tạo user từ CLI", Args: cobra.ExactArgs(1)}
	email := cmd.Flags().String("email", "", "Email user")
	password := cmd.Flags().String("password", "", "Mật khẩu (mặc định SEED_ADMIN_PASSWORD)")
	roleSlug := cmd.Flags().String("role", "admin", "Slug vai trò (mặc định admin)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		e := *email
		if e == "" {
			e = args[0]
		}
		p := *password
		if p == "" {
			p = os.Getenv("SEED_ADMIN_PASSWORD")
		}
		ctx := context.Background()
		pool, err := database.NewPool(ctx, cfg.Cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()
		return user.CreateFromCLI(ctx, pool, e, p, *roleSlug)
	}
	return cmd
}
