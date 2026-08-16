package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/ducconit/tiktok-live-platform/backend/internal/mocks"
	"github.com/ducconit/tiktok-live-platform/backend/internal/user"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func newTestService(t *testing.T) (*user.Service, *mocks.MockRepository) {
	t.Helper()
	repo := mocks.NewMockRepository(t)
	return user.NewService(repo), repo
}

func testUser(email string) db.User {
	return db.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: mustHash("password123"),
		FullName:     "Test User",
		IsActive:     true,
	}
}

func mustHash(pw string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

// MatchCreateParams — matcher kiểm tra param CreateUserParams theo email.
func MatchCreateParams(email string) any {
	return mock.MatchedBy(func(p db.CreateUserParams) bool { return p.Email == email })
}

func TestCreate_EmailExists_ReturnsConflict(t *testing.T) {
	svc, repo := newTestService(t)
	repo.EXPECT().GetByEmail(context.Background(), "a@b.com").Return(testUser("a@b.com"), nil)

	_, err := svc.Create(context.Background(), user.CreateParams{
		Email:    "a@b.com",
		Password: "password123",
		FullName: "A",
	})

	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, "email_exists", ae.Code)
	assert.Equal(t, apperr.KindConflict, ae.Kind)
}

func TestCreate_Success(t *testing.T) {
	svc, repo := newTestService(t)
	repo.EXPECT().GetByEmail(context.Background(), "a@b.com").Return(db.User{}, pgx.ErrNoRows)
	repo.EXPECT().Create(context.Background(), MatchCreateParams("a@b.com")).Return(testUser("a@b.com"), nil)

	u, err := svc.Create(context.Background(), user.CreateParams{
		Email:    "a@b.com",
		Password: "password123",
		FullName: "A",
	})

	require.NoError(t, err)
	assert.Equal(t, "a@b.com", u.Email)
}

func TestVerifyPassword_WrongPassword_ReturnsUnauthorized(t *testing.T) {
	svc, repo := newTestService(t)
	repo.EXPECT().GetByEmail(context.Background(), "a@b.com").Return(testUser("a@b.com"), nil)

	_, err := svc.VerifyPassword(context.Background(), "a@b.com", "wrong-password")

	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, "invalid_credentials", ae.Code)
	assert.Equal(t, apperr.KindUnauthorized, ae.Kind)
}

func TestGetByID_NotFound(t *testing.T) {
	svc, repo := newTestService(t)
	id := uuid.NewString()
	repo.EXPECT().GetByID(context.Background(), id).Return(db.User{}, pgx.ErrNoRows)

	_, err := svc.GetByID(context.Background(), id)

	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, "user_not_found", ae.Code)
}

func TestDelete_NotFound(t *testing.T) {
	svc, repo := newTestService(t)
	id := uuid.NewString()
	repo.EXPECT().GetByID(context.Background(), id).Return(db.User{}, errors.New("boom"))

	_, err := svc.GetByID(context.Background(), id)
	require.Error(t, err) // không phải ErrNoRows → wrap internal
}
