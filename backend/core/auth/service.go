package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/ducconit/tiktok-live-platform/backend/core/database"
	"github.com/ducconit/tiktok-live-platform/backend/db"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// UserRepo — interface user repo mà auth cần (mock được bằng mockery).
type UserRepo interface {
	GetByEmail(ctx context.Context, email string) (db.User, error)
	GetByID(ctx context.Context, id string) (db.User, error)
}

// RBACReader — đọc roles/perms của user để nhét vào claims.
type RBACReader interface {
	GetUserRoles(ctx context.Context, userID string) ([]db.Role, error)
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)
}

// TokenStore — vòng đời refresh token.
type TokenStore interface {
	CreateRefreshToken(ctx context.Context, p db.CreateRefreshTokenParams) (db.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, hash string) (db.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
	RevokeUserRefreshTokens(ctx context.Context, userID string) error
}

// tokenStore — implement TokenStore qua sqlc; read (replica) / write (master).
type tokenStore struct {
	rw *db.Queries
	ro *db.Queries
}

func NewTokenStore(p *database.Pool) TokenStore {
	return &tokenStore{rw: db.New(p.Write()), ro: db.New(p.Read())}
}

func (t *tokenStore) CreateRefreshToken(ctx context.Context, p db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	return t.rw.CreateRefreshToken(ctx, p)
}

func (t *tokenStore) GetRefreshTokenByHash(ctx context.Context, hash string) (db.RefreshToken, error) {
	return t.ro.GetRefreshTokenByHash(ctx, hash)
}

func (t *tokenStore) RevokeRefreshToken(ctx context.Context, id string) error {
	return t.rw.RevokeRefreshToken(ctx, id)
}

func (t *tokenStore) RevokeUserRefreshTokens(ctx context.Context, userID string) error {
	return t.rw.RevokeUserRefreshTokens(ctx, userID)
}

// Tokens — cặp token trả về khi login/refresh.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // giây
}

// Service — nghiệp vụ auth (login/refresh/logout).
type Service struct {
	jwt    *jwtService
	users  UserRepo
	rbac   RBACReader
	tokens TokenStore
	cfg    config.JWTConfig
}

func NewService(cfg config.JWTConfig, users UserRepo, rbac RBACReader, tokens TokenStore) *Service {
	return &Service{jwt: newJWTService(cfg), users: users, rbac: rbac, tokens: tokens, cfg: cfg}
}

// Login — xác thực email/password, trả cặp token.
func (s *Service) Login(ctx context.Context, email, password string) (Tokens, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tokens{}, apperr.Unauthorized("invalid_credentials", "error.bad_credentials")
	}
	if err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}
	if !user.IsActive {
		return Tokens{}, apperr.Forbidden("account_disabled", "error.account_disabled")
	}
	if user.EmailVerifiedAt == nil {
		// Chặn đăng nhập khi chưa xác thực email (admin seed đã verified)
		return Tokens{}, apperr.Forbidden("email_not_verified", "error.email_not_verified")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return Tokens{}, apperr.Unauthorized("invalid_credentials", "error.bad_credentials")
	}
	return s.issuePair(ctx, user)
}

// Refresh — rotate refresh token, trả cặp mới.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	rt, err := s.tokens.GetRefreshTokenByHash(ctx, hashToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return Tokens{}, apperr.Unauthorized("invalid_refresh_token", "error.invalid_refresh_token")
	}
	if err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}
	if rt.RevokedAt != nil || time.Now().After(rt.ExpiresAt) {
		return Tokens{}, apperr.Unauthorized("invalid_refresh_token", "error.refresh_token_expired")
	}

	user, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}
	if !user.IsActive {
		return Tokens{}, apperr.Forbidden("account_disabled", "error.account_disabled")
	}

	// rotate: vô hiệu token cũ trước, cấp cặp mới
	if err := s.tokens.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}
	return s.issuePair(ctx, user)
}

// Logout — vô hiệu refresh token (idempotent).
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	rt, err := s.tokens.GetRefreshTokenByHash(ctx, hashToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperr.WrapInternal(err)
	}
	return s.tokens.RevokeRefreshToken(ctx, rt.ID)
}

// Me — thông tin user hiện tại.
func (s *Service) Me(ctx context.Context, userID string) (db.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, apperr.NotFound("user_not_found", "Không tìm thấy người dùng")
	}
	if err != nil {
		return db.User{}, apperr.WrapInternal(err)
	}
	return u, nil
}

// issuePair — access token (claims roles/perms) + refresh token (random, lưu hash).
func (s *Service) issuePair(ctx context.Context, user db.User) (Tokens, error) {
	roles, err := s.rbac.GetUserRoles(ctx, user.ID)
	if err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}
	perms, err := s.rbac.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}

	roleSlugs := make([]string, 0, len(roles))
	for _, r := range roles {
		roleSlugs = append(roleSlugs, r.ID)
	}

	access, err := s.jwt.sign(user.ID, roleSlugs, perms)
	if err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}
	expiresAt := time.Now().Add(s.cfg.RefreshTTL)
	if _, err := s.tokens.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashToken(refresh),
		ExpiresAt: expiresAt,
	}); err != nil {
		return Tokens{}, apperr.WrapInternal(err)
	}

	return Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.AccessTTL.Seconds()),
	}, nil
}

// hashToken — sha256 hex của refresh token (không lưu token thô trong DB).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// newRefreshToken — 32 bytes random, hex 64 ký tự.
func newRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
