# Storage — multiple disk (kiểu Laravel Filesystem)

`backend/core/storage` — abstraction lưu trữ file theo mô hình **Laravel Filesystem**:

```
Manager ── Disk("public")  →  local driver (thư mục, serve công khai)
       └── Disk("private") →  local driver (chỉ backend truy cập)
       └── Disk("s3")      →  s3 driver (MinIO / AWS S3 compatible)
       └── Disk("")        →  DISK MẶC ĐỊNH (storage.default_disk)
```

App **không bao giờ đụng driver trực tiếp** — chỉ gọi `storage.Disk(name)` → `Disk`
interface. Đổi driver/đường dẫn chỉ cần sửa config, không sửa code.

## Disk interface

```go
type Disk interface {
    Name() string                                    // tên disk
    Put(name string, content []byte) error           // ghi file (tự tạo thư mục cha)
    Get(name string) ([]byte, error)                 // đọc nội dung
    Delete(name string) error                        // xoá (không lỗi nếu không tồn tại)
    Exists(name string) bool
    Size(name string) (int64, error)
    URL(name string) string                          // URL công khai (private → "")
    TemporaryURL(name string, ttl time.Duration) (string, error) // s3: presigned; local: = URL
    Health(ctx context.Context) error                // metrics service_up
}
```

- Tên file = `"thư mục/tên"` (không bắt đầu `/`). **Path traversal tự chặn** (`../`, đường dẫn tuyệt đối).
- Upload ảnh chuẩn: `storage.UploadImage(ctx, disk, fileHeader)` — kiểm tra kích thước (5MB) +
  đuôi cho phép (jpg/jpeg/png/webp/gif), key ngẫu nhiên `avatars/<uuid>.<ext>`.

## Config

```yaml
# backend/config.yml
storage:
  default_disk: public        # disk dùng khi gọi Disk("")
  disks:
    public:                   # local — serve công khai qua route /storage/*
      driver: local
      root: ./storage/public
      url: /storage
    private:                  # local — không có URL công khai (url rỗng)
      driver: local
      root: ./storage/private
    s3:                       # S3-compatible (MinIO) — lazy connect khi dùng
      driver: s3
      endpoint: localhost:9000
      access_key: minioadmin
      secret_key: minioadmin_dev
      use_ssl: false
      bucket: uploads
```

- **Env**: `STORAGE_DEFAULT_DISK`; `MINIO_*` (giữ tên env cũ) → `storage.disks.s3.*`.
- **Defaults (zero-config)**: 2 disk local `public` + `private`, default = `public` — không set
  gì vẫn chạy, file vào `./storage/public`.
- **Lazy s3**: disk s3 KHÔNG kết nối lúc boot (app không chết khi MinIO down) — kết nối ở
  lần `Disk("s3")` đầu tiên, lỗi hiện rõ ở lần dùng đó (không cache lỗi — retry lần sau).

## Serve file công khai (public local disk)

Route `GET /storage/*filepath` serve thư mục của disk `public` (nếu là local có URL prefix)
— tương đương `php artisan storage:link` của Laravel. File public → URL trả về dùng trực tiếp
trên web; file private → `URL()` trả `""`, truy cập qua `TemporaryURL()` (s3 presigned).

## Dùng trong code

```go
disk, err := m.Disk("")        // default (public)
disk, err := m.Disk("private") // private local
disk, err := m.Disk("s3")      // s3 (MinIO) — lazy connect

disk.Put("reports/2026-08.csv", data)        // ghi
data, _ := disk.Get("reports/2026-08.csv")   // đọc
u, _ := disk.TemporaryURL("reports/2026-08.csv", 24*time.Hour) // s3 presigned
```

Ví dụ thực tế: avatar upload (`internal/user/account.go`) → disk `public` → URL
`/storage/avatars/<uuid>.png` serve bởi gin.

## Test

- Local disk: unit test đầy đủ (put/get/delete/url/traversal/health...) — không cần dịch vụ ngoài.
- S3 disk: **integration** — chạy với MinIO thật: set `MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY`
  (+ `MINIO_BUCKET` mặc định `uploads-test`). Thiếu env → `t.Skip` (CI không có MinIO).
