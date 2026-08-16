import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { Toaster } from 'vue-sonner'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import { useAuthStore } from './stores/auth'
import './style.css'

// Đăng xuất toàn cục khi refresh token hết hạn (phát từ response interceptor)
window.addEventListener('auth:logout', () => {
  const auth = useAuthStore()
  auth.logout()
  router.push({ name: 'login' })
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(i18n)
app.use(VueQueryPlugin)
app.component('Toaster', Toaster)
app.mount('#app')
