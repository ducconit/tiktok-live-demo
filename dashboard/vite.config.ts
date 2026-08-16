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
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
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
