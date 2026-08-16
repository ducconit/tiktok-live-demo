import { createApiClient, errorMessage, type Envelope } from "@tiktok-live/api"
import { LOCALE_KEY } from '../i18n'
import i18n from '../i18n'

const TOKEN_KEY = 'gvs_access'
const REFRESH_KEY = 'gvs_refresh'

export function getAccessToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}
export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY)
}
export function setTokens(access: string, refresh: string) {
  localStorage.setItem(TOKEN_KEY, access)
  localStorage.setItem(REFRESH_KEY, refresh)
}
export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_KEY)
}

// Auth chung cho cả 2 client (admin + public) — 401 → refresh → retry → logout.
const auth = {
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
  refreshEndpoint: '/auth/refresh',
  isAuthUrl: (url: string) => url.includes('/auth/login') || url.includes('/auth/refresh'),
  onUnauthorized: () => window.dispatchEvent(new CustomEvent('auth:logout')),
  locale: () => (localStorage.getItem(LOCALE_KEY) === 'en' ? 'en' : 'vi'),
}

// Dashboard dùng namespace admin: /api/v1/admin/auth/login, /api/v1/admin/users...
export const api = createApiClient({
  baseURL: `${import.meta.env.VITE_API_BASE_URL || '/api'}/v1/admin`,
  timeout: 15_000,
  authProvider: auth,
})

// Namespace public (tài khoản của chính mình): /me, /me/avatar, /me/change-password
export const publicApi = createApiClient({
  baseURL: `${import.meta.env.VITE_API_BASE_URL || '/api'}/v1/public`,
  timeout: 15_000,
  authProvider: auth,
})

// Helper lỗi chuẩn envelope 4 key (code/msg) — giữ i18n fallback của dashboard
export { errorMessage }
export const dashboardErrorMessage = (err: unknown) =>
  errorMessage(err, i18n.global.t('errors.generic'))
export type { Envelope }
