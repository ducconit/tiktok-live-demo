package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/ducconit/tiktok-live-platform/backend/internal/mocks"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func fakeRole(id string, slug string, isSystem ...bool) db.Role {
	sys := len(isSystem) > 0 && isSystem[0]
	return db.Role{ID: id, Name: slug, Description: "", IsSystem: sys}
}

func TestCreateRole_Success(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	id := uuid.NewString()
	repo.On("GetRoleByID", mock.Anything, "editor").Return(db.Role{}, pgx.ErrNoRows).Once()
	repo.On("CreateRole", mock.Anything, mock.MatchedBy(func(p db.CreateRoleParams) bool {
		return p.ID == "editor" && p.Name == "Editor"
	})).Return(fakeRole(id, "editor"), nil).Once()

	role, err := svc.CreateRole(context.Background(), RoleInput{ID: "editor", Name: "Editor"})
	require.NoError(t, err)
	assert.Equal(t, id, role.ID)
	repo.AssertExpectations(t)
}

func TestCreateRole_Duplicate_Conflict(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	repo.On("GetRoleByID", mock.Anything, "admin").Return(fakeRole(uuid.NewString(), "admin"), nil).Once()

	_, err := svc.CreateRole(context.Background(), RoleInput{ID: "admin", Name: "Admin"})
	require.Error(t, err)
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindConflict, ae.Kind)
	assert.Equal(t, "role_exists", ae.Code)
}

func TestCreateRole_DBError_Wrapped(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	repo.On("GetRoleByID", mock.Anything, "editor").Return(db.Role{}, errors.New("connection refused")).Once()

	_, err := svc.CreateRole(context.Background(), RoleInput{ID: "editor"})
	require.Error(t, err)
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindInternal, ae.Kind, "lỗi DB → internal, không phải conflict")
}

func TestUpdateRole_NotFound(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	repo.On("GetRoleByID", mock.Anything, mock.Anything).Return(db.Role{}, pgx.ErrNoRows).Once()

	_, err := svc.UpdateRole(context.Background(), uuid.NewString(), RoleInput{Name: "X"})
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindNotFound, ae.Kind)
}

func TestUpdateRole_Success(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	id := uuid.NewString()
	repo.On("GetRoleByID", mock.Anything, id).Return(fakeRole(id, "editor"), nil).Once()
	repo.On("UpdateRole", mock.Anything, mock.MatchedBy(func(p db.UpdateRoleParams) bool {
		return p.ID == id && p.Name == "Editor Mới"
	})).Return(fakeRole(id, "editor"), nil).Once()

	role, err := svc.UpdateRole(context.Background(), id, RoleInput{Name: "Editor Mới"})
	require.NoError(t, err)
	assert.Equal(t, id, role.ID)
}

func TestSetRolePermissions_System_Forbidden(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	id := uuid.NewString()
	repo.On("GetRoleByID", mock.Anything, id).Return(fakeRole(id, "system_admin", true), nil).Once()

	err := svc.SetRolePermissions(context.Background(), id, []string{uuid.NewString()})
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindForbidden, ae.Kind)
	assert.Equal(t, "system_role_permissions", ae.Code)
}

func TestUpdateRole_System_SlugImmutable(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	id := uuid.NewString()
	repo.On("GetRoleByID", mock.Anything, id).Return(fakeRole(id, "system_admin", true), nil).Once()

	// Cố đổi slug → chặn
	_, err := svc.UpdateRole(context.Background(), id, RoleInput{ID: "hacker", Name: "Khác"})
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindForbidden, ae.Kind)
	assert.Equal(t, "system_role_immutable", ae.Code)
}

func TestUpdateRole_System_RenameOK(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	id := uuid.NewString()
	repo.On("GetRoleByID", mock.Anything, id).Return(fakeRole(id, "system_admin", true), nil).Once()
	repo.On("UpdateRole", mock.Anything, mock.Anything).Return(fakeRole(id, "system_admin", true), nil).Once()

	// Đổi TÊN (slug rỗng — handler không gửi slug) → OK, id giữ nguyên
	got, err := svc.UpdateRole(context.Background(), id, RoleInput{Name: "Quản trị hệ thống"})
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}

func TestDeleteRole_NotFound(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	repo.On("GetRoleByID", mock.Anything, mock.Anything).Return(db.Role{}, pgx.ErrNoRows).Once()

	err := svc.DeleteRole(context.Background(), uuid.NewString())
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindNotFound, ae.Kind)
}

func TestDeleteRole_System_Forbidden(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	repo.On("GetRoleByID", mock.Anything, mock.Anything).Return(fakeRole(uuid.NewString(), "system_admin", true), nil).Once()

	err := svc.DeleteRole(context.Background(), uuid.NewString())
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindForbidden, ae.Kind)
	assert.Equal(t, "system_role_protected", ae.Code)
}

func TestDeleteRole_Success(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	id := uuid.NewString()
	repo.On("GetRoleByID", mock.Anything, id).Return(fakeRole(id, "editor"), nil).Once()
	repo.On("DeleteRole", mock.Anything, id).Return(nil).Once()

	require.NoError(t, svc.DeleteRole(context.Background(), id))
}

func TestSetRolePermissions_RoleNotFound(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	repo.On("GetRoleByID", mock.Anything, mock.Anything).Return(db.Role{}, pgx.ErrNoRows).Once()

	err := svc.SetRolePermissions(context.Background(), uuid.NewString(), []string{uuid.NewString()})
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindNotFound, ae.Kind)
}

func TestSetRolePermissions_Success(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	roleID := uuid.NewString()
	permIDs := []string{uuid.NewString(), uuid.NewString()}
	repo.On("GetRoleByID", mock.Anything, roleID).Return(fakeRole(roleID, "editor"), nil).Once()
	repo.On("SetRolePermissions", mock.Anything, roleID, permIDs).Return(nil).Once()

	require.NoError(t, svc.SetRolePermissions(context.Background(), roleID, permIDs))
}

func TestAssignRemoveUserRole_PassThrough(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	u, r := uuid.NewString(), uuid.NewString()
	repo.On("AssignUserRole", mock.Anything, u, r).Return(nil).Once()
	repo.On("RemoveUserRole", mock.Anything, u, r).Return(nil).Once()

	require.NoError(t, svc.AssignUserRole(context.Background(), u, r))
	require.NoError(t, svc.RemoveUserRole(context.Background(), u, r))
}

func TestListRoles_PassThrough(t *testing.T) {
	repo := mocks.NewMockRoleRepository(t)
	svc := NewService(repo)
	roles := []db.Role{fakeRole(uuid.NewString(), "admin"), fakeRole(uuid.NewString(), "user")}
	repo.On("ListRoles", mock.Anything).Return(roles, nil).Once()

	got, err := svc.ListRoles(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}
