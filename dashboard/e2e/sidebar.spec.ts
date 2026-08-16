import { test, expect } from '@playwright/test'
import { loginViaApi } from './helpers'

test.describe('Sidebar (shadcn-vue)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
    await page.goto('/')
  })

  test('render đủ 7 nav items (desktop)', async ({ page, isMobile }) => {
    test.skip(isMobile, 'desktop sidebar')
    // Chờ sidebar thực sự render (redirect sau login hoàn tất)
    await expect(page.locator('[data-sidebar="menu"] a').first()).toBeVisible()
    const items = await page
      .locator('[data-sidebar="menu"] a')
      .allTextContents()
    const labels = items.map((t) => t.trim()).filter(Boolean)
    expect(labels).toContain('Tổng quan')
    expect(labels).toContain('Người dùng')
    expect(labels).toContain('Vai trò & Quyền')
    expect(labels).toContain('API Keys')
    expect(labels).toContain('Cấu hình động')
    expect(labels).toContain('Cache')
    expect(labels).toContain('Cài đặt')
  })

  test('collapse qua rail: rộng → icon-only (desktop)', async ({ page, isMobile }) => {
    test.skip(isMobile, 'desktop sidebar')
    const sidebar = page.locator('[data-sidebar="sidebar"]')
    const w0 = (await sidebar.evaluate((el) => getComputedStyle(el).width)).replace('px', '')
    await page.locator('button[data-sidebar="rail"]').click()
    await expect
      .poll(() => sidebar.evaluate((el) => getComputedStyle(el).width))
      .not.toBe(`${w0}px`)
    const w1 = (await sidebar.evaluate((el) => getComputedStyle(el).width)).replace('px', '')
    expect(Number(w1)).toBeLessThan(Number(w0)) // thu nhỏ
  })

  test('navigate: click Users → URL /users + content hiện', async ({ page, isMobile }) => {
    test.skip(isMobile, 'desktop sidebar')
    await page.locator('[data-sidebar="menu"] a', { hasText: 'Người dùng' }).click()
    await expect(page).toHaveURL(/\/users/)
    // Chờ trang users render (danh sách hoặc title)
    await expect(page.locator('main h1, main h2, main [data-slot="card"]').first()).toBeVisible()
  })

  test('mobile: trigger mở sheet drawer + menu đầy đủ', async ({ page, isMobile }) => {
    test.skip(!isMobile, 'mobile sheet')
    await page.locator('[data-sidebar="trigger"]').click()
    const dialog = page.locator('[role="dialog"]')
    await expect(dialog).toBeVisible()
    const labels = (await dialog.locator('a').allTextContents())
      .map((t) => t.trim())
      .filter(Boolean)
    expect(labels).toContain('Tổng quan')
    expect(labels).toContain('API Keys')
  })

  test('theme toggle trong sidebar footer', async ({ page, isMobile }) => {
    test.skip(isMobile, 'desktop sidebar')
    const footer = page.locator('[data-sidebar="footer"]')
    await expect(footer).toBeVisible()
    await footer.locator('button').first().click() // theme toggle
    await expect(page.locator('html')).toHaveClass(/dark|light/)
  })
})
