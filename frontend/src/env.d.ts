/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_SOCKUDO_HOST?: string
  readonly VITE_SOCKUDO_PORT?: string
  readonly VITE_SOCKUDO_KEY?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
