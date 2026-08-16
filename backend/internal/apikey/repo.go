package apikey

import (
	"context"

	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
)

// Repo — wrapper sqlc cho api_keys; tách read (replica) / write (master).
type Repo struct {
	rw *db.Queries
	ro *db.Queries
}

func NewRepo(p *database.Pool) *Repo {
	return &Repo{rw: db.New(p.Write()), ro: db.New(p.Read())}
}

func (r *Repo) Create(ctx context.Context, p db.CreateAPIKeyParams) (db.ApiKey, error) {
	return r.rw.CreateAPIKey(ctx, p)
}

func (r *Repo) GetByID(ctx context.Context, id string) (db.ApiKey, error) {
	return r.ro.GetAPIKeyByID(ctx, id)
}

func (r *Repo) GetByHash(ctx context.Context, hash string) (db.ApiKey, error) {
	return r.ro.GetAPIKeyByHash(ctx, hash)
}

func (r *Repo) List(ctx context.Context, p db.ListAPIKeysParams) ([]db.ApiKey, error) {
	return r.ro.ListAPIKeys(ctx, p)
}

func (r *Repo) Count(ctx context.Context, q string) (int64, error) {
	return r.ro.CountAPIKeys(ctx, q)
}

func (r *Repo) Update(ctx context.Context, p db.UpdateAPIKeyParams) (db.ApiKey, error) {
	return r.rw.UpdateAPIKey(ctx, p)
}

func (r *Repo) Revoke(ctx context.Context, id string) error {
	return r.rw.RevokeAPIKey(ctx, id)
}

func (r *Repo) TouchLastUsed(ctx context.Context, id string) error {
	return r.rw.TouchAPIKeyLastUsed(ctx, id)
}
