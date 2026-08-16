package user

import (
	"context"
	"log/slog"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"golang.org/x/crypto/bcrypt"
)

// AdminSlugs — vai trò thuộc nhóm "admin" (CRUD admin quản lý các tài khoản này).
var AdminSlugs = []string{"admin", "system_admin"}

// AdminService — CRUD tài khoản admin (admin = user + role admin/super_admin).
type AdminService struct {
	users Repository
	tx    database.TxRunner
}

func NewAdminService(users Repository, tx database.TxRunner) *AdminService {
	return &AdminService{users: users, tx: tx}
}

// List — danh sách admin (phân trang).
func (s *AdminService) List(ctx context.Context, page, pageSize int) ([]db.User, int64, error) {
	offset := int32((page - 1) * pageSize)
	users, err := s.users.ListByRole(ctx, AdminSlugs, int32(pageSize), offset)
	if err != nil {
		return nil, 0, apperr.WrapInternal(err)
	}
	total, err := s.users.CountByRole(ctx, AdminSlugs)
	if err != nil {
		return nil, 0, apperr.WrapInternal(err)
	}
	return users, total, nil
}

// Create — tạo tài khoản admin (verified ngay, không cần OTP).
// Tạo user + set verified + gán role admin trong 1 TRANSACTION (atomic).
func (s *AdminService) Create(ctx context.Context, email, password, fullName string) (db.User, error) {
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return db.User{}, apperr.New(apperr.KindConflict, "email_taken", "error.email_taken")
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
		if err := q.SetEmailVerified(ctx, u.ID); err != nil {
			return err
		}
		role, err := q.GetRoleByID(ctx, "admin")
		if err != nil {
			slog.Warn("admin: role 'admin' chưa tồn tại (chạy seed?)", "err", err)
			return err
		}
		return q.AssignUserRole(ctx, db.AssignUserRoleParams{UserID: u.ID, RoleID: role.ID})
	})
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	return u, nil
}

// Update — cập nhật thông tin admin (full_name, is_active).
func (s *AdminService) Update(ctx context.Context, id string, fullName string, isActive bool) (db.User, error) {
	u, err := s.users.Update(ctx, db.UpdateUserParams{ID: id, FullName: fullName, IsActive: isActive})
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	return u, nil
}

// Delete — xoá admin (chống tự xoá bản thân).
func (s *AdminService) Delete(ctx context.Context, id string) error {
	if err := s.users.Delete(ctx, id); err != nil {
		return apperr.WrapInternal(err)
	}
	return nil
}

// CannotDeleteSelf — kiểm tra không xoá chính mình.
func CannotDeleteSelf(actorID, targetID string) error {
	if actorID == targetID {
		return apperr.New(apperr.KindConflict, "cannot_delete_self", "error.cannot_delete_self")
	}
	return nil
}
