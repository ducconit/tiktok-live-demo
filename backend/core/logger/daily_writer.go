package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// dailyWriter — io.Writer ghi vào file chia theo ngày: <dir>/<prefix>-YYYY-MM-DD.log.
//
//   - Tự tạo thư mục nếu chưa có
//   - Đổi ngày (phát hiện qua time.Now() mỗi Write) → mở file mới, đóng file cũ
//   - Dọn file cũ hơn keepDays (chỉ đụng file cùng prefix, định dạng đúng)
//
// An toàn đồng thời (nhiều goroutine log cùng lúc).
type dailyWriter struct {
	dir      string
	prefix   string
	keepDays int

	mu   sync.Mutex
	day  string // YYYY-MM-DD của file đang mở
	file *os.File
}

func newDailyWriter(dir, prefix string, keepDays int) (*dailyWriter, error) {
	if dir == "" {
		dir = "logs"
	}
	if prefix == "" {
		prefix = "app"
	}
	if keepDays <= 0 {
		keepDays = 14
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	w := &dailyWriter{dir: dir, prefix: prefix, keepDays: keepDays}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

// Write — ghi vào file của ngày hiện tại; rotate nếu đã sang ngày mới.
func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if today != w.day {
		_ = w.file.Close() // lỗi đóng file cũ bỏ qua — rotate vẫn tiếp tục
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	return w.file.Write(p)
}

// open — mở file của ngày hiện tại (append) + dọn file cũ.
func (w *dailyWriter) open() error {
	w.day = time.Now().Format("2006-01-02")
	name := w.prefix + "-" + w.day + ".log"
	f, err := os.OpenFile(filepath.Join(w.dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.cleanup()
	return nil
}

// cleanup — xoá file cùng prefix cũ hơn keepDays (lỗi bỏ qua — không chặn log).
func (w *dailyWriter) cleanup() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -w.keepDays)
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), w.prefix+"-") || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		day, err := time.Parse("2006-01-02", strings.TrimSuffix(strings.TrimPrefix(e.Name(), w.prefix+"-"), ".log"))
		if err != nil {
			continue // không phải file của chúng ta (tên không đúng định dạng)
		}
		if day.Before(cutoff) {
			_ = os.Remove(filepath.Join(w.dir, e.Name()))
		}
	}
}

// Close — đóng file hiện tại.
func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// path — đường dẫn file đang mở (debug/test).
func (w *dailyWriter) path() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return ""
	}
	return w.file.Name()
}
