# TikTok Bar — Release 1.0

> **Ngày:** 2026-08-16 · **Trạng thái:** Stable · **Git tag:** `v1.0.0`

TikTok Bar là nền tảng theo dõi **TikTok LIVE real-time**: nhập `@username` của streamer đang phát
trực tiếp, nhận + hiển thị ngay các sự kiện **gift**, **comment**, **join**, like, follow, share,
battle… cùng thông tin phòng và host — **hoàn toàn tự host, không phụ thuộc bất kỳ dịch vụ bên thứ 3 nào**.

---

## 1. Change log

| Phiên bản | Nội dung |
|-----------|----------|
| **v1.0.0** (release này) | Hoàn thiện toàn bộ từ prototype: self-host signing, WebSocket + long-poll, config, UI fixes |
| dd3d74b — *self-hosted signing polish* | Retry offline (3 lần rồi ngắt), xoá debug harness → test chính thức, fixed temp dir, dùng username của chủ repo, timeout mstoken |
| 447b528 — *remove euler* | **Xoá hoàn toàn Euler Stream** (sign server bên thứ 3) → chuyển sang signing tại chỗ |
| 02f3890 — *v1.0.0 (prototype)* | App đầu tiên: gotiktoklive relay + Euler signing (mốc khởi đầu) |

### Chi tiết các thay đổi chính (sau prototype)

- **Self-host signing** — thay Euler bằng QuickJS chạy `webmssdk.js` của TikTok tại chỗ:
  - `X-Bogus` + `X-Gnarly`: sinh bằng QuickJS (cgo) qua XHR interception.
  - `msToken`: mint qua `mssdk-sg.tiktok.com` (service của TikTok), tự rotate qua `X-Ms-Token`.
  - `ttwid`: warmup từ `www.tiktok.com`.
- **Hai chế độ kết nối** (cấu hình bằng `CONNECTION_MODE`):
  - `websocket` (mới): push server TikTok, realtime, heartbeat 10s, enter-room, gzip, ack.
  - `long_poll` (mặc định): poll `im/fetch` mỗi `POLL_INTERVAL_MS` (mặc định 3000ms), retry 3 lần rồi ngắt.
- **WebSocket mode fallback long-poll** nếu WS lỗi.
- **UI fixes**: nút Dừng dừng hẳn (không tự reconnect — fix race form submit), close WS khi Dừng.
- **Test**: `signer_test.go` (signature + X-Bogus determinism), dọn dead code (Euler, WebSocket cũ, utls).
- **Security/clean**: không còn key/user TikTok nào của người khác; `go.mod` sạch.

---

## 2. Stack công nghệ

### Backend — `server/` (Go 1.26)
| Thành phần | Mục đích |
|-----------|----------|
| `net/http` + `gorilla/websocket` | WebSocket frontend ↔ server, relay events |
| `gotiktoklive` (fork vendored) | Kết nối + parse Webcast (protobuf), sửa bug avatar |
| **QuickJS (C qua cgo)** | Chạy `webmssdk.js` → sinh `X-Bogus` + `X-Gnarly` (không goja vì goja sinh X-Bogus không hợp lệ) |
| `bogdanfinn/tls-client` | Chrome TLS fingerprint (JA3/JA4) để qua WAF của TikTok |
| `gobwas/ws` | WebSocket push server TikTok (chế độ websocket) |
| `google.golang.org/protobuf` | Parse protobuf Webcast |
| `joho/godotenv` | Cấu hình `.env` |

### Frontend — `frontend/` (Vite + React)
| Thành phần | Mục đích |
|-----------|----------|
| Vite 5 | Build + dev server |
| React 18 + TypeScript 5 | UI event feed |
| Tailwind (CDN) | Styling |

---

## 3. Các hoạt động (events) hỗ trợ

Server relay các event sau (qua WebSocket JSON `{ type, data, ts }`):

| `type` | Nghĩa | Dữ liệu chính |
|--------|-------|---------------|
| `chat` | Bình luận | `comment`, `user`, `msgId`, `timestamp` |
| `gift` | Nhận quà | `giftName`, `diamondCount`, `repeatCount`, `repeatEnd`, `user` |
| `member` | Người vào phòng | `user`, `memberCount` |
| `follow` | Follow | `user` |
| `share` | Chia sẻ | `user` |
| `like` | Thả tim | `likeCount`, `totalLikeCount`, `user` |
| `roomUser` | Số người xem | `viewerCount` |
| `questionNew` | Câu hỏi mới | `questionText`, `user` |
| `liveIntro` | Giới thiệu live | `title`, `user` |
| `linkMicBattle` | Battle mic | `users` |
| `linkMicArmies` | Phe battle | `status`, `battles` |
| `status` | Trạng thái | `connecting` / `connected` / `disconnected` / `error` / `ended` / `idle` + `roomInfo` |

---

## 4. Các luồng (flows)

### 4.1 Luồng kết nối (browser → server → TikTok)

```
Browser ──ws──► Server ──signed HTTPS──► webcast.tiktok.com
   │                │                          │
   └── connect ─────┘── warmup(ttwid)          │
                      └── mint msToken         │
                      └── QuickJS sign         │
                      └── TrackUser ───────────┘
                      │      │
                      │   TrackRoom
                      │      ├── im/fetch (initial) → cursor, messages
                      │      ├── [websocket] → push server + im_enter_room + heartbeat
                      │      └── [long_poll] → poll im/fetch mỗi N ms
                      │
                      └── relay events → browser
```

### 4.2 Luồng signing (tự host, không bên thứ 3)

```
1. Warmup:  GET www.tiktok.com      → cookie ttwid (device fingerprint)
2. msToken: POST mssdk-sg.tiktok.com → msToken 148 ký tự (rotate qua X-Ms-Token mỗi response)
3. Sign:    QuickJS (cgo) chạy webmssdk.js + fake-DOM (hybrid-fake-dom.js)
            → XHR interception → X-Bogus + X-Gnarly (gắn vào URL)
4. Fetch:   tls-client (Chrome fingerprint) gửi request đã ký
```

### 4.3 Luồng WebSocket mode (push server TikTok)

```
- Build URL: wss://webcast-ws.<vùng>.tiktok.com/webcast/im/ws_proxy/ws_reuse_supplement/
             ?base_ws_params&room_id&compress=gzip  → sign → wss://
- Connect → gửi im_enter_room
- Heartbeat mỗi 10s (HeartBeatMessage)
- Nhận WebcastPushFrame → gzip → WebcastResponse → events; gửi ack khi cần
- WS lỗi → fallback long-poll
```

### 4.4 Luồng long-poll mode

```
- Mỗi POLL_INTERVAL_MS (mặc định 3000ms): GET im/fetch (đã ký) với cursor hiện tại
- Parse WebcastResponse → events + cursor mới
- Room offline/soft-block: thử 3 lần (1 + 2 retry) rồi phát DisconnectEvent → UI "disconnected"
```

---

## 5. Cấu hình (`.env`)

```env
PORT=3001
LOG_DIR=logs
CONNECTION_MODE=long_poll     # hoặc websocket
POLL_INTERVAL_MS=3000
```

- Yêu cầu build: **`CGO_ENABLED=1`** (QuickJS là C, build qua cgo).
- Không cần API key, không cần tài khoản, không gọi dịch vụ bên thứ 3.

---

## 6. Bảo mật & giới hạn

- **Không dùng Euler / Tik.Tools / bất kỳ sign server bên thứ 3** — toàn bộ signing tại chỗ.
- Giao tiếp mạng chỉ với **TikTok** (`www.tiktok.com`, `webcast.tiktok.com`, `mssdk-sg.tiktok.com`).
- Không lộ key/user TikTok của người khác trong repo.
- **Giới hạn:** TikTok không có API chính thức → dựa trên reverse-engineered Webcast, có thể bị TikTok thay đổi;
  room phải **đang live** mới nhận event; WebSocket mode có thể bị chặn → tự fallback long-poll.

---

## 7. Chạy

```bash
cd server && CGO_ENABLED=1 go run .   # http://localhost:3001
cd frontend && npm install && npm run build   # build UI (hoặc npm run dev cho Vite :5173)
```
