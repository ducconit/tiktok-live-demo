// Package otp — mã xác thực một lần (email verification, password reset).
// OTP 6 chữ số, lưu Redis (TTL giới hạn), resend có cooldown, chặn brute-force.
package otp

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cấu hình OTP — không magic number.
const (
	codeLength     = 6
	codeTTL        = 10 * time.Minute // thời gian sống của mã
	resendCooldown = 60 * time.Second // chờ giữa 2 lần gửi
	maxAttempts    = 5                // tối đa số lần nhập sai
)

// Purpose — loại OTP (truyền vào endpoint resend/verify).
type Purpose string

// Các loại OTP hỗ trợ.
const (
	PurposeVerifyAccount Purpose = "verify-account"
	PurposePasswordReset Purpose = "password-reset"
)

// Lỗi nghiệp vụ (caller map sang apperr/http).
var (
	ErrInvalidCode  = errors.New("otp: mã không đúng")
	ErrExpired      = errors.New("otp: mã đã hết hạn")
	ErrTooManyTries = errors.New("otp: quá nhiều lần thử sai")
	ErrCooldown     = errors.New("otp: gửi lại quá nhanh")
	ErrNotFound     = errors.New("otp: không có mã cho email này")
)

// Service — sinh/kiểm tra OTP qua Redis.
type Service struct {
	rdb *redis.Client
}

func NewService(rdb *redis.Client) *Service {
	return &Service{rdb: rdb}
}

func codeKey(purpose Purpose, email string) string {
	return fmt.Sprintf("otp:code:%s:%s", purpose, email)
}

func cooldownKey(purpose Purpose, email string) string {
	return fmt.Sprintf("otp:cd:%s:%s", purpose, email)
}

func attemptsKey(purpose Purpose, email string) string {
	return fmt.Sprintf("otp:attempts:%s:%s", purpose, email)
}

// Generate — tạo mã mới cho email + purpose. Lỗi ErrCooldown nếu gửi lại quá nhanh.
func (s *Service) Generate(ctx context.Context, purpose Purpose, email string) (string, error) {
	// Cooldown: lần gửi trước còn hiệu lực trong resendCooldown → chặn
	exists, err := s.rdb.Exists(ctx, cooldownKey(purpose, email)).Result()
	if err != nil {
		return "", err
	}
	if exists == 1 {
		return "", ErrCooldown
	}

	code, err := randomCode()
	if err != nil {
		return "", err
	}

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, codeKey(purpose, email), code, codeTTL)
	pipe.Set(ctx, cooldownKey(purpose, email), "1", resendCooldown)
	pipe.Del(ctx, attemptsKey(purpose, email)) // reset số lần thử khi gửi mã mới
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return code, nil
}

// Verify — kiểm tra mã. Đúng → xoá mã + cooldown; sai → tăng attempt, quá maxAttempts → xoá mã.
func (s *Service) Verify(ctx context.Context, purpose Purpose, email, code string) error {
	stored, err := s.rdb.Get(ctx, codeKey(purpose, email)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if stored != code {
		attempts, err := s.rdb.Incr(ctx, attemptsKey(purpose, email)).Result()
		if err != nil {
			return err
		}
		if attempts >= maxAttempts {
			_ = s.rdb.Del(ctx, codeKey(purpose, email), attemptsKey(purpose, email)).Err()
			return ErrTooManyTries
		}
		// mã vẫn còn hiệu lực tới codeTTL — set lại TTL cho attempts key
		_ = s.rdb.Expire(ctx, attemptsKey(purpose, email), codeTTL).Err()
		return ErrInvalidCode
	}

	// Thành công — dọn sạch
	if err := s.rdb.Del(ctx, codeKey(purpose, email), cooldownKey(purpose, email), attemptsKey(purpose, email)).Err(); err != nil {
		slog.Warn("otp: dọn key sau verify lỗi", "err", err)
	}
	return nil
}

// randomCode — sinh mã số ngẫu nhiên an toàn (crypto/rand).
func randomCode() (string, error) {
	const digits = "0123456789"
	buf := make([]byte, codeLength)
	// rand.Int cho từng chữ số (đơn giản, đủ an toàn cho OTP)
	randBytes := make([]byte, codeLength)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("otp: sinh mã ngẫu nhiên: %w", err)
	}
	for i, b := range randBytes {
		buf[i] = digits[int(b)%len(digits)]
	}
	return string(buf), nil
}
