# Config — nguồn & thứ tự ưu tiên

> Khớp với `core/config` — cập nhật khi code đổi.

## Cấu trúc file (2 tầng)

```
.env                    # root — DOCKER COMPOSE (optional): FORWARD_* port, POSTGRES_*, MINIO_ROOT_*, VALKEY_PASSWORD
backend/.env            # BACKEND (optional): mọi biến backend — xem backend/.env.example
backend/config.yml      # BACKEND config YAML (optional) — mẫu backend/config.example.yml
```

- App load `.env` + `config.yml` từ **CWD** — chạy chuẩn: `cd backend && go run ./cmd/app`
  (docker: `backend/.env` + `backend/config.yml` mount vào `/app/`)
- Root `.env` KHÔNG được backend đọc (chỉ docker compose substitute `${VAR}`)

## Thứ tự ưu tiên (CAO → THẤP)

```
① OS env            (luôn thắng — vd SERVER_PORT=8800)
② CONFIG_DSN        (file://config.yml | postgres://... bảng app_config)
③ file .env         (dotenv — map qua envBindings: APP_ENV → app.env...)
④ defaults          (trong code — zero-config, chạy được ngay không cần gì)
```

Điểm mấu chốt: **.env đọc TRƯỚC CONFIG_DSN** — vì CONFIG_DSN thường nằm trong .env,
có env mới biết lấy config từ file hay từ database.

## CONFIG_DSN — nguồn cấu hình chính

| DSN | Ý nghĩa |
|---|---|
| `file://config.yml` (mặc định) | YAML ở pwd — zero-config, không có file → dùng defaults |
| `postgres://user:pass@host:5432/db` | Config trong bảng `app_config(key, value)` — **multi-instance** |

Khi dùng postgres DSN → **Redis/Valkey BẮT BUỘC** (đồng bộ dynamic config giữa các
instance qua pub/sub) — lỗi → retry, không degrade.

## Env bindings (env var → config key)

`APP_ENV` `APP_NAME` `APP_TITLE` · `SERVER_HOST` `SERVER_PORT` `SERVER_RATE_LIMIT(_BURST)` ·
`DATABASE_URL` `DATABASE_REPLICAS` · `JWT_SECRET` `JWT_ACCESS_TTL` `JWT_REFRESH_TTL` ·
`REDIS_URL` · `CACHE_STORE` `CACHE_PREFIX` `CACHE_DEFAULT_TTL` · `MINIO_*` · `MAIL_*` ·
`LOG_LEVEL` `LOG_FILE_ENABLED` `LOG_FILE_DIR` `LOG_FILE_KEEP_DAYS` · `CONFIG_DSN`

## JWT_SECRET

- **Default cố định** `dev-secret-change-me` (dev) — KHÔNG tự sinh random nữa
  (random cũ → restart là mất hết session, khó debug)
- **Production bắt buộc**: `app key:generate` → ghi JWT_SECRET mới vào .env
  (giống `php artisan key:generate`; hỏi xác nhận nếu đã có key, `--force` để ghi đè)

## Dynamic config (đổi runtime, không restart)

- `admin PUT /api/v1/admin/config {key, value}` — set dynamic key + đồng bộ mọi instance (Redis pub/sub)
- Ví dụ: `log.level` → đổi log level runtime; `app.maintenance_mode` → bật/tắt maintenance
- Key bị chặn đổi runtime: `database.*`, `jwt.*`, `redis.*`, `minio.*`, `mail.*`, `cache.*`, `server.host/port`

## Zero-config

Không có .env, không có config.yml → app chạy với defaults (DB dev localhost:5433,
Redis localhost:6380, cache memory, log stdout+file daily...). JWT_SECRET lúc đó là
default dev — **không dùng cho production**.
