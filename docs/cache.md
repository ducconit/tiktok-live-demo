# Cache — multi-store (memory / redis)

`core/cache` — wrapper trên **eko/gocache v4** với API type-safe qua marshaler
(msgpack). App không đụng store trực tiếp — chọn store bằng config, đổi không cần sửa code.

## Store

| Store | Khi nào dùng | Ghi chú |
|---|---|---|
| `memory` (mặc định) | 1 instance, zero-config | Ristretto (dgraph-io) — không cần dịch vụ ngoài |
| `redis` | Nhiều instance sau LB (dùng chung) | go-redis; cần redis client khởi tạo từ config |

- Cấu hình: `CACHE_STORE=memory|redis`, `CACHE_PREFIX` (tiền tố key khi dùng chung
  redis giữa nhiều app), `CACHE_DEFAULT_TTL=5m`.
- **Fail rõ, không ngậm**: `CACHE_STORE=redis` mà redis chưa sẵn sàng → app báo lỗi
  lúc boot (không chạy "tạm"). Chọn `memory` hoặc khởi động redis trước.
- Redis bắt buộc khi `CONFIG_DSN` là database (dynamic config sync pub/sub) — xem docs/config.md.

## API — type-safe

Go không cho method generic → API là **free functions** (gọi `cache.Get[T]`, không
`m.Get[T]`):

```go
// Get — miss → cache.ErrNotFound (check bằng errors.Is)
v, err := cache.Get[Info](m, ctx, "key")
if errors.Is(err, cache.ErrNotFound) { /* miss */ }

// Set — TTL mặc định của store (CACHE_DEFAULT_TTL)
cache.Set[Info](m, ctx, "key", info)

// SetWithTTL — TTL riêng cho key
cache.SetWithTTL[Info](m, ctx, "key", info, 30*time.Second)

// Delete / Clear (toàn bộ store — admin: DELETE /admin/cache)
cache.Delete(m, ctx, "key")
cache.Clear(m, ctx)

// Chọn store cụ thể (mặc định là store trong config)
cache.GetFrom[T](m, "redis", ctx, "key")
```

- Prefix tự động thêm vào key (`m.prefix + key`) — không cần tự nối.
- Ristretto cần cost > 0 → mọi item set cost 1 (tránh "set xong get miss").
- `WithSynchronousSet()` — Set xong Get được ngay (không lag do buffer bất đồng bộ).

## Dùng trong dự án (thực tế)

| Nơi | Key | TTL | Invalidate |
|---|---|---|---|
| API key verify (`internal/apikey`) | `apikey:<hash>` | 1 phút | Update/Revoke/Rotate → xoá key |
| Stats dashboard (`internal/stats`) | theo query | ngắn | — |
| Cache admin (`internal/server/admin.go`) | — | — | `DELETE /admin/cache` xoá toàn bộ |

## Config

```yaml
# backend/config.yml (hoặc env)
cache:
  store: memory        # memory | redis
  prefix: ""           # vd "myapp:" khi dùng chung redis
  default_ttl: 5m
```
```bash
CACHE_STORE=redis CACHE_PREFIX=gvs: CACHE_DEFAULT_TTL=1m ./gvs
```

## Test

- Unit test đầy đủ (`cache_test.go`): get/set/delete/clear/ttl/miss — không cần dịch vụ.
- Benchmark hot path (`benchmark_test.go`).
