package user

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/ducconit/tiktok-live-platform/backend/internal/rbac"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// Repository — interface cho user repo (mock được bằng mockery).
type Repository interface {
	Create(ctx context.Context, p db.CreateUserParams) (db.User, error)
	GetByID(ctx context.Context, id string) (db.User, error)
	GetByEmail(ctx context.Context, email string) (db.User, error)
	List(ctx context.Context, p db.ListUsersParams) ([]db.User, error)
	Count(ctx context.Context, p db.CountUsersParams) (int64, error)
	Update(ctx context.Context, p db.UpdateUserParams) (db.User, error)
	UpdatePassword(ctx context.Context, id string, hash string) error
	SetVerified(ctx context.Context, id string) error
	UpdateName(ctx context.Context, id string, fullName string) (db.User, error)
	UpdateAvatarURL(ctx context.Context, id string, url string) (db.User, error)
	ListByRole(ctx context.Context, slugs []string, pageLimit, pageOffset int32) ([]db.User, error)
	CountByRole(ctx context.Context, slugs []string) (int64, error)
	UpdateLastLogin(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

// Service — nghiệp vụ user (không biết HTTP).
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateParams — input tạo user.
type CreateParams struct {
	Email    string
	Password string
	FullName string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (db.User, error) {
	if _, err := s.repo.GetByEmail(ctx, p.Email); err == nil {
		return db.User{}, apperr.Conflict("email_exists", "error.email_taken")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, apperr.WrapInternal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}

	u, err := s.repo.Create(ctx, db.CreateUserParams{
		Email:        p.Email,
		PasswordHash: string(hash),
		FullName:     p.FullName,
		AvatarUrl:    "",
		IsActive:     true,
	})
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	return u, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (db.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, apperr.NotFound("user_not_found", "error.user_not_found")
	}
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	return u, nil
}

// boolToPgtype — *bool → pgtype.Bool (sqlc narg param sinh pgtype — không override được).
func boolToPgtype(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

func (s *Service) List(ctx context.Context, q string, isActive *bool, page, pageSize int) ([]db.User, int64, error) {
	params := db.ListUsersParams{
		Q:          q,
		IsActive:   boolToPgtype(isActive),
		PageLimit:  int32(pageSize),
		PageOffset: int32((page - 1) * pageSize),
	}
	users, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, apperr.WrapInternal(err)
	}
	total, err := s.repo.Count(ctx, db.CountUsersParams{Q: q, IsActive: boolToPgtype(isActive)})
	if err != nil {
		return nil, 0, apperr.WrapInternal(err)
	}
	return users, total, nil
}

// UpdateParams — chỉ sửa các field cho phép.
type UpdateParams struct {
	FullName  string
	AvatarURL string
	IsActive  bool
}

func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (db.User, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return db.User{}, err
	}
	u, err := s.repo.Update(ctx, db.UpdateUserParams{
		ID:        id,
		FullName:  p.FullName,
		AvatarUrl: p.AvatarURL,
		IsActive:  p.IsActive,
	})
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	return u, nil
}

func (s *Service) ChangePassword(ctx context.Context, id string, newPassword string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.WrapInternal(err)
	}
	return s.repo.UpdatePassword(ctx, id, string(hash))
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// VerifyPassword — kiểm tra mật khẩu (dùng cho auth).
func (s *Service) VerifyPassword(ctx context.Context, email, password string) (db.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, apperr.Unauthorized("invalid_credentials", "error.bad_credentials")
	}
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return db.User{}, apperr.Unauthorized("invalid_credentials", "error.bad_credentials")
	}
	return u, nil
}

func (s *Service) TouchLastLogin(ctx context.Context, id string) {
	if err := s.repo.UpdateLastLogin(ctx, id); err != nil {
		slog.Warn("update last_login", "err", err)
	}
}

// CreateFromCLI — tạo user qua `devtool user:create` (không cần HTTP).
func CreateFromCLI(ctx context.Context, pool *database.Pool, email, password, roleSlug string) error {
	if roleSlug == "" {
		roleSlug = "admin"
	}
	svc := NewService(NewRepo(pool))
	u, err := svc.Create(ctx, CreateParams{
		Email:    email,
		Password: password,
		FullName: email,
	})
	if err != nil {
		return err
	}
	// Tài khoản tạo qua CLI (admin/seed) xác thực email ngay — không cần OTP
	if err := NewRepo(pool).SetVerified(ctx, u.ID); err != nil {
		return err
	}

	// Gán role qua rbac repo (role phải tồn tại từ seed)
	role, err := rbac.NewRepo(pool).GetRoleByID(ctx, roleSlug)
	if err != nil {
		return apperr.BadRequest("role_not_found", "error.role_slug_not_found")
	}
	if err := rbac.NewRepo(pool).AssignUserRole(ctx, u.ID, role.ID); err != nil {
		return err
	}

	slog.Info("user created", "id", u.ID, "email", u.Email, "role", roleSlug)
	return nil
}
