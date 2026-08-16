# TikTok Bar

Nền tảng theo dõi TikTok LIVE: nhập `@username` của streamer đang phát trực tiếp, nhận và hiển thị
real-time các sự kiện **gift (nhận quà)**, **comment**, **lượt người tham gia (join)**, like, follow,
share, battle… cùng thông tin phòng và host.

> **Lưu ý quan trọng:** TikTok **không có API chính thức** cho các sự kiện LIVE. Dự án dùng đường đi
> Webcast reverse-engineered qua thư viện [`gotiktoklive`](https://github.com/steampoweredtaco/gotiktoklive)
> (Go, không chính thức, không liên kết với ByteDance). Xem chi tiết nghiên cứu trong
> [`docs/research/tiktok-live-events.md`](docs/research/tiktok-live-events.md) và
> [`docs/research/tiktok-interactive-games.md`](docs/research/tiktok-interactive-games.md).

> **Release 1.2.0** — server chuyển sang **Gin (gin-gonic)**. Xem [`docs/RELEASE-1.2.md`](docs/RELEASE-1.2.md).
> **Release 1.1.0** — frontend chuyển sang **Vue 3 + Tailwind v4 + shadcn-vue + TanStack Query + axios**,
> thêm room preview + unit/e2e tests. Xem [`docs/RELEASE-1.1.md`](docs/RELEASE-1.1.md).
> Release 1.0: [`docs/RELEASE-1.0.md`](docs/RELEASE-1.0.md).

## Kiến trúc

```
┌────────────┐   WebSocket    ┌──────────────────────────────┐   WebSocket/protobuf   ┌────────────────┐
│  Frontend  │ ─────────────► │  Server (Go)                 │ ─────────────────────► │ webcast.tiktok │
│  (Vue 3 + Vite)             │  gotiktoklive relay          │   (reverse-engineered) │   .com         │
└────────────┘                └──────────────────────────────┘                        └────────────────┘
```

- **`server/`** — Go + **Gin (gin-gonic)** + `gorilla/websocket`. Nhận `{"action":"connect"}` qua WebSocket,
  chạy `gotiktoklive.TrackUser()`, relay các event đã decode tới client.
- **`frontend/`** — Vue 3 + Vite + TypeScript, **Tailwind CSS v4** + **shadcn-vue** + **TanStack Query** +
  **axios**. Ô nhập username + live event feed + room preview.

> **Fork vendored:** `gotiktoklive` được copy vào `server/third_party/gotiktoklive` (kèm `replace`
> trong `go.mod`) để sửa bug `toUser()` — bản gốc check nhầm field avatar (`AvatarLarge`/`AvatarJpg`)
> khiến `ProfilePicture` luôn rỗng. Bản vá đọc `AvatarThumb` để hiển thị avatar thật của viewer.

## Chạy

```bash
# 1. Server (Go)
cd server
go run .                    # http://localhost:3001

# 2. Frontend (tab khác)
cd frontend
bun install
bun run dev                 # http://localhost:5173
```

Mở http://localhost:5173, nhập `@username` của một streamer **đang LIVE** (vd `ducnt.tiktok`), bấm
**Kết nối** → nhận event real-time.

### Signing (self-hosted)

Server tự ký WebSocket hoàn toàn tại chỗ bằng **QuickJS** (chạy `webmssdk.js` của TikTok) —
sinh `X-Bogus` + `X-Gnarly` + `msToken`, **không cần bất kỳ dịch vụ sign bên thứ ba nào**.

Yêu cầu build: `CGO_ENABLED=1` (QuickJS là C, biên dịch qua cgo):

```bash
cd server
CGO_ENABLED=1 go run .
```

## Production build

```bash
cd frontend && bun run build      # tạo frontend/dist
cd ../server && CGO_ENABLED=1 go build -o bin/tiktok-bar . && ./bin/tiktok-bar   # tự serve frontend/dist tại :3001
```

## API / WebSocket

- `GET /api/health` — `{ ok }`
- WebSocket endpoint `/ws`. Tin nhắn **server → client** là JSON `{ type, data, ts }`:
  - `type`: `status` (connecting/connected/disconnected/error/ended/idle), `chat`, `gift`, `member`,
    `like`, `follow`, `share`, `roomUser`, `questionNew`, `liveIntro`, `linkMicBattle`, `linkMicArmies`.
  - `status` connected kèm `roomInfo` (title, owner, userCount) và `roomId`.
- Tin nhắn **client → server** (JSON):
  - `{"action":"connect","username":"..."}` — kết nối tới phòng LIVE.
  - `{"action":"disconnect"}` — ngắt kết nối.

## Xử lý event cần lưu ý (theo nghiên cứu)

- **Gift streak:** gift `Type == 1` gửi nhiều lần, chỉ xử lý `RepeatEnd == true` (bản cuối cùng của
  streak) để tránh đếm trùng.
- **Like:** "not always triggered" trên stream nhiều người xem — coi là lossy.
- **Room auth:** mỗi room bật/tắt các tính năng khác nhau; không phải sự kiện nào cũng được emit.
- **History events:** server lọc bỏ các event `IsHistory()` (batch tin nhắn cũ khi connect) để chỉ
  nhận sự kiện live mới.
