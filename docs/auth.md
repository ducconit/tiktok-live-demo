# Auth — JWT + OTP (đăng nhập, xác thực email, quên mật khẩu)

`core/auth` (JWT + login/refresh/logout) + `core/otp` (mã xác thực một lần) +
`internal/user` (register/verify/forgot/reset/avatar).

## Luồng xác thực tổng quan

```
Register → gửi OTP email → Verify-account → Login → JWT pair (access + refresh)
                                                          │
                                            access: HS256, roles/perms trong claims
                                            refresh: random 256-bit, lưu SHA-256 hash
```

- **Access token**: JWT HS256, TTL `JWT_ACCESS_TTL` (mặc định 15m). Claims:
  `sub` (user id), `roles`, `perms` — middleware check quyền **không đụng DB**.
- **Refresh token**: 32 bytes random (hex 64 ký tự), DB chỉ lưu SHA-256 hash, TTL
  `JWT_REFRESH_TTL` (mặc định 720h = 30 ngày). **Rotate mỗi lần refresh** (token cũ
  revoke trước khi cấp mới) — bắt được replay.
- Logout → revoke refresh token (idempotent).

## Config — bắt buộc set trên production

```bash
# backend/.env — KHÔNG được để default trên production
JWT_SECRET=           # rỗng → tự sinh mỗi lần start (mọi token vô hiệu khi restart!)
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h
```
- Sinh secret: `gvs key:generate` (giống `artisan key:generate` — hỏi xác nhận nếu đã có,
  `--force` để ghi đè; xem docs/config.md).
- Password hash: **bcrypt** (`golang.org/x/crypto/bcrypt`).

## API

### Public — `/api/v1/auth` (và `/api/v1/admin/auth` cho dashboard)

| Method | Route | Mô tả |
|---|---|---|
| POST | `/login` | `{email, password}` → `{access_token, refresh_token, token_type, expires_in}` |
| POST | `/refresh` | `{refresh_token}` → cặp mới (rotate) |
| POST | `/register` | `{email, password, full_name}` → tạo user + gửi OTP xác thực |
| POST | `/verify-account` | `{email, code}` — xác thực email (OTP 6 số) |
| POST | `/resend-otp` | `{email, purpose}` — gửi lại mã (cooldown 60s) |
| POST | `/forgot-password` | `{email}` — gửi OTP purpose `password-reset` |
| POST | `/reset-password` | `{email, code, new_password}` — đặt lại mật khẩu |

### Authed — cần `Authorization: Bearer <access_token>`

| Method | Route | Mô tả |
|---|---|---|
| POST | `/auth/logout` | `{refresh_token}` — revoke refresh |
| GET | `/auth/me` | Thông tin user hiện tại |
| GET/PUT | `/me` | Profile của mình (public namespace) |
| POST | `/me/avatar` | Upload avatar (storage — xem docs/storage.md) |
| POST | `/me/change-password` | Đổi mật khẩu |

### Admin (JWT + permission)

| Method | Route | Permission |
|---|---|---|
| GET/POST/PUT/DELETE | `/admin/users`... | `users.read` / `users.write` |
| POST | `/admin/users/:id/change-password` | `users.write` (hoặc chính user) |

## OTP — quy tắc (`core/otp`)

| Quy tắc | Giá trị |
|---|---|
| Độ dài mã | 6 chữ số (crypto/rand) |
| TTL mã | 10 phút |
| Resend cooldown | 60 giây (mã mới xoá attempts) |
| Max lần nhập sai | 5 → mã bị xoá (`ErrTooManyTries`) |

- Lưu Redis: `otp:code:<purpose>:<email>`, `otp:cd:...` (cooldown), `otp:attempts:...`.
- Purpose: `verify-account` | `password-reset`.
- Redis down → OTP không hoạt động (lỗi rõ, không ngậm); mailer chưa cấu hình → mã
  chỉ in log (app vẫn chạy — dev).

## Middleware (`core/auth/middleware.go`)

```go
// Yêu cầu token hợp lệ + nhét claims vào context
r.Use(auth.RequireAuth(cfg.JWT))

// Yêu cầu permission cụ thể (dùng sau RequireAuth) — check claims, không đụng DB
g.GET("/users", auth.RequirePermission("users.read"), h.list)
```
- Lỗi: thiếu token → 401; token sai/hết hạn → 401; thiếu permission → 403.
- Lấy user id trong handler: `auth.CurrentUserID(c)` hoặc `ctxkey.UserID(c)`.

## Dùng trong code

```go
svc := auth.NewService(cfg.JWT, userRepo, rbacRepo, auth.NewTokenStore(pool))

// login → cặp token (claims roles/perms từ RBAC)
tokens, err := svc.Login(ctx, email, password)

// verify access token (middleware)
claims, err := auth.VerifyAccessToken(token, cfg.JWT.Secret)

// OTP
otpSvc := otp.NewService(rdb)
code, _ := otpSvc.Generate(ctx, otp.PurposeVerifyAccount, email)
err := otpSvc.Verify(ctx, otp.PurposeVerifyAccount, email, code)
```

## Bảo mật & lưu ý

1. **JWT_SECRET production là bắt buộc** — default `dev-secret-change-me` chỉ cho dev.
2. Refresh token **không lưu plaintext** — chỉ hash (giống API key, xem docs/apikey.md).
3. Đổi role/quyền → user phải **login lại** để claims mới (JWT không thể thu hồi ngay).
4. Chặn login khi chưa verify email (`email_not_verified`) và user bị disable
   (`account_disabled`).
5. Rate limit: auth group (`server.auth_rate_limit`, mặc định 10/s) chống brute
   force login/OTP/forgot; API chung `server.rate_limit` (100/s). Store theo
   `server.rate_limit_store` — "" = theo cache store (cache=redis → tự multi-instance).
