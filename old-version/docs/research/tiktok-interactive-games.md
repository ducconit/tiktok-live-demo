# TikTok LIVE Interactive Games & nhận real-time events — Ghi chú nghiên cứu (phần 2)

**Câu hỏi nghiên cứu:** "TikTok LIVE interactive games" (những mini-game chạy trong phòng TikTok
LIVE để người xem tham gia) là gì, ai/ngoài TikTok xây dựng chúng như thế nào, và làm cách nào một
game client nhận được các sự kiện real-time — **gift (nhận quà)**, **participant/join (người tham
gia)**, và **comment**?

Tài liệu này là **phần bổ sung** cho
[`tiktok-live-events.md`](./tiktok-live-events.md) — file đó đã phủ kỹ đường đi không chính thức
"Webcast" (`webcast.tiktok.com`, protobuf-over-WebSocket). Tài liệu này **không lặp lại** phần đó;
nó đi sâu vào **sản phẩm/ứng dụng interactive games và các SDK/API chính thức mới** (nếu có) và
đối chiếu với hiện trạng thực tế.

**Kết luận nhanh:**

1. **Không tồn tại chương trình/platform developer chính thức nào tên "TikTok LIVE Interactive
   Games", "TikTok Live Game SDK", hay "Live Gaming SDK".** Tôi đã đọc toàn bộ cây docs của
   `developers.tiktok.com` (menu điều hướng đầy đủ), tìm kiếm docs (`live game`, `interactive`,
   `live`) — tất cả đều **không có kết quả**; không có mục "LIVE" nào trong menu sản phẩm developer.
2. Thứ **gần nhất là "TikTok Mini Games"** — game chạy *bên trong* app TikTok (không phải trong phòng
   LIVE), có SDK client (`TTMinis.game`), server APIs, hỗ trợ Unity. **Không cái nào lộ sự kiện
   gift/comment/join của phòng LIVE.**
3. Tin tức chính thức (newsroom) chỉ dùng cụm "interactive LIVE" cho **chương trình creator
   (The Game Room, LIVE Fest)**, không phải một nền tảng game.
4. **Bằng chứng mạnh nhất cho thấy TikTok *có* một hệ thống "live game / partnership game" nội bộ**
   là các proto-event **`LiveGameIntroEvent`, `GameRankNotifyEvent`, `PartnershipGameOfflineEvent`,
   `PartnershipDropsUpdateEvent`** xuất hiện trên **đường đi Webcast không chính thức** (được các
   thư viện reverse-engineered ghi nhận) — tức là sự kiện game có tồn tại trong giao thức nội bộ,
   nhưng **không có API/SDK chính thức nào lộ chúng**.
5. Với từng loại sự kiện (gift / join / comment), **ngày hôm nay không có cơ chế chính thức** nào để
   game client nhận được; mọi lựa chọn thực tế đều đi qua đường Webcast reverse-engineered (đã phủ
   trong doc cũ) hoặc gateway managed của bên thứ ba (Euler Stream, v.v.).
6. **Webhooks chính thức của TikTok vẫn chỉ có 4 event** (`authorization.removed`,
   `video.upload.failed`, `video.publish.completed`, `portability.download.ready`) — **không có
   webhook LIVE/game nào được thêm.**

---

## 1. "TikTok LIVE interactive games" — nó là gì (và không phải là gì)

### 1.1 Không có chương trình/platform developer chính thức

Toàn bộ cây docs của TikTok for Developers (đọc từ
<https://developers.tiktok.com/doc/overview/>) gồm: Account/App Management, Our Guidelines,
Integration Essentials (Mobile SDK, Server API, OAuth), Webhooks, Login Kit, Share Kit, Content
Posting API, Data Portability API, Display API, Research Tools, Commercial Content API, Embed,
Monetization, **TikTok Minis Platform**, **Mini Games**, **Mini Dramas**, TikTok GO, Scopes, Legacy
Products.

> **Không có mục "LIVE", "Live", "Interactive Games" nào trong danh sách trên.**

Primary source: <https://developers.tiktok.com/doc/overview/>

Tìm kiếm trong chính docs index của TikTok trả về **trang trống** (không có kết quả index nào):

- <https://developers.tiktok.com/doc/search?q=live%20game> — không có kết quả
- <https://developers.tiktok.com/doc/search?q=interactive> — không có kết quả

### 1.2 TikTok Mini Games — game chạy *bên trong* app, KHÔNG phải trong phòng LIVE

Sản phẩm game developer chính thức duy nhất của TikTok là **TikTok Mini Games**:

> "TikTok mini games are lightweight, cross-border entertainment apps integrated with the
> short-video ecosystem … Users can launch mini games directly in TikTok without downloading them …
> accessible through search, short-video embedding, Minis Center, shared links."

Primary source: <https://developers.tiktok.com/doc/mini-games-overview>

Điểm khác biệt quyết định: mini games mở ra từ **search / short-video / Minis Center / link chia
sẻ** — **không có entry nào từ trong một phòng LIVE**. Tài liệu technical overview xác nhận kiến
trúc runtime:

> "Mini games on TikTok follow a lightweight game format that runs inside the TikTok client. You
> can build games with mainstream engines such as Cocos, Unity, and Laya … The package is loaded and
> executed by the mini game runtime inside the TikTok client … Developers do not need to integrate
> TikTok client capabilities manually, and **should not depend on internal host interfaces that are
> not exposed**."

Primary source: <https://developers.tiktok.com/doc/mini-games-technical-overview>

### 1.3 "Interactive LIVE" trong newsroom = creator programming, không phải platform game

Từ sitemap và các bài newsroom đã fetch, TikTok không công bố một bài giới thiệu chương trình
"interactive games" cho developer. Cụm "interactive LIVE" chỉ xuất hiện trong bối cảnh chương trình
giải trí do TikTok tổ chức:

- **"Gaming goes LIVE on TikTok"** (Nov 2021): ra mắt *The Game Room*, "a monthly LIVE series
  featuring some of TikTok's most prominent creators and global celebrities playing their favorite
  games" — đây là **series phát sóng creator chơi game**, không phải nền tảng để dev build game.
  Primary source: <https://newsroom.tiktok.com/gaming-goes-live-on-tiktok>
- **"TikTok LIVE Game Crown"** (Jul 2026): một sự kiện LIVE âm nhạc ("listening party, fan Q&A…"),
  tên có chữ "game" nhưng là chương trình giải trí. Primary source:
  <https://newsroom.tiktok.com/tiktok-live-game-crown>
- **Bài ra mắt TikTok LIVE tại Việt Nam** (Apr 2021) mô tả các tính năng LIVE chính thức: phát sóng
  LIVE, hiệu ứng/nhãn dán, **Co-host (LIVE nhiều người)** — **không hề nhắc tới game/quiz/trivia.**
  Primary source (tiếng Việt): <https://newsroom.tiktok.com/vi-vn/tiktokgioithieutinhnangtiktoklive>

### 1.4 Bằng chứng TikTok CÓ hệ thống "live game" nội bộ — nhưng chỉ lộ qua đường Webcast

Mặc dù không có API chính thức, các thư viện reverse-engineered ghi nhận **các proto-event liên
quan game trên Webcast** (đọc trực tiếp README của hai thư viện — secondary source):

- `LiveGameIntroEvent` — "Live-game intro message"
- `GameRankNotifyEvent` — "Game-rank notification"
- `PartnershipDropsUpdateEvent` — "Partnership drops campaign update"
- `PartnershipGameOfflineEvent` — "Partnership game taken offline"
- `PartnershipPunishEvent` — "Partnership punishment notice"

Secondary source: README Python `isaackogan/TikTokLive`
(<https://github.com/isaackogan/TikTokLive>) và README Node `zerodytrash/TikTok-Live-Connector`
(<https://github.com/zerodytrash/TikTok-Live-Connector>).

> **Ý nghĩa:** TikTok chắc chắn có một hệ sinh thái "LIVE game / partnership game" trong giao thức
> push nội bộ (streamer host game, viewer tham gia, có hình phạt/quảng bá partnership). Nhưng các
> sự kiện này **không được phơi ra qua bất kỳ API/SDK/webhook chính thức nào** — chỉ nhận được qua
> đường Webcast reverse-engineered (đã phủ trong doc trước).

---

## 2. Cách các "interactive live games" được build hôm nay

### 2.1 Không có SDK chính thức để build game tương tác trong phòng LIVE

- GitHub org chính thức của TikTok (`https://github.com/tiktok`) chỉ ship OpenSDK (Login/Share Kit),
  Business API SDKs và infrastructure — **không có LIVE SDK**. (Đã xác minh ở doc trước, cite:
  <https://github.com/tiktok>)
- Không có trang tài liệu "LIVE API" trong docs nav (mục 1.1).

### 2.2 Mini Games SDK (client, chạy trong app TikTok) — capability, không có LIVE events

Mini Games SDK là JavaScript SDK phía client chạy bên trong app TikTok (prefix `TTMinis.game`):
login/auth, payments, UI triggers, rewarded ads, lifecycle, networking, storage, canvas, audio, v.v.
Nó có một primitive **`connectSocket`** (WebSocket thô tới backend riêng của game), nhưng **không
có event listener nào cho gift/comment/member của phòng LIVE**.

> "Mini Games SDK is a client-side JavaScript SDK that runs inside the TikTok app. It provides
> in-app capabilities like login and authorization, payments, UI triggers, rewarded ads, lifecycle
> hooks, networking, and capability detection. JavaScript APIs … contain the prefix `TTMinis.game`."

Primary source: <https://developers.tiktok.com/doc/mini-games-sdk-overview>
(Đã phủ trong doc trước — giữ ở đây như bối cảnh, không lặp lại.)

### 2.3 TikTok Minis Server APIs — backend-only (identity + commerce), KHÔNG có live events

Mới so với doc trước: TikTok giờ có **TikTok Minis Server APIs** (`open.tiktokapis.com`) — bộ
endpoint **backend-only** cho mini apps / mini games:

> "TikTok Minis Server APIs (open.tiktokapis.com) is a secure, backend-only suite of endpoints that
> manages identity and commerce for TikTok Minis (mini apps and mini games) via OAuth v2 and
> scope-based permissions. It provides OAuth token retrieval, scope-gated user info retrieval, order
> creation and management, and pricing."

Primary source: <https://developers.tiktok.com/doc/minis-server-apis-overview>

Danh mục docs con (từ menu cùng trang): OAuth for TikTok Minis, TikTok User Data API, Payment APIs,
Subscription APIs, Error Codes. **Không có endpoint push/realtime/webhook LIVE nào.**

### 2.4 Build mini game bằng Unity — nhưng vẫn không liên quan LIVE

Docs Mini Games giờ có hướng dẫn Unity đầy đủ (mới so với doc trước): build Unity → IL2CPP →
WebAssembly → mini game package, plugin `com.tiktok.minigame@<version>-Release.unitypackage`, giới
hạn package ≤ 60 MB, DevTool qua CLI `@ttmg/cli` (`ttmg dev`). Primary sources:
<https://developers.tiktok.com/doc/build-mini-games-with-unity>,
<https://developers.tiktok.com/doc/tiktok-sdk-for-unity>.

> Đây là đường đi để ship game *chạy trong TikTok app*, **không phải** để game nhận sự kiện từ phòng
> LIVE.

---

## 3. Cơ chế nhận 3 loại sự kiện (gift / join / comment) — đối chiếu

| Sự kiện | (a) Chính thức (API/SDK/webhook) | (b) Webcast reverse-engineered (đã phủ doc trước) | (c) Gateway managed bên thứ ba |
| --- | --- | --- | --- |
| **Gift (quà)** | Không có | `GiftEvent` / `gift` — kèm streak (`repeatEnd`), diamonds, extended gift info | Có (Euler Stream WebSocket API, v.v.) |
| **Participant join** | Không có | `JoinEvent` / `member` — user identity + `memberCount` | Có |
| **Comment** | Không có | `CommentEvent` / `chat` | Có |

Chi tiết payload của (b) đã nằm ở `tiktok-live-events.md` mục 4 — không lặp lại.

**Điểm cập nhật từ doc trước** (secondary, README Node connector đã fetch lại — branch `ts-rewrite`):

- Schema gift v3: field canonical giờ là `name`, `type`, `image` (alias v2 `gift_name`,
  `gift_type`, `gift_image` vẫn tồn tại để tương thích ngược).
- Có thêm các sự kiện game/battle mới trong schema: `liveGameIntro`, `gameRankNotify`,
  `partnershipDropsUpdate`, `partnershipGameOffline`, `partnershipPunish`, `viewerPicksUpdate`,
  `subPinEvent`, `boostCard`, v.v.

Secondary source: <https://github.com/zerodytrash/TikTok-Live-Connector>

---

## 4. Webhooks chính thức — kiểm tra lại (trả lời Q4)

Đã đọc lại trang `webhooks-events` hiện tại. Vẫn **chính xác 4 event types**, không có loại mới nào
liên quan LIVE/game:

| Event | Ý nghĩa |
| --- | --- |
| `authorization.removed` | user hủy cấp quyền cho app |
| `video.upload.failed` | upload Video Kit thất bại |
| `video.publish.completed` | video được publish |
| `portability.download.ready` | export Data Portability sẵn sàng |

Primary source: <https://developers.tiktok.com/doc/webhooks-events>

> "By default, you are subscribed to all events when a callback URL is configured in the TikTok
> Developer Portal." — tức là 4 sự kiện này là **toàn bộ** danh mục webhook hiện có.

**Kết luận Q4:** không có webhook LIVE/gift/comment/join nào được thêm vào.

---

## 5. Kiến trúc relay sự kiện LIVE tới game client (browser / Unity / Godot / HTML5)

Không có tài liệu chính thức nào của TikTok về kiến trúc này; các pattern dưới đây là **từ cộng
đồng reverse-engineered** (secondary sources, đã fetch và trích dẫn).

### 5.1 Pattern chuẩn: server bridge + Socket.IO / WebSocket riêng

Dự án demo chính thức (của tác giả thư viện Node) **`TikTok-Chat-Reader`** minh họa chính xác pattern
này:

> "A chat reader for TikTok LIVE utilizing TikTok-Live-Connector and **Socket.IO** to forward the
> data to the client. This demo project uses the unofficial TikTok API to retrieve chat comments,
> gifts and other events from TikTok LIVE."

Secondary source: <https://github.com/zerodytrash/TikTok-Chat-Reader>
(Demo: <https://tiktok-chat-reader.zerody.one/>)

Luồng:
1. **Node server** chạy `tiktok-live-connector` (kết nối Webcast).
2. Server **relay** các event đã decode tới browser client qua **Socket.IO**.
3. Browser/HTML5 game client chỉ cần là một WebSocket/Socket.IO client — không cần chạy logic Webcast.

Pattern này áp dụng cho mọi client: browser (HTML5/Phaser/three.js), Unity (dùng `TikTokLiveSharp`
hoặc WebSocket thường), Godot (WebSocket client), v.v.

### 5.2 Phân phối API key an toàn: mint JWT ngắn hạn

README Node connector khuyến nghị (khi chạy connector trong app client-facing như desktop/browser
worker) **không ship API key thô**, mà server mint **JWT ngắn hạn** cho client:

> "When running the connector in a client-facing environment (bundled desktop app, browser worker,
> etc.) you should not ship your raw API key. Mint a short-lived JWT on your server and hand that to
> the client instead." — kèm code `authentication.createJWT(accountId, { limits: { minute, hour,
> day }, expireAfter })` rồi gửi header `x-jwt-key`.

Secondary source: <https://github.com/zerodytrash/TikTok-Live-Connector>

### 5.3 Managed gateway như "WebSocket API" (relay không cần tự host Webcast)

Euler Stream (bên thứ ba, commercial) cung cấp **"WebSocket API — Real-time TikTok LIVE event data
streams"** kèm rate limits công khai, SDK client, và được cả hai thư viện open-source lớn trỏ đến
làm **lựa chọn production**:

> "This is **not** a production-ready API. It is a reverse engineering project. Use the WebSocket
> API for production." — README của cả `TikTokLive` (Python) lẫn `TikTok-Live-Connector` (Node).

Secondary sources: <https://eulerstream.com>, <https://github.com/isaackogan/TikTokLive>,
<https://github.com/zerodytrash/TikTok-Live-Connector>

### 5.4 Lưu ý: Mini Games SDK có `connectSocket` (WebSocket thô)

Nếu game chạy *bên trong* app TikTok (Mini Games), SDK có primitive `connectSocket` để nối tới
backend riêng của game — có thể dùng làm relay từ server bridge của chính bạn. Nhưng đây là WebSocket
chung, **không phải** kênh nhận sự kiện LIVE từ TikTok. Primary source:
<https://developers.tiktok.com/doc/mini-games-sdk-websocket>

---

## 6. Constraints: rate limits, stability, ToS/compliance, region

### 6.1 Rate limits
- **Đường Webcast reverse-engineered:** không có rate limits công khai. README Node khuyến nghị
  "wait a little bit before attempting a reconnect to avoid being rate-limited". Python lib có
  `tiktok_sign_api_key` để "increase rate limits" — tức là giới hạn do **dịch vụ sign của bên thứ
  ba** quản lý, không phải TikTok.
- **Gateway managed (Euler Stream):** có trang rate limits riêng; hiện tại là sản phẩm commercial
  của bên thứ ba.
- **API chính thức của TikTok (không liên quan LIVE):** có rate limits ghi nhận
  (<https://developers.tiktok.com/doc/tiktok-api-v2-rate-limit>) — nhưng không áp dụng cho LIVE.

### 6.2 Stability risk
- Đường Webcast đã từng vỡ khi TikTok đổi giao thức (doc trước trích: "Due to a change on the part
  of TikTok, versions prior v1.1.7 are no longer functional"). Giao thức và schema cũng **thay đổi
  theo version** — README Python hiện nói schema v3 và cảnh báo "if you don't see one you used to
  rely on, it's because TikTok removed it from the schema."
- Không có SLA nào từ TikTok cho đường này.

### 6.3 ToS / compliance
- Đường Webcast là endpoint nội bộ không công bố, không có OAuth/scope/approval — tiềm ẩn rủi ro
  TikTok thay đổi hoặc xử lý abuse (đặc biệt khi **gửi** chat: cần `sessionid` + Euler API key, và
  docs cũ cảnh báo spam có thể bị treo tài khoản).
- Tự host Webcast client có thể vi phạm ToS TikTok; gateway managed thì chuyển rủi ro đó cho bên
  thứ ba (và phụ thuộc vào họ).
- App review của TikTok ("all apps seeking to integrate with our APIs and SDKs in Live are
  reviewed") chỉ áp dụng cho sản phẩm developer chính thức, không có LIVE API để review.

### 6.4 Region availability
- **Mini Games:** doc ghi "Already launched in markets including the U.S., Japan, Indonesia, Turkey,
  Saudi Arabia, Thailand, Brazil, Malaysia, Philippines, and Vietnam" — Việt Nam có. Primary source:
  <https://developers.tiktok.com/doc/mini-games-overview>
- **TikTok LIVE / mini games trong phòng LIVE:** không có tài liệu chính thức nào về region
  availability của "interactive games" (vì sản phẩm đó không tồn tại ở tầng developer). Trải nghiệm
  người dùng LIVE (Co-host, LIVE match/battle, LIVE Events) là tính năng app, không có API.
- **Vietnam:** trang newsroom VN xác nhận LIVE có mặt từ 2020–2021 (Co-host, effects) —
  <https://newsroom.tiktok.com/vi-vn/tiktokgioithieutinhnangtiktoklive>.

---

## 7. Checklist "đã kiểm tra & xác nhận-absent" (để tránh tìm lại)

Những URL sau **đã được fetch và cho kết quả "không tồn tại / không có"** — phân biệt rõ
*confirmed-absent* vs *not-found*:

| Điểm kiểm tra | URL đã fetch | Kết quả |
| --- | --- | --- |
| Sản phẩm developer "LIVE" trong docs nav | `developers.tiktok.com/doc/overview/` | **Không có mục LIVE nào** (confirmed-absent) |
| Docs search "live game" | `developers.tiktok.com/doc/search?q=live%20game` | Không có kết quả |
| Docs search "interactive" | `developers.tiktok.com/doc/search?q=interactive` | Không có kết quả |
| Webhook LIVE/game | `developers.tiktok.com/doc/webhooks-events` | Chỉ 4 event cũ (confirmed-absent) |
| Webhook overview | `developers.tiktok.com/doc/webhooks-overview` | HTTP callback, không phải LIVE stream |
| Chương trình "interactive games" trên newsroom | `newsroom.tiktok.com/sitemap.xml` + bài viết | Chỉ creator programming (The Game Room, LIVE Fest), không có program dev game |
| Help center "LIVE games" | `support.tiktok.com/en/live-gifts-wallet` | Category LIVE chỉ có LIVE match/Events/Gifts/Fan Club — không có "LIVE games" article |
| GitHub org TikTok LIVE SDK | `github.com/tiktok` | Không có (confirmed-absent, đã phủ doc trước) |
| `blastmode.com` (một vendor game third-party từng được nhắc) | `www.blastmode.com` | **Domain bị bỏ / rao bán (parked)** — không còn hoạt động |
| `livepk.com` (vendor game third-party) | `livepk.com` | Transport error — **không xác minh được** (not-found) |

---

## 8. Hàm ý cho `tiktok-bar`

1. **Đừng chờ đợi một "TikTok Live Game SDK" chính thức** — nó không tồn tại trong docs ngày hôm
   nay; thiết kế lớp nhận sự kiện đằng sau một abstraction (giống khuyến nghị ở doc trước).
2. **Nếu mục tiêu là game trong phòng LIVE (nhận gift/comment/join real-time):** con đường thực tế
   duy nhất là (a) tự host Webcast client (Node/Python) + relay qua Socket.IO/WebSocket riêng, hoặc
   (b) dùng gateway managed (Euler Stream WebSocket API). Không có lựa chọn chính thức.
3. **Nếu mục tiêu là game chạy bên trong TikTok (Mini Games):** có đường đi chính thức hoàn chỉnh
   (SDK `TTMinis.game`, server APIs, Unity, DevTool) — nhưng **không nhận được sự kiện LIVE room**;
   phải tự nối backend qua `connectSocket` và tự lấy dữ liệu từ đâu đó (vd Webcast).
4. **Tận dụng bằng chứng `LiveGameIntroEvent`/`GameRankNotifyEvent`/`Partnership*`:**
   giao thức nội bộ đã có khái niệm game — nếu sau này TikTok mở API chính thức, nó sẽ xoay quanh
   các sự kiện này; chuẩn bị mapping sẵn.
5. **Xử lý gift streak-aware** (`repeatEnd`), **coi likes là lossy**, kiểm tra `room_auth` — vẫn đúng
   theo doc trước.
6. **Quản lý rủi ro:** đường Webcast không có SLA, dễ vỡ khi TikTok đổi schema; gateway managed thì
   chuyển rủi ro + phụ thuộc vendor; luôn có JWT ngắn hạn thay vì API key thô nếu client-facing.

---

## Sources

### Primary (TikTok chính thức — đã fetch)

- TikTok for Developers — Docs Overview (toàn bộ cây menu, không có mục LIVE):
  <https://developers.tiktok.com/doc/overview/>
- TikTok for Developers — trang chủ/danh sách sản phẩm: <https://developers.tiktok.com/>
- Docs search — "live game": <https://developers.tiktok.com/doc/search?q=live%20game>
- Docs search — "interactive": <https://developers.tiktok.com/doc/search?q=interactive>
- Webhooks — Events (4 event types, không có LIVE): <https://developers.tiktok.com/doc/webhooks-events>
- Webhooks — Overview: <https://developers.tiktok.com/doc/webhooks-overview/>
- Mini Games — Overview: <https://developers.tiktok.com/doc/mini-games-overview>
- Mini Games — Technical Overview: <https://developers.tiktok.com/doc/mini-games-technical-overview>
- Mini Games SDK — Overview: <https://developers.tiktok.com/doc/mini-games-sdk-overview>
- Mini Games SDK — Websocket (`connectSocket`): <https://developers.tiktok.com/doc/mini-games-sdk-websocket>
- TikTok Minis Server APIs — Overview (backend-only, không có live events):
  <https://developers.tiktok.com/doc/minis-server-apis-overview>
- Build Mini Games With Unity: <https://developers.tiktok.com/doc/build-mini-games-with-unity>
- TikTok SDK for Unity: <https://developers.tiktok.com/doc/tiktok-sdk-for-unity>
- Embed Player: <https://developers.tiktok.com/doc/embed-player>
- Rate Limits (API chính thức, không liên quan LIVE): <https://developers.tiktok.com/doc/tiktok-api-v2-rate-limit>
- Newsroom — "Gaming goes LIVE on TikTok" (The Game Room, series creator):
  <https://newsroom.tiktok.com/gaming-goes-live-on-tiktok>
- Newsroom — "TikTok LIVE Game Crown": <https://newsroom.tiktok.com/tiktok-live-game-crown>
- Newsroom VN — "TikTok chính thức giới thiệu tính năng TikTok LIVE":
  <https://newsroom.tiktok.com/vi-vn/tiktokgioithieutinhnangtiktoklive>
- Newsroom — sitemap (rà soát game/live/quiz, không có program interactive games):
  <https://newsroom.tiktok.com/sitemap.xml>
- TikTok Help Center — "TikTok LIVE, Gifts, and wallet" (không có article "LIVE games"):
  <https://support.tiktok.com/en/live-gifts-wallet>
- TikTok GitHub organization (không có LIVE SDK): <https://github.com/tiktok>
- TikTok gaming hub (trang `tiktok.com/games`, JS-rendered): <https://www.tiktok.com/games>

### Secondary (không chính thức / reverse-engineered / bên thứ ba — đã fetch)

- `isaackogan/TikTokLive` (Python) README — gồm `LiveGameIntroEvent`, `GameRankNotifyEvent`,
  `PartnershipGameOfflineEvent`, schema v3, khuyến nghị Euler Stream cho production:
  <https://github.com/isaackogan/TikTokLive>
- `zerodytrash/TikTok-Live-Connector` (Node, branch `ts-rewrite`) README — signing qua Euler Stream,
  `SignConfig`/JWT, `sessionid`+`tt-target-idc`, đầy đủ game/battle events:
  <https://github.com/zerodytrash/TikTok-Live-Connector>
- `zerodytrash/TikTok-Chat-Reader` — reference architecture relay (connector + Socket.IO tới browser):
  <https://github.com/zerodytrash/TikTok-Chat-Reader>
- Euler Stream — managed "TikTok LIVE API" / WebSocket API / rate limits:
  <https://eulerstream.com>

### Xác nhận bổ sung (secondary — đã trích dẫn inline)

- `blastmode.com` — domain parked/rao bán (vendor game cũ không còn): <https://www.blastmode.com>
- `livepk.com` — transport error, không xác minh được: <https://livepk.com>
