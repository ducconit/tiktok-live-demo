import { test, expect } from '@playwright/test'
import { ADMIN, login, loginViaApi } from './helpers'

test.describe('Auth', () => {
  test('trang login render + title', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('form')).toBeVisible()
    await expect(page.locator('#email')).toHaveValue(ADMIN.email) // default điền sẵn
  })

  test('đăng nhập đúng → vào dashboard', async ({ page, isMobile }) => {
    await login(page)
    if (isMobile) {
      // Mobile: sidebar offcanvas ẩn — check header + trigger mở sheet
      await expect(page.locator('header')).toBeVisible()
      await expect(page.locator('[data-sidebar="trigger"]')).toBeVisible()
    } else {
      await expect(page.locator('[data-sidebar="menu"] a').first()).toBeVisible()
    }
  })

  test('mật khẩu sai → toast lỗi, không vào được', async ({ page }) => {
    await page.goto('/login')
    await page.fill('#password', 'wrong-password')
    await page.click('form button')
    await expect(page.locator('[data-sonner-toast]')).toBeVisible()
    await expect(page).toHaveURL(/\/login/)
  })

  test('user menu ở header: avatar + logout', async ({ page, isMobile }) => {
    test.skip(isMobile, 'header user menu hiện trên desktop')
    await login(page)
    // Avatar dropdown trigger (data-slot) — KHÔNG dùng button.last() vì
    // dropdown content render trong header (Cài đặt/Đăng xuất) nằm sau nó
    const trigger = page.locator('[data-slot="dropdown-menu-trigger"]')
    await expect(trigger).toBeVisible()
    await trigger.click()
    await expect(page.getByText('Đăng xuất')).toBeVisible()
    await page.getByText('Đăng xuất').click()
    await expect(page).toHaveURL(/\/login/)
  })

  test('loginViaApi: access token lưu vào localStorage', async ({ page, isMobile }) => {
    await loginViaApi(page)
    await page.goto('/')
    if (isMobile) {
      await expect(page.locator('[data-sidebar="trigger"]')).toBeVisible()
    } else {
      await expect(page.locator('[data-sidebar="menu"]').first()).toBeVisible()
    }
  })
})
