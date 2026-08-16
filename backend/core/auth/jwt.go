package auth

import (
	"errors"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/golang-jwt/jwt/v5"
)

// Claims — payload access token.
type Claims struct {
	UserID string   `json:"sub"`
	Roles  []string `json:"roles"`
	Perms  []string `json:"perms"`
	jwt.RegisteredClaims
}

// jwtService — sign/verify access token (HS256).
type jwtService struct {
	secret []byte
	ttl    time.Duration
}

// VerifyAccessToken — parse + validate access token (dùng cho middleware).
func VerifyAccessToken(token, secret string) (*Claims, error) {
	j := newJWTService(config.JWTConfig{Secret: secret})
	return j.parse(token)
}

func newJWTService(cfg config.JWTConfig) *jwtService {
	return &jwtService{secret: []byte(cfg.Secret), ttl: cfg.AccessTTL}
}

func (j *jwtService) sign(userID string, roles, perms []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Roles:  roles,
		Perms:  perms,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

// parse — verify + trả Claims; lỗi → Unauthorized.
func (j *jwtService) parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, apperr.Unauthorized("invalid_token", "error.invalid_token")
	}
	return claims, nil
}
