# tiktok-live-platform

Template dự án **"kiểu Laravel"** cho stack Go + Vue — copy + đổi tên là có ngay project mới với:

- **Backend Go 1.26+**: gin (sonic) + sqlc + goose migrations + JWT auth + RBAC + DB master-slave + CLI devtool
- **Admin dashboard**: Vue 3 + Pinia + TanStack Query/Table + shadcn-vue + Tailwind v4 (Bun), dark-first design system **Studio Dark**
- **Zero-config**: chạy được ngay không cần `.env` (mặc định local/development) — env chỉ cần khi deploy/đổi môi trường
- **Agent-first**: OpenAgentsControl (OAC) + context patterns trong repo — AI agents sinh code đúng chuẩn project
- **Dev hoàn toàn qua docker compose**: Postgres 18 + Valkey 9 + MinIO + Mailpit, hot reload (air + vite)

## Quickstart

```bash
docker compose up -d --build postgres valkey minio mailpit backend dashboard  # CORE services
docker compose up -d prometheus grafana loki alloy jaeger                      # MONITORING (optional)
make migrate-up                       # chạy goose migrations
make seed                             # tạo admin (SEED_ADMIN_EMAIL/PASSWORD trong backend/.env)
make admin                            # tạo admin khác: default admin@example.com / admin123
#   override: make admin ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... ADMIN_ROLES=admin,editor
#   hoặc trực tiếp: go run ./cmd/devtool make:admin --name "Quản trị" --email a@b.com --roles admin,editor
# Mở http://localhost:5176 (dashboard admin) — login admin@example.com / admin123
#    http://localhost:5175 (frontend end-user — live monitor)
# (5174 đang bị dự án khác chiếm — đổi qua FORWARD_DASHBOARD_PORT)
# Monitoring: http://localhost:3006 (Grafana) · http://localhost:9092 (Prometheus) · http://localhost:16687 (Jaeger)
```

> API mặc định bind **127.0.0.1:3330** (override `--port <n>` hoặc `SERVER_PORT`). Port Postgres `5433`, Valkey `6380` (tránh đụng service khác — đổi qua `FORWARD_*`).

## Cấu trúc env & config (2 tầng)

```
.env                    # DOCKER COMPOSE (optional): forward port + credentials postgres/minio/valkey/mailpit
backend/.env            # BACKEND (optional): DATABASE_URL, JWT_SECRET, LOG_LEVEL, SEED_ADMIN_*... (xem backend/.env.example)
backend/config.yml      # BACKEND config YAML (optional): cấu trúc theo backend/config.example.yml
```

- App đọc `.env` + `config.yml` từ **CWD** — chạy chuẩn: `cd backend && go run ./cmd/app`
- Trong docker: `backend/.env` + `backend/config.yml` mount vào container (`/app/`)
- Thứ tự ưu tiên: **OS env > CONFIG_DSN (file/database) > .env > defaults** — chi tiết [`docs/config.md`](docs/config.md)
- Dashboard admin: `http://localhost:5176` · Frontend end-user: `http://localhost:5175` (đổi qua `FORWARD_DASHBOARD_PORT` / `FORWARD_FRONTEND_PORT`)

## Hai ứng dụng trong `cmd/`

| Binary | Mục đích |
|---|---|
| `cmd/app` | Ứng dụng chính — deploy staging/production. Binary tên `gvs`. CLI: `key:generate`, `migrate up/down/status`, `logs` (mỗi lệnh 1 file trong `cmd/app/commands/`) |
| `cmd/devtool` | Công cụ dev local — make:migration, seed, user:create, make:crud, config |

```bash
cd backend
go run ./cmd/devtool make:migration create_orders  # TẠO migration (goose v3 — migrations/0000N_<tên>.sql)
go run ./cmd/app migrate up|down|status            # quản lý migration (lệnh nằm trong binary app)
go run ./cmd/devtool seed                          # tạo super_admin
go run ./cmd/devtool user:create --email a@b.com --role admin
go run ./cmd/devtool config:set <key> <value>      # đổi dynamic config (sync mọi instance)
go run ./cmd/devtool make:crud product             # sinh khung CRUD domain mới

# JWT_SECRET — app key:generate (giống laravel artisan, dùng cả production):
go run ./cmd/app key:generate            # hỏi xác nhận nếu đã có key
go run ./cmd/app key:generate --force    # ghi đè không hỏi

# Logs — gvs logs (giống tail): 25 dòng cuối hôm nay | -n <số> | -f theo dõi
go run ./cmd/app logs             # = tail -n 25 logs/<app>-YYYY-MM-DD.log
go run ./cmd/app logs -n 100 -f   # 100 dòng cuối + follow
```

> Tài liệu chi tiết (trong `docs/`):
> - [`docs/migration.md`](docs/migration.md) — quy chuẩn migration (tạo bằng `make:migration`, Up/Down, bất biến)
> - [`docs/lifecycle.md`](docs/lifecycle.md) — startup/shutdown/retry/fail-fast
> - [`docs/config.md`](docs/config.md) — thứ tự nguồn config, CONFIG_DSN, key:generate
> - [`docs/logger.md`](docs/logger.md) — core/logger, file daily, level dynamic
> - [`docs/response.md`](docs/response.md) — chuẩn response 4 key
> - [`docs/monitoring.md`](docs/monitoring.md) — Prometheus metrics (port riêng, auth optional)
> - [`docs/metrics.md`](docs/metrics.md) — Metrics Reference (key + mô tả + alert rules)
> - [`docs/i18n.md`](docs/i18n.md) — Đa ngôn ngữ (go-i18n: message files, workflow merge, Accept-Language)
> - [`docs/storage.md`](docs/storage.md) — Lưu trữ multiple disk kiểu Laravel (local + s3, default disk)
> - [`docs/apikey.md`](docs/apikey.md) — API keys cho integrations (hash, scope, rotate atomic, idempotency)
> - [`docs/rbac.md`](docs/rbac.md) — Roles & Permissions (system_admin immutable, claims trong JWT)
> - [`docs/cache.md`](docs/cache.md) — Cache multi-store (memory Ristretto | redis, type-safe)
> - [`docs/auth.md`](docs/auth.md) — Auth JWT + OTP (login/refresh/logout, verify email, reset password)

Muốn thêm entrypoint khác (vd desktop app)? Tạo package mới trong `cmd/` — mỗi thư mục là 1 binary.

## Zero-config

Mọi giá trị có default trong code — app chạy không cần config:

- Server: `127.0.0.1:3330`, môi trường `development`
- DB: `postgres://app:app_password_dev@localhost:5433/tiktok_live_platform`
- JWT: thiếu `JWT_SECRET` → tự sinh ngẫu nhiên mỗi lần start (set khi deploy)

## Nguồn cấu hình: CONFIG_DSN

Config chính nạp từ **CONFIG_DSN** (kiểu "data source name") — mặc định `file://config.yml` (file YAML ở thư mục chạy lệnh), hoặc lưu trong **Postgres**:

```bash
# 1) File YAML (mặc định) — copy mẫu rồi sửa
cp config.example.yml config.yml

# 2) Lưu config trong DB — đổ YAML vào bảng app_config rồi trỏ DSN
CONFIG_DSN=postgres://app:pass@localhost:5433/tiktok_live_platform?sslmode=disable \
  go run ./cmd/devtool config:import config.yml     # upsert từng key vào bảng
CONFIG_DSN=postgres://app:pass@localhost:5433/tiktok_live_platform?sslmode=disable \
  go run ./cmd/app                                  # app đọc config từ DB
```

Thứ tự merge: **defaults (code) → CONFIG_DSN → .env → OS env** (env thắng cuối cùng).
Mọi instance đọc cùng nguồn (file hoặc DB) → đồng nhất khi chạy sau load balancer.

## DB master-slave

`core/database.Pool` tách read/write:

- `DATABASE_URL` — master (write), bắt buộc
- `DATABASE_REPLICAS` — comma-separated URLs read-only, tuỳ chọn
- Không set replica → mọi read chạy trên master (single-node)

Repo dùng `pool.Write()` cho INSERT/UPDATE/DELETE, `pool.Read()` cho SELECT (round-robin qua replicas).

## Agent-first (OpenAgentsControl)

```bash
# Cài OAC (1 lần): cài OpenCode + agents
curl -fsSL https://raw.githubusercontent.com/darrenhinde/OpenAgentsControl/main/install.sh | bash -s developer

# Làm việc trong repo — agents tự load context patterns (.opencode/context/project/)
opencode --agent OpenAgent
```

Context files commit trong repo: `stack.md`, `backend-patterns.md`, `dashboard-patterns.md` — agents học chuẩn (envelope response, repo/service/handler, sqlc, shadcn UI...) và sinh code khớp ngay từ đầu.

## Cấu trúc

```
backend/
  core/             # framework tái dùng (tách được thành module riêng)
    config/         # viper zero-config, tự sinh JWT secret
    database/       # Pool master-slave + goose + seeder
    response/       # envelope {success,data,meta,error}
    apperr/         # lỗi nghiệp vụ → HTTP status
    validation/     # validator wrapper
    auth/           # JWT access/refresh + RequireAuth/RequirePermission
  internal/         # business logic (user, rbac, stats, server wiring)
  cmd/app/          # ứng dụng deploy
  cmd/devtool/      # CLI dev tools
  db/               # sqlc generated (không sửa tay)
  migrations/       # goose SQL (embed)
dashboard/          # Vue 3 admin
.opencode/context/  # OAC patterns
docker-compose.yml  # dev environment
```

## Tạo project mới từ template

```bash
cd backend && go run ./cmd/devtool new:project myapp "My App"
cd ../myapp && cp .env.example .env && cp config.example.yml config.yml
docker compose up -d --build
```

Tạo project mới từ template — chạy từ root skeleton (logic trong `core/template`, sau này tách CLI riêng chỉ cần import):

```bash
cd backend && go run ./cmd/devtool new:project myapp "My App"
# hoặc: make new-project NAME=myapp TITLE="My App"
cd ../myapp && cp .env.example .env && cp config.example.yml config.yml
docker compose up -d --build
```

Tự đổi module Go, DB name, branding dashboard, `git init` sẵn (kèm OAC context).

## Dynamic config (đồng bộ mọi instance qua Redis pub/sub)

Config gồm 2 loại: **static** (khởi động: DB, JWT, mail...) và **dynamic** (đổi runtime, không restart):

```bash
cd backend
go run ./cmd/devtool config:set server.rate_limit 250   # ví dụ: giới hạn request/s
```

Cơ chế (mỗi instance chạy sau load balancer đều đồng bộ):

```
Instance A: SetDynamic("server.rate_limit", 250)
  1. set local (koanf) — áp dụng ngay trên A
  2. persist vào Redis state key (last-write-wins theo timestamp)
  3. publish lên channel gvs:config:sync:<app>
Instance B (subscriber): nhận → set local + persist state
Instance C (mới join): đọc state key → đồng bộ ngay từ đầu
```

- Không có Redis → `SetDynamic` chỉ ảnh hưởng local (zero-config vẫn chạy)
- API trong code: `mgr.SetDynamic(key, val)`, `mgr.GetDynamic(key)`, `mgr.Int(key, def)`, `mgr.OnChange(key, fn)`
- Ví dụ dùng thật: rate limit đọc `server.rate_limit` mỗi request — đổi runtime không restart
- State: Redis key `gvs:config:state:<app>` (JSON), conflict giải quyết last-write-wins

## Cache multi-store (eko/gocache)

`core/cache` — wrapper type-safe trên eko/gocache v4, hỗ trợ **nhiều store**:

| Store | Khi nào dùng |
|---|---|
| `memory` (mặc định) | Ristretto — zero-config, 1 instance, tốc độ cao |
| `redis` | Nhiều instance sau load balancer — cache dùng chung |

```go
// Cấu hình: config.yml → cache: {store: redis, prefix: "myapp:", default_ttl: 5m}
// hoặc env CACHE_STORE / CACHE_PREFIX / CACHE_DEFAULT_TTL

// Dùng trong code (free functions — Go không cho method generic)
if v, err := cache.Get[Stats](cm, ctx, "stats:overview"); err == nil { ... }
_ = cache.SetWithTTL(cm, ctx, "stats:overview", out, 30*time.Second)
_ = cache.Set(cm, ctx, "user:42", u)          // TTL mặc định 5m
_ = cache.Delete(cm, ctx, "user:42")
// Miss → cache.ErrNotFound (errors.Is); GetFrom(cm, "redis", ctx, key) chọn store
```

Ví dụ dùng thật trong skeleton: `/api/v1/admin/stats` cache 30s. Ristretto commit **async** — Set xong Get ngay có thể miss (eventual, tương tự redis).

## Build (version/branch-aware)

```bash
make build   # → backend/bin/app
```

- **hash**: git short commit · **date**: thời điểm build (UTC) — inject qua ldflags
- **version** theo thứ tự ưu tiên:
  1. `VERSION=2.5 make build` — chỉ định tay
  2. Branch `release-*` (production): phần sau prefix — `release-1.0` → `1.0`
  3. Branch khác (main = staging): **tag gần nhất** (`git describe --tags`, bỏ prefix `v`)
  4. Không có tag (build lần đầu): `1.0.0`

Expose qua `/healthz` + log startup:
```json
{"status":"ok","version":"1.0.0","build_hash":"0fb27bc","build_date":"2026-08-12T00:52:56Z","environment":"development"}
```

## API structure (namespace)

Mọi API có prefix `/api/{version}` (mặc định `v1`) + namespace:

| Namespace | Dành cho | Auth |
|---|---|---|
| `/api/v1/public/*` | Ứng dụng client (mobile, web, bên thứ 3) | Tuỳ endpoint |
| `/api/v1/admin/*` | Admin dashboard (đặc thù riêng) | Tuỳ endpoint |
| `/api/v1/integrations/*` | Server-server | **Tất cả cần API key** (`X-API-Key` hoặc `Authorization: Bearer gvs_...`) |

Mỗi namespace có `GET /config` — thông tin công khai (version, environment, build hash/date, `maintenance_mode` — dynamic config, set qua `devtool config:set app.maintenance_mode true`). Riêng `/integrations/config` phải auth trước.

Ví dụ: `/api/v1/admin/auth/login`, `/api/v1/public/auth/login`, `/api/v1/admin/users`, `/api/v1/integrations/config`.

**API keys** (`/api/v1/admin/api-keys`, permission `api_keys.*`): tạo/liệt kê/sửa/thu hồi/xoay. Key dạng `gvs_<env>_<256-bit>` — chỉ hiển thị **ĐÚNG 1 lần** lúc tạo/rotate; backend chỉ lưu SHA-256 hash. Scopes tùy chọn (`["orders.read", ...]`), hết hạn tùy chọn; rotate vô hiệu key cũ ngay lập tức. Xem dashboard → **API Keys**.

**OpenAPI spec tự động**: `GET /api/v1/openapi.json` (public) — generate runtime từ gin routes (không bao giờ lệch), kèm components (Envelope, User, ApiKey, Role, Permission) + security schemes (bearerAuth JWT / apiKeyAuth). Import vào Postman/Insomnia/Stoplight để duyệt API. **Bật/tắt bằng config**: `OPENAPI_ENABLED=false` (hoặc `openapi.enabled: false`) — mặc định BẬT, tắt ở production không cần sửa code.

## Auth & tài khoản (OTP flow)

**Public** (`/api/v1/public/auth`):
- `POST /register` — đăng ký (role user, chưa verify) → gửi OTP email
- `POST /verify-account` `{email, otp}` — xác thực tài khoản
- `POST /resend-otp` `{email, type: verify-account|password-reset}` — gửi lại mã (cooldown 60s)
- `POST /forgot-password` `{email}` — gửi OTP đặt lại mật khẩu
- `POST /reset-password` `{email, otp, new_password}` — đặt lại mật khẩu

OTP: 6 chữ số, Redis TTL 10 phút, chống brute (5 lần sai → vô hiệu mã). Email qua **Mailpit** (dev, http://localhost:8025) — SMTP config `mail.*`. Login chặn tài khoản chưa verify (`email_not_verified`).

**Tài khoản** (`/api/v1/public`, cần token): `GET/PUT /me`, `POST /me/avatar` (upload → MinIO, trả URL), `POST /me/change-password`.

**Admin** (`/api/v1/admin`, cần token + permission):
- `GET/POST/PUT/DELETE /admins` — CRUD tài khoản admin (admin = user + role admin)
- `DELETE /cache` — xoá toàn bộ cache (cache.delete); `GET /cache` — thông tin store
- `GET /config/dynamic` — danh sách dynamic keys; `PUT /config` — set 1 key (config.*)
- `GET/POST/PUT/DELETE /api-keys` + `POST /api-keys/:id/rotate` — quản lý API keys (api_keys.*)
- `PUT /config` `{key, value}` — remote config (config.write): đổi dynamic key + sync mọi instance; chặn key static nguy hiểm (`database.*`, `jwt.*`, `redis.*`, `minio.*`, `mail.*`, `cache.*`, `server.host/port`)

## Config (DSN + zero-config)

- **Mọi config có default** trong code — config thiếu/không khai báo → lấy default hợp lệ (test `TestLoad_EveryFieldHasDefault` đảm bảo không field zero). File/env chỉ override.
- **DSN đầy đủ thay vì thông số rời**: `database.url` (`postgres://user:***@host:5432/db`), `redis.url` (`redis://user:pass@host:6379/0` — `Redis.Client()` tự parse). Không tách host/port/username/password.
- Redis DSN sai → lỗi rõ lúc start; ping fail → dynamic config disabled (zero-config vẫn chạy).

## Chuẩn response (4 key)

MỌI response: `{ "code", "msg", "data", "meta" }` — `code` luôn string:

| code | Trường hợp |
|---|---|
| `"0"` | thành công (msg bỏ trống, data = dữ liệu/null, danh sách = `[]`) |
| `"400"` | bad request (fallback validation khi không rõ field) |
| `"401"` / `"403"` / `"404"` | unauthorized / forbidden / not found |
| `"413"` | file quá lớn |
| `"422"` | validation — `meta` = `{ field: "message lỗi" }` (tên field theo tag json) |
| `"429"` | rate limit |
| `"500"` | internal (msg mặc định "Ops! Đang có lỗi xảy ra...") |
| `"503"` | maintenance — chặn mọi API trừ `/config` |

Pagination: `meta = { limit, page, total }`. Status code header theo chuẩn HTTP.

## Logging (core/logger)

Engine: **`log/slog`** (stdlib) — JSON handler. Mặc định ghi **2 nơi**: stdout (console) + **file theo ngày** `logs/<app>-YYYY-MM-DD.log` (tự tạo thư mục, tự xoá file cũ hơn `log.file_keep_days`).

- **Level**: `LOG_LEVEL` env hoặc `log.level` config (debug|info|warn|error, default `info`) — **đổi runtime** qua `admin PUT /config {key: "log.level", value: "error"}` (đồng bộ mọi instance qua Redis pub/sub, không cần restart)
- **File**: `LOG_FILE_ENABLED` (default true), `LOG_FILE_DIR` (default `logs`), `LOG_FILE_KEEP_DAYS` (default 14)
- **Quy tắc log** (middleware HTTP tự động): ≥500 → `ERROR` (kèm code/error/details — lỗi hệ thống), 400-499 → `WARN` (lỗi nghiệp vụ — có code), còn lại → `INFO`. Log JSON ghi đủ `method/path/status/latency_ms/ip/request_id`
- Lưu ý: khi level = error, log request của chính request đổi level cũng bị chặn (middleware log sau handler) — bình thường

## Convention dev

- MỌI service chạy qua docker compose (không cài binary ad-hoc)
- Migration bằng goose (annotation `-- +goose Up/Down`), query bằng sqlc (SQL thuần → type-safe)
- Không hardcode secret; staging/prod dùng `compose.yml` (gitignored)
- Build backend với tag sonic: `go build -tags sonic ./...`
- Design system: Studio Dark (dark-first, indigo→violet) — tokens trong `dashboard/src/assets/design-tokens.css`
