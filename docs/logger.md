# Logger — core/logger

> Khớp với `core/logger` + `cmd/app/main.go` — cập nhật khi code đổi.

## Engine

**`log/slog`** (stdlib Go) — JSON handler. **KHÔNG** dùng zerolog/logrus/zap
(so sánh slog vs zerolog: zerolog thắng về perf cực độ + LevelWriter, nhưng slog
đủ nhanh cho app HTTP, zero dependency, stdlib bền vững — skeleton chọn slog).

## Ghi ra đâu (mặc định: 2 nơi)

| Kênh | Mặc định | Ghi gì |
|---|---|---|
| **stdout** (console) | luôn bật | toàn bộ theo level |
| **file theo ngày** `logs/<app>-YYYY-MM-DD.log` | **BẬT** (`log.file_enabled=true`) | toàn bộ theo level (JSON) |

- File: tự tạo thư mục, tự **xoá file cũ hơn** `log.file_keep_days` (mặc định 14),
  thread-safe, append qua nhiều lần ghi.
- Cùng 1 JSON handler cho cả 2 kênh (`io.MultiWriter`).

## Log level

| Cấu hình | Giá trị |
|---|---|
| `LOG_LEVEL` env / `log.level` config | `debug` \| `info` \| `warn` \| `error` — mặc định `info` |
| Đổi runtime | `admin PUT /api/v1/admin/config {key: "log.level", value: "error"}` — đồng bộ mọi instance qua Redis pub/sub, **không restart** |

> Lưu ý: khi level = error, log của chính request đổi level cũng bị chặn
> (middleware log SAU handler) — hành vi bình thường.

## Quy tắc log (middleware HTTP tự động)

| HTTP status | Level | Kèm theo |
|---|---|---|
| ≥ 500 | `ERROR` | code, error chi tiết, details (lỗi hệ thống — bắt buộc ghi) |
| 400–499 | `WARN` | code, error (lỗi nghiệp vụ/client) |
| còn lại | `INFO` | — |

Mọi log JSON ghi đủ: `method, path, status, latency_ms, ip, request_id`.

## Config keys

```
log.level          # debug|info|warn|error (default info)
log.file_enabled   # ghi thêm file daily (default true)
log.file_dir       # thư mục (default "logs")
log.file_keep_days # giữ N ngày, xoá file cũ (default 14)
```

## Dùng trong code

```go
// main.go — khởi tạo + set default (toàn bộ app dùng slog.Info/Warn/Error/...)
lg, err := logger.New(logger.Config{Level: mgr.Cfg.Log.Level, FileEnabled: ...})
slog.SetDefault(lg.Logger())
mgr.OnChange("log.level", func(v any) { ... lg.SetLevel(lvl) ... })

// Mọi nơi khác — dùng trực tiếp slog:
slog.Info("order created", "order_id", id)
slog.Error("db connect fail", "err", err)
```
