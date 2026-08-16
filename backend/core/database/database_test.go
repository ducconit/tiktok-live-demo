package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/retry"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDBDSN — DSN postgres dev (docker compose, forward 5433).
// Test skip khi không có DB (CI) — case được biết nhưng chưa kiểm chứng được.
func testDBDSN(t *testing.T) string {
	t.Helper()
	if retry.SkipRetry() {
		t.Skip("CI/test không có postgres — skip integration (case đã biết, chưa kiểm chứng)")
	}
	dsn := "postgres://app:app_password_dev@localhost:5433/tiktok_live_platform?sslmode=disable"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := NewPool(ctx, config.DatabaseConfig{URL: dsn})
	if err != nil {
		t.Skipf("postgres không khả dụng (%v) — skip integration", err)
	}
	pool.Close()
	return dsn
}

// TestMigrate_UpDownStatus — migrate lên schema + status.
// KHÔNG chạy down ở đây: migration có thể destructive (00010 đổi type role id —
// down không khôi phục được dữ liệu gốc, test từng làm hỏng id role trên DB dev).
// Down được kiểm chứng thủ công khi cần (goose down + verify).
func TestMigrate_UpDownStatus(t *testing.T) {
	dsn := testDBDSN(t)
	ctx := context.Background()

	require.NoError(t, Migrate(ctx, dsn), "migrate up phải thành công (idempotent — chạy lại cũng OK)")
	require.NoError(t, MigrateStatus(ctx, dsn), "status phải chạy được")
}

// TestCreateMigration_SequentialFile — goose tạo file 0000N đúng chuẩn.
// goose.Create ghi vào dir "migrations" (relative CWD) — test phải chạy từ backend/.
func TestCreateMigration_SequentialFile(t *testing.T) {
	dsn := testDBDSN(t)
	// Test chạy với cwd = core/database → nhảy lên backend/ để có thư mục migrations/
	wd, err := os.Getwd()
	require.NoError(t, err)
	if filepath.Base(wd) == "database" {
		require.NoError(t, os.Chdir("../.."))
		t.Cleanup(func() { _ = os.Chdir(wd) })
	}

	name := "test_create_migration"
	path, err := CreateMigration(context.Background(), dsn, name)
	require.NoError(t, err, "tạo migration phải thành công")
	assert.Contains(t, path, "migrations/")
	t.Cleanup(func() { _ = os.Remove(path) })
}

// TestSeedAdmin_DefaultRolesAndAdmin — seed tạo roles/permissions/admin (idempotent).
func TestSeedAdmin_DefaultRolesAndAdmin(t *testing.T) {
	dsn := testDBDSN(t)
	ctx := context.Background()

	pool, err := NewPool(ctx, config.DatabaseConfig{URL: dsn})
	require.NoError(t, err)
	defer pool.Close()

	require.NoError(t, SeedAdmin(ctx, pool, "seed-admin-test@test.dev", "password123"))
	require.NoError(t, SeedAdmin(ctx, pool, "seed-admin-test@test.dev", "password123"), "seed chạy 2 lần vẫn OK")
}

// TestPool_WithTx_CommitRollback — transaction: rollback khi fn lỗi (cần DB thật).
func TestPool_WithTx_CommitRollback(t *testing.T) {
	dsn := testDBDSN(t)
	ctx := context.Background()
	pool, err := NewPool(ctx, config.DatabaseConfig{URL: dsn})
	require.NoError(t, err)
	defer pool.Close()

	emailRollback := "tx-rollback-" + timestampSuffix() + "@test.dev"
	// ROLLBACK: tạo user trong tx rồi trả lỗi → user KHÔNG tồn tại sau đó
	err = pool.WithTx(ctx, func(q *db.Queries) error {
		if _, err := q.CreateUser(ctx, db.CreateUserParams{
			Email:        emailRollback,
			PasswordHash: "x",
			FullName:     "Tx",
			IsActive:     true,
		}); err != nil {
			return err
		}
		return assert.AnError // bắt buộc rollback
	})
	require.Error(t, err)
	err = pool.Read().QueryRow(ctx, "SELECT 1 FROM users WHERE email=$1", emailRollback).Scan(nil)
	assert.Error(t, err, "user tạo trong tx bị rollback phải KHÔNG tồn tại")

	// COMMIT: fn không lỗi → user tồn tại
	emailCommit := "tx-commit-" + timestampSuffix() + "@test.dev"
	require.NoError(t, pool.WithTx(ctx, func(q *db.Queries) error {
		_, err := q.CreateUser(ctx, db.CreateUserParams{
			Email:        emailCommit,
			PasswordHash: "x",
			FullName:     "Tx",
			IsActive:     true,
		})
		return err
	}))
	var cnt int
	require.NoError(t, pool.Read().QueryRow(ctx, "SELECT count(*) FROM users WHERE email=$1", emailCommit).Scan(&cnt))
	assert.Equal(t, 1, cnt, "user tạo trong tx commit phải tồn tại")

	// Cleanup
	_, _ = pool.Write().Exec(ctx, "DELETE FROM users WHERE email LIKE 'tx-rollback-%' OR email LIKE 'tx-commit-%'")
}

func timestampSuffix() string {
	return time.Now().Format("150405.000000")
}
