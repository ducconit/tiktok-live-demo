<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { Save, SlidersHorizontal } from 'lucide-vue-next'
import { configApi } from '@/api'
import { errorMessage } from '@/api/client'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

const { t } = useI18n()
const queryClient = useQueryClient()

const { data: cfg, isLoading } = useQuery({
  queryKey: ['config-dynamic'],
  queryFn: () => configApi.dynamic(),
})

// Object.entries → key luôn string (tránh lỗi type của v-for destructure)
const entries = computed(() => Object.entries(cfg.value ?? {}))

// Key cấm đổi runtime (khớp backend blocklistedConfigKeys)
const PROTECTED_PREFIXES = ['database.', 'jwt.', 'redis.', 'minio.', 'mail.', 'cache.']
const PROTECTED_EXACT = ['server.host', 'server.port']

function isProtected(key: string): boolean {
  if (PROTECTED_EXACT.includes(key)) return true
  return PROTECTED_PREFIXES.some((p) => key.startsWith(p))
}

// ---- Sửa 1 key ----
const editing = ref<{ key: string; value: string } | null>(null)
const valueInput = ref('')

const saveMutation = useMutation({
  mutationFn: ({ key, value }: { key: string; value: string }) => {
    // Ép kiểu theo giá trị hiện tại (bool/num giữ nguyên kiểu)
    const current = cfg.value?.[key]
    let parsed: unknown = value
    if (typeof current === 'boolean') parsed = value === 'true' || value === '1'
    else if (typeof current === 'number' && value.trim() !== '') parsed = Number(value)
    return configApi.set(key, parsed)
  },
  onSuccess: (_d, vars) => {
    toast.success(`Đã cập nhật ${vars.key} — đồng bộ mọi instance`)
    editing.value = null
    queryClient.invalidateQueries({ queryKey: ['config-dynamic'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

function startEdit(key: string, value: unknown) {
  editing.value = { key, value: String(value) }
  valueInput.value = String(value)
}

function displayValue(v: unknown): string {
  if (v === null) return 'null'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}
</script>

<template>
  <div class="space-y-4">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('config.title') }}</h1>
      <p class="text-sm text-muted-foreground">
        Remote config — đổi runtime, tự đồng bộ mọi instance qua Redis pub/sub (không cần restart)
      </p>
    </div>

    <Card>
      <CardHeader>
        <CardTitle class="text-base">{{ t('dashboard.recentKeys') }}</CardTitle>
        <CardDescription>
          Keys có prefix <code class="rounded bg-muted px-1">database. jwt. redis. minio. mail. cache.</code> hoặc
          <code class="rounded bg-muted px-1">server.host/port</code> bị chặn đổi runtime.
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-2">
        <Skeleton v-for="i in 4" v-if="isLoading" :key="i" class="h-10 w-full" />

        <div
          v-for="[key, value] in entries"
          :key="key"
          class="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <code class="truncate rounded bg-muted px-2 py-0.5 text-xs font-medium">{{ key }}</code>
              <Badge v-if="isProtected(key)" variant="secondary">{{ t('config.protected') }}</Badge>
              <Badge v-else variant="success">dynamic</Badge>
            </div>
            <div class="mt-1 truncate font-mono text-sm text-muted-foreground">
              {{ displayValue(value) }}
            </div>
          </div>

          <!-- Form sửa inline -->
          <div v-if="editing?.key === key" class="flex items-center gap-2">
            <Input v-model="valueInput" class="w-40 font-mono" @keyup.enter="saveMutation.mutate({ key, value: valueInput })" />
            <Button size="sm" variant="gradient" :disabled="saveMutation.isPending.value" @click="saveMutation.mutate({ key, value: valueInput })">
              <Save class="h-3.5 w-3.5" />
            </Button>
            <Button size="sm" variant="ghost" @click="editing = null">{{ t('common.cancel') }}</Button>
          </div>
          <Button v-else size="sm" variant="outline" :disabled="isProtected(key)" @click="startEdit(key, value)">
            <SlidersHorizontal class="h-3.5 w-3.5" /> Sửa
          </Button>
        </div>

        <p v-if="!isLoading && Object.keys(cfg ?? {}).length === 0" class="py-8 text-center text-sm text-muted-foreground">
          Chưa có dynamic config nào — đổi giá trị qua API hoặc dashboard để tạo key đầu tiên
        </p>
      </CardContent>
    </Card>
  </div>
</template>
