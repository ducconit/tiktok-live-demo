// Package retry — retry đơn giản với backoff (không dep).
//
// Dùng khi khởi động cùng docker compose: postgres/valkey chưa sẵn sàng
// ngay → app retry thay vì chết ngay.
package retry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Config — tuỳ chọn retry.
type Config struct {
	Attempts    int           // số lần thử TỐI ĐA (>=1). Mặc định 10.
	InitialWait time.Duration // chờ giữa 2 lần đầu. Mặc định 2s (tăng dần ×2, tối đa 30s).
	Force       bool          // luôn retry kể cả môi trường CI/test (dùng trong test đơn vị)
}

// SkipRetry — môi trường CI/CD hoặc test: KHÔNG retry, báo lỗi ngay
// (retry chỉ dành cho runtime thật — vd khởi động cùng docker compose).
func SkipRetry() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	switch strings.ToLower(os.Getenv("APP_ENV")) {
	case "test", "testing", "ci":
		return true
	}
	return false
}

// Do — chạy fn; lỗi → chờ backoff → thử lại, hết attempts → trả lỗi cuối.
// ctx bị cancel → dừng ngay (trả ctx.Err()).
// attempt=1 → gọi 1 lần, không retry.
// CI/test (SkipRetry) → gọi 1 lần, báo lỗi luôn (trừ khi Force).
func Do(ctx context.Context, cfg Config, what string, fn func() error) error {
	if cfg.Attempts <= 0 {
		cfg.Attempts = 10
	}
	if SkipRetry() && !cfg.Force {
		cfg.Attempts = 1
	}
	if cfg.InitialWait <= 0 {
		cfg.InitialWait = 2 * time.Second
	}
	wait := cfg.InitialWait

	var err error
	for attempt := 1; attempt <= cfg.Attempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == cfg.Attempts {
			break
		}
		slog.Warn("retry: tạm thất bại, thử lại", "what", what, "attempt", attempt, "max", cfg.Attempts, "next_in", wait.Round(time.Second).String(), "err", err)

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w (dừng retry vì context cancelled)", what, ctx.Err())
		case <-time.After(wait):
		}
		if wait < 30*time.Second {
			wait *= 2
		}
	}
	return fmt.Errorf("%s: %w (sau %d lần thử)", what, err, cfg.Attempts)
}
