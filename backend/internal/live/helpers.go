package live

import (
	"net/url"
	"strings"

	"github.com/ducconit/tiktok-live-platform/backend/core/apperr"
)

// normalizeUsername — bỏ @, path, query từ input (chấp nhận cả URL profile).
func normalizeUsername(raw string) string {
	s := strings.TrimSpace(raw)
	if u, err := url.Parse(s); err == nil && u.Path != "" && strings.Contains(s, "/") && strings.Contains(s, ".") {
		for _, part := range strings.Split(strings.Trim(u.Path, "/"), "/") {
			if strings.HasPrefix(part, "@") {
				return strings.TrimPrefix(part, "@")
			}
		}
	}
	s = strings.TrimPrefix(s, "@")
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return s
}

// classifyLiveError — map lỗi track TikTok sang AppError (message ID i18n).
func classifyLiveError(err error) *apperr.AppError {
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "429"):
		return apperr.New(apperr.KindTooManyRequests, "429", "error.live_rate_limit")
	case strings.Contains(lower, "offline") || strings.Contains(lower, "not live"):
		return apperr.New(apperr.KindNotFound, "404", "error.live_offline")
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		return apperr.New(apperr.KindServiceUnavailable, "503", "error.live_timeout")
	default:
		return apperr.New(apperr.KindServiceUnavailable, "503", "error.live_connect_failed")
	}
}
