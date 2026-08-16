package user

import (
	"context"

	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
)

// Repo — wrapper sqlc cho user; tách read (replica) / write (master).
type Repo struct {
	rw *db.Queries // master — INSERT/UPDATE/DELETE
	ro *db.Queries // replica (hoặc master nếu không có replica) — SELECT
}

func NewRepo(p *database.Pool) *Repo {
	return &Repo{rw: db.New(p.Write()), ro: db.New(p.Read())}
}

func (r *Repo) Create(ctx context.Context, p db.CreateUserParams) (db.User, error) {
	return r.rw.CreateUser(ctx, p)
}

func (r *Repo) GetByID(ctx context.Context, id string) (db.User, error) {
	return r.ro.GetUserByID(ctx, id)
}

func (r *Repo) GetByEmail(ctx context.Context, email string) (db.User, error) {
	return r.ro.GetUserByEmail(ctx, email)
}

func (r *Repo) List(ctx context.Context, p db.ListUsersParams) ([]db.User, error) {
	return r.ro.ListUsers(ctx, p)
}

func (r *Repo) Count(ctx context.Context, p db.CountUsersParams) (int64, error) {
	return r.ro.CountUsers(ctx, p)
}

func (r *Repo) Update(ctx context.Context, p db.UpdateUserParams) (db.User, error) {
	return r.rw.UpdateUser(ctx, p)
}

func (r *Repo) UpdatePassword(ctx context.Context, id string, hash string) error {
	return r.rw.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: id, PasswordHash: hash})
}

func (r *Repo) SetVerified(ctx context.Context, id string) error {
	return r.rw.SetEmailVerified(ctx, id)
}

func (r *Repo) UpdateName(ctx context.Context, id string, fullName string) (db.User, error) {
	return r.rw.UpdateProfile(ctx, db.UpdateProfileParams{ID: id, FullName: fullName})
}

func (r *Repo) UpdateAvatarURL(ctx context.Context, id string, url string) (db.User, error) {
	return r.rw.UpdateAvatar(ctx, db.UpdateAvatarParams{ID: id, AvatarUrl: url})
}

func (r *Repo) ListByRole(ctx context.Context, slugs []string, pageLimit, pageOffset int32) ([]db.User, error) {
	return r.ro.ListUsersByRole(ctx, db.ListUsersByRoleParams{Slugs: slugs, PageLimit: pageLimit, PageOffset: pageOffset})
}

func (r *Repo) CountByRole(ctx context.Context, slugs []string) (int64, error) {
	return r.ro.CountUsersByRole(ctx, slugs)
}

func (r *Repo) UpdateLastLogin(ctx context.Context, id string) error {
	return r.rw.UpdateUserLastLogin(ctx, id)
}

func (r *Repo) Delete(ctx context.Context, id string) error {
	return r.rw.DeleteUser(ctx, id)
}
