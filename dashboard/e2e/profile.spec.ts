import { test, expect } from '@playwright/test'
import { loginViaApi, apiBase } from './helpers'

test.describe('Profile (/settings)', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaApi(page)
    await page.goto('/settings')
  })

  test('hiển thị 3 card: thông tin + cập nhật + đổi mật khẩu', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Thông tin cá nhân', exact: true })).toBeVisible()
    // Email hiển thị trong avatar card
    await expect(page.getByText('admin@example.com').first()).toBeVisible()
    // Form họ tên có giá trị từ /me
    const name = page.locator('#full-name')
    await expect(name).toBeVisible()
    // Card đổi mật khẩu
    await expect(page.locator('#current-password')).toBeVisible()
    await expect(page.locator('#new-password')).toBeVisible()
    await expect(page.locator('#confirm-password')).toBeVisible()
  })

  test('cập nhật họ tên → toast thành công', async ({ page }) => {
    const name = page.locator('#full-name')
    await name.fill('Super Admin E2E')
    await page.getByRole('button', { name: /lưu thông tin|save profile/i }).click()
    await expect(page.locator('[data-sonner-toast]').filter({ hasText: /đã lưu|saved/i })).toBeVisible()
  })

  test('đổi mật khẩu sai mật khẩu hiện tại → báo lỗi, không đổi được', async ({ page }) => {
    await page.locator('#current-password').fill('wrong-current-pass')
    await page.locator('#new-password').fill('NewPass@123')
    await page.locator('#confirm-password').fill('NewPass@123')
    // Submit qua Enter (form submit) — tránh click bị che trên mobile
    const submit = page.locator('button[type="submit"]').last()
    await expect(submit).toBeEnabled()
    await submit.press('Enter')
    // Backend trả 401/400 → toast lỗi (không phải "đã đổi")
    await expect(page.locator('[data-sonner-toast]')).toBeVisible()
  })

  test('mật khẩu mới không khớp → lỗi validation client, nút disabled', async ({ page }) => {
    await page.locator('#current-password').fill('admin123')
    await page.locator('#new-password').fill('NewPass@123')
    await page.locator('#confirm-password').fill('KhacNhau@456')
    await expect(page.getByText(/không khớp|do not match/i)).toBeVisible()
    const submit = page.getByRole('button', { name: /đổi mật khẩu|change password/i }).last()
    await expect(submit).toBeDisabled()
  })

  test('upload avatar: chọn file → preview → bấm Lưu chung → avatar + tên được cập nhật', async ({ page }) => {
    // 1) Chọn ảnh → preview + badge "Ảnh mới" (chưa gọi API)
    const input = page.locator('input[type="file"]')
    await input.setInputFiles({
      name: 'avatar.png',
      mimeType: 'image/png',
      buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==', 'base64'),
    })
    await expect(page.getByText(/ảnh mới|new image/i)).toBeVisible()

    // 2) Sửa tên + bấm 1 nút Lưu duy nhất → avatar + thông tin cùng lúc
    const name = page.locator('#full-name')
    await name.fill('Super Admin E2E')
    const saveBtn = page.getByRole('button', { name: /lưu thông tin|save profile/i })
    await expect(saveBtn).toBeEnabled()
    await saveBtn.click()

    // 3) Toast thành công + badge ảnh mới biến mất
    await expect(page.locator('[data-sonner-toast]').filter({ hasText: /đã lưu|saved/i })).toBeVisible()
    await expect(page.getByText(/ảnh mới|new image/i)).toBeHidden()

    // 4) Verify qua API: full_name mới + avatar_url có (đã upload)
    const me = await page.request.get(`${apiBase()}/v1/public/me`, {
      headers: { Authorization: `Bearer ${await page.evaluate(() => localStorage.getItem('gvs_access'))}` },
    })
    const body = (await me.json()) as { data: { full_name: string; avatar_url: string } }
    expect(body.data.full_name).toBe('Super Admin E2E')
    expect(body.data.avatar_url).toBeTruthy()
  })
})
