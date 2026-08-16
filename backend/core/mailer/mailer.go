// Package mailer — gửi email qua SMTP (go-mail), dev dùng Mailpit (compose).
package mailer

import (
	"context"
	"fmt"
	"time"

	"github.com/ducconit/tiktok-live-platform/backend/core/config"
	"github.com/wneessen/go-mail"
)

// smtpTimeout — giới hạn gửi email.
const smtpTimeout = 10 * time.Second

// Mailer — client SMTP đóng gói (host/port/user/pass/from từ config).
type Mailer struct {
	client *mail.Client
	from   string
}

// New — tạo client SMTP. Chỉ lỗi khi cấu hình sai (không kết nối ngay).
func New(cfg config.MailConfig) (*Mailer, error) {
	opts := []mail.Option{
		mail.WithPort(int(cfg.Port)),
		mail.WithTimeout(smtpTimeout),
	}
	// TLS policy: auto (local → NoTLS, remote → STARTTLS bắt buộc — an toàn mặc định)
	// | none (Mailpit/dev SMTP không STARTTLS) | starttls (ép TLSMandatory)
	switch cfg.TLSPolicy {
	case "none":
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	case "starttls":
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	default: // auto
		if cfg.Host == "localhost" || cfg.Host == "127.0.0.1" {
			opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
		}
	}
	if cfg.Username != "" || cfg.Password != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.Username), mail.WithPassword(cfg.Password))
	}
	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("mailer: init: %w", err)
	}
	return &Mailer{client: client, from: cfg.From}, nil
}

func (m *Mailer) Close() error { return m.client.Close() }

// Send — gửi email HTML đơn giản.
func (m *Mailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	msg := mail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("mailer: from: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("mailer: to: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, htmlBody)

	if err := m.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("mailer: send %s: %w", to, err)
	}
	return nil
}

// Health — ping SMTP (dial + close) — dùng cho metrics service_up.
func (m *Mailer) Health(ctx context.Context) error {
	if err := m.client.DialWithContext(ctx); err != nil {
		return fmt.Errorf("mailer: dial: %w", err)
	}
	return m.client.Close()
}
