// Package commands — các lệnh CLI của binary app (key:generate, migrate...).
// Mỗi lệnh 1 file; cmd/app/main.go chỉ là entry point (wiring).
package commands

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

// key:generate — giống `php artisan key:generate` (laravel):
//   - Sinh JWT_SECRET mới (32 bytes → hex 64 ký tự)
//   - Tìm key hiện có: file config (nếu CONFIG_DSN là file) → .env
//   - Đã có key → HỎI xác nhận thay thế (trừ khi --force)
//   - Ghi vào .env (env ưu tiên cao nhất — luôn thắng mọi nguồn khác)

var jwtSecretLine = regexp.MustCompile(`(?m)^JWT_SECRET=(.*)$`)

// devDefaultSecret — giá trị default trong code (xem core/config defaults).
const devDefaultSecret = "dev-secret-change-me"

func KeygenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key:generate",
		Short: "Sinh JWT_SECRET mới và ghi vào .env (chạy trên production sau khi deploy)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			envFile := cmd.Flags().Lookup("env").Value.String()
			force, _ := cmd.Flags().GetBool("force")

			// Đọc config để biết nguồn (file config có chứa jwt.secret không)
			mgr, err := config.Load(envFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Key hiện có? → hỏi xác nhận thay thế (giống laravel)
			if existing := findExistingKey(envFile, mgr); existing != "" && !force {
				if !confirmReplace() {
					fmt.Println("Đã huỷ — giữ nguyên key hiện tại.")
					return nil
				}
			}

			buf := make([]byte, 32)
			if _, err := rand.Read(buf); err != nil {
				return fmt.Errorf("random: %w", err)
			}
			secret := hex.EncodeToString(buf)

			if err := writeEnvSecret(envFile, secret); err != nil {
				return err
			}
			fmt.Println("Application key set successfully (JWT_SECRET).")
			fmt.Printf("JWT_SECRET=%s\n", secret)
			return nil
		},
	}
	cmd.Flags().String("env", ".env", "đường dẫn file .env để ghi JWT_SECRET")
	cmd.Flags().Bool("force", false, "ghi đè key hiện tại KHÔNG hỏi xác nhận")
	return cmd
}

// findExistingKey — tìm key đang dùng theo thứ tự: file config (nếu DSN là file) → .env.
// Trả "" nếu chưa có key thật (chỉ có default trong code).
func findExistingKey(envFile string, mgr *config.Manager) string {
	// 1) File config (CONFIG_DSN = file://...)
	if dsn := mgr.K.String("config.dsn"); strings.HasPrefix(dsn, "file://") {
		path := strings.TrimPrefix(dsn, "file://")
		if path != "" {
			if s := readYAMLSecret(path); s != "" && s != devDefaultSecret {
				return s
			}
		}
	}
	// 2) File .env
	if data, err := os.ReadFile(envFile); err == nil {
		if m := jwtSecretLine.FindSubmatch(data); len(m) > 1 {
			s := strings.TrimSpace(string(m[1]))
			if s != "" && s != devDefaultSecret {
				return s
			}
		}
	}
	return ""
}

// readYAMLSecret — đọc jwt.secret từ file YAML config (không load toàn bộ).
func readYAMLSecret(path string) string {
	tmp := koanf.New(".")
	if err := tmp.Load(file.Provider(path), yaml.Parser()); err != nil {
		return ""
	}
	return tmp.String("jwt.secret")
}

// confirmReplace — hỏi y/N trên terminal (giống laravel "Are you sure?").
func confirmReplace() bool {
	fmt.Print("JWT_SECRET đã tồn tại. Thay thế bằng key mới? [y/N]: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

// writeEnvSecret — thay/append dòng JWT_SECRET= trong file .env (tạo file nếu thiếu).
func writeEnvSecret(path, secret string) error {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("đọc %s: %w", path, err)
	}

	line := "JWT_SECRET=" + secret
	if jwtSecretLine.Match(content) {
		content = jwtSecretLine.ReplaceAll(content, []byte(line))
	} else {
		// Thêm vào cuối, đảm bảo xuống dòng sạch
		if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
			content = append(content, '\n')
		}
		content = append(content, line+"\n"...)
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("ghi %s: %w", path, err)
	}
	return nil
}
