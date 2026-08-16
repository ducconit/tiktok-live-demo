package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/jackc/pgx/v5"
)

// APIKeyRepository — interface cho api key repo (mock được bằng mockery).
// Tên riêng (không phải "Repository") để tránh trùng mock file với user.Repository.
type APIKeyRepository interface {
	Create(ctx context.Context, p db.CreateAPIKeyParams) (db.ApiKey, error)
	GetByID(ctx context.Context, id string) (db.ApiKey, error)
	GetByHash(ctx context.Context, hash string) (db.ApiKey, error)
	List(ctx context.Context, p db.ListAPIKeysParams) ([]db.ApiKey, error)
	Count(ctx context.Context, q string) (int64, error)
	Update(ctx context.Context, p db.UpdateAPIKeyParams) (db.ApiKey, error)
	Revoke(ctx context.Context, id string) error
	TouchLastUsed(ctx context.Context, id string) error
}

// CreateParams — input tạo API key.
type CreateParams struct {
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
	CreatedBy string
}

// Created — kết quả tạo/rotate: plaintext key (hiển thị ĐÚNG 1 lần) + bản ghi.
type Created struct {
	Key    string // plaintext — KHÔNG lưu lại, chỉ trả cho client 1 lần
	Record db.ApiKey
}

// Service — nghiệp vụ API key (không biết HTTP).
//
// Bảo mật: chỉ lưu SHA-256 hash của key. Plaintext sinh ngẫu nhiên (256-bit)
// và chỉ trả về đúng 1 lần lúc tạo/rotate. Middleware verify qua hash lookup.
type Service struct {
	repo  APIKeyRepository
	tx    database.TxRunner // transaction (pool.WithTx) — mock được cho test
	cache *cache.Manager
	env   string // app.env — tiền tố key (vd "live", "dev")
}

func NewService(repo APIKeyRepository, tx database.TxRunner, cm *cache.Manager, env string) *Service {
	return &Service{repo: repo, tx: tx, cache: cm, env: env}
}

// ---- Sinh key + hash ----

const (
	keyPrefixFmt  = "gvs_%s_" // gvs_live_ | gvs_dev_
	keyBytes      = 32        // 256-bit entropy
	cacheTTL      = 1 * time.Minute
	cacheKeyFmt   = "apikey:hash:%s"
	prefixShowLen = 20 // đủ hiển thị "gvs_development_ab12cd..." (env + 6 ký tự key)
)

// generateKey — key dạng gvs_<env>_<base64url(32 bytes)>, không padding.
func (s *Service) generateKey() (string, error) {
	buf := make([]byte, keyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf(keyPrefixFmt, s.env) + base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashKey — SHA-256 hex của plaintext (giá trị lưu trong DB).
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// displayPrefix — phần hiển thị an toàn: gvs_live_ab12... (8 ký tự đầu).
func displayPrefix(key string) string {
	if len(key) <= prefixShowLen {
		return key
	}
	return key[:prefixShowLen]
}

// ---- CRUD ----

// Create — tạo API key mới, trả plaintext ĐÚNG 1 lần.
func (s *Service) Create(ctx context.Context, p CreateParams) (Created, error) {
	key, err := s.generateKey()
	if err != nil {
		return Created{}, apperr.WrapInternal(err)
	}
	scopes := p.Scopes
	if scopes == nil {
		scopes = []string{} // NOT NULL DEFAULT '{}' — không gửi NULL
	}
	rec, err := s.repo.Create(ctx, db.CreateAPIKeyParams{
		Name:      strings.TrimSpace(p.Name),
		KeyHash:   hashKey(key),
		KeyPrefix: displayPrefix(key),
		Scopes:    scopes,
		ExpiresAt: p.ExpiresAt,
		IsActive:  true,
		CreatedBy: strPtr(p.CreatedBy),
	})
	if err != nil {
		return Created{}, apperr.WrapInternal(err)
	}
	return Created{Key: key, Record: rec}, nil
}

// List — danh sách API keys (phân trang, tìm theo tên).
func (s *Service) List(ctx context.Context, q string, page, pageSize int) ([]db.ApiKey, int64, error) {
	offset := int32((page - 1) * pageSize)
	items, err := s.repo.List(ctx, db.ListAPIKeysParams{
		Q:          q,
		PageLimit:  int32(pageSize),
		PageOffset: offset,
	})
	if err != nil {
		return nil, 0, apperr.WrapInternal(err)
	}
	total, err := s.repo.Count(ctx, q)
	if err != nil {
		return nil, 0, apperr.WrapInternal(err)
	}
	return items, total, nil
}

// Get — lấy API key theo id (404 nếu không tồn tại).
func (s *Service) Get(ctx context.Context, id string) (db.ApiKey, error) {
	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ApiKey{}, apperr.NotFound("api_key_not_found", "error.api_key_not_found")
		}
		return db.ApiKey{}, apperr.WrapInternal(err)
	}
	return rec, nil
}

// UpdateParams — input cập nhật API key.
type UpdateParams struct {
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
	IsActive  bool
}

// Update — cập nhật thông tin (name, scopes, expires_at, is_active).
func (s *Service) Update(ctx context.Context, id string, p UpdateParams) (db.ApiKey, error) {
	rec, err := s.repo.Update(ctx, db.UpdateAPIKeyParams{
		ID:        id,
		Name:      strings.TrimSpace(p.Name),
		Scopes:    p.Scopes,
		ExpiresAt: p.ExpiresAt,
		IsActive:  p.IsActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ApiKey{}, apperr.NotFound("api_key_not_found", "error.api_key_not_found")
		}
		return db.ApiKey{}, apperr.WrapInternal(err)
	}
	s.invalidate(rec.KeyHash)
	return rec, nil
}

// Revoke — vô hiệu hoá API key (is_active=false, revoked_at=now).
func (s *Service) Revoke(ctx context.Context, id string) error {
	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("api_key_not_found", "error.api_key_not_found")
		}
		return apperr.WrapInternal(err)
	}
	if err := s.repo.Revoke(ctx, id); err != nil {
		return apperr.WrapInternal(err)
	}
	s.invalidate(rec.KeyHash)
	return nil
}

// Rotate — vô hiệu key hiện tại + tạo key mới cùng name/scopes/expiry.
// Cả 2 thao tác trong 1 TRANSACTION (atomic — lỗi giữa chừng → rollback,
// không xảy ra trường hợp key cũ chết mà key mới chưa tạo).
func (s *Service) Rotate(ctx context.Context, id string, createdBy string) (Created, error) {
	rec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Created{}, apperr.NotFound("api_key_not_found", "error.api_key_not_found")
		}
		return Created{}, apperr.WrapInternal(err)
	}

	key, err := s.generateKey()
	if err != nil {
		return Created{}, apperr.WrapInternal(err)
	}

	var created db.ApiKey
	err = s.tx.WithTx(ctx, func(q *db.Queries) error {
		// 1) Giết key cũ (revoke — thao tác SQL, không qua repo để dùng chung tx)
		if err := q.RevokeAPIKey(ctx, id); err != nil {
			return err
		}
		// 2) Tạo key mới kế thừa name/scopes/expiry
		scopes := rec.Scopes
		if scopes == nil {
			scopes = []string{}
		}
		created, err = q.CreateAPIKey(ctx, db.CreateAPIKeyParams{
			Name:      rec.Name,
			KeyHash:   hashKey(key),
			KeyPrefix: displayPrefix(key),
			Scopes:    scopes,
			ExpiresAt: rec.ExpiresAt,
			IsActive:  true,
			CreatedBy: strPtr(createdBy),
		})
		return err
	})
	if err != nil {
		return Created{}, apperr.WrapInternal(err)
	}

	// Key cũ đã revoke → xoá cache entry của nó (tránh dùng lại được)
	s.invalidate(rec.KeyHash)
	return Created{Key: key, Record: created}, nil
}

// ---- Lookup (middleware) ----

// Info — thông tin API key sau khi verify (đưa vào context).
type Info struct {
	ID     string
	Name   string
	Scopes []string
}

// Lookup — verify key: hash → tìm bản ghi → check active/expired/revoked.
// Cache ngắn hạn (1m) để giảm DB hit trên request nóng.
func (s *Service) Lookup(ctx context.Context, key string) (Info, error) {
	h := hashKey(key)

	// Cache hit? (s.cache có thể nil khi app chạy zero-config không cache — bỏ qua cache)
	if s.cache != nil {
		if info, err := cache.Get[Info](s.cache, ctx, cacheKey(h)); err == nil {
			return info, nil
		} else if !errors.Is(err, cache.ErrNotFound) {
			slog.Warn("apikey: cache get", "err", err)
		}
	}

	rec, err := s.repo.GetByHash(ctx, h)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Info{}, apperr.Unauthorized("invalid_api_key", "error.invalid_api_key")
		}
		return Info{}, apperr.WrapInternal(err)
	}
	if !rec.IsActive || rec.RevokedAt != nil {
		return Info{}, apperr.Unauthorized("api_key_revoked", "error.api_key_revoked")
	}
	if rec.ExpiresAt != nil && time.Now().After(*rec.ExpiresAt) {
		return Info{}, apperr.Unauthorized("api_key_expired", "error.api_key_expired")
	}

	info := Info{ID: rec.ID, Name: rec.Name, Scopes: rec.Scopes}
	// Cache (mặc định TTL 1m). Miss cache → DB, không cache lỗi (chỉ cache thành công).
	if s.cache != nil {
		if err := cache.Set[Info](s.cache, ctx, cacheKey(h), info); err != nil {
			slog.Warn("apikey: cache set", "err", err)
		}
	}

	// last_used_at — async, không chặn request (lỗi chỉ log).
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.repo.TouchLastUsed(ctx, rec.ID); err != nil {
			slog.Debug("apikey: touch last_used", "id", rec.ID, "err", err)
		}
	}()

	return info, nil
}

// invalidate — xoá cache entry khi key đổi/revoke.
func (s *Service) invalidate(hash string) {
	if s.cache == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cache.Delete(s.cache, ctx, cacheKey(hash)); err != nil && !errors.Is(err, cache.ErrNotFound) {
		slog.Warn("apikey: cache delete", "err", err)
	}
}

func cacheKey(hash string) string {
	return strings.ReplaceAll(cacheKeyFmt, "%s", hash)
}
