# API Keys — integrations (server-server)

`internal/apikey` — quản lý API key cho máy chủ ngoài (server-server) gọi vào namespace
`/api/v1/integrations/*`. Khác JWT (người dùng), API key là **máy → máy**, vô hiệu được
từng key (revoke) và giới hạn theo **scope**.

## Khái niệm cốt lõi

| Thứ | Giá trị |
|---|---|
| Plaintext key | `gvs_<env>_<base64url(32 bytes)>` — 256-bit entropy, **chỉ hiển thị ĐÚNG 1 lần** lúc tạo/rotate |
| Lưu DB | Chỉ **SHA-256 hash** của key (`key_hash`) — DB lộ cũng không lấy lại được key |
| Prefix hiển thị | `gvs_live_ab12...` (20 ký tự đầu) — dùng nhận diện key nào trong admin, không phải secret |
| Cache verify | 1 phút (memory/redis) — giảm DB hit trên request nóng; đổi/revoke → invalidate ngay |
| last_used_at | Cập nhật **async** (goroutine, timeout 3s) — không chặn request |

## Bảo mật

- Key không bao giờ lưu plaintext — sinh xong hash ngay, trả client 1 lần rồi thôi.
- Middleware verify qua `GetByHash` → check `is_active` / `revoked_at` / `expires_at`.
- **Revoke + Rotate atomic trong 1 transaction** (`pool.WithTx`) — không xảy ra trường
  hợp key cũ chết mà key mới chưa tạo.
- Invalid key / revoked / expired → 401 với mã lỗi riêng (`invalid_api_key`,
  `api_key_revoked`, `api_key_expired`).

## Quản trị — namespace `/api/v1/admin` (JWT + permission)

| Method | Route | Permission | Mô tả |
|---|---|---|---|
| GET | `/api-keys?q=&page=&page_size=` | `api_keys.read` | Danh sách (tìm theo tên, phân trang) |
| GET | `/api-keys/:id` | `api_keys.read` | Chi tiết 1 key |
| POST | `/api-keys` | `api_keys.write` | Tạo key → trả `key` (plaintext) **+ record** |
| PUT | `/api-keys/:id` | `api_keys.write` | Sửa name / scopes / expires_at / is_active |
| DELETE | `/api-keys/:id` | `api_keys.write` | Revoke (is_active=false, revoked_at=now) |
| POST | `/api-keys/:id/rotate` | `api_keys.write` | Giết key cũ + tạo key mới (atomic, kế thừa name/scopes/expiry) → trả plaintext mới |

Body tạo:
```json
{ "name": "Billing webhook", "scopes": ["orders.read"], "expires_at": null }
```
- `name` bắt buộc 3–64 ký tự; `scopes` tối đa 10; `expires_at` RFC3339, null = vĩnh viễn.
- Response `201` khi tạo: `{ "key": "gvs_live_xxx", ...record }` — **lưu `key` ngay**,
  không xem lại được.

## Gọi API integrations — namespace `/api/v1/integrations/*`

TẤT CẢ route trong namespace này cần API key (middleware `RequireAPIKey`), kể cả
`GET /config`. Gửi key qua header (ưu tiên Authorization):

```
Authorization: Bearer gvs_live_xxx
# hoặc
X-API-Key: gvs_live_xxx
```

**Scope check** (`RequireAPIScope`): route khai báo scope cần thiết, key thiếu scope → 403.

### Idempotency (chống xử lý 2 lần)

Với `POST/PUT/PATCH` trong `/integrations`, client gửi header:

```
Idempotency-Key: <uuid bất kỳ>
```

- Key mới → thực thi, lưu response (status + body) vào bảng `idempotency_keys` (TTL 24h).
- **Cùng key + cùng method + path** → trả lại response CŨ, không chạy lại handler
  (tránh "trừ tiền 2 lần" khi mạng lỗi → client retry).
- Response `5xx` KHÔNG được lưu — client retry với key khác.
- Cùng key nhưng khác method/path → hash khác → thực thi bình thường.

## Dùng trong code

```go
// Service (nghiệp vụ, không biết HTTP)
svc := apikey.NewService(repo, pool /*TxRunner*/, cm /*cache*/, "live" /*app.env*/)

created, _ := svc.Create(ctx, apikey.CreateParams{Name: "x", Scopes: []string{"orders.read"}})
_ = created.Key // plaintext — gửi cho client, không lưu

// Middleware verify (namespace integrations)
intg.Use(apikey.RequireAPIKey(svc))
g.GET("/orders", apikey.RequireAPIScope("orders.read"), handler)
```

## Config liên quan

- Cache: `CACHE_STORE=memory|redis` (verify key dùng cache 1 phút — xem docs/cache.md).
- `APP_ENV` quyết định tiền tố key (`gvs_live_` / `gvs_dev_`...).
- Seed permissions: `api_keys.read`, `api_keys.write` — gán cho role qua admin RBAC
  (xem docs/rbac.md).

## Test

- Unit test đầy đủ (`service_test.go`): create/rotate/revoke/lookup/expired — repo mock
  (mockery), không cần DB.
- Cache test riêng (`cache_test.go`) — verify cache hit/miss/invalidate.
- Benchmark hot path (`benchmark_test.go`) — `BenchmarkLookup`.
