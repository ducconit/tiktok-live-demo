import { test, expect, type Page, type WebSocketRoute } from "@playwright/test";

// Mock mode: intercept REST (/api/v1/public/live/*) + Sockudo WebSocket
// (ws://localhost:6002) — không cần TikTok thật, không cần backend/docker.
const LIVE_USER = process.env.E2E_LIVE_USER ?? "mock.live";
const OFFLINE_USER = process.env.E2E_OFFLINE_USER ?? "mock.offline";

function send(ws: WebSocketRoute, payload: unknown) {
  ws.send(JSON.stringify(payload));
}

function parseMsg(message: unknown): { event?: string; data?: { channel?: string } } {
  let raw: string;
  if (typeof message === "string") raw = message;
  else if (message && typeof (message as { text?: () => string }).text === "function") {
    raw = (message as { text: () => string }).text();
  } else raw = String(message);
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

async function mockSockudoWs(page: Page) {
  await page.routeWebSocket("**/app/**", (ws) => {
    // Server → client: báo connection established trước khi client dám subscribe.
    send(ws, {
      event: "pusher:connection_established",
      data: { socket_id: "mock.socket.1234", activity_timeout: 30 },
    });

    let timer: ReturnType<typeof setInterval> | undefined;
    ws.onMessage((message) => {
      const data = parseMsg(message);
      if (data.event === "pusher:subscribe") {
        const channel = data.data?.channel ?? "";
        send(ws, { event: "pusher_internal:subscription_succeeded", channel, data: {} });
        const events = [
          { type: "chat", data: { comment: "hello mock 👋", user: { uniqueId: "mock.user", nickname: "Mock User" } }, ts: Date.now() },
          { type: "gift", data: { giftName: "Rose", repeatCount: 1, diamondCount: 1, user: { uniqueId: "mock.user", nickname: "Mock User" } }, ts: Date.now() },
          { type: "member", data: { user: { uniqueId: "mock.user2", nickname: "Mock User 2" }, memberCount: 1235 }, ts: Date.now() },
        ];
        let i = 0;
        timer = setInterval(() => {
          const e = events[i % events.length];
          send(ws, { event: "event", channel, data: JSON.stringify(e) });
          i++;
        }, 400);
      }
    });
    ws.onClose(() => {
      if (timer) clearInterval(timer);
    });
  });
}

async function mockRest(page: Page) {
  // Room preview — GET /api/v1/public/live/{user}
  await page.route("**/api/v1/public/live/*", (route) => {
    const user = decodeURIComponent(route.request().url().split("/").pop() ?? "");
    const live = user === LIVE_USER;
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        code: "0",
        msg: "",
        data: live
          ? { live: true, title: `Mock LIVE — ${user}`, userCount: 1234, owner: { uniqueId: user, nickname: user } }
          : { live: false },
        meta: {},
      }),
    });
  });

  // Connect — POST /api/v1/public/live/{user}/connect
  await page.route("**/connect", async (route) => {
    const url = route.request().url();
    const user = decodeURIComponent(url.split("/").filter(Boolean).at(-2) ?? "");
    if (user === LIVE_USER) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          code: "0",
          msg: "",
          data: {
            connected: true,
            roomId: "1234567890123456789",
            roomInfo: { title: "Mock LIVE", owner: { uniqueId: user, nickname: user }, userCount: 1234 },
          },
          meta: {},
        }),
      });
    } else {
      await route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({
          code: "404",
          msg: "User này hiện không đang LIVE.",
          data: null,
          meta: {},
        }),
      });
    }
  });

  // Disconnect — POST /api/v1/public/live/{user}/disconnect
  await page.route("**/disconnect", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ code: "0", msg: "", data: { ok: true }, meta: {} }),
    }),
  );
}

test.beforeEach(async ({ page }) => {
  await mockRest(page);
  await mockSockudoWs(page);
  await page.goto("/");
});

test("app loads with title + connect bar", async ({ page }) => {
  await expect(page.getByText("TikTok Live Platform")).toBeVisible();
  await expect(page.getByPlaceholder("tiktok username")).toBeVisible();
  await expect(page.getByRole("button", { name: "Kết nối" })).toBeVisible();
});

test("connect to LIVE user shows events; Dừng stops and does NOT auto-reconnect", async ({ page }) => {
  await page.getByPlaceholder("tiktok username").fill(LIVE_USER);
  await page.getByRole("button", { name: "Kết nối" }).click();

  const stopBtn = page.getByRole("button", { name: "Dừng" });
  await expect(stopBtn).toBeVisible({ timeout: 10_000 });

  // Mock events arrive qua Sockudo channel (chat event đầu tiên).
  await expect(page.getByText("hello mock 👋")).toBeVisible({ timeout: 10_000 });

  // Dừng → idle, không auto-reconnect.
  await stopBtn.click();
  const connectBtn = page.getByRole("button", { name: "Kết nối" });
  await expect(connectBtn).toBeVisible();
  await page.waitForTimeout(3500);
  await expect(connectBtn).toBeVisible();
  await expect(page.getByRole("button", { name: "Dừng" })).toHaveCount(0);
});

test("connect to OFFLINE user shows error (không LIVE)", async ({ page }) => {
  await page.getByPlaceholder("tiktok username").fill(OFFLINE_USER);
  await page.getByRole("button", { name: "Kết nối" }).click();

  await expect(page.getByText(/không đang LIVE|không LIVE/)).toBeVisible({ timeout: 10_000 });
});

test("room preview shows live status via /api/v1/public/live", async ({ page }) => {
  const input = page.getByPlaceholder("tiktok username");

  await input.fill(LIVE_USER);
  await expect(page.getByText(/Đang LIVE/)).toBeVisible({ timeout: 8_000 });

  await input.fill(OFFLINE_USER);
  await expect(page.getByText(/Không live/)).toBeVisible({ timeout: 8_000 });
});
