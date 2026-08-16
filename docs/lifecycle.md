# Lifecycle — từ lúc chạy app

> Tài liệu khớp với `cmd/app/main.go` + `core/*` — cập nhật khi code đổi.

## 1. Khởi động (tuần tự, fail-fast có điều kiện)

```
main() → cobra (flag --port) → run(ctx)
  │
  ├─ ① config.Load(".env")
  │     thứ tự: defaults → .env → CONFIG_DSN (file|database) → OS env (cao nhất)
  │     JWT_SECRET: default cố định "dev-secret-change-me" — production chạy `app key:generate`
  │
  ├─ ② logger.New → stdout + file daily (logs/<app>-YYYY-MM-DD.log)
  │     slog.SetDefault + OnChange("log.level") → đổi level runtime
  │
  ├─ ③ database.NewPool (master + replicas)
  │     LỖI → retry 15 lần × backoff 2s (chờ postgres cùng compose)
  │     CI/test (APP_ENV=test|CI=true) → BÁO LỖI NGAY, không retry
  │
  ├─ ④ Redis (Valkey)
  │     CONFIG_DSN=database → BẮT BUỘC: retry 15 lần (đồng bộ config giữa instance), lỗi → chết
  │     Nguồn khác (file/defaults) → optional: ping fail → nil (zero-config, app vẫn chạy)
  │     CI/test → BÁO LỖI NGAY, không retry
  │
  ├─ ⑤ cache.New — store memory (Ristretto) MẶC ĐỊNH | redis
  │     store=redis mà redis chưa sẵn sàng → lỗi RÕ (không ngậm)
  │
  ├─ ⑥ mgr.InitDynamic — Redis pub/sub (load state + subscribe); không redis → local-only
  │
  ├─ ⑦ server.New — gin + middleware chain:
  │     requestID → logger → Recovery → CORS → rateLimit → gzip → maintenance
  │     + setupRoutes (public/admin/integrations + openapi.json) + snapshot routes
  │
  ├─ ⑧ goroutine: ListenAndServe (lỗi → gửi serverErr chan)
  │
  └─ ⑨ select: serverErr | SIGINT/SIGTERM
```

### Fail-fast vs degrade (chủ ý)

| Thành phần | Lỗi | Hành vi |
|---|---|---|
| Config DSN = postgres | connect fail | **retry 10×** (bảng app_config chưa sẵn) |
| Database master | connect fail | **retry 15×** — app không chạy nổi nếu không DB |
| Redis + CONFIG_DSN=database | ping fail | **retry 15×** — bắt buộc để đồng bộ config giữa instance |
| Redis + nguồn khác | ping fail | **degrade**: rdb=nil, dynamic config local-only |
| Cache store=redis + redis nil | — | **lỗi rõ**: đổi CACHE_STORE=memory hoặc bật valkey |

> CI/CD (`CI=true`) hoặc test (`APP_ENV=test|testing`) → **KHÔNG retry**: báo lỗi ngay
> (retry chỉ dành cho runtime thật — vd khởi động cùng docker compose). Xem `core/retry`.

## 2. Runtime

- Serve request qua gin (middleware chain).
- Goroutine nền: subscriber loop (Redis pub/sub — dynamic config), goroutine ngắn hạn (apikey touch last_used...).

## 3. Shutdown (graceful, timeout 10s)

```
nhận SIGINT/SIGTERM (hoặc lỗi server) → "shutting down..."
  │
  ├─ srv.Shutdown(ctx 10s) — ngừng nhận request mới, chờ request đang xử lý xong
  │
  ├─ defer chain (LIFO):
  │     mgr.Close()  → dừng subscriber (cancel context)
  │     cm.Close()   → đóng Ristretto/redis cache
  │     rdb.Close()  → đóng Redis client
  │     pool.Close() → đóng DB pool
  │     lg.Close()   → đóng file log daily
  │
  └─ exit 0 (hoặc exit 1 kèm lỗi server)
```

> Lỗi `ListenAndServe` (vd bind port thất bại) → **shutdown sạch** qua serverErr chan,
> KHÔNG `os.Exit(1)` — os.Exit bỏ qua defer → connection rò (DB/redis/file log không đóng).

## 4. key:generate (giống laravel)

```
app key:generate              # chạy trên production sau deploy
app key:generate --force      # ghi đè không hỏi
app key:generate --env .env.production
```

- Sinh JWT_SECRET 32 bytes → hex 64 ký tự, **ghi vào .env** (env ưu tiên cao nhất)
- Kiểm tra key hiện có: file config (nếu CONFIG_DSN=file) → .env — đã có → **hỏi xác nhận y/N** trước khi thay thế
