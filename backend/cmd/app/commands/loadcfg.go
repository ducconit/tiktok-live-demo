package commands

import (
	"github.com/ducconit/tiktok-live-platform/backend/core/config"
)

// loadCfg — config chung cho mọi lệnh (đọc .env nếu có + env shell).
// Zero-config: không có .env/config.yml → dùng defaults (xem core/config).
func loadCfg() (*config.Manager, error) {
	return config.Load(".env")
}
