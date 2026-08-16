# Migration — quy chuẩn & workflow

> Engine: **goose v3** (`github.com/pressly/goose/v3`). File migration là SQL thuần
> trong `backend/migrations/` (embed vào binary qua `migrations.FS`).

## Tạo migration — BẮT BUỘC dùng lệnh (đúng chuẩn)

```bash
cd backend
go run ./cmd/devtool make:migration <tên>
# vd:
go run ./cmd/devtool make:migration create_orders
go run ./cmd/devtool make:migration add_index_users_email
```

Lệnh này **dùng goose v3** (`goose.Create` + `SetSequential(true)`) để tạo:

```
migrations/00009_create_orders.sql   ← version = max(goose_db_version) + 1, pad 5 số
```

File sinh ra đúng template chuẩn goose:

```sql
-- +goose Up
SELECT 'up SQL query';

-- +goose Down
SELECT 'down SQL query';
```

**Viết SQL thật vào giữa 2 annotation** — không sửa tên file/annotation.

### Quy ước đặt tên

| Tiền tố | Dùng khi | Ví dụ |
|---|---|---|
| `create_` | tạo bảng | `create_orders` |
| `add_` | thêm cột/index/constraint | `add_index_users_email` |
| `alter_` | đổi cấu trúc (type, rename...) | `alter_orders_total` |
| `drop_` | xoá | `drop_orders_legacy` |
| `seed_` | dữ liệu mặc định (thay vì sửa seed code) | `seed_api_key_perms` |

Tên snake_case, lowercase, mô tả HÀNH ĐỘNG (không đặt tên ngày tháng — version đã có).

## Quy tắc vàng

1. **Migration là BẤT BIẾN** — một khi đã `migrate up` (kể cả local dev) thì **KHÔNG sửa**
   file đó. Sai sót → tạo migration MỚI sửa lại (vd `alter_...`).
2. **Luôn viết Down** — mọi migration phải có phần `-- +goose Down` hoàn chỉnh
   (rollback được). Không viết `SELECT 'down'` rồi bỏ đó.
3. **NOT NULL cần default** — thêm cột NOT NULL vào bảng có dữ liệu → phải kèm
   `DEFAULT` (hoặc chia 2 bước: thêm nullable → backfill → set NOT NULL).
4. **Index tên rõ ràng** — `idx_<bảng>_<cột>` / `uq_<bảng>_<cột>` (unique).
5. **FK có ON DELETE** — khai báo rõ `ON DELETE CASCADE` hoặc `SET NULL` (không để mặc định mơ hồ).
6. **Tách seed khỏi migration** — dữ liệu seed dev (`super_admin`, roles, permissions)
   nằm trong `devtool seed`, không nhét vào migration (migration chỉ seed dữ liệu
   BẮT BUỘC cho hệ thống — vd permission mới cho 1 role).

## Chạy migration

```bash
go run ./cmd/app migrate up        # áp dụng toàn bộ chưa chạy (lệnh nằm trong binary app)
go run ./cmd/app migrate down      # lùi 1 bước
go run ./cmd/app migrate status    # xem trạng thái (applied/pending)
# hoặc qua Makefile: make migrate / make migrate-down / make migrate-status
```

Lưu ý: `make:migration` lấy version từ **DB** → trước khi tạo hãy `migrate up` cho
DB luôn ở mới nhất (tránh version trùng file chưa apply).

## Embed & đóng gói

`backend/migrations/migrations.go` embed toàn bộ thư mục `migrations/` vào binary
(`go:embed migrations/*.sql`). File mới tạo bằng `make:migration` **tự động** nằm
trong embed khi build — không cần đăng ký gì thêm.

## Luồng phát hành chuẩn

```
1. devtool make:migration create_orders   # tạo file (goose v3)
2. viết SQL Up/Down
3. go run ./cmd/app migrate up            # kiểm tra local
4. go run ./cmd/app migrate down && migrate up   # verify rollback + re-apply
5. commit + push → CI (Compose config job kiểm tra migration syntax)
```

## Kiến trúc liên quan

- `core/database/database.go` — `Migrate` / `MigrateDown` / `MigrateStatus` / `CreateMigration` (goose wrapper)
- `core/database/seeder.go` — seed dev (roles, perms, admin)
- Bảng `goose_db_version` — goose tự quản lý version đã áp dụng (không sửa tay)
