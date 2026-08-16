package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalDisk — driver lưu trên đĩa (thư mục root).
//   - public: URL = cfg.URL + name (serve qua route /storage — xem server.go)
//   - private: URL = "" (không công khai — chỉ backend truy cập)
type LocalDisk struct {
	name string
	root string // đường dẫn tuyệt đối (normalize lúc New)
	url  string // prefix URL công khai ("" = private)
}

// NewLocalDisk — root tương đối → tuyệt đối; tự tạo thư mục root nếu chưa có.
func NewLocalDisk(name, root, url string) (*LocalDisk, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("local disk %q: root %q: %w", name, root, err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("local disk %q: tạo root: %w", name, err)
	}
	return &LocalDisk{name: name, root: abs, url: url}, nil
}

func (d *LocalDisk) Name() string { return d.name }

// Root — đường dẫn tuyệt đối (dùng để serve /storage public).
func (d *LocalDisk) Root() string { return d.root }

// path — map tên file → đường dẫn tuyệt đối, chặn path traversal.
func (d *LocalDisk) path(name string) (string, error) {
	if err := validatePath(name); err != nil {
		return "", err
	}
	p := filepath.Join(d.root, filepath.FromSlash(name))
	// Double-check: phải nằm TRONG root (filepath.Join đã chuẩn hoá ../)
	if !strings.HasPrefix(p, d.root) {
		return "", ErrInvalidPath
	}
	return p, nil
}

func (d *LocalDisk) Put(name string, content []byte) error {
	p, err := d.path(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("local disk: tạo thư mục: %w", err)
	}
	return os.WriteFile(p, content, 0o644)
}

func (d *LocalDisk) Get(name string) ([]byte, error) {
	p, err := d.path(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func (d *LocalDisk) Delete(name string) error {
	p, err := d.path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (d *LocalDisk) Exists(name string) bool {
	p, err := d.path(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

func (d *LocalDisk) Size(name string) (int64, error) {
	p, err := d.path(name)
	if err != nil {
		return 0, err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// URL — public disk: url prefix + name; private: "" (không công khai).
func (d *LocalDisk) URL(name string) string {
	if d.url == "" {
		return ""
	}
	return d.url + "/" + name
}

// TemporaryURL — local không presign: trả URL công khai ("" nếu private).
func (d *LocalDisk) TemporaryURL(name string, _ time.Duration) (string, error) {
	return d.URL(name), nil
}

// Health — root tồn tại + ghi được (tạo/xoá file temp).
func (d *LocalDisk) Health(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	tmp, err := os.CreateTemp(d.root, ".health-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	_ = tmp.Close()
	return os.Remove(name)
}
