package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/internal/user"
	"github.com/spf13/cobra"
)

// makeAdminCmd — tạo admin từ CLI (giống `php artisan make:admin`).
//
//	# Tạo admin với nhiều role (slug, cách nhau bằng dấu phẩy)
//	go run ./cmd/devtool make:admin --name "Quản trị" --email admin@example.com --roles admin,editor
//
// Mặc định (Makefile `make admin`): --email admin@example.com --password admin123 --roles admin.
func makeAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make:admin",
		Short: "Tạo tài khoản admin (email xác thực ngay, gán nhiều role trong 1 transaction)",
		Example: `  go run ./cmd/devtool make:admin --name "Quản trị" --email admin@example.com --roles admin,editor
  make admin                                # default admin@example.com / admin123`,
	}
	name := cmd.Flags().String("name", "", "Họ tên (mặc định \"Admin\")")
	email := cmd.Flags().String("email", "", "Email (bắt buộc; make admin mặc định admin@example.com)")
	roles := cmd.Flags().String("roles", "admin", "Slug vai trò, phân tách bằng dấu phẩy (mặc định admin)")
	password := cmd.Flags().String("password", "", "Mật khẩu (tối thiểu 8 ký tự; mặc định SEED_ADMIN_PASSWORD)")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if *email == "" {
			return fmt.Errorf("--email là bắt buộc (vd: make:admin --email admin@example.com)")
		}
		p := *password
		if p == "" {
			p = os.Getenv("SEED_ADMIN_PASSWORD")
		}
		if p == "" {
			p = "admin123" // khớp Makefile `make admin` default
		}

		cfg, err := loadCfg()
		if err != nil {
			return err
		}
		pool, err := database.NewPool(cmd.Context(), cfg.Cfg.Database)
		if err != nil {
			return err
		}
		defer pool.Close()

		return user.CreateAdminFromCLI(context.Background(), pool, user.AdminCLIParams{
			Email:    *email,
			Password: p,
			FullName: *name,
			Roles:    splitRoles(*roles),
		})
	}
	return cmd
}

// splitRoles — "admin,editor" → ["admin", "editor"] (bỏ rỗng/space).
func splitRoles(s string) []string {
	var out []string
	for _, r := range strings.Split(s, ",") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}
