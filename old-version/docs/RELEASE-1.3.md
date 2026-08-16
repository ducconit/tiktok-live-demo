# TikTok Bar — Release 1.3.0

> **Ngày:** 2026-08-16 · **Trạng thái:** Stable · **Git tag:** `v1.3.0`

## Thay đổi chính: WebSocket relay → **Sockudo**

Chuyển hẳn service WebSocket từ **Go server tự relay (gorilla/websocket)** sang **Sockudo**
(Pusher-compatible WebSocket server, Rust). Frontend nhận events realtime qua Sockudo, Go server
chỉ đóng vai trò **publisher** + **control (REST)**.

## Kiến trúc mới

```
Browser ──WS (6001)──► Sockudo            ◄── HTTP publish (HMAC) ── Go server ── TikTok
   │                      ▲                                        │
   └── POST /api/connect ─┘────────────────────────────────────────┘
```

- **Go server** publish events lên Sockudo channel `user_<username>` (Pusher HTTP API + HMAC-SHA256).
- **Browser** (`@sockudo/client` + `@sockudo/client/vue`) subscribe `user_<username>` → nhận events.
- **Control**: `POST /api/connect` (bắt đầu track, trả connected status) + `POST /api/disconnect`.
- **Room preview** giữ nguyên: `GET /api/room/{username}`.

## Docker Compose (mới)

```yaml
services:
  sockudo:   # sockudo/sockudo:latest — WS server (6001) + metrics (9601)
  redis:     # redis:7-alpine (optional backend)
```

Khởi động: `docker compose up -d` → Sockudo tại `http://localhost:6001` (app `demo-app`/`demo-key`/`demo-secret`).

## Thay đổi chi tiết

| | v1.2.0 | v1.3.0 |
|--|--------|--------|
| Realtime | Go `/ws` relay (gorilla) | **Sockudo** (Pusher-compatible) |
| Go server | relay events → client | **publisher** (HMAC) + REST control |
| Frontend | `LiveSocket` (WebSocket tới :3001/ws) | **`@sockudo/client` + `@sockudo/client/vue`** (subscribe channel) |
| Control | WS command | **REST** `POST /api/connect` / `POST /api/disconnect` |
| Docker | — | **`docker-compose.yml`** (sockudo + redis) |

## Giữ nguyên

- **Self-host signing** (QuickJS): X-Bogus + X-Gnarly + msToken.
- `CONNECTION_MODE` (websocket/long_poll), `POLL_INTERVAL_MS`, fix nút Dừng, avatar.
- REST: `/api/health`, `/api/room/{username}`.
- Frontend: Vue 3 + Tailwind v4 + shadcn-vue + TanStack Query + axios.

## Testing

- **Unit (vitest)**: 6 tests (api/axios + ConnectBar).
- **E2E (Playwright)**: 4 tests — mock REST + mock Sockudo WS (`page.route` + `routeWebSocket`, Pusher protocol:
  `pusher:connection_established` → `pusher:subscribe` → `pusher_internal:subscription_succeeded` → events);
  production qua `E2E_MOCK=0` + `E2E_LIVE_USER`/`E2E_OFFLINE_USER`.
- **Manual verify**: connect `conchofanny` (live) → nhận `liveIntro`/`member`/`roomUser` realtime qua Sockudo.

## Verify

```
go build ./... → OK · go vet → OK
vitest → 6/6 · playwright e2e → 4/4
docker compose up -d → Sockudo UP (/up = OK)
```

## Config mới (`.env`)

```env
SOCKUDO_URL=http://localhost:6001
SOCKUDO_APP_ID=demo-app
SOCKUDO_APP_KEY=demo-key
SOCKUDO_APP_SECRET=demo-secret
```
