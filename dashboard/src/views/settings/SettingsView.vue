<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { usersApi } from '@/api'
import { errorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

const { t } = useI18n()
const auth = useAuthStore()

const profileForm = reactive({
  full_name: auth.user?.full_name ?? '',
  avatar_url: auth.user?.avatar_url ?? '',
})

const saveProfile = useMutation({
  mutationFn: async () => {
    if (!auth.user) return
    await usersApi.update(auth.user.id, {
      full_name: profileForm.full_name,
      is_active: auth.user.is_active,
    })
  },
  onSuccess: async () => {
    await auth.fetchMe()
    toast.success(t('settings.saved'))
  },
  onError: (err) => toast.error(errorMessage(err)),
})

const pwForm = reactive({ new_password: '', confirm: '' })
const pwError = ref('')

const changePw = useMutation({
  mutationFn: async () => {
    if (!auth.user) return
    await usersApi.changePassword(auth.user.id, pwForm.new_password)
  },
  onSuccess: () => {
    toast.success(t('settings.passwordChanged'))
    pwForm.new_password = ''
    pwForm.confirm = ''
    pwError.value = ''
  },
  onError: (err) => toast.error(errorMessage(err)),
})

function onSubmitPassword() {
  if (pwForm.new_password.length < 8) {
    pwError.value = t('settings.pwTooShort')
    return
  }
  if (pwForm.new_password !== pwForm.confirm) {
    pwError.value = t('settings.pwMismatch')
    return
  }
  pwError.value = ''
  changePw.mutate()
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">{{ t('settings.title') }}</h1>
      <p class="text-sm text-muted-foreground">{{ t('settings.profileDesc') }}</p>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <!-- Profile -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">{{ t('settings.profile') }}</CardTitle>
          <CardDescription>{{ t('users.emailReadonly') }}</CardDescription>
        </CardHeader>
        <CardContent>
          <form class="space-y-4" @submit.prevent="saveProfile.mutate()">
            <div class="space-y-2">
              <Label>{{ t('common.email') }}</Label>
              <Input :model-value="auth.user?.email" disabled />
            </div>
            <div class="space-y-2">
              <Label for="p-name">{{ t('users.fullName') }}</Label>
              <Input id="p-name" v-model="profileForm.full_name" />
            </div>
            <div class="space-y-2">
              <Label for="p-avatar">{{ t('settings.avatarUrl') }}</Label>
              <Input id="p-avatar" v-model="profileForm.avatar_url" placeholder="https://..." />
            </div>
            <div class="flex items-center justify-between">
              <Badge :variant="auth.user?.is_active ? 'success' : 'destructive'">
                {{ auth.user?.is_active ? t('common.active') : t('common.disabled') }}
              </Badge>
              <Button type="submit" variant="gradient" :disabled="saveProfile.isPending.value">
                {{ saveProfile.isPending.value ? t('settings.saving') : t('settings.saveProfile') }}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <!-- Change password -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">{{ t('settings.changePassword') }}</CardTitle>
          <CardDescription>{{ t('settings.changePasswordDesc') }}</CardDescription>
        </CardHeader>
        <CardContent>
          <form class="space-y-4" @submit.prevent="onSubmitPassword">
            <div class="space-y-2">
              <Label for="pw-new">{{ t('settings.newPassword') }}</Label>
              <Input id="pw-new" v-model="pwForm.new_password" type="password" />
            </div>
            <div class="space-y-2">
              <Label for="pw-confirm">{{ t('settings.confirmPassword') }}</Label>
              <Input id="pw-confirm" v-model="pwForm.confirm" type="password" />
              <p v-if="pwError" class="text-xs text-destructive">{{ pwError }}</p>
            </div>
            <Button type="submit" variant="gradient" :disabled="changePw.isPending.value">
              {{ changePw.isPending.value ? t('settings.changing') : t('settings.changePassword') }}
            </Button>
          </form>
          <p class="mt-4 text-xs text-muted-foreground">
            Tham gia từ: {{ formatDateTime(auth.user?.created_at ?? '') }}
          </p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
