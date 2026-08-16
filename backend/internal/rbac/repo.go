package rbac

import (
	"context"

	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
)

// Repo — wrapper sqlc cho roles/permissions/user_roles; read (replica) / write (master).
type Repo struct {
	rw *db.Queries
	ro *db.Queries
}

func NewRepo(p *database.Pool) *Repo {
	return &Repo{rw: db.New(p.Write()), ro: db.New(p.Read())}
}

func (r *Repo) ListRoles(ctx context.Context) ([]db.Role, error) {
	return r.ro.ListRoles(ctx)
}

func (r *Repo) GetRoleByID(ctx context.Context, id string) (db.Role, error) {
	return r.ro.GetRoleByID(ctx, id)
}

func (r *Repo) CreateRole(ctx context.Context, p db.CreateRoleParams) (db.Role, error) {
	return r.rw.CreateRole(ctx, p)
}

func (r *Repo) UpdateRole(ctx context.Context, p db.UpdateRoleParams) (db.Role, error) {
	return r.rw.UpdateRole(ctx, p)
}

func (r *Repo) DeleteRole(ctx context.Context, id string) error {
	return r.rw.DeleteRole(ctx, id)
}

func (r *Repo) ListPermissions(ctx context.Context) ([]db.Permission, error) {
	return r.ro.ListPermissions(ctx)
}

func (r *Repo) ListRolePermissions(ctx context.Context, roleID string) ([]db.Permission, error) {
	return r.ro.ListRolePermissions(ctx, roleID)
}

func (r *Repo) SetRolePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return r.rw.SetRolePermissions(ctx, db.SetRolePermissionsParams{
		RoleID:  roleID,
		Column2: permissionIDs,
	})
}

func (r *Repo) AssignUserRole(ctx context.Context, userID string, roleID string) error {
	return r.rw.AssignUserRole(ctx, db.AssignUserRoleParams{UserID: userID, RoleID: roleID})
}

func (r *Repo) RemoveUserRole(ctx context.Context, userID string, roleID string) error {
	return r.rw.RemoveUserRole(ctx, db.RemoveUserRoleParams{UserID: userID, RoleID: roleID})
}

func (r *Repo) GetUserRoles(ctx context.Context, userID string) ([]db.Role, error) {
	return r.ro.GetUserRoles(ctx, userID)
}

func (r *Repo) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	return r.ro.GetUserPermissions(ctx, userID)
}
