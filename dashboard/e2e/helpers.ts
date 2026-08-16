import { expect, type Page } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'

// Tài khoản admin mặc định của skeleton (make admin / devtool seed).
export const ADMIN = {
  email: 'admin@example.com',
  password: 'admin123',
}

// API backend thật — đọc VITE_API_BASE_URL từ dashboard/.env (browser gọi thẳng
// backend, KHÔNG qua vite proxy — nên e2e cũng gọi thẳng).
export function apiBase(): string {
  try {
    const env = fs.readFileSync(path.join(process.cwd(), '.env'), 'utf8')
    const m = env.match(/^VITE_API_BASE_URL=(.+)$/m)
    if (m) return m[1].trim()
  } catch {
    /* fallback */
  }
  return 'http://127.0.0.1:3330/api'
}

// Đăng nhập qua UI (trang login thật) — dùng trong test cần auth.
export async function login(page: Page) {
  await page.goto('/login')
  await page.fill('#password', ADMIN.password)
  await page.click('form button')
  // Rời khỏi trang login (không phải vẫn ở /login)
  await expect(page).not.toHaveURL(/\/login$/, { timeout: 15_000 })
}

// Đăng nhập nhanh qua API (set token trực tiếp vào localStorage) —
// tránh lệ thuộc UI login khi test trang khác.
export async function loginViaApi(page: Page) {
  const res = await page.request.post(`${apiBase()}/v1/admin/auth/login`, {
    data: { email: ADMIN.email, password: ADMIN.password },
  })
  expect(res.ok()).toBeTruthy()
  const { data } = (await res.json()) as {
    data: { access_token: string; refresh_token: string }
  }
  await page.addInitScript(
    ([at, rt]) => {
      localStorage.setItem('gvs_access', at)
      localStorage.setItem('gvs_refresh', rt)
    },
    [data.access_token, data.refresh_token],
  )
}
