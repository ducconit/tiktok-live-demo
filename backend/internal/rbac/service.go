package rbac

import (
	"context"
	"errors"
	"regexp"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/jackc/pgx/v5"
)

// roleIDRe — id role do admin tự điền: chữ thường, số, gạch dưới/gạch ngang.
var roleIDRe = regexp.MustCompile(`^[a-z0-9_-]{2,64}$`)

// RoleRepository — interface repo RBAC (mock được cho unit test; *Repo implement).
type RoleRepository interface {
	ListRoles(ctx context.Context) ([]db.Role, error)
	GetRoleByID(ctx context.Context, id string) (db.Role, error)
	CreateRole(ctx context.Context, p db.CreateRoleParams) (db.Role, error)
	UpdateRole(ctx context.Context, p db.UpdateRoleParams) (db.Role, error)
	DeleteRole(ctx context.Context, id string) error
	ListPermissions(ctx context.Context) ([]db.Permission, error)
	ListRolePermissions(ctx context.Context, roleID string) ([]db.Permission, error)
	SetRolePermissions(ctx context.Context, roleID string, permissionIDs []string) error
	AssignUserRole(ctx context.Context, userID string, roleID string) error
	RemoveUserRole(ctx context.Context, userID string, roleID string) error
}

// Service — nghiệp vụ RBAC.
type Service struct {
	repo RoleRepository
}

func NewService(repo RoleRepository) *Service {
	return &Service{repo: repo}
}

type RoleInput struct {
	ID          string // admin tự điền (vd system_admin, editor) — immutable sau khi tạo
	Name        string
	Description string
}

func (s *Service) ListRoles(ctx context.Context) ([]db.Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) CreateRole(ctx context.Context, in RoleInput) (db.Role, error) {
	if !roleIDRe.MatchString(in.ID) {
		return db.Role{}, apperr.BadRequest("invalid_role_id", "error.invalid_role_id")
	}
	if _, err := s.repo.GetRoleByID(ctx, in.ID); err == nil {
		return db.Role{}, apperr.Conflict("role_exists", "error.role_exists")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.Role{}, apperr.WrapInternal(err)
	}
	return s.repo.CreateRole(ctx, db.CreateRoleParams{
		ID:          in.ID,
		Name:        in.Name,
		Description: in.Description,
	})
}

func (s *Service) UpdateRole(ctx context.Context, id string, in RoleInput) (db.Role, error) {
	role, err := s.repo.GetRoleByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Role{}, apperr.NotFound("role_not_found", "error.role_not_found")
	}
	if err != nil {
		return db.Role{}, apperr.WrapInternal(err)
	}
	// Role hệ thống: CHỈ đổi tên/mô tả được (slug không nằm trong UpdateRoleParams
	// — immutable sẵn; description vô hại). Chặn đổi slug nếu cố tình truyền khác.
	if role.IsSystem && in.ID != "" && in.ID != role.ID {
		return db.Role{}, apperr.Forbidden("system_role_immutable", "error.system_role_immutable")
	}
	return s.repo.UpdateRole(ctx, db.UpdateRoleParams{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
	})
}

func (s *Service) DeleteRole(ctx context.Context, id string) error {
	role, err := s.repo.GetRoleByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NotFound("role_not_found", "error.role_not_found")
	}
	if err != nil {
		return apperr.WrapInternal(err)
	}
	if role.IsSystem {
		return apperr.Forbidden("system_role_protected", "error.system_role_protected")
	}
	return s.repo.DeleteRole(ctx, id)
}

func (s *Service) SetRolePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NotFound("role_not_found", "error.role_not_found")
	}
	if err != nil {
		return apperr.WrapInternal(err)
	}
	// Role hệ thống: quyền immutable (đã seed đủ — không sửa được)
	if role.IsSystem {
		return apperr.Forbidden("system_role_permissions", "error.system_role_permissions")
	}
	return s.repo.SetRolePermissions(ctx, roleID, permissionIDs)
}

func (s *Service) AssignUserRole(ctx context.Context, userID string, roleID string) error {
	return s.repo.AssignUserRole(ctx, userID, roleID)
}

func (s *Service) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
	return s.repo.RemoveUserRole(ctx, userID, roleID)
}
