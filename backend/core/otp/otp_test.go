package otp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisForTest — kết nối valkey dev (compose). Skip nếu không có.
func redisForTest(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6380"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("không có valkey dev (compose) — skip integration")
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestGenerate_Verify_OK(t *testing.T) {
	svc := NewService(redisForTest(t))
	ctx := context.Background()
	email := "otp-test@example.com"

	code, err := svc.Generate(ctx, PurposeVerifyAccount, email)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(code) != codeLength {
		t.Fatalf("mã phải %d chữ số, got %q", codeLength, code)
	}
	if err := svc.Verify(ctx, PurposeVerifyAccount, email, code); err != nil {
		t.Fatalf("verify đúng phải pass: %v", err)
	}
	// Mã dùng 1 lần — verify lại phải fail
	if err := svc.Verify(ctx, PurposeVerifyAccount, email, code); !errors.Is(err, ErrNotFound) {
		t.Fatalf("verify lần 2 phải fail (mã đã xoá), got %v", err)
	}
}

func TestGenerate_Cooldown(t *testing.T) {
	svc := NewService(redisForTest(t))
	ctx := context.Background()
	email := "otp-cooldown@example.com"

	if _, err := svc.Generate(ctx, PurposePasswordReset, email); err != nil {
		t.Fatalf("generate lần 1: %v", err)
	}
	_, err := svc.Generate(ctx, PurposePasswordReset, email)
	if !errors.Is(err, ErrCooldown) {
		t.Fatalf("generate ngay sau đó phải ErrCooldown, got %v", err)
	}
	// Purpose khác — không dính cooldown
	if _, err := svc.Generate(ctx, PurposeVerifyAccount, email); err != nil {
		t.Fatalf("purpose khác không được dính cooldown: %v", err)
	}
}

func TestVerify_WrongCode_TooManyTries(t *testing.T) {
	svc := NewService(redisForTest(t))
	ctx := context.Background()
	email := "otp-bruteforce@example.com"

	if _, err := svc.Generate(ctx, PurposePasswordReset, email); err != nil {
		t.Fatalf("generate: %v", err)
	}
	var last error
	for i := 0; i < maxAttempts+1; i++ {
		last = svc.Verify(ctx, PurposePasswordReset, email, "000000")
	}
	if !errors.Is(last, ErrTooManyTries) && !errors.Is(last, ErrNotFound) {
		t.Fatalf("sau quá nhiều lần sai phải TooManyTries (hoặc NotFound sau khi mã bị xoá), got %v", last)
	}
}
