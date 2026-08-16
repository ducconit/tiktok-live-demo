package main

import (
	"net/url"
	"strings"
)

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

func friendlyError(err error) string {
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "429"):
		return "Đã đạt giới hạn request. Đợi vài phút rồi thử lại."
	case strings.Contains(lower, "offline") || strings.Contains(lower, "not live"):
		return "User này hiện không đang LIVE."
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		return "Kết nối quá hạn (timeout). Thử lại sau."
	default:
		return msg
	}
}
