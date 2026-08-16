import { defineConfig, devices } from '@playwright/test'

// E2E test cho dashboard — chạy dev server tự động (webServer), đăng nhập thật.
// Yêu cầu: backend + postgres + valkey đang chạy (docker compose / make dev)
// + VITE_API_BASE_URL trỏ đúng backend (dashboard/.env).
//
// Chạy:  cd dashboard && bun run test:e2e
//        bun run test:e2e -- --ui          (giao diện)
//        bun run test:e2e -- --headed      (thấy browser)
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
    url: 'http://127.0.0.1:5173/login',
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
    // Mobile dùng Chromium + viewport iPhone (không dùng devices['iPhone 13']
    // vì nó là WebKit — máy chỉ cài chromium)
    {
      name: 'mobile',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 390, height: 844 },
        isMobile: true,
        hasTouch: true,
      },
    },
  ],
})
