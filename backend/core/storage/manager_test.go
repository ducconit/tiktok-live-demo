package storage

import (
	"errors"
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCfg — config đầy đủ: default public + public/private local + s3.
func testCfg(t *testing.T) config.StorageConfig {
	t.Helper()
	return config.StorageConfig{
		DefaultDisk: "public",
		Disks: map[string]config.DiskConfig{
			"public":  {Driver: "local", Root: t.TempDir() + "/public", URL: "/storage"},
			"private": {Driver: "local", Root: t.TempDir() + "/private"},
			"s3":      {Driver: "s3", Endpoint: "localhost:9000", AccessKey: "x", SecretKey: "y", Bucket: "uploads"},
		},
	}
}

func TestNewManager_DefaultDisk(t *testing.T) {
	m, err := NewManager(testCfg(t))
	require.NoError(t, err)
	assert.Equal(t, "public", m.DefaultDiskName())
	assert.ElementsMatch(t, []string{"public", "private", "s3"}, m.DiskNames())

	d, err := m.Disk("") // rỗng → default
	require.NoError(t, err)
	assert.Equal(t, "public", d.Name())
}

func TestNewManager_DiskNotFound(t *testing.T) {
	m, err := NewManager(testCfg(t))
	require.NoError(t, err)
	_, err = m.Disk("nonexistent")
	assert.True(t, errors.Is(err, ErrDiskNotFound))
}

func TestNewManager_DefaultDiskMissing(t *testing.T) {
	cfg := testCfg(t)
	cfg.DefaultDisk = "backups" // không có trong disks → lỗi rõ
	_, err := NewManager(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default disk")
}

func TestNewManager_UnsupportedDriver(t *testing.T) {
	cfg := testCfg(t)
	cfg.Disks["weird"] = config.DiskConfig{Driver: "ftp", Root: "/tmp/x"}
	_, err := NewManager(cfg)
	assert.True(t, errors.Is(err, ErrUnsupportedDriver))
}

func TestNewManager_NoDisks(t *testing.T) {
	_, err := NewManager(config.StorageConfig{DefaultDisk: "public", Disks: map[string]config.DiskConfig{}})
	require.Error(t, err) // default không có → lỗi (config sai phải lộ)
}

func TestManager_Health_UnknownDisk(t *testing.T) {
	m, err := NewManager(testCfg(t))
	require.NoError(t, err)
	err = m.Health(t.Context(), "nope")
	assert.True(t, errors.Is(err, ErrDiskNotFound))
}
