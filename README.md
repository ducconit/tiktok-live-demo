# TikTok Bar

Nền tảng theo dõi TikTok LIVE: nhập `@username` của streamer đang phát trực tiếp, nhận và hiển thị
real-time các sự kiện **gift (nhận quà)**, **comment**, **lượt người tham gia (join)**, like, follow,
share, battle… cùng thông tin phòng và host.

> **Lưu ý quan trọng:** TikTok **không có API chính thức** cho các sự kiện LIVE. Dự án dùng đường đi
> Webcast reverse-engineered qua thư viện [`gotiktoklive`](https://github.com/steampoweredtaco/gotiktoklive)
> (Go, không chính thức, không liên kết với ByteDance). Xem chi tiết nghiên cứu trong
> [`docs/research/tiktok-live-events.md`](docs/research/tiktok-live-events.md) và
> [`docs/research/tiktok-interactive-games.md`](docs/research/tiktok-interactive-games.md).

## Kiến trúc

```
┌────────────┐   WebSocket    ┌──────────────────────────────┐   WebSocket/protobuf   ┌────────────────┐
│  Frontend  │ ─────────────► │  Server (Go)                 │ ─────────────────────► │ webcast.tiktok │
│  (Vite+React)               │  gotiktoklive relay          │   (reverse-engineered) │   .com         │
└────────────┘                └──────────────────────────────┘                        └────────────────┘
```

- **`server/`** — Go + `net/http` + `gorilla/websocket`. Nhận `{"action":"connect"}` qua WebSocket,
  chạy `gotiktoklive.TrackUser()`, relay các event đã decode tới client.
- **`frontend/`** — Vite + React + TypeScript, Tailwind (CDN). Ô nhập username + live event feed.

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
npm install
npm run dev                 # http://localhost:5173
```

Mở http://localhost:5173, nhập `@username` của một streamer **đang LIVE** (vd `nhu2hand2`), bấm
**Kết nối** → nhận event real-time.

### Cấu hình signing (tùy chọn)

`gotiktoklive` ủy thác việc ký WebSocket cho sign server của Euler Stream (`tiktok.eulerstream.com`)
theo mặc định. **Không cần API key** vẫn kết nối được (anonymous), nhưng bị rate-limit và có thể
không ổn định. Tạo key tại https://www.eulerstream.com để tăng giới hạn và độ ổn định:

```env
# server/.env
SIGN_API_KEY=your-euler-stream-key
# SIGN_SERVER_URL=https://your-custom-sign-server.com   # tùy chọn
```

## Production build

```bash
cd frontend && npm run build      # tạo frontend/dist
cd ../server && go build -o bin/tiktok-bar . && ./bin/tiktok-bar   # tự serve frontend/dist tại :3001
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
