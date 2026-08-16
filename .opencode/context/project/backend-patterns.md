# Backend Patterns — sinh code đúng chuẩn skeleton

## Domain mới (vd: product)

1. `db/queries/products.sql` — viết SQL thuần, named params (`sqlc.arg(x)`, `sqlc.narg(x)` cho nullable), `-- name: Xxx :one/:many/:exec`
2. `sqlc generate` → code vào `db/`
3. `internal/product/repo.go`:
   ```go
   type Repo struct {
       rw *db.Queries // master — ghi
       ro *db.Queries // replica — đọc (không có replica = master)
   }
   func NewRepo(p *database.Pool) *Repo {
       return &Repo{rw: db.New(p.Write()), ro: db.New(p.Read())}
   }
   ```
   SELECT dùng `r.ro`, INSERT/UPDATE/DELETE dùng `r.rw`.
4. `internal/product/service.go` — Service nhận interface (mock được):
   ```go
   type Repository interface { ... }
   type Service struct { repo Repository }
   ```
   Trả lỗi `apperr.Conflict("email_exists", "...")`, `apperr.NotFound("user_not_found", "...")`...
5. `internal/product/handler.go` — Handler + `RegisterRoutes(g *gin.RouterGroup)`; DTO cho response (KHÔNG trả struct DB chứa hash/secret); body struct với `validate:"required,email,min=..."`; `validation.ValidateStruct` → `apperr.BadRequest(...).WithDetails(fields)`; parse id `uuid.Parse` → `apperr.BadRequest("invalid_id", ...)`
6. Wire vào `internal/server/routes.go` buildServices + đăng ký route

## Auth & permission

- Route public: `/api/v1/auth/login`, `/api/v1/auth/refresh`
- Route cần token: group sau `auth.RequireAuth(cfg.JWT)`; permission per-route: `auth.RequirePermission("users.read")`
- Permission convention: `<resource>.<action>` — users.read, users.write, roles.read, roles.write, settings.read, settings.write, dashboard.view
- Claims chứa roles + perms (không query DB mỗi request)

## Database

- Chỉ dùng qua `database.Pool` (core): `pool.Write()` cho ghi, `pool.Read()` cho đọc
- Migration: goose — file mới trong `migrations/`, chạy `go run ./cmd/app migrate up` (lệnh migrate nằm trong binary app — cmd/app/commands/migrate.go)
- Query mới: viết trong `db/queries/*.sql`, named params, KHÔNG dùng `$1` trực tiếp (trừ param đơn)

## CLI devtool

- Lệnh mới: thêm vào `cmd/devtool/` — migrate, seed, key:generate, user:create, make:crud
- `make:crud <domain>` sinh khung repo/service/handler từ template

## Test

- Mock interface qua mockery (`.mockery.yaml`, chạy `mockery`) → `internal/mocks/`
- Unit test service với `mocks.NewMockXxx(t)` + `.EXPECT()`
- Pattern test: `package xxx_test`, `newTestService(t)`, assert qua testify
