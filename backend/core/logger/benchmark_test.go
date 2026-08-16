package logger

import (
	"io"
	"log/slog"
	"testing"
)

// benchLogger — logger chuẩn, ghi ra io.Discard (không đo I/O disk).
func benchLogger(b *testing.B) *Manager {
	b.Helper()
	lv := &slog.LevelVar{}
	lv.Set(slog.LevelInfo)
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: lv})
	return &Manager{level: lv, logger: slog.New(handler)}
}

func BenchmarkManager_Info_OneField(b *testing.B) {
	m := benchLogger(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.logger.Info("request", "path", "/api/v1/users")
	}
}

func BenchmarkManager_Info_FourFields(b *testing.B) {
	m := benchLogger(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.logger.Info("http",
			"method", "GET", "path", "/api/v1/admin/users",
			"status", 200, "latency_ms", 12,
		)
	}
}

// BenchmarkSlog_JSONHandler_Attrs — baseline slog thuần (so sánh với wrapper).
func BenchmarkSlog_JSONHandler_Attrs(b *testing.B) {
	h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo})
	lg := slog.New(h)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lg.Info("msg", slog.Int("n", i), slog.String("s", "value"))
	}
}
