# Dashboard Patterns — Vue 3 + shadcn-vue + Tailwind v4

## Cấu trúc

```
src/api/           # client.ts (axios + auto refresh token) + index.ts (typed API per domain)
src/stores/        # Pinia setup stores (auth, theme)
src/router/        # routes + guard (auth + permission qua meta.perm)
src/views/<domain>/  # page + component dialog
src/components/ui/ # shadcn-style UI kit (reka-ui v2 — Root hậu tố)
src/components/layout/
src/lib/           # utils (cn, format), app.ts (APP_TITLE)
```

## Quy tắc

1. **API mới**: thêm vào `src/api/index.ts` — function typed, gọi `api.get/post/...` từ client, trả `res.data.data` (unwrapped envelope)
2. **Dữ liệu page**: dùng TanStack Vue Query — `useQuery({ queryKey: computed(...), queryFn })`; mutation dùng `useMutation` + `queryClient.invalidateQueries`
3. **Form**: dialog component riêng (`XxxFormDialog.vue`) nhận `v-model:open` + `submitting` prop, emit `submit`; validate thủ công + toast lỗi
4. **UI components**: luôn dùng từ `@/components/ui/*` (Button, Input, Card, Dialog, Table, Badge, Select, Checkbox, Switch, Avatar, Skeleton...) — KHÔNG tự viết HTML thô
5. **Design system Studio Dark**: dark-first; class `bg-background`, `text-foreground`, `bg-primary`, `text-muted-foreground`, `border-input`... (tokens trong `src/assets/design-tokens.css`)
6. **Naming**: file kebab-case; components PascalCase; function camelCase; type PascalCase
7. **Toast**: `toast.success/error` từ vue-sonner; lỗi API qua `errorMessage(err)` từ client
8. **Type**: định nghĩa trong `src/api/types.ts` khớp backend envelope + snake_case JSON
9. Build: `bun run typecheck` (vue-tsc) + `bun run build`
