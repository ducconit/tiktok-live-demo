<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { Plus, Pencil, Trash2, Shield } from 'lucide-vue-next'
import { rbacApi } from '@/api'
import { errorMessage } from '@/api/client'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
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
import type { Permission, Role } from '@/api/types'

const { t } = useI18n()
const queryClient = useQueryClient()

const { data: roles, isLoading } = useQuery({
  queryKey: ['roles'],
  queryFn: rbacApi.roles,
})

const { data: permissions } = useQuery({
  queryKey: ['permissions'],
  queryFn: rbacApi.permissions,
})

const editingId = ref<string | null>(null)

const { refetch: refetchRolePerms } = useQuery({
  queryKey: ['role-permissions', editingId],
  queryFn: () => rbacApi.rolePermissions(editingId.value!),
  enabled: computed(() => !!editingId.value),
})

const editing = computed<Role | null>(
  () => roles.value?.find((r) => r.id === editingId.value) ?? null,
)

// ---- Create/Edit role ----
const roleDialog = ref(false)
const roleForm = ref({ slug: '', name: '', description: '' })
const saveRole = useMutation({
  mutationFn: () =>
    editingId.value
      ? rbacApi.updateRole(editingId.value, roleForm.value)
      : rbacApi.createRole(roleForm.value),
  onSuccess: () => {
    toast.success(t('roles.saved'))
    roleDialog.value = false
    queryClient.invalidateQueries({ queryKey: ['roles'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

function openRoleDialog(role?: Role) {
  editingId.value = role?.id ?? null
  roleForm.value = role
    ? { slug: role.slug, name: role.name, description: role.description }
    : { slug: '', name: '', description: '' }
  roleDialog.value = true
}

// ---- Permissions assignment ----
const selectedPerms = ref<string[]>([])
const permDialogRole = ref<Role | null>(null)

function openPermDialog(role: Role) {
  permDialogRole.value = role
  editingId.value = role.id
  refetchRolePerms().then(({ data }) => {
    selectedPerms.value = (data ?? []).map((p: Permission) => p.slug)
  })
}

const savePerms = useMutation({
  mutationFn: async (ids: string[]) => {
    if (!permDialogRole.value) return
    const perms = permissions.value ?? []
    const idList = ids
      .map((slug) => perms.find((p) => p.slug === slug)?.id)
      .filter((id): id is string => !!id)
    await rbacApi.setRolePermissions(permDialogRole.value.id, idList)
  },
  onSuccess: () => {
    toast.success(t('roles.permsUpdated'))
    queryClient.invalidateQueries({ queryKey: ['roles'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

// ---- Delete ----
const deleteRole = useMutation({
  mutationFn: (id: string) => rbacApi.deleteRole(id),
  onSuccess: () => {
    toast.success(t('roles.deleted'))
    queryClient.invalidateQueries({ queryKey: ['roles'] })
  },
  onError: (err) => toast.error(errorMessage(err)),
})

function togglePerm(slug: string, checked: boolean) {
  if (checked) {
    selectedPerms.value = [...selectedPerms.value, slug]
  } else {
    selectedPerms.value = selectedPerms.value.filter((s) => s !== slug)
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">{{ t('roles.title') }}</h1>
        <p class="text-sm text-muted-foreground">{{ t('roles.subtitle') }}</p>
      </div>
      <Button variant="gradient" @click="openRoleDialog()">
        <Plus class="h-4 w-4" /> Thêm vai trò
      </Button>
    </div>

    <div class="rounded-xl border bg-card">
      <Table>
        <TableHeader>
          <TableRow class="hover:bg-transparent">
            <TableHead>{{ t('users.role') }}</TableHead>
            <TableHead>{{ t('common.description') }}</TableHead>
            <TableHead class="w-24">{{ t('roles.permissions') }}</TableHead>
            <TableHead class="w-32 text-right">{{ t('common.actions') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-if="isLoading">
            <TableCell colspan="4" class="space-y-2">
              <Skeleton v-for="i in 3" :key="i" class="h-9 w-full" />
            </TableCell>
          </TableRow>
          <TableRow v-for="role in roles ?? []" :key="role.id">
            <TableCell>
              <div class="flex items-center gap-2">
                <Shield class="h-4 w-4 text-primary" />
                <span class="font-medium">{{ role.name }}</span>
                <Badge variant="secondary" class="font-mono text-xs">{{ role.slug }}</Badge>
              </div>
            </TableCell>
            <TableCell class="text-sm text-muted-foreground">{{ role.description || '—' }}</TableCell>
            <TableCell>
              <Button variant="ghost" size="sm" @click="openPermDialog(role)">{{ t('roles.assign') }}</Button>
            </TableCell>
            <TableCell class="text-right">
              <div class="flex justify-end gap-1">
                <Button variant="ghost" size="icon" title="Sửa" @click="openRoleDialog(role)">
                  <Pencil class="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  title="Xoá"
                  :disabled="role.slug === 'super_admin'"
                  @click="deleteRole.mutate(role.id)"
                >
                  <Trash2 class="h-4 w-4 text-destructive" />
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <!-- Role dialog -->
    <Dialog v-model:open="roleDialog">
      <DialogContent class="max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editing ? t('roles.editTitle') : t('roles.addTitle') }}</DialogTitle>
          <DialogDescription>{{ t('roles.dialogDesc') }}</DialogDescription>
        </DialogHeader>
        <form class="space-y-4" @submit.prevent="saveRole.mutate()">
          <div class="space-y-2">
            <Label for="r-slug">{{ t('roles.slug') }}</Label>
            <Input id="r-slug" v-model="roleForm.slug" placeholder="editor" :disabled="!!editing" />
            <p class="text-xs text-muted-foreground">Ví dụ: editor, moderator — không đổi được khi đã tạo</p>
          </div>
          <div class="space-y-2">
            <Label for="r-name">{{ t('common.name') }}</Label>
            <Input id="r-name" v-model="roleForm.name" placeholder="Biên tập viên" />
          </div>
          <div class="space-y-2">
            <Label for="r-desc">{{ t('common.description') }}</Label>
            <Input id="r-desc" v-model="roleForm.description" placeholder="Quản lý nội dung" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="roleDialog = false">{{ t('common.cancel') }}</Button>
            <Button type="submit" variant="gradient" :disabled="saveRole.isPending.value">
              {{ saveRole.isPending.value ? t('settings.saving') : t('common.save') }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Permissions dialog -->
    <Dialog :open="!!permDialogRole" @update:open="(v) => !v && (permDialogRole = null)">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>Phân quyền — {{ permDialogRole?.name }}</DialogTitle>
          <DialogDescription>{{ t('roles.assignDialogDesc') }}</DialogDescription>
        </DialogHeader>
        <div class="grid gap-2 max-h-80 overflow-y-auto pr-2">
          <label
            v-for="perm in permissions ?? []"
            :key="perm.id"
            class="flex cursor-pointer items-center gap-3 rounded-md border px-3 py-2.5 transition-colors hover:bg-muted/50"
          >
            <Checkbox
              :checked="selectedPerms.includes(perm.slug)"
              @update:checked="(v: boolean) => togglePerm(perm.slug, v)"
            />
            <div class="flex-1">
              <div class="text-sm font-medium">{{ perm.name }}</div>
              <div class="text-xs text-muted-foreground font-mono">{{ perm.slug }}</div>
            </div>
          </label>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="permDialogRole = null">{{ t('common.cancel') }}</Button>
          <Button
            variant="gradient"
            :disabled="savePerms.isPending.value"
            @click="savePerms.mutate(selectedPerms)"
          >
            {{ savePerms.isPending.value ? t('settings.saving') : t('roles.savePerms') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
