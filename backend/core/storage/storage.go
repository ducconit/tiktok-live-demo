// Package storage — multiple disk (kiểu Laravel Filesystem).
//
//	Manager → Disk(name) → Put/Get/Delete/Exists/Size/URL
//
// Disk là đơn vị lưu trữ độc lập với driver riêng:
//   - local: thư mục trên đĩa (public → serve qua web, private → chỉ backend)
//   - s3:    S3-compatible (MinIO, AWS S3...) — bucket + credentials
//
// Mặc định 2 disk local: public (URL công khai /storage) + private.
// Disk mặc định (config storage.default_disk = "public") khi gọi Disk("").
package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

// contentType — ước lượng Content-Type từ đuôi file (chuẩn cho cả S3 + serve local).
func contentType(name string) string {
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// Lỗi chuẩn — dùng errors.Is để phân loại.
var (
	// ErrDiskNotFound — disk không tồn tại trong config.
	ErrDiskNotFound = errors.New("storage: disk không tồn tại")
	// ErrFileTooLarge — file vượt giới hạn kích thước.
	ErrFileTooLarge = errors.New("storage: file quá lớn (tối đa 5 MB)")
	// ErrUnsupportedDriver — driver không được hỗ trợ (chỉ local | s3).
	ErrUnsupportedDriver = errors.New("storage: driver không hỗ trợ (local | s3)")
	// ErrInvalidPath — tên file chứa đường dẫn không hợp lệ (vd ../).
	ErrInvalidPath = errors.New("storage: tên file không hợp lệ")
)

// Disk — 1 disk lưu trữ (Laravel Storage facade). Tên file = "thư mục/tên" (không bắt đầu /).
type Disk interface {
	// Name — tên disk (khớp key trong config storage.disks).
	Name() string
	// Put — ghi file (ghi đè nếu tồn tại; tự tạo thư mục cha).
	Put(name string, content []byte) error
	// Get — đọc nội dung file.
	Get(name string) ([]byte, error)
	// Delete — xoá file (không lỗi nếu không tồn tại).
	Delete(name string) error
	// Exists — file có tồn tại không.
	Exists(name string) bool
	// Size — kích thước file (bytes).
	Size(name string) (int64, error)
	// URL — URL truy cập công khai (public disk); private disk trả chuỗi rỗng.
	URL(name string) string
	// TemporaryURL — URL truy cập tạm thời (s3: presigned; local: = URL).
	TemporaryURL(name string, ttl time.Duration) (string, error)
	// Health — kiểm tra disk sẵn sàng (dùng cho metrics service_up).
	Health(ctx context.Context) error
}

// maxAvatarSize — giới hạn upload ảnh (không magic number).
const maxAvatarSize = 5 << 20 // 5MB

// AllowedImageExts — đuôi file ảnh được phép upload.
var AllowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
}

// UploadImage — lưu file ảnh (avatar...) từ multipart vào disk, trả key + URL.
// Key = "<folder>/<sha256-16><ext>" — hash nội dung: dedup + không đoán tên.
// folder ví dụ: "users/<userID>/avatars".
func UploadImage(ctx context.Context, disk Disk, folder string, fh *multipart.FileHeader) (key, url string, err error) {
	if fh.Size > maxAvatarSize {
		return "", "", ErrFileTooLarge
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !AllowedImageExts[ext] {
		return "", "", fmt.Errorf("storage: định dạng %q không hỗ trợ (%s)", ext, "jpg/jpeg/png/webp/gif")
	}

	file, err := fh.Open()
	if err != nil {
		return "", "", fmt.Errorf("storage: mở file: %w", err)
	}
	defer func() { _ = file.Close() }()

	buf := make([]byte, fh.Size)
	if _, err := io.ReadFull(file, buf); err != nil {
		return "", "", fmt.Errorf("storage: đọc file: %w", err)
	}

	sum := sha256.Sum256(buf)
	key = fmt.Sprintf("%s/%x%s", folder, sum[:8], ext)
	if err := disk.Put(key, buf); err != nil {
		return "", "", fmt.Errorf("storage: put %s: %w", key, err)
	}
	return key, disk.URL(key), nil
}

// validatePath — tên file hợp lệ: không bắt đầu bằng /, không chứa .. (path traversal).
func validatePath(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return ErrInvalidPath
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." || seg == "." {
			return ErrInvalidPath
		}
	}
	return nil
}
