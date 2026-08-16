package storage

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// s3TestConfig — đọc config từ env (chạy thật với MinIO local/CI):
//
//	MINIO_ENDPOINT (mặc định localhost:9000), MINIO_ACCESS_KEY, MINIO_SECRET_KEY,
//	MINIO_BUCKET (mặc định uploads-test — không đụng bucket thật).
func s3TestConfig(t *testing.T) (config.DiskConfig, bool) {
	t.Helper()
	endpoint := os.Getenv("MINIO_ENDPOINT")
	access := os.Getenv("MINIO_ACCESS_KEY")
	secret := os.Getenv("MINIO_SECRET_KEY")
	if endpoint == "" || access == "" || secret == "" {
		return config.DiskConfig{}, false
	}
	return config.DiskConfig{
		Driver: "s3", Endpoint: endpoint, AccessKey: access, SecretKey: secret,
		Bucket: os.Getenv("MINIO_BUCKET"),
	}, true
}

// TestS3Disk_Integration — verify thật với S3-compatible (MinIO).
// Skip khi không có MINIO_* env (CI không có MinIO — case đã biết, chưa kiểm chứng ở CI).
func TestS3Disk_Integration(t *testing.T) {
	cfg, ok := s3TestConfig(t)
	if !ok {
		t.Skip("thiếu MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY — skip integration (case: S3 Put/Get/Delete; chưa kiểm chứng ở CI)")
	}

	d, err := NewS3Disk("s3-test", cfg)
	require.NoError(t, err)

	key := "test/unit-" + strings.ReplaceAll(time.Now().Format("150405.000"), ".", "") + ".txt"

	// Put → Get → Size → Exists → URL → TemporaryURL → Delete
	require.NoError(t, d.Put(key, []byte("s3 hello")))
	got, err := d.Get(key)
	require.NoError(t, err)
	assert.Equal(t, "s3 hello", string(got))

	size, err := d.Size(key)
	require.NoError(t, err)
	assert.Equal(t, int64(8), size)

	assert.True(t, d.Exists(key))
	assert.True(t, strings.Contains(d.URL(key), cfg.Bucket+"/"+key))

	u, err := d.TemporaryURL(key, time.Minute)
	require.NoError(t, err)
	assert.True(t, strings.Contains(u, "X-Amz-Signature"), "presigned URL phải có signature")

	require.NoError(t, d.Delete(key))
	assert.False(t, d.Exists(key))

	// Health
	require.NoError(t, d.Health(context.Background()))
}

// TestS3Disk_PathTraversal — chặn trước khi gọi S3.
func TestS3Disk_PathTraversal(t *testing.T) {
	d := &S3Disk{} // không cần client — validatePath chặn trước
	err := d.Put("../evil.txt", []byte("x"))
	assert.ErrorIs(t, err, ErrInvalidPath)
	_, err = d.Get("/abs.txt")
	assert.ErrorIs(t, err, ErrInvalidPath)
}
