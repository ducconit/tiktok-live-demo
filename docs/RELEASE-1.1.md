# TikTok Bar — Release 1.1.0

> **Ngày:** 2026-08-16 · **Trạng thái:** Stable · **Git tag:** `v1.1.0`

## Thay đổi chính: Frontend viết lại bằng Vue

Bản 1.1.0 **migrate toàn bộ frontend từ React sang Vue** với stack hiện đại, giữ nguyên chức năng
(theo dõi TikTok LIVE real-time) và **backend Go không đổi**.

### Stack frontend mới

| Công nghệ | Vai trò |
|-----------|---------|
| **Vue 3** (Composition API + `<script setup>` + TS) | Framework |
| **Vite** | Build + dev server |
| **Tailwind CSS v4** (`@tailwindcss/vite`, CSS-first `@theme`) | Styling |
| **shadcn-vue** (reka-ui) | UI components (Button, Input, Card, Badge, Avatar) |
| **@tanstack/vue-query** | Server state / data fetching |
| **axios** | HTTP client |
| **vitest** + @vue/test-utils | Unit tests |
| **Playwright** | E2E tests |
| **bun** | Package manager |

### Tính năng mới / cải tiến

- **Room preview**: gõ username → debounce → gọi `/api/room/<user>` (TanStack Query + axios) → hiển
  thị "Đang LIVE — tiêu đề" hoặc "Không live" TRƯỚC khi kết nối.
- **REST endpoint mới** phía server: `GET /api/room/{username}` (trả preview room, không cần kết nối).
- **Dark theme** qua shadcn-vue CSS variables (primary cyan, destructive red — giữ bảng màu cũ).
- Giữ nguyên **fix nút Dừng** (button `type="button"`, không auto-reconnect).

### Testing

- **Unit (vitest)**: 10 tests — `socket.test.ts` (LiveSocket: queue/connect/disconnect/events),
  `api.test.ts` (axios mock adapter: room preview), `ConnectBar.test.ts` (connect/disconnect/Dừng).
- **E2E (Playwright)**: 4 tests chạy **mock network** (Playwright `page.route` + `page.routeWebSocket`
  — không gọi TikTok):
  - App load, connect → events realtime → **Dừng không auto-reconnect**,
  - offline user → error "không LIVE", room preview live status.
- **Production e2e**: đặt `E2E_MOCK=0` + `E2E_LIVE_USER` / `E2E_OFFLINE_USER` để chạy với TikTok thật.

## Chi tiết so với v1.0.0

| | v1.0.0 | v1.1.0 |
|--|--------|--------|
| Frontend | React 18 | **Vue 3** |
| Styling | Tailwind CDN | **Tailwind v4 + shadcn-vue** |
| Data | WebSocket thuần | WebSocket + **TanStack Query + axios** (REST) |
| Test | Chưa có | **vitest (10) + Playwright e2e (4)** |
| Package manager | npm | **bun** |
| REST API | `/api/health` | + `/api/room/{username}` |
