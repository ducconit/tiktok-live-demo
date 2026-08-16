// Package template — tạo project mới từ skeleton (copy + đổi tên + git init).
// Nằm ở core/ để sau này tách thành CLI riêng chỉ cần import package này.
package template

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Options — tham số tạo project mới.
type Options struct {
	Name        string // bắt buộc: chữ thường, số, dấu gạch ngang (vd: my-app)
	Title       string // tên hiển thị (branding dashboard); rỗng = Name
	TemplateDir string // thư mục template (thường là pwd khi chạy devtool)
	TargetDir   string // nơi tạo project (mặc định: cạnh TemplateDir)
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Placeholder skeleton — ghép chuỗi để KHÔNG bị replace khi template tự nhân bản
// (literal thuần "tiktok-live-platform" trong code này sẽ bị new:project đổi tên → gãy).
var (
	skeletonName = "tiktok-live-" + "platform"
	skeletonDB   = "tiktok_live_" + "platform"
)

// exclusions — thư mục/file không copy (git, deps, build, secret).
var exclusions = map[string]bool{
	".git":         true,
	"node_modules": true,
	"dist":         true,
	"tmp":          true,
	".env":         true,
	"config.yml":   true,
}

// NewProject — copy template, đổi placeholder, git init + commit đầu tiên.
func NewProject(o Options) (string, error) {
	if o.Name == "" {
		return "", fmt.Errorf("thiếu tên project — vd: devtool new:project myapp")
	}
	if !nameRe.MatchString(o.Name) {
		return "", fmt.Errorf("tên project chỉ gồm chữ thường, số và dấu gạch ngang (vd: myapp, my-app)")
	}
	if o.Title == "" {
		o.Title = o.Name
	}
	if o.TemplateDir == "" {
		return "", fmt.Errorf("thiếu TemplateDir")
	}
	if o.TargetDir == "" {
		o.TargetDir = filepath.Join(filepath.Dir(o.TemplateDir), o.Name)
	}
	if _, err := os.Stat(o.TargetDir); err == nil {
		return "", fmt.Errorf("%s đã tồn tại", o.TargetDir)
	}

	dbName := strings.ReplaceAll(o.Name, "-", "_")

	// 1) Copy cây thư mục (bỏ exclusions)
	if err := copyTree(o.TemplateDir, o.TargetDir); err != nil {
		return "", fmt.Errorf("copy template: %w", err)
	}

	// 2) Đổi placeholder
	if err := replaceInTree(o.TargetDir, map[string]string{
		skeletonName:    o.Name,
		skeletonDB:      dbName,
		"{{APP_TITLE}}": o.Title,
	}); err != nil {
		return "", fmt.Errorf("đổi placeholder: %w", err)
	}

	// 3) git init + commit đầu tiên
	gitInit(o.TargetDir)

	return o.TargetDir, nil
}

// copyTree — copy toàn bộ src → dst, bỏ exclusions.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if exclusions[d.Name()] || exclusions[rel] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// replaceInTree — thay placeholder trong mọi file text (chỉ ghi khi file có chứa).
func replaceInTree(root string, repl map[string]string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		changed := false
		for from, to := range repl {
			if strings.Contains(content, from) {
				content = strings.ReplaceAll(content, from, to)
				changed = true
			}
		}
		if changed {
			return os.WriteFile(path, []byte(content), d.Type().Perm())
		}
		return nil
	})
}

// gitInit — git init + commit đầu tiên; lỗi commit (thiếu identity) chỉ warning.
func gitInit(dir string) {
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Stdout = nil
		cmd.Stderr = nil
		return cmd.Run()
	}
	if err := run("init", "-b", "main", "-q"); err != nil {
		slog.Warn("git init thất bại (bỏ qua)", "err", err)
		return
	}
	if err := run("add", "-A"); err == nil {
		msg := "chore: init from " + skeletonName + " template"
		if err := run("commit", "-qm", msg); err != nil {
			slog.Warn("git commit đầu tiên bỏ qua (thiếu git identity?)", "err", err)
		}
	}
}
