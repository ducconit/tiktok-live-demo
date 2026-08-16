package database

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrate chạy goose lên schema theo databaseURL.
// Đã embed migrations nên không cần file hệ thống — dir luôn là ".".
func Migrate(ctx context.Context, databaseURL string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// MigrateStatus in trạng thái migration ra stdout (dùng cho CLI status).
func MigrateStatus(ctx context.Context, databaseURL string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	return goose.Status(db, ".")
}

// MigrateDown lùi 1 bước migration (CLI down).
func MigrateDown(ctx context.Context, databaseURL string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	return goose.Down(db, ".")
}

// CreateMigration — tạo file migration mới ĐÚNG CHUẨN goose v3.
//
//   - SetSequential(true) → version = max(goose_db_version) + 1, pad 5 số
//     → `migrations/00009_<tên>.sql` (khớp chuẩn 00001..00008 hiện tại)
//   - Template chuẩn goose: `-- +goose Up` / `-- +goose Down`
//
// Yêu cầu DB đã migrate tới mới nhất trước khi tạo (version tính từ DB).
// Trả về tên file đã tạo.
func CreateMigration(ctx context.Context, databaseURL, name string) (string, error) {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return "", fmt.Errorf("set goose dialect: %w", err)
	}
	goose.SetSequential(true)

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return "", fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	if err := goose.Create(db, "migrations", name, "sql"); err != nil {
		return "", fmt.Errorf("goose create: %w", err)
	}
	// Trả về đường dẫn file vừa tạo (file .sql mới nhất trong migrations/)
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return "", fmt.Errorf("đọc migrations/: %w", err)
	}
	var newest string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if newest == "" || e.Name() > newest {
			newest = e.Name()
		}
	}
	return "migrations/" + newest, nil
}
