import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// loadEnv: Vite config chạy ở Node context — KHÔNG tự đọc .env như client code.
// Đọc dashboard/.env (VITE_API_BASE_URL) — env dashboard đặt trong dashboard/.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  return {
    plugins: [vue(), tailwindcss()],
    resolve: {
      // Monorepo: ép dùng chung 1 bản dep (tránh 2 copy vue/reka-ui…)
      dedupe: ['vue', 'reka-ui', '@vueuse/core', 'class-variance-authority', 'clsx', 'tailwind-merge', 'vee-validate', 'vue-input-otp', 'vue-sonner', 'embla-carousel-vue', '@unovis/vue', '@lucide/vue', '@internationalized/date', 'lucide-vue-next'],
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      // Monorepo: cho phép vite dev đọc source trong packages/ (workspace)
      fs: {
        allow: ['.', '../packages'],
      },
      port: 5173,
      host: true,
      // Proxy API sang backend (dev, tránh CORS rườm rà)
      proxy: {
        '/api': {
          target: env.VITE_API_BASE_URL || 'http://localhost:3330',
          changeOrigin: true,
        },
      },
    },
  }
})
