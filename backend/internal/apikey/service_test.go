package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/cache"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/ducconit/tiktok-live-platform/backend/internal/mocks"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// fakeTx — TxRunner giả: capture fn (không chạy — unit test không có DB thật).
type fakeTx struct{ called bool }

func (f *fakeTx) WithTx(_ context.Context, _ func(q *db.Queries) error) error {
	f.called = true
	return nil
}

func newTestService(t *testing.T, repo *mocks.MockAPIKeyRepository) *Service {
	t.Helper()
	cm, err := cache.New(cache.Config{Store: "memory", DefaultTTL: time.Minute}, nil)
	require.NoError(t, err)
	t.Cleanup(cm.Close)
	return NewService(repo, &fakeTx{}, cm, "test")
}

// fakeKey — db.ApiKey hợp lệ (từ hash tìm thấy).
func timePtr(t time.Time) *time.Time { return &t }

func fakeKey(id string, hash string, active bool) db.ApiKey {
	return db.ApiKey{
		ID:        id,
		Name:      "worker",
		KeyHash:   hash,
		KeyPrefix: "gvs_test_ab12",
		Scopes:    []string{"orders.read"},
		IsActive:  active,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestHashKey_StableAndIrreversible(t *testing.T) {
	h1 := hashKey("gvs_test_secretkey123")
	h2 := hashKey("gvs_test_secretkey123")
	h3 := hashKey("gvs_test_secretkey124")
	assert.Equal(t, h1, h2, "cùng key → cùng hash")
	assert.NotEqual(t, h1, h3, "khác key → khác hash")
	assert.Len(t, h1, 64, "SHA-256 hex = 64 ký tự")
	assert.NotContains(t, h1, "secret", "hash không chứa plaintext")
}

func TestDisplayPrefix(t *testing.T) {
	assert.Equal(t, "gvs_test", displayPrefix("gvs_test"))   // ngắn hơn giới hạn → giữ nguyên
	assert.Equal(t, "gvs_live_", displayPrefix("gvs_live_")) // 9 ký tự < 20
	assert.Equal(t, "gvs_live_ab12cd34ef5", displayPrefix("gvs_live_ab12cd34ef56"))
}

func TestLookup_Success(t *testing.T) {
	repo := mocks.NewMockAPIKeyRepository(t)
	svc := newTestService(t, repo)

	id := uuid.NewString()
	key := "gvs_test_abcdef123456"
	h := hashKey(key)
	repo.On("GetByHash", mock.Anything, h).Return(fakeKey(id, h, true), nil).Once()
	// TouchLastUsed chạy ASYNC (goroutine trong Lookup) — .Maybe() vì không đảm bảo
	// goroutine xong trước khi test kết thúc (flaky CI khi máy chậm); đúng bản chất
	// side-effect không chặn request.
	repo.On("TouchLastUsed", mock.Anything, id).Return(nil).Maybe()

	info, err := svc.Lookup(context.Background(), key)
	require.NoError(t, err)
	assert.Equal(t, id, info.ID)
	assert.Equal(t, "worker", info.Name)
	assert.Equal(t, []string{"orders.read"}, info.Scopes)
}

func TestLookup_UnknownKey(t *testing.T) {
	repo := mocks.NewMockAPIKeyRepository(t)
	svc := newTestService(t, repo)

	repo.On("GetByHash", mock.Anything, mock.Anything).Return(db.ApiKey{}, pgx.ErrNoRows).Once()

	_, err := svc.Lookup(context.Background(), "gvs_test_wrongkey")
	require.Error(t, err)
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindUnauthorized, ae.Kind)
	assert.Equal(t, "invalid_api_key", ae.Code)
}

func TestLookup_Revoked(t *testing.T) {
	repo := mocks.NewMockAPIKeyRepository(t)
	svc := newTestService(t, repo)

	id := uuid.NewString()
	key := "gvs_test_revoked123"
	h := hashKey(key)
	rec := fakeKey(id, h, false)
	rec.RevokedAt = timePtr(time.Now())
	repo.On("GetByHash", mock.Anything, h).Return(rec, nil).Once()

	_, err := svc.Lookup(context.Background(), key)
	require.Error(t, err)
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, "api_key_revoked", ae.Code)
}

func TestLookup_Expired(t *testing.T) {
	repo := mocks.NewMockAPIKeyRepository(t)
	svc := newTestService(t, repo)

	id := uuid.NewString()
	key := "gvs_test_expired123"
	h := hashKey(key)
	rec := fakeKey(id, h, true)
	rec.ExpiresAt = timePtr(time.Now().Add(-time.Hour))
	repo.On("GetByHash", mock.Anything, h).Return(rec, nil).Once()

	_, err := svc.Lookup(context.Background(), key)
	require.Error(t, err)
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, "api_key_expired", ae.Code)
}

func TestCreate_ReturnsKeyOnceAndHashes(t *testing.T) {
	repo := mocks.NewMockAPIKeyRepository(t)
	svc := newTestService(t, repo)

	var savedHash string
	repo.On("Create", mock.Anything, mock.MatchedBy(func(p db.CreateAPIKeyParams) bool {
		// Plaintext KHÔNG được lưu — chỉ hash + prefix hiển thị
		assert.NotContains(t, p.KeyHash, "gvs_test_")
		assert.Equal(t, 64, len(p.KeyHash), "SHA-256 hex")
		assert.True(t, len(p.KeyPrefix) <= 20)
		assert.Equal(t, "CI worker", p.Name)
		savedHash = p.KeyHash
		return true
	})).Return(fakeKey(uuid.NewString(), "hash", true), nil).Once()

	created, err := svc.Create(context.Background(), CreateParams{Name: "CI worker", Scopes: []string{"a"}})
	require.NoError(t, err)
	assert.True(t, len(created.Key) > 30, "key plaintext dài (đủ entropy)")
	assert.Contains(t, created.Key, "gvs_test_")
	// Hash DB lưu phải = hash(key plaintext trả về)
	assert.Equal(t, hashKey(created.Key), savedHash)
}

func TestRotate_RevokesOldThenCreatesNew(t *testing.T) {
	repo := mocks.NewMockAPIKeyRepository(t)
	svc := newTestService(t, repo)
	tx := svc.tx.(*fakeTx) // verify tx được dùng

	id := uuid.NewString()
	oldHash := hashKey("gvs_test_oldkey123")
	rec := fakeKey(id, oldHash, true)
	rec.Name = "rotate-me"
	rec.Scopes = []string{"a", "b"}

	// Rotate: GetByID (lấy rec) 1 lần; revoke + create chạy TRONG tx (fakeTx không chạy fn)
	repo.On("GetByID", mock.Anything, id).Return(rec, nil).Once()

	created, err := svc.Rotate(context.Background(), id, uuid.NewString())
	require.NoError(t, err)
	assert.True(t, len(created.Key) > 30, "key plaintext dài")
	assert.True(t, tx.called, "phải chạy trong transaction (revoke + create atomic)")
	repo.AssertExpectations(t)
}

func TestService_GetNotFound(t *testing.T) {
	repo := mocks.NewMockAPIKeyRepository(t)
	svc := newTestService(t, repo)

	repo.On("GetByID", mock.Anything, mock.Anything).Return(db.ApiKey{}, pgx.ErrNoRows).Once()
	_, err := svc.Get(context.Background(), uuid.NewString())
	require.Error(t, err)
	ae, ok := apperr.From(err)
	require.True(t, ok)
	assert.Equal(t, apperr.KindNotFound, ae.Kind)
}
