import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
    dedupe: ['vue', 'reka-ui', '@vueuse/core', 'class-variance-authority', 'clsx', 'tailwind-merge', 'vee-validate', 'vue-input-otp', 'vue-sonner', 'embla-carousel-vue', '@unovis/vue', '@lucide/vue', '@internationalized/date', 'lucide-vue-next'],
  },
  test: {
    environment: "jsdom",
    globals: true,
    exclude: ["e2e/**", "node_modules/**", "dist/**"],
    // Workspace packages chứa source .ts/.vue — bắt vitest transform (không externalize)
    server: {
      deps: {
        inline: ["@tiktok-live/ui", "@tiktok-live/api", "@tiktok-live/styles"],
      },
    },
  },
});
