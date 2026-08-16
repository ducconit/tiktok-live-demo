import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import type { Envelope } from './types'
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

export const api = axios.create({
  // Dashboard dùng namespace admin: /api/v1/admin/auth/login, /api/v1/admin/users...
  baseURL: `${import.meta.env.VITE_API_BASE_URL || '/api'}/v1/admin`,
  timeout: 15_000,
})

// Namespace public (tài khoản của chính mình): /me, /me/avatar, /me/change-password
export const publicApi = axios.create({
  baseURL: `${import.meta.env.VITE_API_BASE_URL || '/api'}/v1/public`,
  timeout: 15_000,
})

// ---- Request: gắn Bearer + Accept-Language (backend trả msg đúng ngôn ngữ) ----
function attachAuth(instance: typeof api) {
  instance.interceptors.request.use((cfg) => {
    const token = getAccessToken()
    if (token) cfg.headers.Authorization = `Bearer ${token}`
    cfg.headers['Accept-Language'] = localStorage.getItem(LOCALE_KEY) === 'en' ? 'en' : 'vi'
    return cfg
  })
}
attachAuth(api)
attachAuth(publicApi)

// ---- Response: 401 → refresh 1 lần → retry; fail → logout ----
// Áp cho cả api (admin) và publicApi (tài khoản) — dùng chung refresh state.
let isRefreshing = false
let waiters: ((ok: boolean) => void)[] = []

function onRefreshed(ok: boolean) {
  waiters.forEach((w) => w(ok))
  waiters = []
}

async function tryRefresh(): Promise<boolean> {
  const refresh = getRefreshToken()
  if (!refresh) return false
  try {
    const res = await axios.post<Envelope<{ access_token: string; refresh_token: string }>>(
      `${api.defaults.baseURL}/auth/refresh`,
      { refresh_token: refresh },
    )
    const t = res.data.data
    if (!t) return false
    setTokens(t.access_token, t.refresh_token)
    return true
  } catch {
    clearTokens()
    return false
  }
}

function attachRefresh(instance: typeof api) {
  instance.interceptors.response.use(
    (res) => res,
    async (error: AxiosError<Envelope>) => {
      const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
      const isAuthCall = original.url?.includes('/auth/login') || original.url?.includes('/auth/refresh')

      if (error.response?.status === 401 && original && !original._retry && !isAuthCall) {
        if (isRefreshing) {
          return new Promise((resolve) => {
            waiters.push((ok) => {
              if (ok) {
                original._retry = true
                original.headers.Authorization = `Bearer ${getAccessToken()}`
                resolve(instance(original))
              } else {
                resolve(Promise.reject(error))
              }
            })
          })
        }

        original._retry = true
        isRefreshing = true
        const ok = await tryRefresh()
        onRefreshed(ok)
        isRefreshing = false

        if (ok) {
          original.headers.Authorization = `Bearer ${getAccessToken()}`
          return instance(original)
        }
        // refresh fail → đá về login
        window.dispatchEvent(new CustomEvent('auth:logout'))
        return Promise.reject(error)
      }
      return Promise.reject(error)
    },
  )
}
attachRefresh(api)
attachRefresh(publicApi)

// ---- Helper lỗi chuẩn envelope 4 key (code/msg) ----
export function errorMessage(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const env = err.response?.data as Envelope | undefined
    if (env?.msg) return env.msg
    // validation 422: meta = { field: message } — lấy message đầu tiên
    if (env?.code === '422' && env.meta && typeof env.meta === 'object') {
      const first = Object.values(env.meta as Record<string, string>)[0]
      if (first) return first
    }
    if (err.code === 'ECONNABORTED') return i18n.global.t('errors.timeout')
  }
  return i18n.global.t('errors.generic')
}
