package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/ctxkey"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/core/mailer"
	"github.com/ducconit/tiktok-live-platform/backend/core/otp"
	"github.com/ducconit/tiktok-live-platform/backend/core/storage"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"golang.org/x/crypto/bcrypt"
)

// Lỗi nghiệp vụ tài khoản (map sang apperr với code ổn định).
var (
	ErrEmailTaken         = apperr.New(apperr.KindConflict, "email_taken", "error.email_taken")
	ErrEmailNotVerified   = apperr.New(apperr.KindForbidden, "email_not_verified", "error.email_not_verified")
	ErrEmailNotFound      = apperr.New(apperr.KindNotFound, "email_not_found", "error.email_not_found")
	ErrWrongPassword      = apperr.New(apperr.KindUnauthorized, "wrong_password", "error.wrong_password")
	ErrMailUnavailable    = apperr.New(apperr.KindInternal, "mail_unavailable", "error.mail_unavailable")
	ErrStorageUnavailable = apperr.New(apperr.KindInternal, "storage_unavailable", "error.storage_unavailable")
)

// AccountService — flow tài khoản công khai: register, verify email (OTP),
// forgot/reset password (OTP), profile (me/update/avatar/change-password).
type AccountService struct {
	users   Repository
	tx      database.TxRunner // transaction (pool.WithTx) — mock được cho test
	otp     *otp.Service
	mail    *mailer.Mailer // nil → không gửi được (lỗi rõ, không nuốt)
	storage *storage.Manager
}

func NewAccountService(users Repository, tx database.TxRunner, otpSvc *otp.Service, mail *mailer.Mailer, st *storage.Manager) *AccountService {
	return &AccountService{users: users, tx: tx, otp: otpSvc, mail: mail, storage: st}
}

// Register — tạo tài khoản (role "user", chưa verify) + gửi OTP xác thực email.
// Tạo user + gán role trong 1 TRANSACTION (atomic — lỗi giữa chừng → rollback,
// không để tài khoản mồ côi không role).
func (s *AccountService) Register(ctx context.Context, email, password, fullName string) (db.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return db.User{}, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}

	var u db.User
	err = s.tx.WithTx(ctx, func(q *db.Queries) error {
		u, err = q.CreateUser(ctx, db.CreateUserParams{
			Email:        email,
			PasswordHash: string(hash),
			FullName:     fullName,
			IsActive:     true,
		})
		if err != nil {
			return err
		}
		// Gán role mặc định "user" — lỗi → rollback cả user (atomic)
		role, err := q.GetRoleByID(ctx, ctxkey.DefaultUserRole)
		if err != nil {
			slog.Warn("account: role 'user' chưa có trong seed?", "err", err)
			return apperr.WrapInternal(err)
		}
		return q.AssignUserRole(ctx, db.AssignUserRoleParams{UserID: u.ID, RoleID: role.ID})
	})
	if err != nil {
		if ae, ok := apperr.From(err); ok {
			return db.User{}, ae
		}
		return db.User{}, apperr.WrapInternal(err)
	}

	if err := s.sendOTP(ctx, otp.PurposeVerifyAccount, email, fullName); err != nil {
		return db.User{}, err
	}
	return u, nil
}

// VerifyAccount — xác thực email bằng OTP.
func (s *AccountService) VerifyAccount(ctx context.Context, email, code string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return ErrEmailNotFound
	}
	if err := s.otp.Verify(ctx, otp.PurposeVerifyAccount, email, code); err != nil {
		return mapOTPError(err)
	}
	if err := s.users.SetVerified(ctx, u.ID); err != nil {
		return apperr.WrapInternal(err)
	}
	return nil
}

// ResendVerification — gửi lại OTP xác thực (cooldown chặn spam).
func (s *AccountService) ResendVerification(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return ErrEmailNotFound
	}
	if u.EmailVerifiedAt == nil { // chưa verify → gửi OTP xác thực
		return s.sendOTP(ctx, otp.PurposeVerifyAccount, email, u.FullName)
	}
	// Đã verified — coi như thành công (không lộ trạng thái)
	slog.Info("account: resend verification cho email đã xác thực", "email", email)
	return nil
}

// ForgotPassword — gửi OTP đặt lại mật khẩu.
func (s *AccountService) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Không lộ email tồn tại — trả thành công nhưng không gửi (chống scan)
		slog.Info("account: forgot password cho email không tồn tại", "email", email)
		return nil
	}
	return s.sendOTP(ctx, otp.PurposePasswordReset, email, u.FullName)
}

// ResetPassword — xác thực OTP + đặt mật khẩu mới.
func (s *AccountService) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return ErrEmailNotFound
	}
	if err := s.otp.Verify(ctx, otp.PurposePasswordReset, email, code); err != nil {
		return mapOTPError(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.WrapInternal(err)
	}
	if err := s.users.UpdatePassword(ctx, u.ID, string(hash)); err != nil {
		return apperr.WrapInternal(err)
	}
	return nil
}

// UpdateProfile — cập nhật thông tin tài khoản (full_name...).
func (s *AccountService) UpdateProfile(ctx context.Context, id string, fullName string) (db.User, error) {
	u, err := s.users.UpdateName(ctx, id, fullName)
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	return u, nil
}

// UpdateAvatar — upload ảnh lên disk public + lưu URL.
// File nằm ở: users/<userID>/avatars/<sha256-16>.<ext> (storage.UploadImage),
// URL trả về theo driver của disk đang dùng (local: /storage/..., s3: full URL).
func (s *AccountService) UpdateAvatar(ctx context.Context, id string, fh *multipart.FileHeader) (string, error) {
	if s.storage == nil {
		return "", ErrStorageUnavailable
	}
	disk, err := s.storage.Disk("public")
	if err != nil {
		return "", ErrStorageUnavailable
	}
	folder := fmt.Sprintf("users/%s/avatars", id)
	_, url, err := storage.UploadImage(ctx, disk, folder, fh)
	if err != nil {
		if errors.Is(err, storage.ErrFileTooLarge) {
			// Chuẩn response: 413 file too large
			return "", apperr.New(apperr.KindPayloadTooLarge, "413", "error.file_too_large")
		}
		return "", apperr.WrapInternal(err)
	}
	if _, err := s.users.UpdateAvatarURL(ctx, id, url); err != nil {
		return "", apperr.WrapInternal(err)
	}
	return url, nil
}

// ChangePassword — đổi mật khẩu (phải đúng mật khẩu hiện tại).
func (s *AccountService) ChangePassword(ctx context.Context, id string, current, newPassword string) error {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return apperr.WrapInternal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)) != nil {
		return ErrWrongPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.WrapInternal(err)
	}
	if err := s.users.UpdatePassword(ctx, id, string(hash)); err != nil {
		return apperr.WrapInternal(err)
	}
	return nil
}

// sendOTP — sinh mã + gửi email; mail nil → log mã ra console (dev-friendly) + lỗi rõ.
func (s *AccountService) sendOTP(ctx context.Context, purpose otp.Purpose, email, fullName string) error {
	code, err := s.otp.Generate(ctx, purpose, email)
	if err != nil {
		if errors.Is(err, otp.ErrCooldown) {
			return apperr.New(apperr.KindTooManyRequests, "otp_cooldown", "error.otp_cooldown")
		}
		return apperr.WrapInternal(err)
	}

	if s.mail == nil {
		slog.Warn("account: mailer chưa cấu hình — OTP chỉ in ra log (dev)", "email", email, "otp", code, "purpose", purpose)
		return nil
	}

	subject, body := otpEmail(purpose, code, fullName)
	if err := s.mail.Send(ctx, email, subject, body); err != nil {
		slog.Error("account: gửi email OTP thất bại", "email", email, "err", err)
		return ErrMailUnavailable
	}
	return nil
}

func mapOTPError(err error) error {
	switch {
	case errors.Is(err, otp.ErrNotFound), errors.Is(err, otp.ErrExpired):
		return apperr.New(apperr.KindInvalid, "otp_invalid", "error.otp_invalid")
	case errors.Is(err, otp.ErrInvalidCode):
		return apperr.New(apperr.KindInvalid, "otp_invalid", "error.otp_wrong")
	case errors.Is(err, otp.ErrTooManyTries):
		return apperr.New(apperr.KindTooManyRequests, "otp_too_many_tries", "error.otp_too_many_tries")
	default:
		return apperr.WrapInternal(err)
	}
}

// otpEmail — nội dung email OTP (dev-friendly).
func otpEmail(purpose otp.Purpose, code, fullName string) (subject, body string) {
	action := "xác thực tài khoản"
	if purpose == otp.PurposePasswordReset {
		action = "đặt lại mật khẩu"
	}
	subject = "Mã " + action + " của bạn"
	body = "<p>Xin chào <b>" + fullName + "</b>,</p>" +
		"<p>Mã " + action + " của bạn là:</p>" +
		"<h2 style=\"letter-spacing:4px\">" + code + "</h2>" +
		"<p>Mã có hiệu lực trong 10 phút. Không chia sẻ mã này với bất kỳ ai.</p>"
	return subject, body
}
