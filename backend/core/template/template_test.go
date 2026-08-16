package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeTemplate — dựng 1 template tạm nhỏ với đủ placeholder.
func makeTemplate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":              "module github.com/ducconit/tiktok-live-platform/backend\n",
		".env.example":        "POSTGRES_DB=tiktok_live_platform\nAPP_NAME=tiktok-live-platform\n",
		"dashboard/app.ts":    "export const APP_TITLE = '{{APP_TITLE}}'\n",
		"backend/core/x.go":   "package x\n// tạo từ tiktok-live-platform\n",
		"keep/plain.txt":      "không có placeholder\n",
		".env":                "SECRET=không được copy\n",
		"node_modules/pkg.js": "giả lập dep\n",
		"dist/bundle.js":      "giả lập build\n",
	}
	for path, content := range files {
		fp := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestNewProject(t *testing.T) {
	tpl := makeTemplate(t)
	target := filepath.Join(t.TempDir(), "myapp")

	got, err := NewProject(Options{
		Name:        "myapp",
		Title:       "My App",
		TemplateDir: tpl,
		TargetDir:   target,
	})
	if err != nil {
		t.Fatalf("NewProject() error = %v", err)
	}
	if got != target {
		t.Errorf("got dir = %s, want %s", got, target)
	}

	// Module path đã đổi
	gomod, _ := os.ReadFile(filepath.Join(target, "go.mod"))
	if !strings.Contains(string(gomod), "github.com/ducconit/myapp/backend") {
		t.Errorf("go.mod chưa đổi tên: %s", gomod)
	}
	// DB name đổi
	envEx, _ := os.ReadFile(filepath.Join(target, ".env.example"))
	if !strings.Contains(string(envEx), "POSTGRES_DB=myapp") {
		t.Errorf(".env.example chưa đổi DB: %s", envEx)
	}
	// Title đổi
	appTS, _ := os.ReadFile(filepath.Join(target, "dashboard/app.ts"))
	if !strings.Contains(string(appTS), "'My App'") {
		t.Errorf("APP_TITLE chưa đổi: %s", appTS)
	}
	// Không còn placeholder sót
	if _, err := os.Stat(filepath.Join(target, ".env")); err == nil {
		t.Error(".env phải bị loại khỏi copy")
	}
	if _, err := os.Stat(filepath.Join(target, "node_modules")); err == nil {
		t.Error("node_modules phải bị loại khỏi copy")
	}
	if _, err := os.Stat(filepath.Join(target, "dist")); err == nil {
		t.Error("dist phải bị loại khỏi copy")
	}
	// Git init đã chạy
	if _, err := os.Stat(filepath.Join(target, ".git")); err != nil {
		t.Error("thiếu .git — chưa git init")
	}
}

func TestNewProject_InvalidName(t *testing.T) {
	for _, name := range []string{"MyApp", "my_app", "1app", ""} {
		if _, err := NewProject(Options{Name: name, TemplateDir: t.TempDir(), TargetDir: filepath.Join(t.TempDir(), name)}); err == nil {
			t.Errorf("tên %q phải bị từ chối", name)
		}
	}
}

func TestNewProject_TargetExists(t *testing.T) {
	tpl := makeTemplate(t)
	target := t.TempDir() // đã tồn tại
	if _, err := NewProject(Options{Name: "myapp", TemplateDir: tpl, TargetDir: target}); err == nil {
		t.Fatal("target đã tồn tại phải trả lỗi")
	}
}
