import { test, expect, type Page, type WebSocketRoute } from "@playwright/test";

// Mock mode (CI): intercept /api/room/* + /ws in the browser — no TikTok.
// Production e2e: set E2E_MOCK=0 + E2E_LIVE_USER / E2E_OFFLINE_USER to hit real TikTok.
const MOCK = process.env.E2E_MOCK !== "0";
const LIVE_USER = process.env.E2E_LIVE_USER ?? "mock.live";
const OFFLINE_USER = process.env.E2E_OFFLINE_USER ?? "mock.offline";

function send(ws: WebSocketRoute, payload: unknown) {
  ws.send(JSON.stringify(payload));
}

function parseWsMessage(message: unknown): { action?: string; username?: string } {
  let raw: string;
  if (typeof message === "string") {
    raw = message;
  } else if (message && typeof (message as { text?: () => string }).text === "function") {
    raw = (message as { text: () => string }).text();
  } else {
    raw = String(message);
  }
  try {
    return JSON.parse(raw) as { action?: string; username?: string };
  } catch {
    return {};
  }
}

async function setupMockNetwork(page: Page) {
  if (!MOCK) return;

  // Mock REST: /api/room/<user>
  await page.route("**/api/room/*", (route) => {
    const user = decodeURIComponent(route.request().url().split("/").pop() ?? "");
    const live = user === LIVE_USER;
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(
        live
          ? { live: true, title: `Mock LIVE — ${user}`, userCount: 1234, owner: { uniqueId: user, nickname: user } }
          : { live: false },
      ),
    });
  });

  // Mock WS: /ws
  await page.routeWebSocket("**/ws", (ws) => {
    ws.onMessage((message) => {
      const data = parseWsMessage(message);
      if (data.action === "connect") {
        if (data.username === LIVE_USER) {
          send(ws, { type: "status", data: { state: "connecting", username: data.username }, ts: Date.now() });
          setTimeout(() => {
            send(ws, {
              type: "status",
              data: {
                state: "connected",
                roomId: "1234567890123456789",
                roomInfo: { title: "Mock LIVE", owner: { uniqueId: data.username, nickname: data.username }, userCount: 1234 },
              },
              ts: Date.now(),
            });
            const events = [
              { type: "chat", data: { comment: "hello mock 👋", user: { uniqueId: "mock.user", nickname: "Mock User" } } },
              { type: "gift", data: { giftName: "Rose", repeatCount: 1, diamondCount: 1, user: { uniqueId: "mock.user", nickname: "Mock User" } } },
              { type: "member", data: { user: { uniqueId: "mock.user2", nickname: "Mock User 2" }, memberCount: 1235 } },
            ];
            let i = 0;
            const timer = setInterval(() => {
              const e = events[i % events.length];
              send(ws, { type: e.type, data: e.data, ts: Date.now() });
              i++;
            }, 400);
            ws.onClose(() => clearInterval(timer));
          }, 400);
        } else {
          send(ws, { type: "status", data: { state: "error", message: "User này hiện không đang LIVE." }, ts: Date.now() });
        }
      } else if (data.action === "disconnect") {
        send(ws, { type: "status", data: { state: "idle" }, ts: Date.now() });
      }
    });
  });
}

test.beforeEach(async ({ page }) => {
  await setupMockNetwork(page);
  await page.goto("/");
});

test("app loads with title + connect bar", async ({ page }) => {
  await expect(page.getByText("TikTok Bar")).toBeVisible();
  await expect(page.getByPlaceholder("tiktok username")).toBeVisible();
  await expect(page.getByRole("button", { name: "Kết nối" })).toBeVisible();
});

test("connect to LIVE user shows events; Dừng stops and does NOT auto-reconnect", async ({ page }) => {
  await page.getByPlaceholder("tiktok username").fill(LIVE_USER);
  await page.getByRole("button", { name: "Kết nối" }).click();

  const stopBtn = page.getByRole("button", { name: "Dừng" });
  await expect(stopBtn).toBeVisible({ timeout: 10_000 });

  // Real-time mock events arrive.
  await expect(page.getByText(/hello mock|Rose/)).toBeVisible({ timeout: 10_000 });

  // Dừng → idle, and must NOT auto-reconnect.
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

test("room preview shows live status via /api/room", async ({ page }) => {
  const input = page.getByPlaceholder("tiktok username");

  await input.fill(LIVE_USER);
  await expect(page.getByText(/Đang LIVE/)).toBeVisible({ timeout: 8_000 });

  await input.fill(OFFLINE_USER);
  await expect(page.getByText(/Không live/)).toBeVisible({ timeout: 8_000 });
});
