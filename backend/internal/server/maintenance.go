package server

import (
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
	"github.com/ducconit/tiktok-live-platform/backend/core/response"
	"github.com/gin-gonic/gin"
)

// maintenance — chặn mọi API bằng 503 khi app.maintenance_mode bật (dynamic config),
// ngoại trừ /config (client cần biết hệ thống đang bảo trì).
func (s *Server) maintenance() gin.HandlerFunc {
	return func(c *gin.Context) {
		if maintenanceOn(s.cfg.GetDynamic("app.maintenance_mode")) &&
			!strings.HasSuffix(c.Request.URL.Path, "/config") {
			// Chuẩn response: 503 service unavailable
			response.Error(c, apperr.New(apperr.KindServiceUnavailable, "503",
				"Hệ thống đang bảo trì, vui lòng thử lại sau"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// maintenanceOn — true khi dynamic value là bool true hoặc string "true"
// (config:set lưu string trong DB).
func maintenanceOn(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	}
	return false
}
