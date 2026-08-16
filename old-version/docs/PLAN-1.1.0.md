# Kế hoạch v1.1.0 — React → Vue + stack mới

> **Mục tiêu:** Viết lại toàn bộ frontend bằng **Vue 3** với stack hiện đại, giữ nguyên chức năng
> (theo dõi TikTok LIVE real-time, WebSocket, các fix UI) và **backend Go không đổi**.

## Stack mục tiêu

| Công nghệ | Vai trò | Ghi chú |
|-----------|---------|---------|
| **Vite** | Build + dev server | v6 (mới nhất) |
| **Vue 3** | Framework | Composition API + `<script setup>` + TypeScript |
| **Tailwind CSS v4** | Styling | Dùng plugin `@tailwindcss/vite` + cấu hình CSS-first (`@theme`) — **không còn CDN** |
| **shadcn-vue** | UI components | `button`, `input`, `card`, `badge`, `avatar`… (Radix Vue + Tailwind) |
| **@tanstack/vue-query** | Server state / data fetching | Dùng cho REST API (room info, health, tìm kiếm…) |
| **axios** | HTTP client | Gọi REST API, dùng chung với vue-query |
| *(đề xuất)* **vue-router** | Routing | Nếu cần nhiều trang (dashboard, cài đặt…) |
| *(đề xuất)* **pinia** | State management | Nếu cần chia sẻ state phức tạp |

> **WebSocket vẫn dùng `socket.ts`** (class thuần TS, không phụ thuộc framework) — giữ nguyên, chỉ
> bọc thành composable `useLiveSocket` để dùng trong Vue.

## Kiến trúc mới

```
frontend/src/
├── main.ts                 # Vue entry: createApp + QueryClientProvider
├── App.vue                 # Layout chính + quản lý kết nối
├── composables/
│   └── useLiveSocket.ts    # bọc LiveSocket (socket.ts) cho Vue
│   └── useRoomQuery.ts     # TanStack Query: gọi REST room info
├── services/
│   ├── socket.ts           # (giữ nguyên) class LiveSocket
│   └── api.ts              # axios instance + API endpoints
├── lib/
│   └── utils.ts            # cn() của shadcn-vue (clsx + tailwind-merge)
├── components/
│   ├── ui/                 # shadcn-vue: button, input, card, badge, avatar…
│   ├── ConnectBar.vue      # ô username + nút Kết nối/Dừng
│   ├── RoomCard.vue        # thông tin phòng + host
│   ├── EventFeed.vue       # danh sách event + auto-scroll
│   └── EventRow.vue        # 1 dòng event (badge + avatar + nội dung)
└── types.ts                # (giữ nguyên)
```

## Ánh xạ UI hiện tại → shadcn-vue

| UI cũ (Tailwind tay) | shadcn-vue |
|----------------------|-----------|
| `<button>` thủ công | `<Button variant="destructive">` (Dừng) / `<Button>` (Kết nối) |
| `<input>` thủ công | `<Input>` |
| `<section>` phòng | `<Card>` |
| badge event | `<Badge>` |
| avatar | `<Avatar>` |

## Ánh xạ React → Vue

| React | Vue 3 |
|-------|-------|
| `useState` | `ref()` |
| `useEffect([])` | `onMounted` + `onBeforeUnmount` |
| `useEffect([deps])` | `watch(...)` |
| `useMemo` | `computed` |
| `useRef` (DOM) | template `ref` |
| props | `defineProps` |
| `onClick` | `@click` |
| `value`+`onChange` | `v-model` |
| *không có* | **TanStack Query** `useQuery` cho REST |
| *không có* | **axios** thay thế fetch thủ công |

## Các bước

1. **Setup**: `npm create vite@latest` (vue-ts) → thêm `tailwindcss v4` + `@tailwindcss/vite` +
   `shadcn-vue` init + `@tanstack/vue-query` + `axios`.
2. **Tailwind v4**: `@import "tailwindcss"` + `@theme` khai báo màu custom (`ttcyan`, `ttred`,
   `ink`, `panel`, `edge`) — giữ nguyên bảng màu hiện tại.
3. **Reuse**: `socket.ts`, `types.ts` giữ nguyên.
4. **Port logic**: `App.vue` (ref/computed/watch + onMounted subscription),
   `ConnectBar.vue`, `RoomCard.vue`, `EventFeed.vue`, `EventRow.vue` → dùng shadcn-vue.
5. **REST layer**: `api.ts` (axios) + `useRoomQuery.ts` (TanStack Query) — sẵn sàng cho API mới.
6. **Build + test**: `npm run build` → dist; test connect → events → **Dừng (không tự reconnect)**.
7. **Release v1.1.0**: cập nhật docs, bump UI version, tag + push.

## Rủi ro / lưu ý

- ⚠️ **Fix nút Dừng**: phải giữ đúng pattern (button `type="button"` + `@click`, form `@submit.prevent`)
  để không tái diễn auto-reconnect.
- **Tailwind v4 khác v3/CDN**: cấu hình CSS-first, không dùng `tailwind.config.js`; màu custom qua `@theme`.
- **shadcn-vue** cần **Radix Vue** + `class-variance-authority` + `clsx` + `tailwind-merge` (init tự cài).
- **TanStack Query + axios** dùng cho REST; WebSocket vẫn qua `socket.ts` (không qua axios/query).
- Backend Go **không đổi** (vẫn serve `frontend/dist`).
- Có thể giữ bản React cũ (đã tag v1.0.0) để rollback nếu cần.
