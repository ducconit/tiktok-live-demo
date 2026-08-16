// Package logger — wrapper slog chuẩn cho app: stdout + file daily + level động.
//
//   - Engine: log/slog (stdlib) — JSON handler, level config hoá qua slog.LevelVar
//   - File: mỗi ngày 1 file (vd logs/app-2026-08-12.log), tự tạo thư mục,
//     tự dọn file cũ hơn keep_days
//   - Level: đổi runtime qua SetLevel (nối với dynamic config log.level)
//
// Zero-config: Config zero → stdout debug (hành vi cũ). Bật file qua FileEnabled.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config — cấu hình logger (từ core/config LogConfig).
type Config struct {
	Level        string // debug | info | warn | error — mặc định "info"
	FileEnabled  bool   // ghi thêm vào file daily
	FileDir      string // thư mục chứa file log (vd "logs")
	FileKeepDays int    // giữ N ngày, xoá file cũ hơn
	AppName      string // prefix tên file (vd "app" → app-2026-08-12.log)
}

// Manager — logger + level động.
type Manager struct {
	level  *slog.LevelVar
	logger *slog.Logger
	file   *dailyWriter // nil khi FileEnabled=false
}

// New — khởi tạo logger theo config.
func New(cfg Config) (*Manager, error) {
	level, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	lv := &slog.LevelVar{}
	lv.Set(level)

	var writers []io.Writer
	writers = append(writers, os.Stdout)

	var fw *dailyWriter
	if cfg.FileEnabled {
		fw, err = newDailyWriter(cfg.FileDir, cfg.AppName, cfg.FileKeepDays)
		if err != nil {
			return nil, fmt.Errorf("logger: init daily file: %w", err)
		}
		writers = append(writers, fw)
	}

	handler := slog.NewJSONHandler(io.MultiWriter(writers...), &slog.HandlerOptions{Level: lv})
	return &Manager{level: lv, logger: slog.New(handler), file: fw}, nil
}

// Logger — slog.Logger đã cấu hình (gọi slog.SetDefault(Logger()) ở main).
func (m *Manager) Logger() *slog.Logger { return m.logger }

// SetLevel — đổi level runtime (dynamic config log.level). Không panic.
func (m *Manager) SetLevel(l slog.Level) { m.level.Set(l) }

// Close — đóng file log (gọi khi shutdown).
func (m *Manager) Close() error {
	if m.file != nil {
		return m.file.Close()
	}
	return nil
}

// ParseLevel — "debug"|"info"|"warn"|"error" → slog.Level (không phân biệt hoa thường).
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("logger: level %q không hợp lệ (debug|info|warn|error)", s)
	}
}
