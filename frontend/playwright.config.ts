import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  use: {
    baseURL: "http://localhost:3001",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
  // Serve the app. Reuses the running Go server (:3001, serves frontend/dist);
  // in CI it builds + starts the Go server. Network to TikTok is NOT needed:
  // tests intercept /api/room/* + /ws in the browser (mock mode) unless
  // E2E_MOCK=0 (real TikTok, requires E2E_LIVE_USER / E2E_OFFLINE_USER).
  webServer: {
    command:
      "cd ../server && CGO_ENABLED=1 go build -o /tmp/tiktok-bar-e2e . && /tmp/tiktok-bar-e2e",
    port: 3001,
    reuseExistingServer: true,
    timeout: 60_000,
  },
});
