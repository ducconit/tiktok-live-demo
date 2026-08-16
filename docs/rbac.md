# RBAC — Roles & Permissions

`internal/rbac` — phân quyền theo **role → permission**, gán role cho user. Quyền được
nhúng thẳng vào **JWT claims** lúc login (không query DB mỗi request).

## Mô hình

```
User ──< user_roles >── Role ──< role_permissions >── Permission
```

- **Role**: có `id` (slug do admin tự điền, vd `editor`, `moderator`), `name`, `description`,
  cờ `is_system`.
- **Permission**: seed cố định theo tính năng (vd `users.read`, `users.write`,
  `api_keys.write`, `roles.write`...) — danh sách xem ở `GET /admin/permissions`.
- **Gán quyền**: role × permission (nhiều-nhiều); user có thể mang nhiều role → hợp quyền.

## Role hệ thống `system_admin` — BẤT BIẾN

Role `system_admin` được seed sẵn (qua `devtool seed`) với **toàn quyền**. Quy tắc:

| Thao tác | Cho phép? |
|---|---|
| Xoá role system | ❌ 403 `system_role_protected` |
| Sửa quyền của role system | ❌ 403 `system_role_permissions` (đã seed đủ) |
| Đổi slug role system | ❌ 403 `system_role_immutable` (id không nằm trong update) |
| Đổi tên / mô tả role system | ✅ (vô hại) |

Role thường có `id` tự do — **slug immutable sau khi tạo** (chỉ sửa name/description).
Quy tắc `id`: `^[a-z0-9_-]{2,64}$` (chữ thường, số, gạch dưới/gạch ngang).

## JWT claims

Access token chứa sẵn quyền — middleware không đụng DB:

```json
{
  "sub": "user-id",
  "roles": ["system_admin"],
  "perms": ["users.read", "users.write", "roles.write", "..."]
}
```

- Login/refresh → `auth.Service.issuePair` đọc roles + permissions → sign vào claims.
- **Đổi role/quyền → user phải login lại** (hoặc refresh) để nhận claims mới.
- Middleware check: `auth.RequirePermission("users.write")` — so khớp chuỗi trong claims.

## API — namespace `/api/v1/admin` (JWT + permission per-route)

| Method | Route | Permission | Mô tả |
|---|---|---|---|
| GET | `/roles` | `roles.read` | Danh sách roles |
| POST | `/roles` | `roles.write` | Tạo role (`id`, `name`, `description`) |
| PUT | `/roles/:id` | `roles.write` | Sửa name/description (slug immutable) |
| DELETE | `/roles/:id` | `roles.write` | Xoá role (chặn role system) |
| GET | `/roles/:id/permissions` | `roles.read` | Quyền của role |
| PUT | `/roles/:id/permissions` | `roles.write` | Set lại toàn bộ quyền role (`permission_ids`) |
| GET | `/permissions` | `roles.read` | Danh sách permission có sẵn |
| POST | `/users/:id/roles` | `users.write` | Gán role cho user (`role_id`) |
| DELETE | `/users/:id/roles/:roleId` | `users.write` | Gỡ role khỏi user |

## Seed & CLI

- `devtool seed` — tạo `system_admin` + roles/permissions nền tảng (idempotent).
- `make admin` — tạo user admin nhanh (mặc định `admin@example.com/admin123`, role
  `system_admin`); override: `make admin ADMIN_EMAIL=... ADMIN_ROLES=editor,admin`.
- Permission mới cho hệ thống: **migration** `seed_...` (dữ liệu BẮT BUỘC), không nhét
  vào seed dev (xem docs/migration.md quy tắc 6).

## Dùng trong code

```go
// Handler — gắn middleware permission ngay trên route
g.GET("/roles", auth.RequirePermission("roles.read"), h.listRoles)
g.POST("/roles", auth.RequirePermission("roles.write"), h.createRole)

// Service — check trước khi hành động phá hoại
if role.IsSystem {
    return apperr.Forbidden("system_role_protected", "error.system_role_protected")
}
```

## Test

- Unit test service (`service_test.go`): tạo/đổi/xoá role, chặn role system
  (protected/immutable/permissions), conflict trùng slug — repo mock.
- RBACReader dùng chung cho auth: mock qua mockery (`internal/mocks`).
