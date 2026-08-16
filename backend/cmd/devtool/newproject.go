package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ducconit/tiktok-live-platform/backend/core/template"
	"github.com/spf13/cobra"
)

// newProjectCmd — tạo project mới từ template tiktok-live-platform.
// Chạy từ root skeleton: go run ./cmd/devtool new:project myapp "My App"
// (Logic nằm ở core/template — sau này tách CLI riêng chỉ cần import package đó.)
func newProjectCmd() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "new:project <name> [\"Title\"]",
		Short: "Tạo project mới từ template (copy + đổi tên + git init)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			title := name
			if len(args) == 2 {
				title = args[1]
			}
			pwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := findTemplateRoot(pwd)
			if err != nil {
				return err
			}
			dir, err := template.NewProject(template.Options{
				Name:        name,
				Title:       title,
				TemplateDir: root,
				TargetDir:   target,
			})
			if err != nil {
				return err
			}

			fmt.Printf("\n✅ Xong! Project mới: %s\n", dir)
			fmt.Println("   Bước tiếp theo:")
			fmt.Println("   1. cd " + name + " && cp .env.example .env && cp config.example.yml config.yml")
			fmt.Println("   2. Sửa JWT_SECRET (chạy: cd backend && go run ./cmd/devtool key:generate)")
			fmt.Println("   3. docker compose up -d --build")
			fmt.Println("   4. make migrate-up && make seed")
			fmt.Println("   5. Mở http://localhost:5173 — đăng nhập admin@example.com")
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "thư mục đích (mặc định: cạnh template)")
	return cmd
}

// findTemplateRoot — đi lên từ pwd tìm thư mục root skeleton
// (có docker-compose.yml — chạy devtool từ backend/ hay root đều được).
func findTemplateRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("không tìm thấy template root (thiếu docker-compose.yml từ %s trở lên)", start)
		}
		dir = parent
	}
}
