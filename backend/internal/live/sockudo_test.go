package live

import (
	"testing"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
)

// TestPublishToSockudo — publish một event tới Sockudo thật (localhost:6002,
// app demo-app/demo-key/demo-secret — chạy qua docker compose).
// Đây là integration test: bỏ qua nếu Sockudo không chạy (go test không có compose).
func TestPublishToSockudo(t *testing.T) {
	cfg := config.LiveConfig{
		SockudoURL:       "http://localhost:6002",
		SockudoAppID:     "demo-app",
		SockudoAppKey:    "demo-key",
		SockudoAppSecret: "demo-secret",
	}
	p := newSockudoPublisher(cfg)
	if err := p.publish("user_publish_test", "event", statusEvent("idle")); err != nil {
		t.Fatalf("publish thất bại: %v", err)
	}
}
