# TikTok Bar — Release 1.2.0

> **Ngày:** 2026-08-16 · **Trạng thái:** Stable · **Git tag:** `v1.2.0`

## Thay đổi chính: Server chuyển sang **Gin (gin-gonic)**

Backend Go chuyển từ `net/http` + `http.ServeMux` thuần sang **Gin framework** — giữ nguyên toàn bộ
chức năng (Webcast relay, self-host signing, WebSocket/long-poll, REST API).

## Thay đổi

| | v1.1.0 | v1.2.0 |
|--|--------|--------|
| HTTP framework | `net/http` + `http.ServeMux` | **Gin (gin-gonic)** |
| REST API | handler thủ công | `r.GET("/api/room/:username")` (gin Context, `c.JSON`, `c.Param`) |
| WebSocket | `mux.HandleFunc("/ws")` | `r.GET("/ws")` (gorilla/websocket trong gin handler) |
| Static frontend | `http.FileServer` + SPA thủ công | `r.Static("/assets")` + `r.NoRoute` SPA fallback |
| JSON response | `json.NewEncoder` | `c.JSON` (gin) |
| Middleware | không | `gin.Recovery()` (bắt panic) |
| Chế độ | — | `gin.ReleaseMode` (GIN_MODE=debug để dev) |

## Giữ nguyên (không đổi)

- **Self-host signing** (QuickJS): X-Bogus + X-Gnarly + msToken, không bên thứ 3.
- **Hai chế độ kết nối**: `CONNECTION_MODE=websocket` | `long_poll`, `POLL_INTERVAL_MS`.
- **Fix nút Dừng** (frontend, không auto-reconnect), **avatar** (shadcn/reka-ui).
- **REST**: `GET /api/health`, `GET /api/room/{username}` (cùng output).
- **Tests**: vitest 10 + Playwright 4 (mock network).

## Verify

```
go build ./...  → OK
go vet ./...    → OK
vitest          → 10/10
playwright e2e  → 4/4
```

- `GET /api/health` → `{"ok":true}`
- `GET /api/room/mock.live` → `{"live":false}`
- `/ws` (WS handshake) hoạt động; HTTP thường trả 400 (đúng).
