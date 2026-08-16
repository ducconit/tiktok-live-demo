package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// SeedAdmin tạo tài khoản system_admin đầu tiên (idempotent — chạy lại không nhân đôi).
// Mật khẩu từ env SEED_ADMIN_PASSWORD, hash bcrypt runtime (không hardcode).
func SeedAdmin(ctx context.Context, pool *Pool, email, password string) error {
	if email == "" || password == "" {
		return fmt.Errorf("SEED_ADMIN_EMAIL và SEED_ADMIN_PASSWORD phải được set")
	}

	q := db.New(pool.Write())

	_, err := q.GetUserByEmail(ctx, email)
	if err == nil {
		slog.Info("admin đã tồn tại, bỏ qua", "email", email)
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check admin: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
		FullName:     "Super Admin",
		AvatarUrl:    "",
		IsActive:     true,
	})
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	role, err := q.GetRoleByID(ctx, "system_admin")
	if err != nil {
		return fmt.Errorf("get system_admin role (chạy migrate trước): %w", err)
	}
	if err := q.AssignUserRole(ctx, db.AssignUserRoleParams{
		UserID: user.ID,
		RoleID: role.ID,
	}); err != nil {
		return fmt.Errorf("assign role: %w", err)
	}

	slog.Info("admin đã được tạo", "email", email, "id", user.ID)
	return nil
}
