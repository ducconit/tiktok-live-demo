package apikey

import (
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/db"
)

// DTO — response an toàn: KHÔNG bao giờ lộ key_hash, chỉ hiển thị prefix.
type DTO struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"` // vd "gvs_live_ab12..."
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at"`
	IsActive   bool       `json:"is_active"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedBy  string     `json:"created_by"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CreatedDTO — trả key plaintext ĐÚNG 1 lần (tạo/rotate) + thông tin key.
type CreatedDTO struct {
	Key string `json:"key"` // hiển thị 1 lần, KHÔNG lưu lại ở client sau khi copy
	DTO
}

// ToDTO map db.ApiKey → DTO.
func ToDTO(k db.ApiKey) DTO {
	return DTO{
		ID:         k.ID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     k.Scopes,
		ExpiresAt:  k.ExpiresAt,
		IsActive:   k.IsActive,
		LastUsedAt: k.LastUsedAt,
		CreatedBy:  derefStr(k.CreatedBy),
		RevokedAt:  k.RevokedAt,
		CreatedAt:  k.CreatedAt,
		UpdatedAt:  k.UpdatedAt,
	}
}

func ToDTOs(keys []db.ApiKey) []DTO {
	out := make([]DTO, 0, len(keys))
	for _, k := range keys {
		out = append(out, ToDTO(k))
	}
	return out
}
