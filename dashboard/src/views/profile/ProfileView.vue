<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { Camera, KeyRound, Loader2, Save, UserRound } from 'lucide-vue-next'
import { accountApi } from '@/api'
import { errorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/lib/utils'
import { Button } from "@tiktok-live/ui/button"
import { Input } from "@tiktok-live/ui/input"
import { Label } from "@tiktok-live/ui/label"
import { Avatar, AvatarFallback, AvatarImage } from "@tiktok-live/ui/avatar"
import { Badge } from "@tiktok-live/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@tiktok-live/ui/card"

const { t } = useI18n()
const auth = useAuthStore()
const qc = useQueryClient()

// ---- Dữ liệu profile (từ GET /me — namespace public) ----
const { data: user, isLoading } = useQuery({
  queryKey: ['account', 'me'],
  queryFn: accountApi.me,
  staleTime: 30_000,
})

// ---- Lưu toàn bộ thông tin cá nhân (1 nút Lưu duy nhất) ----
// Flow: nếu có ảnh mới (pendingAvatar) → upload avatar trước → rồi PUT /me.
const fullName = ref('')
watch(user, (u) => {
  if (u) fullName.value = u.full_name
})

const saveProfile = useMutation({
  mutationFn: async () => {
    // 1) Upload avatar nếu có ảnh mới được chọn (preview)
    if (pendingAvatar.value) {
      await accountApi.uploadAvatar(pendingAvatar.value)
    }
    // 2) Cập nhật thông tin cá nhân
    return accountApi.updateMe({ full_name: fullName.value.trim() })
  },
  onSuccess: async (u) => {
    await auth.fetchMe()
    qc.setQueryData(['account', 'me'], u)
    cancelAvatar()
    toast.success(t('settings.saved'))
  },
  onError: (e) => toast.error(errorMessage(e)),
})

// ---- Form đổi mật khẩu (POST /me/change-password) ----
const pwForm = ref({ current_password: '', new_password: '', confirm_password: '' })
const pwError = computed(() => {
  if (pwForm.value.new_password.length > 0 && pwForm.value.new_password.length < 8) return t('settings.pwTooShort')
  if (pwForm.value.confirm_password && pwForm.value.new_password !== pwForm.value.confirm_password) return t('settings.pwMismatch')
  return ''
})

const changePassword = useMutation({
  mutationFn: () =>
    accountApi.changePassword({
      current_password: pwForm.value.current_password,
      new_password: pwForm.value.new_password,
    }),
  onSuccess: () => {
    pwForm.value = { current_password: '', new_password: '', confirm_password: '' }
    toast.success(t('settings.passwordChanged'))
  },
  onError: (e) => toast.error(errorMessage(e)),
})

// ---- Upload avatar (POST /me/avatar — multipart field "avatar") ----
// Flow: chọn file → xem trước (preview local) → bấm "Lưu" chung (saveProfile)
// sẽ upload avatar + cập nhật thông tin trong 1 lần.
const avatarInput = ref<HTMLInputElement | null>(null)
const pendingAvatar = ref<File | null>(null)
const avatarPreview = ref<string | null>(null)

function onAvatarPick(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  // Preview local trước — chưa upload gì cả
  if (avatarPreview.value) URL.revokeObjectURL(avatarPreview.value)
  pendingAvatar.value = file
  avatarPreview.value = URL.createObjectURL(file)
  // Cho phép chọn lại cùng file sau khi hủy/lưu
  input.value = ''
}

function cancelAvatar() {
  if (avatarPreview.value) URL.revokeObjectURL(avatarPreview.value)
  pendingAvatar.value = null
  avatarPreview.value = null
}

// Ảnh hiển thị: preview local (nếu đang chọn) — nếu không thì avatar từ server
const avatarSrc = computed(() => avatarPreview.value || displayUser.value?.avatar_url || '')

const initials = computed(() => (fullName.value || user.value?.email || '?').slice(0, 2).toUpperCase())
const displayUser = computed(() => user.value ?? auth.user)
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('settings.profile') }}</h1>
      <p class="mt-1 text-sm text-muted-foreground">{{ t('settings.profileDesc') }}</p>
    </div>

    <!-- Card 1: Avatar + thông tin tóm tắt -->
    <Card class="overflow-hidden">
      <div class="h-24 bg-gradient-to-r from-indigo-500/20 via-violet-500/15 to-transparent" />
      <CardContent class="-mt-12 flex flex-col items-start gap-4 sm:flex-row sm:items-end">
        <div class="relative">
          <Avatar class="h-24 w-24 border-4 border-background shadow-lg">
            <AvatarImage :src="avatarSrc" />
            <AvatarFallback class="bg-gradient-to-br from-indigo-500 to-violet-500 text-2xl font-bold text-white">
              {{ initials }}
            </AvatarFallback>
          </Avatar>
          <button
            type="button"
            class="absolute bottom-1 right-1 flex h-7 w-7 items-center justify-center rounded-full bg-primary text-primary-foreground shadow transition-opacity hover:opacity-80 disabled:opacity-50"
            :title="t('settings.changeAvatar')"
            @click="avatarInput?.click()"
          >
            <Camera class="h-3.5 w-3.5" />
          </button>
          <input ref="avatarInput" type="file" accept="image/*" class="hidden" @change="onAvatarPick" />
        </div>
        <div class="flex-1 space-y-1 pb-1">
          <div class="flex items-center gap-2">
            <h2 class="text-lg font-semibold">{{ displayUser?.full_name || '—' }}</h2>
            <Badge variant="success">{{ displayUser?.email }}</Badge>
          </div>
          <p class="text-sm text-muted-foreground">
            {{ t('settings.memberSince') }} {{ displayUser?.created_at ? formatDateTime(displayUser.created_at) : '—' }}
          </p>
          <p v-if="isLoading" class="text-xs text-muted-foreground">{{ t('common.loading') }}</p>

          <!-- Đã chọn ảnh mới (chưa lưu) — nhắc + nút bỏ chọn -->
          <div v-if="avatarPreview" class="mt-3 flex items-center gap-2">
            <Badge variant="warning">{{ t('settings.avatarPending') }}</Badge>
            <Button size="sm" variant="ghost" :disabled="saveProfile.isPending.value" @click="cancelAvatar">
              {{ t('common.cancel') }}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Card 2: Cập nhật thông tin cá nhân (1 nút Lưu — avatar + thông tin) -->
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base">
          <UserRound class="h-4 w-4 text-primary" />
          {{ t('settings.personalInfo') }}
        </CardTitle>
        <CardDescription>{{ t('settings.personalInfoDesc') }}</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="saveProfile.mutate()">
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-2">
              <Label for="full-name">{{ t('settings.fullName') }}</Label>
              <Input id="full-name" v-model="fullName" :placeholder="t('settings.fullName')" required minlength="2" maxlength="120" />
            </div>
            <div class="space-y-2">
              <Label>{{ t('common.email') }}</Label>
              <Input :model-value="displayUser?.email || ''" disabled />
            </div>
          </div>
          <div class="flex justify-end">
            <Button type="submit" variant="gradient" :disabled="saveProfile.isPending.value || !fullName.trim()">
              <Loader2 v-if="saveProfile.isPending.value" class="h-4 w-4 animate-spin" />
              <Save v-else class="h-4 w-4" />
              {{ saveProfile.isPending.value ? t('settings.saving') : t('settings.saveProfile') }}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>

    <!-- Card 3: Đổi mật khẩu -->
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base">
          <KeyRound class="h-4 w-4 text-primary" />
          {{ t('settings.changePassword') }}
        </CardTitle>
        <CardDescription>{{ t('settings.changePasswordDesc') }}</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="changePassword.mutate()">
          <div class="space-y-2">
            <Label for="current-password">{{ t('settings.currentPassword') }}</Label>
            <Input id="current-password" v-model="pwForm.current_password" type="password" autocomplete="current-password" required />
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-2">
              <Label for="new-password">{{ t('settings.newPassword') }}</Label>
              <Input id="new-password" v-model="pwForm.new_password" type="password" autocomplete="new-password" required minlength="8" />
            </div>
            <div class="space-y-2">
              <Label for="confirm-password">{{ t('settings.confirmPassword') }}</Label>
              <Input id="confirm-password" v-model="pwForm.confirm_password" type="password" autocomplete="new-password" required />
            </div>
          </div>
          <p v-if="pwError" class="text-sm text-destructive">{{ pwError }}</p>
          <div class="flex justify-end">
            <Button
              type="submit"
              variant="gradient"
              :disabled="
                changePassword.isPending.value ||
                !!pwError ||
                !pwForm.current_password ||
                pwForm.new_password.length < 8 ||
                !pwForm.confirm_password
              "
            >
              <Loader2 v-if="changePassword.isPending.value" class="h-4 w-4 animate-spin" />
              <KeyRound v-else class="h-4 w-4" />
              {{ changePassword.isPending.value ? t('settings.changing') : t('settings.changePassword') }}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
