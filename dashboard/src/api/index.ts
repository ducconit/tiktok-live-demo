import { api, publicApi } from './client'
import type { ApiKey, ApiKeyCreated, CacheInfo, DynamicConfig, Envelope, Paginated, Permission, Role, Stats, Tokens, User } from './types'

// ---- Auth ----
export const authApi = {
  login: async (email: string, password: string) => {
    const res = await api.post<Envelope<Tokens>>('/auth/login', { email, password })
    return res.data.data!
  },
  refresh: async (refresh_token: string) => {
    const res = await api.post<Envelope<Tokens>>('/auth/refresh', { refresh_token })
    return res.data.data!
  },
  logout: (refresh_token: string) => api.post('/auth/logout', { refresh_token }),
  me: async () => {
    const res = await api.get<Envelope<User>>('/auth/me')
    return res.data.data!
  },
}

// ---- Tài khoản của chính mình (namespace /v1/public) ----
export const accountApi = {
  me: async () => {
    const res = await publicApi.get<Envelope<User>>('/me')
    return res.data.data!
  },
  updateMe: async (body: { full_name: string }) => {
    const res = await publicApi.put<Envelope<User>>('/me', body)
    return res.data.data!
  },
  changePassword: async (body: { current_password: string; new_password: string }) => {
    const res = await publicApi.post<Envelope<{ message: string }>>('/me/change-password', body)
    return res.data.data
  },
  uploadAvatar: async (file: File) => {
    const form = new FormData()
    form.append('avatar', file)
    const res = await publicApi.post<Envelope<{ avatar_url: string }>>('/me/avatar', form)
    return res.data.data!
  },
}

// ---- Users ----
export const usersApi = {
  list: async (params: { page: number; page_size: number; q?: string; is_active?: boolean }) => {
    const res = await api.get<Envelope<User[]>>('/users', { params })
    return { items: res.data.data!, meta: res.data.meta! } as Paginated<User>
  },
  create: (body: { email: string; password: string; full_name: string }) =>
    api.post<Envelope<User>>('/users', body),
  update: (id: string, body: { full_name: string; is_active: boolean }) =>
    api.put<Envelope<User>>(`/users/${id}`, body),
  remove: (id: string) => api.delete(`/users/${id}`),
  changePassword: (id: string, new_password: string) =>
    api.post(`/users/${id}/change-password`, { new_password }),
}

// ---- RBAC ----
export const rbacApi = {
  roles: async () => {
    const res = await api.get<Envelope<Role[]>>('/roles')
    return res.data.data!
  },
  createRole: (body: { slug: string; name: string; description: string }) =>
    api.post<Envelope<Role>>('/roles', body),
  updateRole: (id: string, body: { slug: string; name: string; description: string }) =>
    api.put<Envelope<Role>>(`/roles/${id}`, body),
  deleteRole: (id: string) => api.delete(`/roles/${id}`),
  permissions: async () => {
    const res = await api.get<Envelope<Permission[]>>('/permissions')
    return res.data.data!
  },
  rolePermissions: async (roleId: string) => {
    const res = await api.get<Envelope<Permission[]>>(`/roles/${roleId}/permissions`)
    return res.data.data!
  },
  setRolePermissions: (roleId: string, permission_ids: string[]) =>
    api.put(`/roles/${roleId}/permissions`, { permission_ids }),
  assignUserRole: (userId: string, role_id: string) =>
    api.post(`/users/${userId}/roles`, { role_id }),
  removeUserRole: (userId: string, roleId: string) => api.delete(`/users/${userId}/roles/${roleId}`),
}

// ---- Stats ----
export const statsApi = {
  get: async () => {
    const res = await api.get<Envelope<Stats>>('/stats')
    return res.data.data!
  },
}

// ---- API keys (integrations) ----
export const apiKeysApi = {
  list: async (params: { page: number; page_size: number; q?: string }) => {
    const res = await api.get<Envelope<ApiKey[]>>('/api-keys', { params })
    return { items: res.data.data!, meta: res.data.meta! } as Paginated<ApiKey>
  },
  create: (body: { name: string; scopes?: string[]; expires_at?: string | null }) =>
    api.post<Envelope<ApiKeyCreated>>('/api-keys', body),
  update: (id: string, body: { name?: string; scopes?: string[]; expires_at?: string | null; is_active?: boolean }) =>
    api.put<Envelope<ApiKey>>(`/api-keys/${id}`, body),
  revoke: (id: string) => api.delete(`/api-keys/${id}`),
  rotate: (id: string) => api.post<Envelope<ApiKeyCreated>>(`/api-keys/${id}/rotate`),
}

// ---- Cache ----
export const cacheApi = {
  info: async () => {
    const res = await api.get<Envelope<CacheInfo>>('/cache')
    return res.data.data!
  },
  clear: () => api.delete('/cache'),
}

// ---- Remote config (dynamic) ----
export const configApi = {
  dynamic: async () => {
    const res = await api.get<Envelope<DynamicConfig>>('/config/dynamic')
    return res.data.data!
  },
  set: (key: string, value: unknown) => api.put('/config', { key, value }),
}
