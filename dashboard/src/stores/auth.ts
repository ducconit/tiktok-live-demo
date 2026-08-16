import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { authApi } from '@/api'
import { clearTokens, getAccessToken, getRefreshToken, setTokens } from '@/api/client'
import type { User } from '@/api/types'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(getAccessToken())
  const refreshToken = ref<string | null>(getRefreshToken())
  const user = ref<User | null>(null)
  const loading = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value)
  const isSuperAdmin = computed(() => user.value?.email === 'admin@example.com') // placeholder; roles trong claims

  async function login(email: string, password: string) {
    loading.value = true
    try {
      const tokens = await authApi.login(email, password)
      setTokens(tokens.access_token, tokens.refresh_token)
      accessToken.value = tokens.access_token
      refreshToken.value = tokens.refresh_token
      await fetchMe()
    } finally {
      loading.value = false
    }
  }

  async function fetchMe() {
    if (!accessToken.value) return
    user.value = await authApi.me()
  }

  function logout() {
    const rt = refreshToken.value
    clearTokens()
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    if (rt) {
      authApi.logout(rt).catch(() => {})
    }
  }

  return { accessToken, refreshToken, user, loading, isAuthenticated, isSuperAdmin, login, fetchMe, logout }
})
