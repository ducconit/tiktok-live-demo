package commands

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// LogsCmd — xem log app kiểu `tail`:
//
//	gvs logs          # 25 dòng cuối của file log hôm nay
//	gvs logs -n 100   # 100 dòng cuối
//	gvs logs -f       # follow — in dòng mới liên tục (Ctrl+C để thoát)
//
// File log daily: <log.file_dir>/<app.name>-YYYY-MM-DD.log (xem core/logger).
// Không cần DB — chỉ đọc file; app chưa chạy hôm nay → báo rõ, không lỗi mơ hồ.
func LogsCmd() *cobra.Command {
	var (
		follow bool
		lines  int
	)
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Xem log app (giống tail) — mặc định 25 dòng cuối hôm nay",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// config.Load log qua slog stdout — tạm discard để output logs SẠCH
			// (chỉ log file của app, không lẫn log khởi tạo của chính lệnh).
			slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
			cfg, err := loadCfg()
			if err != nil {
				return err
			}
			dir := cfg.Cfg.Log.FileDir
			if dir == "" {
				dir = "logs"
			}
			name := cfg.Cfg.App.Name
			path := filepath.Join(dir, fmt.Sprintf("%s-%s.log", name, time.Now().Format("2006-01-02")))
			return tailFile(path, lines, follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "f", "f", false, "theo dõi log mới (giống tail -f)")
	cmd.Flags().IntVarP(&lines, "n", "n", 25, "số dòng cuối hiển thị")
	return cmd
}

// tailFile — in N dòng cuối của file; follow=true thì tiếp tục in dòng mới.
// Dùng polling (500ms) — đủ cho log app, không thêm dependency fsnotify.
// Hỗ trợ rotate: file nhỏ lại / bị xoá → tự mở lại (log daily của core/logger).
func tailFile(path string, n int, follow bool) error {
	if n < 1 {
		n = 1
	}

	// In N dòng cuối (nếu file tồn tại)
	printTail(path, n)

	if !follow {
		return nil
	}

	fmt.Printf("⏳ Đang theo dõi %s (Ctrl+C để thoát)...\n", path)
	lastSize := fileSize(path)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		f, err := os.Open(path)
		if err != nil {
			// File chưa có / bị rotate tạm — chờ vòng sau
			lastSize = 0
			continue
		}
		st, err := f.Stat()
		if err != nil {
			_ = f.Close()
			continue
		}
		// File mới (rotate: size nhỏ hơn lastSize hoặc mtime mới hơn) → đọc từ đầu
		if st.Size() < lastSize {
			lastSize = 0
		}
		if st.Size() > lastSize {
			if _, err := f.Seek(lastSize, io.SeekStart); err == nil {
				_, _ = io.Copy(os.Stdout, f)
			}
			lastSize = st.Size()
		}
		_ = f.Close()
	}
	return nil
}

// printTail — in N dòng cuối của file (file không tồn tại → báo rõ).
func printTail(path string, n int) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("⚠️  Chưa có log hôm nay: %s\n   (chạy app trước: gvs / go run ./cmd/app)\n", path)
			return
		}
		fmt.Printf("⚠️  Không đọc được log: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()

	// Đọc hết file, giữ ring buffer N dòng cuối (file log không quá lớn)
	lines := make([]string, 0, n)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024) // dòng log dài (stack trace) không bị cắt
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	for _, l := range lines {
		fmt.Println(l)
	}
}

// fileSize — kích thước file hiện tại (0 nếu không tồn tại).
func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}
