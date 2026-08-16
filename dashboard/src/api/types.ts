// ---- Types khớp chuẩn backend (envelope 4 key + snake_case) ----
// Chuẩn: { code, msg, data, meta } — code "0" = thành công, lỗi = HTTP status string.

export interface Meta {
  limit: number
  page: number
  total: number
}

export interface Envelope<T = unknown> {
  code: string // "0" = OK; lỗi: "400", "401", ..., "500", "503"
  msg: string
  data?: T
  meta?: Meta | Record<string, string> // phân trang | validation field errors
}

export interface User {
  id: string
  email: string
  full_name: string
  avatar_url: string
  is_active: boolean
  last_login_at: string | null
  created_at: string
  updated_at: string
}

export interface Role {
  id: string
  slug: string
  name: string
  description: string
  created_at: string
}

export interface Permission {
  id: string
  slug: string
  name: string
  description: string
}

export interface Tokens {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface Paginated<T> {
  items: T[]
  meta: Meta
}

export interface Stats {
  total_users: number
  active_users: number
  recent_users: number
  signups_by_day: { day: string; count: number }[]
  role_distribution: { role: string; count: number }[]
}

// ---- API keys (integrations) ----
export interface ApiKey {
  id: string
  name: string
  key_prefix: string // vd "gvs_development_ab12..." — KHÔNG bao giờ có key đầy đủ
  scopes: string[]
  expires_at: string | null
  is_active: boolean
  last_used_at: string | null
  created_by: string | null
  revoked_at: string | null
  created_at: string
  updated_at: string
}

export interface ApiKeyCreated {
  key: string // plaintext — chỉ hiện ĐÚNG 1 lần lúc tạo/rotate
  id: string
  name: string
  key_prefix: string
  scopes: string[]
  expires_at: string | null
  is_active: boolean
  last_used_at: string | null
  created_by: string | null
  revoked_at: string | null
  created_at: string
  updated_at: string
}

// ---- Cache ----
export interface CacheInfo {
  stores: string[]
  default: string
  prefix: string
}

// ---- Remote config ----
export interface DynamicConfig {
  [key: string]: unknown
}
