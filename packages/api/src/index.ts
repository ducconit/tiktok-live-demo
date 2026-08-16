import axios, { AxiosError, type AxiosRequestConfig, type InternalAxiosRequestConfig } from 'axios'

// Envelope chuẩn backend: { code, msg, data, meta } — code "0" = thành công.
export interface Envelope<T = unknown> {
  code: string
  msg: string
  data: T
  meta: Record<string, unknown> | null
}

// Auth tuỳ chọn — app nào có JWT (dashboard) truyền vào, app public (frontend) bỏ qua.
export interface ApiAuth {
  getAccessToken: () => string | null
  getRefreshToken: () => string | null
  setTokens: (access: string, refresh: string) => void
  clearTokens: () => void
  refreshEndpoint: string // vd "/auth/refresh" (relative — gọi trên cùng baseURL)
  isAuthUrl?: (url: string) => boolean // url không refresh khi 401 (login/refresh)
  onUnauthorized?: () => void // refresh fail → đá về login
  locale?: () => string // Accept-Language
}

export interface CreateApiClientOptions extends AxiosRequestConfig {
  // JWT auth provider (dashboard); bỏ qua với client public (frontend)
  authProvider?: ApiAuth
}

/**
 * createApiClient — axios instance dùng chung:
 * - Không auth → instance thuần (frontend end-user, route public).
 * - Có auth  → request gắn Bearer + Accept-Language; 401 → refresh 1 lần → retry;
 *              refresh fail → onUnauthorized().
 */
export function createApiClient(options: CreateApiClientOptions) {
  const client = axios.create(options)
  const auth = options.authProvider
  if (!auth) return client

  client.interceptors.request.use((cfg) => {
    const token = auth.getAccessToken()
    if (token) cfg.headers.Authorization = `Bearer ${token}`
    if (auth.locale) cfg.headers['Accept-Language'] = auth.locale()
    return cfg
  })

  // ---- 401 → refresh 1 lần → retry; fail → onUnauthorized ----
  let isRefreshing = false
  let waiters: ((ok: boolean) => void)[] = []

  const onRefreshed = (ok: boolean) => {
    waiters.forEach((w) => w(ok))
    waiters = []
  }

  const tryRefresh = async (): Promise<boolean> => {
    const refresh = auth.getRefreshToken()
    if (!refresh) return false
    try {
      const res = await axios.post<Envelope<{ access_token: string; refresh_token: string }>>(
        `${client.defaults.baseURL ?? ''}${auth.refreshEndpoint}`,
        { refresh_token: refresh },
      )
      const t = res.data.data
      if (!t) return false
      auth.setTokens(t.access_token, t.refresh_token)
      return true
    } catch {
      auth.clearTokens()
      return false
    }
  }

  client.interceptors.response.use(
    (res) => res,
    async (error: AxiosError<Envelope>) => {
      const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
      const isAuthCall = original?.url
        ? auth.isAuthUrl
          ? auth.isAuthUrl(original.url)
          : original.url.includes('/auth/')
        : true

      if (error.response?.status === 401 && original && !original._retry && !isAuthCall) {
        if (isRefreshing) {
          return new Promise((resolve) => {
            waiters.push((ok) => {
              if (ok) {
                original._retry = true
                original.headers.Authorization = `Bearer ${auth.getAccessToken()}`
                resolve(client(original))
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
          original.headers.Authorization = `Bearer ${auth.getAccessToken()}`
          return client(original)
        }
        auth.onUnauthorized?.()
        return Promise.reject(error)
      }
      return Promise.reject(error)
    },
  )
  return client
}

// Lấy message lỗi từ envelope (code != 0) — fallback khi không parse được.
export function errorMessage(err: unknown, fallback = 'Lỗi hệ thống'): string {
  if (axios.isAxiosError(err)) {
    const env = err.response?.data as Envelope | undefined
    if (env?.msg) return env.msg
    // validation 422: meta = { field: message } — lấy message đầu tiên
    if (env?.code === '422' && env.meta && typeof env.meta === 'object') {
      const first = Object.values(env.meta as Record<string, string>)[0]
      if (first) return first
    }
    if (err.code === 'ECONNABORTED') return 'Kết nối quá hạn'
  }
  return fallback
}
