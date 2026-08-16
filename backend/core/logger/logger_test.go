package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want slog.Level
		ok   bool
	}{
		{"debug", slog.LevelDebug, true},
		{"DEBUG", slog.LevelDebug, true},
		{"info", slog.LevelInfo, true},
		{"", slog.LevelInfo, true}, // mặc định info
		{"warn", slog.LevelWarn, true},
		{"warning", slog.LevelWarn, true},
		{"error", slog.LevelError, true},
		{"trace", 0, false},
		{"123", 0, false},
	}
	for _, c := range cases {
		got, err := ParseLevel(c.in)
		if c.ok {
			assert.NoError(t, err, c.in)
			assert.Equal(t, c.want, got, c.in)
		} else {
			assert.Error(t, err, c.in)
		}
	}
}

func TestNew_LevelAndDynamic(t *testing.T) {
	m, err := New(Config{Level: "error"})
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	ctx := context.Background()
	assert.False(t, m.Logger().Enabled(ctx, slog.LevelInfo), "level error → info bị chặn")
	assert.True(t, m.Logger().Enabled(ctx, slog.LevelError))

	m.SetLevel(slog.LevelDebug)
	assert.True(t, m.Logger().Enabled(ctx, slog.LevelDebug), "SetLevel → debug được phép")
}

func TestNew_InvalidLevel(t *testing.T) {
	_, err := New(Config{Level: "trace"})
	assert.Error(t, err)
}

func TestNew_FileEnabledCreatesDir(t *testing.T) {
	dir := t.TempDir()
	m, err := New(Config{Level: "info", FileEnabled: true, FileDir: dir, AppName: "app"})
	require.NoError(t, err)
	defer func() { _ = m.Close() }()

	assert.FileExists(t, m.file.path())
	assert.True(t, strings.HasSuffix(m.file.path(), time.Now().Format("2006-01-02")+".log"))
}

func TestDailyWriter_WritesToDailyFile(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyWriter(dir, "app", 14)
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	n, err := w.Write([]byte("hello-logger\n"))
	require.NoError(t, err)
	assert.Equal(t, len("hello-logger\n"), n)

	today := time.Now().Format("2006-01-02")
	content, err := os.ReadFile(filepath.Join(dir, "app-"+today+".log"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "hello-logger")
}

func TestDailyWriter_RotatesOnNewDay(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyWriter(dir, "app", 14)
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	_, err = w.Write([]byte("day-one\n"))
	require.NoError(t, err)
	first := w.path()

	// Giả lập sang ngày khác (internal day khác hôm nay) → Write phải rotate
	w.mu.Lock()
	w.day = "2000-01-01"
	w.mu.Unlock()

	_, err = w.Write([]byte("day-two\n"))
	require.NoError(t, err)

	today := time.Now().Format("2006-01-02")
	assert.True(t, strings.HasSuffix(w.path(), today+".log"), "phải rotate sang file hôm nay")
	assert.Equal(t, first, w.path(), "cùng ngày → cùng file (không tạo file trùng)")
	content, _ := os.ReadFile(filepath.Join(dir, "app-"+today+".log"))
	assert.Contains(t, string(content), "day-two", "ghi tiếp vào file hôm nay")
}

func TestDailyWriter_CleansOldFiles(t *testing.T) {
	dir := t.TempDir()
	// File cũ (15 ngày trước) + file hôm nay
	old := filepath.Join(dir, "app-2020-01-01.log")
	today := filepath.Join(dir, "app-"+time.Now().Format("2006-01-02")+".log")
	require.NoError(t, os.WriteFile(old, []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(today, []byte("y"), 0o644))
	// File lạ không cùng prefix — không được đụng
	stranger := filepath.Join(dir, "other-2020-01-01.log")
	require.NoError(t, os.WriteFile(stranger, []byte("z"), 0o644))

	w, err := newDailyWriter(dir, "app", 14)
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	// open() gọi cleanup — file cũ cùng prefix phải bị xoá
	_, err = os.Stat(old)
	assert.True(t, os.IsNotExist(err), "file cũ hơn keepDays phải bị xoá")
	_, err = os.Stat(stranger)
	assert.NoError(t, err, "file khác prefix không được đụng")
}

func TestDailyWriter_AppendsAcrossWrites(t *testing.T) {
	dir := t.TempDir()
	w, err := newDailyWriter(dir, "app", 14)
	require.NoError(t, err)
	defer func() { _ = w.Close() }()

	_, _ = w.Write([]byte("line-1\n"))
	_, _ = w.Write([]byte("line-2\n"))

	today := time.Now().Format("2006-01-02")
	content, _ := os.ReadFile(filepath.Join(dir, "app-"+today+".log"))
	assert.Contains(t, string(content), "line-1")
	assert.Contains(t, string(content), "line-2")
}
