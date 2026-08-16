package mailer

import (
	"context"
	"testing"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_LocalhostNoTLS — New không kết nối SMTP (chỉ parse config) → test được.
func TestNew_LocalhostNoTLS(t *testing.T) {
	m, err := New(config.MailConfig{Host: "localhost", Port: 1025, From: "no-reply@test.dev"})
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "no-reply@test.dev", m.from)
	t.Cleanup(func() { _ = m.Close() })
}

// TestNew_WithAuth — cấu hình username/password → client có SMTPAuth.
func TestNew_WithAuth(t *testing.T) {
	m, err := New(config.MailConfig{
		Host: "localhost", Port: 1025, From: "a@b.dev",
		Username: "user", Password: "pass",
	})
	require.NoError(t, err)
	assert.NotNil(t, m)
	t.Cleanup(func() { _ = m.Close() })
}

// TestNew_TLSPolicy — policy none/starttls chấp nhận (không connect).
func TestNew_TLSPolicy(t *testing.T) {
	for _, policy := range []string{"none", "starttls", "auto"} {
		m, err := New(config.MailConfig{Host: "mailpit", Port: 1025, From: "a@b.dev", TLSPolicy: policy})
		require.NoError(t, err, "policy %s", policy)
		assert.NotNil(t, m, "policy %s", policy)
		t.Cleanup(func() { _ = m.Close() })
	}
}

// TestNew_InvalidHost — host rỗng → lỗi init (không connect).
func TestNew_InvalidHost(t *testing.T) {
	_, err := New(config.MailConfig{Host: "", Port: 1025})
	require.Error(t, err)
}

// TestSend_NeedsSMTPServer — SKIP: cần SMTP server thật (Mailpit/dev).
// Case được biết (gửi email thành công với body HTML) nhưng chưa kiểm chứng
// trong CI — chạy manual với `docker compose up mailpit`.
func TestSend_NeedsSMTPServer(t *testing.T) {
	// Thử connect Mailpit (compose dev) — có thì test thật, không có thì skip
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	m, err := New(config.MailConfig{Host: "localhost", Port: 1025, From: "no-reply@test.dev"})
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	err = m.Send(ctx, "to@test.dev", "Subject", "<p>Hello</p>")
	if err != nil {
		t.Skipf("SMTP không khả dụng (%v) — skip (case: gửi email HTML qua SMTP)", err)
	}
	// Nếu connect được — verify không lỗi (đã pass qua err == nil)
	assert.NoError(t, err)
}
