package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/auth"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/ducconit/tiktok-live-platform/backend/internal/mocks"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTestService(t *testing.T) (*auth.Service, *mocks.MockUserRepo, *mocks.MockRBACReader, *mocks.MockTokenStore) {
	t.Helper()
	users := mocks.NewMockUserRepo(t)
	rbac := mocks.NewMockRBACReader(t)
	tokens := mocks.NewMockTokenStore(t)
	svc := auth.NewService(config.JWTConfig{
		Secret:     "test-secret-0123456789abcdef0123456789abcdef",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
	}, users, rbac, tokens)
	return svc, users, rbac, tokens
}

func activeUser() db.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	return db.User{
		ID:              uuid.NewString(),
		Email:           "a@b.com",
		PasswordHash:    string(hash),
		IsActive:        true,
		EmailVerifiedAt: timePtr(time.Now()), // verified để qua login
	}
}

func validRefreshToken() db.RefreshToken {
	return db.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    uuid.NewString(),
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func timePtr(t time.Time) *time.Time { return &t }

func TestLogin_Success(t *testing.T) {
	svc, users, rbac, tokens := newTestService(t)
	u := activeUser()

	users.EXPECT().GetByEmail(context.Background(), "a@b.com").Return(u, nil)
	rbac.EXPECT().GetUserRoles(context.Background(), u.ID).Return([]db.Role{{ID: "admin"}}, nil)
	rbac.EXPECT().GetUserPermissions(context.Background(), u.ID).Return([]string{"users.read"}, nil)
	tokens.EXPECT().CreateRefreshToken(context.Background(), mock.Anything).Return(db.RefreshToken{}, nil)

	pair, err := svc.Login(context.Background(), "a@b.com", "password123")

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.Equal(t, 900, pair.ExpiresIn)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, users, _, _ := newTestService(t)
	users.EXPECT().GetByEmail(context.Background(), "a@b.com").Return(activeUser(), nil)

	_, err := svc.Login(context.Background(), "a@b.com", "wrong")

	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, "invalid_credentials", ae.Code)
	assert.Equal(t, apperr.KindUnauthorized, ae.Kind)
}

func TestLogin_DisabledAccount(t *testing.T) {
	svc, users, _, _ := newTestService(t)
	u := activeUser()
	u.IsActive = false
	users.EXPECT().GetByEmail(context.Background(), "a@b.com").Return(u, nil)

	_, err := svc.Login(context.Background(), "a@b.com", "password123")

	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, "account_disabled", ae.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, users, _, _ := newTestService(t)
	users.EXPECT().GetByEmail(context.Background(), "x@y.com").Return(db.User{}, pgx.ErrNoRows)

	_, err := svc.Login(context.Background(), "x@y.com", "password123")

	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, apperr.KindUnauthorized, ae.Kind)
}

func TestRefresh_Rotate(t *testing.T) {
	svc, users, rbac, tokens := newTestService(t)
	rt := validRefreshToken()
	u := activeUser()

	tokens.EXPECT().GetRefreshTokenByHash(context.Background(), mock.Anything).Return(rt, nil)
	users.EXPECT().GetByID(context.Background(), rt.UserID).Return(u, nil)
	tokens.EXPECT().RevokeRefreshToken(context.Background(), rt.ID).Return(nil)
	rbac.EXPECT().GetUserRoles(context.Background(), u.ID).Return([]db.Role{{ID: "admin"}}, nil)
	rbac.EXPECT().GetUserPermissions(context.Background(), u.ID).Return([]string{"users.read"}, nil)
	tokens.EXPECT().CreateRefreshToken(context.Background(), mock.Anything).Return(db.RefreshToken{}, nil)

	pair, err := svc.Refresh(context.Background(), "refresh-token-value")

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

func TestRefresh_Revoked(t *testing.T) {
	svc, _, _, tokens := newTestService(t)
	rt := validRefreshToken()
	rt.RevokedAt = timePtr(time.Now())
	tokens.EXPECT().GetRefreshTokenByHash(context.Background(), mock.Anything).Return(rt, nil)

	_, err := svc.Refresh(context.Background(), "refresh-token-value")

	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, "invalid_refresh_token", ae.Code)
}

func TestLogout_UnknownToken_NoError(t *testing.T) {
	svc, _, _, tokens := newTestService(t)
	tokens.EXPECT().GetRefreshTokenByHash(context.Background(), mock.Anything).Return(db.RefreshToken{}, pgx.ErrNoRows)

	err := svc.Logout(context.Background(), "whatever")
	require.NoError(t, err)
}
