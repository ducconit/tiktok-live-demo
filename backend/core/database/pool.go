package database

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/retry"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool — read-write pool:
//   - master: 1 (write), bắt buộc — DATABASE_URL
//   - replicas: N (read), tuỳ chọn — DATABASE_REPLICAS (comma-separated)
//   - Không có replica → Read() trả về master (mọi query chạy trên 1 DB).
type Pool struct {
	master   *pgxpool.Pool
	replicas []*pgxpool.Pool
	rr       atomic.Uint64
}

// NewPool mở master + các replica (nếu cấu hình).
//
// Master BẮT BUỘC — retry khi postgres chưa sẵn sàng (khởi động cùng docker compose:
// app thường chạy trước postgres vài giây). Hết attempts → trả lỗi.
// Replica tuỳ chọn — retry ngắn (5 lần), fail → lỗi rõ (config replica sai).
func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*Pool, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL không được để trống")
	}

	master, err := connectWithRetry(ctx, cfg.URL, "master db", 15, 2*time.Second)
	if err != nil {
		return nil, err
	}

	p := &Pool{master: master}
	for _, u := range cfg.ReplicaURLs() {
		r, err := connectWithRetry(ctx, u, "replica db", 5, time.Second)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("replica db %s: %w", u, err)
		}
		p.replicas = append(p.replicas, r)
	}

	if len(p.replicas) == 0 {
		slog.Info("database: không có replica — read sẽ dùng master (single-node mode)")
	} else {
		slog.Info("database: master + replicas", "replicas", len(p.replicas))
	}
	return p, nil
}

// connectWithRetry — pgxpool.New + Ping với retry backoff (pgxpool.New lazy — Ping mới connect thật).
func connectWithRetry(ctx context.Context, url, what string, attempts int, wait time.Duration) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	err := retry.Do(ctx, retry.Config{Attempts: attempts, InitialWait: wait}, "connect "+what, func() error {
		p, err := pgxpool.New(ctx, url)
		if err != nil {
			return fmt.Errorf("connect %s: %w", what, err)
		}
		if err := p.Ping(ctx); err != nil {
			p.Close()
			return fmt.Errorf("ping %s: %w", what, err)
		}
		pool = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// Write — connection dành cho ghi (INSERT/UPDATE/DELETE) — luôn master.
func (p *Pool) Write() *pgxpool.Pool { return p.master }

// Read — connection dành cho đọc (SELECT) — round-robin qua replicas;
// không có replica → master.
func (p *Pool) Read() *pgxpool.Pool {
	if len(p.replicas) == 0 {
		return p.master
	}
	i := p.rr.Add(1)
	return p.replicas[int(i)%len(p.replicas)]
}

// TxRunner — đối tượng chạy được transaction (Pool implement; mock được cho service test).
type TxRunner interface {
	WithTx(ctx context.Context, fn func(q *db.Queries) error) error
}

// WithTx — chạy fn trong 1 transaction (master). Commit nếu fn không lỗi,
// ROLLBACK nếu lỗi — dùng cho flow multi-step cần atomic
// (vd tạo user + gán role, rotate key: revoke + tạo mới).
func (p *Pool) WithTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := p.master.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	q := db.New(tx)
	if err := fn(q); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("tx rollback: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Ping kiểm tra master (readiness).
func (p *Pool) Ping(ctx context.Context) error { return p.master.Ping(ctx) }

// Close đóng toàn bộ connection.
func (p *Pool) Close() {
	p.master.Close()
	for _, r := range p.replicas {
		r.Close()
	}
}
