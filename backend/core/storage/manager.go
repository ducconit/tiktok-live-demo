package storage

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
)

// Manager — registry các disk + disk mặc định (Laravel Storage facade).
//
//	storage.NewManager(cfg) → m.Disk("public") / m.Disk("") = default
//
// Local disk init NGAY (rẻ, config sai phải lộ lúc boot); s3 disk init LAZY
// (kết nối khi Disk() gọi lần đầu — app không chết khi MinIO/S3 đang down,
// lỗi hiện rõ ở lần dùng đầu tiên).
type Manager struct {
	defaultDisk string
	disks       map[string]Disk              // đã init (local eager + s3 sau lần dùng đầu)
	s3Cfg       map[string]config.DiskConfig // s3 chưa init (lazy)
	mu          sync.Mutex
}

// NewManager — build local disks + đăng ký s3 (chưa kết nối).
func NewManager(cfg config.StorageConfig) (*Manager, error) {
	m := &Manager{
		defaultDisk: cfg.DefaultDisk,
		disks:       make(map[string]Disk, len(cfg.Disks)),
		s3Cfg:       make(map[string]config.DiskConfig),
	}
	if m.defaultDisk == "" {
		m.defaultDisk = "public"
	}

	for name, dc := range cfg.Disks {
		if dc.Driver == "s3" {
			// Lazy — chưa kết nối (MinIO có thể đang down; app vẫn boot)
			m.s3Cfg[name] = dc
			slog.Debug("storage s3 disk registered (lazy)", "disk", name)
			continue
		}
		disk, err := newDisk(name, dc)
		if err != nil {
			return nil, fmt.Errorf("storage: disk %q: %w", name, err)
		}
		m.disks[name] = disk
		slog.Debug("storage disk ready", "disk", name, "driver", dc.Driver)
	}
	if _, ok := m.disks[m.defaultDisk]; !ok {
		// default có thể là s3 (chưa init) — kiểm tra cả 2 nguồn
		if _, ok := m.s3Cfg[m.defaultDisk]; !ok {
			return nil, fmt.Errorf("storage: default disk %q không có trong storage.disks", m.defaultDisk)
		}
	}
	return m, nil
}

// newDisk — chọn driver theo config (local | s3).
func newDisk(name string, dc config.DiskConfig) (Disk, error) {
	switch dc.Driver {
	case "", "local":
		return NewLocalDisk(name, dc.Root, dc.URL)
	case "s3":
		return NewS3Disk(name, dc)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedDriver, dc.Driver)
	}
}

// Disk — lấy disk theo tên; name rỗng → default disk.
// s3 disk được kết nối ở lần gọi đầu (lỗi hiện tại đây, không cache — lần sau retry).
// Lỗi ErrDiskNotFound nếu không tồn tại.
func (m *Manager) Disk(name string) (Disk, error) {
	if name == "" {
		name = m.defaultDisk
	}
	if d, ok := m.disks[name]; ok {
		return d, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.disks[name]; ok { // double-check (race giữa 2 caller)
		return d, nil
	}
	dc, ok := m.s3Cfg[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrDiskNotFound, name)
	}
	d, err := newDisk(name, dc)
	if err != nil {
		return nil, err
	}
	m.disks[name] = d
	slog.Info("storage s3 disk connected", "disk", name)
	return d, nil
}

// DefaultDiskName — tên disk mặc định (config storage.default_disk).
func (m *Manager) DefaultDiskName() string {
	return m.defaultDisk
}

// DiskNames — danh sách tên disk (sắp xếp — deterministic, dùng cho log/test).
func (m *Manager) DiskNames() []string {
	names := make(map[string]bool, len(m.disks)+len(m.s3Cfg))
	for n := range m.disks {
		names[n] = true
	}
	for n := range m.s3Cfg {
		names[n] = true
	}
	out := make([]string, 0, len(names))
	for n := range names {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Health — health của 1 disk theo tên (metrics service_up; không tồn tại → lỗi).
func (m *Manager) Health(ctx context.Context, name string) error {
	d, err := m.Disk(name)
	if err != nil {
		return err
	}
	return d.Health(ctx)
}
