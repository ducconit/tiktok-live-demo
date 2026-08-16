package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/core/validation"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// AdminCLIParams — tham số tạo admin qua CLI (`make:admin`).
type AdminCLIParams struct {
	Email    string
	Password string
	FullName string
	Roles    []string // slug vai trò (vd ["admin", "editor"]) — role phải tồn tại (seed)
}

// CreateAdminFromCLI — tạo admin + gán NHIỀU role trong 1 transaction (atomic:
// lỗi bất kỳ bước → rollback, không để user mồ côi không role).
//
//	Điểm khác CreateFromCLI: hỗ trợ full_name + nhiều role + transaction.
//	Email xác thực ngay (admin tạo qua CLI không cần OTP).
func CreateAdminFromCLI(ctx context.Context, pool *database.Pool, p AdminCLIParams) error {
	// Validate đầu vào — lỗi rõ, không đợi DB
	fields := validation.ValidateStruct(&struct {
		Email    string `validate:"required,email"`
		Password string `validate:"required,min=8"`
	}{Email: p.Email, Password: p.Password})
	if fields != nil {
		msgs := validation.FieldsMap(fields, "vi")
		// Lấy message đầu tiên (field ẩn danh — key email/password)
		for _, m := range msgs {
			return fmt.Errorf("make:admin — %s", m)
		}
	}
	p.Email = strings.ToLower(strings.TrimSpace(p.Email))
	if p.FullName == "" {
		p.FullName = "Admin"
	}
	if len(p.Roles) == 0 {
		p.Roles = []string{"system_admin"}
	}

	return pool.WithTx(ctx, func(q *db.Queries) error {
		// Email đã tồn tại? (không ghi đè — an toàn)
		if _, err := q.GetUserByEmail(ctx, p.Email); err == nil {
			return fmt.Errorf("email %q đã tồn tại — không ghi đè (dùng email khác)", p.Email)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		u, err := q.CreateUser(ctx, db.CreateUserParams{
			Email:        p.Email,
			PasswordHash: string(hash),
			FullName:     p.FullName,
			AvatarUrl:    "",
			IsActive:     true,
		})
		if err != nil {
			return fmt.Errorf("create admin: %w", err)
		}

		// Admin CLI — xác thực email ngay
		if err := q.SetEmailVerified(ctx, u.ID); err != nil {
			return fmt.Errorf("verify email: %w", err)
		}

		// Gán tất cả role — lỗi role không tồn tại → rollback toàn bộ
		for _, slug := range p.Roles {
			role, err := q.GetRoleByID(ctx, slug)
			if err != nil {
				return fmt.Errorf("role %q không tồn tại (chạy migrate + seed trước): %w", slug, err)
			}
			if err := q.AssignUserRole(ctx, db.AssignUserRoleParams{UserID: u.ID, RoleID: role.ID}); err != nil {
				return fmt.Errorf("assign role %q: %w", slug, err)
			}
		}

		slog.Info("admin created", "id", u.ID, "email", u.Email, "name", p.FullName, "roles", p.Roles)
		return nil
	})
}

// EnsureUUID — không dùng trực tiếp; giữ cho API ổn định nếu cần mở rộng.
var _ = uuid.UUID{}
