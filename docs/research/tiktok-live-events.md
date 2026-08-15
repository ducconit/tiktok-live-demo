# TikTok LIVE Events cho Game/App Client — Ghi chú nghiên cứu

**Câu hỏi nghiên cứu:** Làm thế nào để một client (game/app) nhận được các sự kiện real-time từ
TikTok LIVE interactive games — cụ thể là **sự kiện nhận quà (gift)**, **sự kiện người xem/người
tham gia vào phòng (viewer/participant join)**, và **sự kiện comment**?

**Kết luận nhanh:** Tại thời điểm viết tài liệu này, **TikTok không công bố một API chính thức,
công khai nào cho các sự kiện real-time của TikTok LIVE** (gifts, likes, comments, viewer joins,
shares, follows, battles). Không có sản phẩm "TikTok Live Interaction API", "LIVE Events API",
hay "TikTok Connect" nào trên `developers.tiktok.com`. Sản phẩm Webhooks chính thức của TikTok là
HTTP-callback và **không** bao phủ các sự kiện LIVE stream. Mọi thứ nhận được các sự kiện này
hiện nay đều đi qua **dịch vụ push nội bộ "Webcast" không được công bố** (`webcast.tiktok.com`) —
một giao thức protobuf-over-WebSocket được reverse-engineer, **không có SDK chính thức, không có
tài liệu chính thức, không có SLA, và không có rate limits được công bố**.

Tài liệu này ghi lại (a) những gì TikTok *chính thức* cung cấp, (b) cách đường đi Webcast không
chính thức thực sự hoạt động (dùng các thư viện open-source phổ biến làm nguồn tham chiếu thứ
cấp), và (c) cấu trúc payload cho gifts, comments, và joins.

---

## 1. Những gì TikTok chính thức cung cấp (primary sources)

### 1.1 Bề mặt sản phẩm TikTok for Developers

Danh sách sản phẩm developer đầy đủ hiển thị trên trang chủ `developers.tiktok.com`. Danh sách
"Featured products" và "More products" chỉ chứa:

- Login Kit
- Share Kit
- Content Posting API
- Embed Videos
- Webhooks
- Data Portability API
- Scopes
- Green Screen Kit
- Display API
- Research Tools (Research API / Commercial Content API)
- TikTok Shop (liên kết ra `partner.tiktokshop.com`)
- TikTok for Business (liên kết ra `business-api.tiktok.com`)

> **Không có sản phẩm LIVE interaction / LIVE events / LIVE games nào trong danh sách này.**

Primary source: <https://developers.tiktok.com/>

> "Featured products — Login Kit, Share Kit, Content Posting API, Embed Videos. …
> More products — Webhooks … Data Portability API … Scopes … Green Screen Kit … Display API …
> Research Tools … Commercial Content API."

### 1.2 Webhooks chính thức của TikTok — HTTP callbacks, KHÔNG phải LIVE event stream

Cơ chế real-time chính thức của TikTok là **webhooks** (HTTPS POST callbacks), không phải
WebSocket.

Primary source: <https://developers.tiktok.com/doc/webhooks-overview/>

> "Webhook is a subscription that notifies your application via a callback URL when an event
> happens in TikTok. Rather than requiring you to pull information via API, you can use webhooks
> to get information on events that occur. Notifications are delivered via HTTPS POST in JSON
> format to the callback url configured for your app in the Developer Portal."

Các yêu cầu/giới hạn chính từ cùng trang:

- Callback URL phải được đăng ký trong TikTok Developer Portal.
- Callback URL phải **phản hồi ngay lập tức bằng HTTP 200** để xác nhận đã nhận.
- Endpoint của callback URL **bắt buộc dùng HTTPS**.
- Nếu không trả về 200, TikTok sẽ retry với **exponential backoff lên đến 72 giờ**.
- Delivery là **"at least once"** — cùng một sự kiện có thể đến nhiều lần, vì vậy handler phải
  **idempotent**.

### 1.3 Các loại sự kiện webhook chính thức — không có cái nào là LIVE events

Danh sách sự kiện chính thức rất nhỏ và **không** bao gồm gifts, comments, likes, hay joins.

Primary source: <https://developers.tiktok.com/doc/webhooks-events>

Các loại sự kiện được ghi nhận:

- `authorization.removed` — user hủy cấp quyền cho app
- `video.upload.failed` — một Video Kit upload thất bại
- `video.publish.completed` — một Video Kit upload được publish
- `portability.download.ready` — một Data Portability API export sẵn sàng

> "Webhooks let you subscribe to events and receive notice when an event occurs. … By default,
> you are subscribed to all events when a callback URL is configured in the TikTok Developer
> Portal."

Cấu trúc request-body của webhook chính thức (liên quan nếu bạn xây dựng trên webhooks thật của
TikTok):

| Key | Type | Mô tả |
| --- | --- | --- |
| `client_key` | string | key duy nhất được cấp cho partner |
| `event` | string | tên sự kiện |
| `create_time` | int64 | UTC epoch time tính bằng giây |
| `user_openid` | string | định danh duy nhất của TikTok user (từ `/oauth/access_token/`) |
| `content` | string | chuỗi JSON đã serialize chứa thông tin sự kiện |

**Kết luận:** sản phẩm Webhooks chính thức dành cho các sự kiện vòng đời Login Kit / Content
Posting / Data Portability — nó **không** phải kênh cho LIVE gift/comment/join events.

### 1.4 Bề mặt "LIVE" chính thức — chỉ là embed player (playback, không có event stream)

Tài liệu developer của TikTok đề cập "Embed LIVE Player" cho phép nhúng một LIVE room vào iframe và
"hỗ trợ tùy biến player qua URL và một giao diện tùy chọn cho host pages cần điều khiển playback."
Đây là bề mặt **player/embed**, không phải event API — nó không lộ ra các sự kiện gift, comment,
hay join cho host page.

Primary source (search index snippet của `developers.tiktok.com`):
<https://developers.tiktok.com/doc/search?q=live>

> "The TikTok Embed LIVE Player lets you embed a TikTok LIVE room in an iframe. It supports
> URL-based player customization and an optional interface for host pages that need to control
> playback."

(URL tài liệu trực tiếp hiện trả về 404; snippet trên là từ chính docs search của TikTok.)

### 1.5 API v2 chính thức (host tại `open.tiktokapis.com`) — không có LIVE endpoints

Họ API chính thức hiện tại (OAuth 2.0, host tại `open.tiktokapis.com`) chỉ lộ các endpoint
user-info / video list / video query trong bề mặt v2 được ghi nhận. Không có LIVE endpoints.

Primary source: <https://developers.tiktok.com/doc/tiktok-api-v2-introduction/>

> "Our new API is hosted at `open.tiktokapis.com`. Learn more about our new API endpoints:
> Get User Info, List Videos, Query Videos."

### 1.6 Scopes reference chính thức — không có LIVE scopes

Bảng scopes liệt kê các chủ đề cho Local Service, Portability, Research, `user.info.*`, `video.list`,
`video.publish`, `video.upload` — **không có scope đọc live-stream**.

Primary source: <https://developers.tiktok.com/doc/tiktok-api-scopes>

### 1.7 GitHub org chính thức của TikTok — không có LIVE SDK

GitHub organization đã xác minh của TikTok (`@tiktok`) công bố OpenSDK (Login/Share Kit) và
Business API SDKs, cùng các dự án infrastructure. **Không có TikTok LIVE client SDK** trong org này.

Primary source: <https://github.com/tiktok>

Các repo liên quan quan sát được:

- `tiktok-opensdk-ios` / `tiktok-opensdk-android` (Login Kit + Share Kit)
- `tiktok-business-api-sdk` (Python)
- `tiktok-business-unity-sdk`, `tiktok-business-ios-sdk`, `tiktok-business-android-sdk`
- một số dự án infrastructure (sparkling, pnpm-sync, knit, sparo, v.v.)

### 1.8 Mini Games SDK — SDK client-side cho game chạy *bên trong* TikTok

TikTok có **Mini Games SDK** (`TTMinis.game`), nhưng đây là **JavaScript SDK phía client chạy bên
trong app TikTok** dành cho các mini game được publish *lên* TikTok. Bề mặt API của nó là login,
payments, ads, storage, audio, canvas, touch, và networking chung — nó **không** lộ ra các event
listener cho LIVE gift/comment/join.

Primary sources:

- <https://developers.tiktok.com/doc/mini-games-overview>
- <https://developers.tiktok.com/doc/mini-games-sdk-overview>

> "Mini Games SDK is a client-side JavaScript SDK that runs inside the TikTok app. It provides
> in-app capabilities like login and authorization, payments, UI triggers, rewarded ads,
> lifecycle hooks, networking, and capability detection. JavaScript APIs contained within the
> Mini Games SDK contain the prefix `TTMinis.game`."

SDK có bao gồm một WebSocket primitive chung:

> `connectSocket` — "Creates a WebSocket connection to the specified WSS URL with optional
> headers, protocols, and timeout, returning a SocketTask" (kèm `onOpen`, `onMessage`,
> `onClose`, `onError`, `send`).

Đây là một WebSocket client thô (dành cho backend riêng của game), **không phải** API đăng ký
TikTok LIVE event.

---

## 2. Cách client LIVE events thực sự nhận được hiện nay (đường đi "Webcast" không chính thức)

Vì không có API chính thức, hệ sinh thái kết nối trực tiếp đến **dịch vụ push Webcast nội bộ**
của TikTok mà web client TikTok sử dụng. Hai open-source client phổ biến nhất mô tả rõ điều này.
**Coi phần này là tài liệu tham chiếu thứ cấp/reverse-engineering — nó không phải tài liệu chính
thức của TikTok và không liên kết với ByteDance.**

### 2.1 Các thư viện tự mô tả về chúng

Secondary source — README của `isaackogan/TikTokLive` (Python):
<https://github.com/isaackogan/TikTokLive>

> "The #1 TikTok LIVE API Client for Python **(Unofficial, Unaffiliated with ByteDance Ltd.)** —
> Connect to any TikTok LIVE stream and receive real-time chats, gifts, likes, follows & more!
> With this library you can connect to any TikTok livestream and fetch all data available to
> users in a stream using just a creator's `@unique_id`."
>
> "**Note:** This is **not** a production-ready API. It is a reverse engineering project."

Nó cũng lộ các endpoint bên dưới trong `WebDefaults`:

- `tiktok_webcast_url` — "The TikTok livestream URL (`https://webcast.tiktok.com`) where
  livestreams can be accessed from."
- `tiktok_sign_url` — "The signature server used to generate tokens to connect to TikTokLive."

Secondary source — README của `zerodytrash/TikTok-Live-Connector` (Node.js):
<https://github.com/zerodytrash/TikTok-Live-Connector>

> "A Node.js library to receive live stream events such as comments and gifts in realtime from
> TikTok LIVE **by connecting to TikTok's internal Webcast push service**. This package includes
> a wrapper that connects to the Webcast service using just the username (`@uniqueId`). …
> No credentials are required."
>
> "**NOTE:** This is not an official API. It's a reverse engineering project."

### 2.2 Cơ chế kết nối (reverse-engineered)

Từ README của Node connector:

1. Bạn bắt đầu chỉ với một creator `@unique_id` (ví dụ từ
   `https://www.tiktok.com/@officialgeilegisela/live` → `officialgeilegisela`).
2. Thư viện fetch room info từ web API của TikTok để resolve `roomId`.
3. Nó kết nối đến Webcast push service (`webcast.tiktok.com`), upgrade lên WebSocket khi TikTok
   cung cấp (option `enableWebsocketUpgrade`, mặc định `true`); ngược lại nó fallback sang request
   polling (`requestPollingIntervalMs`, mặc định `1000` ms).
4. Các frame đến là **protobuf-encoded**; sự kiện `rawData` lộ ra tên message-type đã decode và
   binary payload:
   > "Triggered every time a protobuf encoded webcast message arrives. You can deserialize the
   > binary object … with `protobufjs`."
5. Cần một bước **signature/token generation** để tạo ra các tham số kết nối đã xác thực. Trong
   `TikTokLive`, việc này được thực hiện qua một "signature server" (`tiktok_sign_url`), và trong
   Node connector qua `signProviderOptions` (`host` của một signing server tùy chỉnh). Các dịch vụ
   managed của bên thứ ba (Euler Stream, Tik.Tools, v.v.) bán signing này như một hosted service.

### 2.3 Authentication / subscription

- **Quyền truy cập read-only không cần bất kỳ TikTok credentials nào** — chỉ cần `@unique_id` của
  streamer.
  > "No credentials are required." (README Node connector)
- **Gửi chat message** yêu cầu giá trị cookie `sessionid` của tài khoản.
  > "`sessionId` … the current Session ID of your TikTok account (**sessionid** cookie value) if
  > you want to send automated chat messages via the `sendMessage()` function."
- **Không có đăng ký app chính thức, OAuth flow, cấp scope, hay phê duyệt** cho đường đi này — nó
  là một endpoint không được công bố hướng tới consumer.

### 2.4 Rate limits, permissions, và yêu cầu phê duyệt

- **Review/approval của TikTok for Developers chính thức** chỉ áp dụng cho các sản phẩm chính thức
  (Login Kit, Content Posting, v.v.), không áp dụng cho LIVE events, vì không có LIVE API nào ở đó.
  Primary source: <https://developers.tiktok.com/doc/our-guidelines-developer-guidelines>
  > "all apps seeking to integrate with our APIs and SDKs in Live are reviewed."
  > "We do not provide an official review timeline or any guarantees for approval."
  > "Respect our API throttling limits."
- **Đường đi Webcast không chính thức:** không có rate limits được công bố. Node connector cảnh báo
  chờ trước khi reconnect "to avoid being rate-limited" khi disconnect, và cảnh báo rằng `sendMessage`
  spam "can lead to the suspension of your TikTok account." README của `TikTokLive` ghi chú một API
  key để "increase rate limits" qua sign service của nó — tức là limits được quản lý bởi các dịch
  vụ sign của bên thứ ba, không được TikTok ghi nhận.
- **Rủi ro bất ổn giao thức:** đường đi này đã từng hỏng và có thể hỏng lại.
  > "Due to a change on the part of TikTok, versions prior v1.1.7 are no longer functional."
  (README Node connector)

---

## 3. Các loại sự kiện có sẵn trên đường đi Webcast

### 3.1 Inventory sự kiện (Python `TikTokLive` — secondary source)

Custom events: `ConnectEvent`, `DisconnectEvent`, `LiveEndEvent`, `LivePauseEvent`,
`LiveUnpauseEvent`, `FollowEvent`, `ShareEvent`, `SuperFanEvent`, `SuperFanJoinEvent`,
`SuperFanBoxEvent`, `WebsocketResponseEvent`, `UnknownEvent`.

Proto events bao gồm (trích lọc, liên quan đến game):

- `CommentEvent` — viewer gửi chat comment
- `GiftEvent` — gift được gửi
- `JoinEvent` — viewer tham gia livestream
- `LikeEvent` — nhận likes
- `SocialEvent` — share/follow (cũng được lộ dưới dạng `FollowEvent`/`ShareEvent`)
- `BarrageEvent` — viewer "VIP" (theo gifting level) tham gia chat room
- `RoomUserSeqEvent` — thông tin viewer-count hiện tại
- `LinkMicBattleEvent` — một battle được bắt đầu
- `LinkMicArmiesEvent` — một user trong battle nhận điểm
- `EmoteChatEvent`, `EnvelopeEvent`, `PollEvent`, `QuestionNewEvent`, `GoalUpdateEvent`,
  `RankTextEvent`, `RankUpdateEvent`, `RoomEvent`, `RoomPinEvent`, `SubNotifyEvent`,
  `GameRankNotifyEvent`, `LiveGameIntroEvent`, và các event khác.

Secondary source: <https://github.com/isaackogan/TikTokLive>

### 3.2 Inventory sự kiện (Node `TikTok-Live-Connector` — secondary source)

Control events: `connected`, `disconnected`, `streamEnd`, `rawData`, `websocketConnected`, `error`.

Message events: `member` (viewer join), `chat` (comment), `gift`, `roomUser` (viewer count +
danh sách top gifter), `like`, `social` (share/follow), `emote`, `envelope` (treasure chest),
`questionNew`, `linkMicBattle`, `linkMicArmies`, `liveIntro`, `subscribe`.

Derived custom events: `follow`, `share`.

Secondary source: <https://github.com/zerodytrash/TikTok-Live-Connector>

---

## 4. Cấu trúc payload — gifts, comments, joins

Dưới đây là các payload **đã chuẩn hóa** do Node connector phát ra sau khi decode protobuf bên
dưới. Tên field phản ánh mapping của thư viện, không phải schema chính thức của TikTok.

### 4.1 Sự kiện comment (`chat`)

```js
tiktokLiveConnection.on('chat', data => {
    console.log(`${data.uniqueId} writes: ${data.comment}`);
});
```

Các field chính:

| Field | Ý nghĩa |
| --- | --- |
| `comment` | nội dung chat |
| `userId` | user id số của TikTok của người gửi |
| `secUid` | secure uid (đã redact) của người gửi |
| `uniqueId` | `@handle` của người gửi |
| `nickname` | tên hiển thị của người gửi |
| `profilePictureUrl` | URL avatar của người gửi |
| `followRole` | `0` = không, `1` = follower, `2` = friends |
| `userBadges[]` | badge moderator / top-gifter / subscriber, v.v. |
| `isModerator` / `isNewGifter` / `isSubscriber` | cờ vai trò |
| `topGifterRank` | hạng hoặc null |
| `msgId` | message id |
| `createTime` | epoch ms dạng string |

### 4.2 Sự kiện gift (`gift`)

```js
tiktokLiveConnection.on('gift', data => {
    if (data.giftType === 1 && !data.repeatEnd) {
        // streak đang diễn ra — temporary/partial
    } else {
        // streak kết thúc hoặc gift không thể streak — repeat_count cuối cùng
    }
});
```

Các field chính:

| Field | Ý nghĩa |
| --- | --- |
| `giftId` | id loại gift (ví dụ `5953`) |
| `giftName` | tên gift dễ đọc (khi bật `enableExtendedGiftInfo`) |
| `giftType` | `1` = gift có thể streak |
| `repeatCount` | số gift trong streak hiện tại |
| `repeatEnd` | `true` ở sự kiện cuối của streak |
| `diamondCount` | giá gift tính bằng diamonds (khi có extended info) |
| `giftPictureUrl` | ảnh gift (khi có extended info) |
| `describe` | mô tả dễ đọc, ví dụ `"Sent Nevalyashka doll"` |
| `monitorExtra` | raw envelope chứa `room_id`, `msg_id`, `gift_id`, `repeat_count`, `repeat_end`, `to_user_id`, `from_user_id`, `log_id`, v.v. |
| `userId` / `uniqueId` / `nickname` / `secUid` / `profilePictureUrl` | danh tính người gửi |
| `receiverUserId` | người nhận (có thể là guest broadcaster trong battles) |
| `msgId` / `createTime` / `timestamp` | message id và timestamps |
| `extendedGiftInfo` | entry catalog gift tĩnh đầy đủ khi bật `enableExtendedGiftInfo` |

Lưu ý streak (quan trọng cho game logic):

> "Users have the capability to send gifts in a streak. … even if the user sends a `giftType: 1`
> gift only once, you will receive the event twice. Once with `repeatEnd: false` and once with
> `repeatEnd: true`."

### 4.3 Sự kiện viewer join (`member`)

```js
tiktokLiveConnection.on('member', data => {
    console.log(`${data.uniqueId} joins the stream!`);
});
```

Các field chính:

| Field | Ý nghĩa |
| --- | --- |
| `userId` / `secUid` / `uniqueId` / `nickname` / `profilePictureUrl` | danh tính người tham gia |
| `followRole` | `0` = không, `1` = follower, `2` = friends |
| `userBadges[]` | badges (moderator, v.v.) |
| `isModerator` / `isNewGifter` / `isSubscriber` | cờ vai trò |
| `topGifterRank` | hạng top-gifter hoặc null |
| `msgId` / `createTime` | message id và epoch ms |
| `displayType` | ví dụ `"live_room_enter_toast"` |
| `label` | template đã localized, ví dụ `"{0:user} joined"` |

### 4.4 Các payload khác hữu ích cho game

- `like`: `{ likeCount, totalLikeCount, userId, uniqueId, … }` — lưu ý
  > "For streams with many viewers, this event is not always triggered by TikTok."
- `social` / `follow` / `share`: identity + `displayType`
  (`pm_main_follow_message_viewer_2` cho follow, `pm_mt_guidance_share` cho share) +
  `label` ví dụ `"{0:user} followed the host"`.
- `roomUser`: `{ viewerCount, topViewers[] }` — viewer count và bảng xếp hạng top gifter.
- `linkMicBattle`: `{ battleUsers[] }` với `userId`/`uniqueId`/`nickname` cho host vs guest.
- `linkMicArmies`: `{ battleStatus, battleArmies[] }` với `points` và `participants[]` từng bên.

### 4.5 Feature toggles theo từng room

Trạng thái `connected` của Node connector bao gồm `roomInfo.room_auth` — map cho biết room cụ thể
bật tính năng nào (ví dụ `Chat`, `Gift`, `Digg` (likes), `Share`, `Viewers`, `UserCount`,
`Rank`, `Poll`, `Battle`). Đây là tín hiệu runtime hữu ích để biết một sự kiện cụ thể có được emit
trong room hay không.

Ví dụ (từ README): `room_auth: { Chat: true, Gift: true, Digg: true, Share: true,
Viewers: false, UserCount: 0, … }`.

---

## 5. SDK / thư viện chính thức cho LIVE (câu trả lời: không có)

- **Không tồn tại TikTok LIVE client SDK chính thức.** GitHub org của TikTok chỉ ship OpenSDK
  (Login/Share) và Business API SDKs. Primary source: <https://github.com/tiktok>
- **Mini Games SDK phía client không phải LIVE event SDK** — nó dành cho game chạy *bên trong*
  app TikTok và chỉ lộ WebSocket networking chung. Primary source:
  <https://developers.tiktok.com/doc/mini-games-sdk-overview>
- **Các lựa chọn tích hợp cho game client, trong thực tế:**

  1. **Các thư viện Webcast client không chính thức** (reverse-engineered, secondary sources):
     - Python: `TikTokLive` — <https://github.com/isaackogan/TikTokLive>
     - Node.js: `TikTok-Live-Connector` — <https://github.com/zerodytrash/TikTok-Live-Connector>
     - Ports: Java (`jwdeveloper/TikTok-Live-Java`), C#/Unity (`frankvHoof93/TikTokLiveSharp`),
       Go, Rust.
  2. **WebSocket gateway do bên thứ ba quản lý** (commercial, secondary): dịch vụ chạy signing +
     connection server phía server và lộ ra một WebSocket sạch (ví dụ Euler Stream, Tik.Tools).
     Đây không phải sản phẩm của TikTok.
  3. **Game client nên chạy các thư viện này trong một server/bridge process**, sau đó relay
     events tới game qua Socket.IO/WebSocket riêng — README Node khuyến nghị rõ pattern này cho
     browser clients.

---

## 6. Consumer LIVE vs official developer APIs — khác biệt chính

| Khía cạnh | TikTok for Developers chính thức | Consumer LIVE ("Webcast") path |
| --- | --- | --- |
| Product | Login/Share/Posting/Display/Research/Webhooks/Mini Games | Không có (undocumented) |
| Transport | HTTPS REST + HTTPS webhooks | WebSocket (có polling fallback) + protobuf frames |
| Event types | 4 webhook events (auth/video/portability) | ~30+ events gồm gifts, chat, joins, likes, battles |
| Auth | OAuth 2.0, đăng ký app, review/approval, scopes | Chỉ `@unique_id` (đọc); `sessionid` để gửi |
| Rate limits | Được ghi nhận theo từng API + hướng dẫn throttling | Không được ghi nhận; do signing/anti-bot systems thực thi |
| SLA / stability | Chính thức, được hỗ trợ | Không có SLA; hỏng khi TikTok đổi giao thức |
| SDKs | OpenSDK (iOS/Android), Business SDKs | Không chính thức; thư viện cộng đồng reverse-engineered |
| Permissions | Scopes + app review | Không có (truy cập không chính thức dữ liệu room công khai) |

**Khác biệt liên quan trực tiếp đến dự án này:** bề mặt developer chính thức *không* có cách nào
đăng ký LIVE gift/comment/join events, trong khi consumer Webcast path thì có — nhưng đường đi sau
không được hỗ trợ, không có tài liệu, và đòi hỏi signature generation mà TikTok không cung cấp
công khai.

---

## 7. Hàm ý / khuyến nghị cho `tiktok-bar`

1. **Đừng lập kế hoạch dựa trên "TikTok Live Interaction API" chính thức.** Thiết kế layer thu
   nhận sự kiện đằng sau một abstraction để có thể đổi transport nếu sau này TikTok ship một cái.
2. **Để nhận gift/comment/join events thực sự hiện nay**, các lựa chọn thực tế là:
   - tự host một Webcast client không chính thức (Node hoặc Python) + signing server riêng, hoặc
   - dùng một WebSocket gateway được quản lý (bên thứ ba, không phải TikTok).
3. **Xử lý gifts theo streak-aware** (`giftType === 1` và `repeatEnd`) để tránh đếm trùng.
4. **Coi likes là lossy** ("not always triggered" trên các stream nhiều viewer) — đừng dựa vào
   số likes chính xác cho game logic.
5. **Kiểm tra `room_auth` / `roomInfo`** khi connect để biết room thực sự bật những sự kiện nào.
6. **Rủi ro compliance:** đường đi này dùng các endpoint không được công bố; TikTok có thể thay
   đổi hoặc hạn chế bất kỳ lúc nào và tài khoản có thể bị xử lý vì abuse (đặc biệt là tự động gửi
   chat messages).

---

## Sources

### Primary (TikTok chính thức)

- TikTok for Developers — trang chủ/danh sách sản phẩm: <https://developers.tiktok.com/>
- TikTok Webhooks — Overview: <https://developers.tiktok.com/doc/webhooks-overview/>
- TikTok Webhooks — Events: <https://developers.tiktok.com/doc/webhooks-events>
- Scopes Reference: <https://developers.tiktok.com/doc/tiktok-api-scopes>
- Migrating to the new API (`open.tiktokapis.com`): <https://developers.tiktok.com/doc/tiktok-api-v2-introduction/>
- Developer Guidelines (app review / approval): <https://developers.tiktok.com/doc/our-guidelines-developer-guidelines>
- Overview of TikTok Mini Games: <https://developers.tiktok.com/doc/mini-games-overview>
- Mini Games SDK Overview: <https://developers.tiktok.com/doc/mini-games-sdk-overview>
- TikTok docs search — "live" (snippet Embed LIVE Player): <https://developers.tiktok.com/doc/search?q=live>
- TikTok GitHub organization (SDK chính thức): <https://github.com/tiktok>
- TikTok docs overview (developer kits / API references): <https://developers.tiktok.com/doc/overview/>

### Secondary (không chính thức / reverse-engineered — chỉ tham chiếu chéo)

- `isaackogan/TikTokLive` (Python) README: <https://github.com/isaackogan/TikTokLive>
- `zerodytrash/TikTok-Live-Connector` (Node.js) README: <https://github.com/zerodytrash/TikTok-Live-Connector>

### Xác nhận bổ sung (secondary, đã trích dẫn inline)

- Tik.Tools — "Is there an official TikTok LIVE API? No.": <https://tik.tools/>
- Các trang tài liệu TikTok for Developers — Content Posting API / Webhooks đã tham chiếu ở trên.
