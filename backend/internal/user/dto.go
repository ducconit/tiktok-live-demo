package user

import (
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/db"
)

// UserDTO — response an toàn (KHÔNG bao giờ lộ password_hash).
type UserDTO struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	AvatarURL   string     `json:"avatar_url"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToDTO map db.User → UserDTO.
func ToDTO(u db.User) UserDTO {
	return UserDTO{
		ID:          u.ID,
		Email:       u.Email,
		FullName:    u.FullName,
		AvatarURL:   u.AvatarUrl,
		IsActive:    u.IsActive,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func ToDTOs(users []db.User) []UserDTO {
	out := make([]UserDTO, 0, len(users))
	for _, u := range users {
		out = append(out, ToDTO(u))
	}
	return out
}
