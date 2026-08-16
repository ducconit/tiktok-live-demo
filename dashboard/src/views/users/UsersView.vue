<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { Plus, Trash2, Search } from 'lucide-vue-next'
import { usersApi } from '@/api'
import { errorMessage } from '@/api/client'
import { formatDateTime } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
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
import UserFormDialog from './UserFormDialog.vue'

const { t } = useI18n()
const queryClient = useQueryClient()

const page = ref(1)
const q = ref('')
const isActiveFilter = ref<string>('')

const { data, isLoading, isFetching } = useQuery({
  queryKey: computed(() => ['users', page.value, q.value, isActiveFilter.value]),
  queryFn: () =>
    usersApi.list({
      page: page.value,
      page_size: 10,
      q: q.value || undefined,
      is_active: isActiveFilter.value === '' ? undefined : isActiveFilter.value === 'true',
    }),
})

const deleteMutation = useMutation({
  mutationFn: (id: string) => usersApi.remove(id),
  onSuccess: () => {
    toast.success(t('users.deleted'))
    queryClient.invalidateQueries({ queryKey: ['users'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

const createMutation = useMutation({
  mutationFn: (body: { email: string; password: string; full_name: string }) => usersApi.create(body),
  onSuccess: () => {
    toast.success(t('users.created'))
    dialogOpen.value = false
    queryClient.invalidateQueries({ queryKey: ['users'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

const dialogOpen = ref(false)
const deleting = ref<{ id: string; email: string } | null>(null)
const totalPages = computed(() => {
  const meta = data.value?.meta
  if (!meta) return 1
  return Math.ceil(meta.total / meta.limit) // meta mới: limit/page/total
})
const total = computed(() => data.value?.meta.total ?? 0)

function onSearch() {
  page.value = 1
}

function confirmDelete() {
  if (deleting.value) {
    deleteMutation.mutate(deleting.value.id)
    deleting.value = null
  }
}

function initialsOf(email: string) {
  return email.slice(0, 2).toUpperCase()
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">{{ t('users.title') }}</h1>
        <p class="text-sm text-muted-foreground">{{ total }} người dùng</p>
      </div>
      <Button variant="gradient" @click="dialogOpen = true">
        <Plus class="h-4 w-4" /> Thêm người dùng
      </Button>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="relative flex-1">
        <Search class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          v-model="q"
          placeholder="Tìm theo email hoặc tên..."
          class="pl-8"
          @keyup.enter="onSearch"
        />
      </div>
      <select
        v-model="isActiveFilter"
        class="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
        @change="onSearch"
      >
        <option value="">{{ t('common.all') }}</option>
        <option value="true">{{ t('common.active') }}</option>
        <option value="false">{{ t('common.disabled') }}</option>
      </select>
    </div>

    <!-- Table -->
    <div class="rounded-xl border bg-card">
      <Table>
        <TableHeader>
          <TableRow class="hover:bg-transparent">
            <TableHead>{{ t('users.title') }}</TableHead>
            <TableHead>{{ t('common.status') }}</TableHead>
            <TableHead>{{ t('users.lastLogin') }}</TableHead>
            <TableHead>{{ t('common.createdAt') }}</TableHead>
            <TableHead class="w-20 text-right">{{ t('common.actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-if="isLoading">
            <TableCell colspan="5" class="space-y-2">
              <Skeleton v-for="i in 5" :key="i" class="h-9 w-full" />
            </TableCell>
          </TableRow>
          <TableRow v-else-if="data?.items.length === 0">
            <TableCell colspan="5" class="py-10 text-center text-muted-foreground">
              Không có dữ liệu
            </TableCell>
          </TableRow>
          <TableRow v-for="u in data?.items ?? []" v-else :key="u.id">
            <TableCell>
              <div class="flex items-center gap-3">
                <div class="flex h-9 w-9 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500/20 to-violet-500/20 text-xs font-semibold text-indigo-400">
                  {{ initialsOf(u.email) }}
                </div>
                <div>
                  <div class="font-medium">{{ u.full_name || u.email }}</div>
                  <div class="text-xs text-muted-foreground">{{ u.email }}</div>
                </div>
              </div>
            </TableCell>
            <TableCell>
              <Badge :variant="u.is_active ? 'success' : 'destructive'">
                {{ u.is_active ? t('common.active') : t('common.disabled') }}
              </Badge>
            </TableCell>
            <TableCell class="text-sm text-muted-foreground">{{ formatDateTime(u.last_login_at ?? '') }}</TableCell>
            <TableCell class="text-sm text-muted-foreground">{{ formatDateTime(u.created_at) }}</TableCell>
            <TableCell class="text-right">
              <div class="flex justify-end gap-1">
                <Button variant="ghost" size="icon" title="Xoá" @click="deleting = { id: u.id, email: u.email }">
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
        <Button variant="outline" size="sm" :disabled="page <= 1 || isFetching" @click="page--">
          Trước
        </Button>
        <Button variant="outline" size="sm" :disabled="page >= totalPages || isFetching" @click="page++">
          Sau
        </Button>
      </div>
    </div>

    <!-- Create dialog -->
    <UserFormDialog v-model:open="dialogOpen" :submitting="createMutation.isPending.value" @submit="createMutation.mutate" />

    <!-- Delete confirm -->
    <Dialog :open="!!deleting" @update:open="(v) => !v && (deleting = null)">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ t('users.deleteConfirm') }}</DialogTitle>
          <DialogDescription>
            Người dùng <span class="font-medium text-foreground">{{ deleting?.email }}</span> sẽ bị xoá vĩnh viễn. Hành động này không thể hoàn tác.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="deleting = null">{{ t('common.cancel') }}</Button>
          <Button variant="destructive" :disabled="deleteMutation.isPending.value" @click="confirmDelete">
            {{ deleteMutation.isPending.value ? t('common.deleting') : t('common.delete') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
