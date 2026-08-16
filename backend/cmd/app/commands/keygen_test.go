package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteEnvSecret_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, writeEnvSecret(path, "secret-1"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "JWT_SECRET=secret-1\n", string(data))
}

func TestWriteEnvSecret_ReplacesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte("APP_ENV=development\nJWT_SECRET=old-secret\nOTHER=1\n"), 0o600))

	require.NoError(t, writeEnvSecret(path, "new-secret"))
	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), "JWT_SECRET=new-secret\n")
	assert.NotContains(t, string(data), "old-secret")
	assert.Contains(t, string(data), "APP_ENV=development", "giữ nguyên các dòng khác")
	assert.Contains(t, string(data), "OTHER=1")
}

func TestWriteEnvSecret_AppendsWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte("APP_ENV=production"), 0o600)) // không có newline cuối

	require.NoError(t, writeEnvSecret(path, "s3"))
	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), "APP_ENV=production\nJWT_SECRET=s3\n")
}

func TestFindExistingKey_FromEnvFile(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(envFile, []byte("JWT_SECRET=env-secret\n"), 0o600))

	// Không có config file → đọc .env
	mgr, err := config.Load("") // defaults
	require.NoError(t, err)
	assert.Equal(t, "env-secret", findExistingKey(envFile, mgr))
}

func TestFindExistingKey_ConfigFileFirst(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(envFile, []byte("JWT_SECRET=env-secret\n"), 0o600))

	cfgFile := filepath.Join(dir, "config.yml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("jwt:\n  secret: yaml-secret\n"), 0o600))

	// CONFIG_DSN trỏ file config → ưu tiên secret từ config file
	t.Setenv("CONFIG_DSN", "file://"+cfgFile)
	mgr, err := config.Load("")
	require.NoError(t, err)
	assert.Equal(t, "yaml-secret", findExistingKey(envFile, mgr))
}

func TestFindExistingKey_ReturnsEmptyWhenOnlyDefault(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(envFile, []byte("JWT_SECRET=dev-secret-change-me\n"), 0o600))

	mgr, err := config.Load("")
	require.NoError(t, err)
	assert.Equal(t, "", findExistingKey(envFile, mgr), "default trong code không tính là key thật")
}
