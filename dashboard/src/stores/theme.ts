import { defineStore } from 'pinia'
import { ref } from 'vue'

const THEME_KEY = 'gvs_theme'

export type Theme = 'dark' | 'light'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>((localStorage.getItem(THEME_KEY) as Theme) || 'dark')

  function apply(t: Theme) {
    theme.value = t
    localStorage.setItem(THEME_KEY, t)
    document.documentElement.classList.toggle('dark', t === 'dark')
    document.documentElement.classList.toggle('light', t === 'light')
  }

  function toggle() {
    apply(theme.value === 'dark' ? 'light' : 'dark')
  }

  // Áp dụng ngay khi store khởi tạo
  apply(theme.value)

  return { theme, apply, toggle }
})
