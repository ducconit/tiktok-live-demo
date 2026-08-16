<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { Database, Trash2 } from 'lucide-vue-next'
import { cacheApi } from '@/api'
import { errorMessage } from '@/api/client'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const queryClient = useQueryClient()

const { data: info, isLoading } = useQuery({
  queryKey: ['cache-info'],
  queryFn: () => cacheApi.info(),
})

const confirmOpen = ref(false)

const clearMutation = useMutation({
  mutationFn: () => cacheApi.clear(),
  onSuccess: () => {
    toast.success(t('cache.cleared'))
    confirmOpen.value = false
    queryClient.invalidateQueries()
  },
  onError: (err) => toast.error(errorMessage(err)),
})
</script>

<template>
  <div class="space-y-4">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('cache.title') }}</h1>
      <p class="text-sm text-muted-foreground">{{ t('cache.subtitle') }}</p>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <!-- Thông tin store -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">{{ t('dashboard.storeInUse') }}</CardTitle>
          <CardDescription>Cache được cấu hình qua CACHE_STORE (memory | redis)</CardDescription>
        </CardHeader>
        <CardContent>
          <div v-if="isLoading" class="space-y-2">
            <Skeleton v-for="i in 2" :key="i" class="h-8 w-full" />
          </div>
          <dl v-else class="space-y-3 text-sm">
            <div class="flex items-center justify-between">
              <dt class="text-muted-foreground">Store mặc định</dt>
              <dd><Badge variant="success">{{ info?.default }}</Badge></dd>
            </div>
            <div class="flex items-center justify-between">
              <dt class="text-muted-foreground">Prefix key</dt>
              <dd><code class="rounded bg-muted px-2 py-0.5">{{ info?.prefix || t('cache.emptyPrefix') }}</code></dd>
            </div>
            <div class="flex items-center justify-between">
              <dt class="text-muted-foreground">Danh sách store</dt>
              <dd class="flex gap-1">
                <Badge v-for="s in info?.stores ?? []" :key="s" variant="outline">{{ s }}</Badge>
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <!-- Xoá cache -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">{{ t('cache.clearAll') }}</CardTitle>
          <CardDescription>
            Clear mọi key trong mọi store. Dữ liệu sẽ được tải lại từ DB ở request kế tiếp — thường an toàn,
            nhưng có thể gây chậm tức thời (cache stampede).
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div class="flex items-center gap-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3">
            <Database class="h-5 w-5 text-destructive" />
            <p class="flex-1 text-sm text-muted-foreground">
              Xoá cache thường dùng sau khi đổi dữ liệu nền hoặc nghi ngờ dữ liệu cũ bị kẹt.
            </p>
          </div>
          <Button variant="destructive" class="mt-4 gap-2" @click="confirmOpen = true">
            <Trash2 class="h-4 w-4" /> Xoá toàn bộ cache
          </Button>
        </CardContent>
      </Card>
    </div>

    <!-- Confirm dialog -->
    <Dialog :open="confirmOpen" @update:open="(v) => !v && (confirmOpen = false)">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('cache.clearConfirm') }}</DialogTitle>
          <DialogDescription>
            Mọi key trong mọi store (memory/redis) sẽ bị xoá. Hành động này không thể hoàn tác.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="confirmOpen = false">{{ t('common.cancel') }}</Button>
          <Button variant="destructive" :disabled="clearMutation.isPending.value" @click="clearMutation.mutate()">
            {{ clearMutation.isPending.value ? t('common.deleting') : t('cache.clearBtn') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
