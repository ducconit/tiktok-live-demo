<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { Copy, KeyRound, Plus, RefreshCw, Search, Trash2 } from 'lucide-vue-next'
import { apiKeysApi } from '@/api'
import type { ApiKey, ApiKeyCreated } from '@/api/types'
import { errorMessage } from '@/api/client'
import { formatDateTime } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const { t } = useI18n()
const queryClient = useQueryClient()

const page = ref(1)
const q = ref('')

const { data, isLoading } = useQuery({
  queryKey: computed(() => ['api-keys', page.value, q.value]),
  queryFn: () => apiKeysApi.list({ page: page.value, page_size: 10, q: q.value || undefined }),
})

const total = computed(() => data.value?.meta.total ?? 0)
const totalPages = computed(() => {
  const meta = data.value?.meta
  if (!meta) return 1
  return Math.ceil(meta.total / meta.limit)
})

// ---- Create ----
const createOpen = ref(false)
const createForm = reactive({ name: '', scopes: '', expires_at: '' })
const createdKey = ref<ApiKeyCreated | null>(null)

const createMutation = useMutation({
  mutationFn: () =>
    apiKeysApi.create({
      name: createForm.name,
      scopes: createForm.scopes
        ? createForm.scopes.split(',').map((s) => s.trim()).filter(Boolean)
        : undefined,
      expires_at: createForm.expires_at ? new Date(createForm.expires_at).toISOString() : null,
    }),
  onSuccess: (res) => {
    createdKey.value = res.data.data!
    toast.success(t('apiKeys.created'))
    queryClient.invalidateQueries({ queryKey: ['api-keys'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

function closeCreate() {
  createOpen.value = false
  createdKey.value = null
  createForm.name = ''
  createForm.scopes = ''
  createForm.expires_at = ''
}

function copyKey() {
  const k = createdKey.value?.key ?? rotateResult.value?.key
  if (!k) return
  navigator.clipboard.writeText(k)
  toast.success(t('apiKeys.copied'))
}

// ---- Rotate / Revoke / Toggle ----
const rotating = ref<ApiKey | null>(null)
const rotateResult = ref<ApiKeyCreated | null>(null)

const rotateMutation = useMutation({
  mutationFn: (id: string) => apiKeysApi.rotate(id),
  onSuccess: (res) => {
    rotateResult.value = res.data.data!
    queryClient.invalidateQueries({ queryKey: ['api-keys'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

function confirmRotate() {
  if (rotating.value) {
    rotateMutation.mutate(rotating.value.id)
    rotating.value = null
  }
}

const revoking = ref<ApiKey | null>(null)
const revokeMutation = useMutation({
  mutationFn: (id: string) => apiKeysApi.revoke(id),
  onSuccess: () => {
    toast.success(t('apiKeys.revoked'))
    queryClient.invalidateQueries({ queryKey: ['api-keys'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

function confirmRevoke() {
  if (revoking.value) {
    revokeMutation.mutate(revoking.value.id)
    revoking.value = null
  }
}

const toggleMutation = useMutation({
  mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
    apiKeysApi.update(id, { is_active }),
  onError: (err) => toast.error(errorMessage(err)),
})

function onToggle(k: ApiKey, val: boolean) {
  toggleMutation.mutate({ id: k.id, is_active: val })
}

function keyStatus(k: ApiKey): { label: string; variant: 'success' | 'destructive' | 'secondary' } {
  if (k.revoked_at) return { label: t('apiKeys.revokedBadge'), variant: 'destructive' }
  if (!k.is_active) return { label: t('apiKeys.disabledBadge'), variant: 'secondary' }
  if (k.expires_at && new Date(k.expires_at) < new Date()) return { label: t('apiKeys.expiredBadge'), variant: 'destructive' }
  return { label: t('common.active'), variant: 'success' }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">{{ t('apiKeys.title') }}</h1>
        <p class="text-sm text-muted-foreground">
          {{ total }} key · dùng cho namespace <code class="rounded bg-muted px-1">/integrations</code>
        </p>
      </div>
      <Button variant="gradient" @click="createOpen = true">
        <Plus class="h-4 w-4" /> Tạo API key
      </Button>
    </div>

    <!-- Toolbar -->
    <div class="relative max-w-sm">
      <Search class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
      <Input
        v-model="q"
        placeholder="Tìm theo tên..."
        class="pl-8"
        @keyup.enter="page = 1"
      />
    </div>

    <!-- Table -->
    <div class="rounded-xl border bg-card">
      <Table>
        <TableHeader>
          <TableRow class="hover:bg-transparent">
            <TableHead>{{ t('common.name') }}</TableHead>
            <TableHead>{{ t('apiKeys.key') }}</TableHead>
            <TableHead>{{ t('apiKeys.scopes') }}</TableHead>
            <TableHead>{{ t('common.status') }}</TableHead>
            <TableHead>{{ t('apiKeys.lastUsed') }}</TableHead>
            <TableHead>{{ t('common.createdAt') }}</TableHead>
            <TableHead class="w-36 text-right">{{ t('common.actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-if="isLoading">
            <TableCell colspan="7" class="space-y-2">
              <Skeleton v-for="i in 4" :key="i" class="h-9 w-full" />
            </TableCell>
          </TableRow>
          <TableRow v-else-if="data?.items.length === 0">
            <TableCell colspan="7" class="py-10 text-center text-muted-foreground">
              Chưa có API key nào
            </TableCell>
          </TableRow>
          <TableRow v-for="k in data?.items ?? []" v-else :key="k.id">
            <TableCell>
              <div class="flex items-center gap-2">
                <KeyRound class="h-4 w-4 text-muted-foreground" />
                <span class="font-medium">{{ k.name }}</span>
              </div>
            </TableCell>
            <TableCell>
              <code class="rounded bg-muted px-2 py-0.5 text-xs">{{ k.key_prefix }}...</code>
            </TableCell>
            <TableCell>
              <div class="flex flex-wrap gap-1">
                <Badge v-if="k.scopes.length === 0" variant="secondary">{{ t('apiKeys.unlimited') }}</Badge>
                <Badge v-for="s in k.scopes" :key="s" variant="outline">{{ s }}</Badge>
              </div>
            </TableCell>
            <TableCell>
              <Badge :variant="keyStatus(k).variant">{{ keyStatus(k).label }}</Badge>
            </TableCell>
            <TableCell class="text-sm text-muted-foreground">{{ formatDateTime(k.last_used_at ?? '') }}</TableCell>
            <TableCell class="text-sm text-muted-foreground">{{ formatDateTime(k.created_at) }}</TableCell>
            <TableCell>
              <div class="flex items-center justify-end gap-2">
                <Switch
                  :checked="k.is_active && !k.revoked_at"
                  :disabled="!!k.revoked_at"
                  @update:checked="(v: boolean | undefined) => onToggle(k, !!v)"
                />
                <Button
                  variant="ghost"
                  size="icon"
                  title="Xoay key (key cũ chết ngay)"
                  @click="rotating = k"
                >
                  <RefreshCw class="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" title="Thu hồi" @click="revoking = k">
                  <Trash2 class="h-4 w-4 text-destructive" />
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <!-- Pagination -->
    <div class="flex items-center justify-between text-sm text-muted-foreground">
      <span>Trang {{ page }} / {{ totalPages }}</span>
      <div class="flex gap-2">
        <Button variant="outline" size="sm" :disabled="page <= 1" @click="page--">{{ t('common.previous') }}</Button>
        <Button variant="outline" size="sm" :disabled="page >= totalPages" @click="page++">{{ t('common.next') }}</Button>
      </div>
    </div>

    <!-- Create dialog -->
    <Dialog :open="createOpen" @update:open="(v) => !v && closeCreate()">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('apiKeys.createDialogTitle') }}</DialogTitle>
          <DialogDescription>
            Key sẽ chỉ được hiển thị ĐÚNG 1 lần — hãy copy và lưu ở nơi an toàn.
          </DialogDescription>
        </DialogHeader>

        <!-- Step 2: hiện key vừa tạo -->
        <div v-if="createdKey" class="space-y-3">
          <div class="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-3">
            <div class="mb-1 flex items-center justify-between">
              <span class="text-xs font-medium text-emerald-600">API key mới của bạn</span>
              <Button variant="ghost" size="sm" class="h-7 gap-1" @click="copyKey">
                <Copy class="h-3.5 w-3.5" /> Copy
              </Button>
            </div>
            <code class="block break-all rounded bg-background p-2 text-xs">{{ createdKey.key }}</code>
            <p class="mt-2 text-xs text-muted-foreground">
              Key này sẽ không hiển thị lại. Nếu mất, bạn phải xoay key.
            </p>
          </div>
          <DialogFooter>
            <Button variant="gradient" @click="closeCreate">{{ t('common.done') }}</Button>
          </DialogFooter>
        </div>

        <!-- Step 1: form tạo -->
        <form v-else class="space-y-4" @submit.prevent="createMutation.mutate()">
          <div class="space-y-2">
            <Label for="ak-name">{{ t('common.name') }}</Label>
            <Input id="ak-name" v-model="createForm.name" placeholder="vd: CI worker, webhook service..." required minlength="3" />
          </div>
          <div class="space-y-2">
            <Label for="ak-scopes">{{ t('apiKeys.scopesHint') }}</Label>
            <Input id="ak-scopes" v-model="createForm.scopes" placeholder="vd: orders.read, orders.write" />
          </div>
          <div class="space-y-2">
            <Label for="ak-exp">{{ t('apiKeys.expires') }}</Label>
            <Input id="ak-exp" v-model="createForm.expires_at" type="datetime-local" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="closeCreate">{{ t('common.cancel') }}</Button>
            <Button type="submit" variant="gradient" :disabled="createMutation.isPending.value">
              {{ createMutation.isPending.value ? t('apiKeys.creating') : t('apiKeys.create') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Rotate confirm -->
    <Dialog :open="!!rotating" @update:open="(v) => !v && (rotating = null)">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('apiKeys.rotate') }}</DialogTitle>
          <DialogDescription>
            Key hiện tại của <span class="font-medium text-foreground">{{ rotating?.name }}</span> sẽ bị vô hiệu
            NGAY LẬP TỨC và một key mới được tạo. Các hệ thống đang dùng key cũ sẽ lỗi cho đến khi cập nhật.
          </DialogDescription>
        </DialogHeader>

        <div v-if="rotateResult" class="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-3">
          <div class="mb-1 flex items-center justify-between">
            <span class="text-xs font-medium text-emerald-600">Key mới (hiện 1 lần)</span>
            <Button variant="ghost" size="sm" class="h-7 gap-1" @click="copyKey">
              <Copy class="h-3.5 w-3.5" /> Copy
            </Button>
          </div>
          <code class="block break-all rounded bg-background p-2 text-xs">{{ rotateResult.key }}</code>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="rotating = null">{{ t('common.cancel') }}</Button>
          <Button variant="destructive" :disabled="rotateMutation.isPending.value || !!rotateResult" @click="confirmRotate">
            {{ rotateMutation.isPending.value ? 'Đang xoay...' : 'Xoay key' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Revoke confirm -->
    <Dialog :open="!!revoking" @update:open="(v) => !v && (revoking = null)">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('apiKeys.revoke') }}</DialogTitle>
          <DialogDescription>
            Key <span class="font-medium text-foreground">{{ revoking?.name }}</span> sẽ bị vô hiệu vĩnh viễn.
            Có thể tạo lại bằng nút xoay, nhưng key cũ không bao giờ hoạt động lại.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="revoking = null">{{ t('common.cancel') }}</Button>
          <Button variant="destructive" :disabled="revokeMutation.isPending.value" @click="confirmRevoke">
            {{ revokeMutation.isPending.value ? t('apiKeys.revoking') : t('apiKeys.revokeBtn') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
