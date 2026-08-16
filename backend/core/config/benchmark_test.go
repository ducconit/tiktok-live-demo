package config

import (
	"testing"
)

// benchManager — Manager defaults (không .env, không DSN — đọc trong memory).
func benchManager(b *testing.B) *Manager {
	b.Helper()
	m, err := Load("")
	if err != nil {
		b.Fatal(err)
	}
	return m
}

func BenchmarkManager_String(b *testing.B) {
	m := benchManager(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if s := m.String("server.port", "3300"); s == "" {
			b.Fatal("empty")
		}
	}
}

func BenchmarkManager_Int(b *testing.B) {
	m := benchManager(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Int("log.file_keep_days", 14)
	}
}

func BenchmarkManager_GetDynamic(b *testing.B) {
	m := benchManager(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.GetDynamic("log.level")
	}
}
