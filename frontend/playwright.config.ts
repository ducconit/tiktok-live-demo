import { defineConfig, devices } from '@playwright/test'

// E2E test cho frontend (end-user live monitor) — mock mode (CI-safe):
// chặn REST (/api/v1/public/live/*) + Sockudo WebSocket (ws://localhost:6002)
// trong browser — KHÔNG cần TikTok thật, không cần backend/docker.
//
// Chạy:  cd frontend && bun run test:e2e
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  timeout: 30_000,
  expect: { timeout: 10_000 },

  // Dev server: bun dev trên 5173 (port vite.config). Đã có server chạy → reuse.
  webServer: {
    command: 'bun run dev',
    url: 'http://127.0.0.1:5173/',
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },

  use: {
    baseURL: 'http://127.0.0.1:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'] } },
  ],
})
