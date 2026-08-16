# Project Context: tiktok-live-platform

> Load bởi OAC agents trước khi sinh code. Giữ file <200 dòng (MVI).

## Mô tả

Template dự án "kiểu Laravel": backend Go + admin dashboard Vue, dev qua docker compose.
Project mới = `devtool new:project` (core/template — copy + đổi tên).

## Stack

- **Backend**: Go 1.26+, gin (build tag `sonic`), sqlc (SQL thuần → type-safe), goose v3 migrations, golang-jwt/v5, viper, cobra, pgx/v5, mockery + testify
- **Dashboard**: Vue 3 (script setup lang=ts), Pinia, Vue Router, TanStack Vue Query, TanStack Vue Table, Axios, shadcn-vue style UI (reka-ui v2 — components dùng hậu tố `Root`), Tailwind v4, Bun
- **DB**: Postgres 18 (master + replica read-only tuỳ chọn), Valkey 9 (redis)
- **Dev**: docker-compose.yml (postgres18/valkey9/minio/mailpit/backend air/dashboard vite)

## Layout backend

```
core/            # framework tái dùng mọi project (tách được thành module riêng)
  config/        # viper, zero-config defaults, tự sinh JWT secret khi thiếu
  database/      # Pool master-slave (Write()/Read()), goose migrations, seeder
  response/      # Envelope {success, data, meta, error}
  apperr/        # sentinel errors → HTTP status
  validation/    # validator wrapper (tiếng Việt message)
  auth/          # JWT access/refresh, RequireAuth, RequirePermission
internal/        # business logic của app
  server/        # gin engine + middlewares + DI + routes
  user/ rbac/ stats/ mocks/
cmd/
  app/           # ứng dụng deploy (--port flag, bind 127.0.0.1:3330 mặc định)
  devtool/       # CLI dev: migrate, seed, key:generate, user:create, make:crud
db/              # sqlc generated
migrations/      # goose SQL (embed)
```

## Quy tắc chính

1. KHÔNG sửa code trong `db/` (sqlc generated) — sửa `db/queries/*.sql` rồi `sqlc generate`
2. Migration mới: `migrations/0000N_name.sql` với `-- +goose Up` / `-- +goose Down`
3. Domain mới: `internal/<domain>/` gồm repo.go (wrapper sqlc, tách rw/ro) + service.go (business, interface cho mock) + handler.go (DTO, validation, response envelope)
4. Response LUÔN qua envelope (`response.OK/OKWithMeta/Created/NoContent/Error`)
5. Lỗi nghiệp vụ qua `apperr` (NotFound/Conflict/Unauthorized/Forbidden/BadRequest) — KHÔNG trả raw error
6. Secret KHÔNG hardcode — `.env` gitignored; zero-config nghĩa là mọi thứ có default chạy được
7. Build: `go build -tags sonic ./...`; test: `go test ./...`; format: gofmt
