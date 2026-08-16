<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { errorMessage } from '@/api/client'
import { APP_TITLE } from '@/lib/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const email = ref('admin@example.com')
const password = ref('')
const submitting = ref(false)

async function onSubmit() {
  if (!email.value || !password.value) {
    toast.error(t('auth.login.fillBoth'))
    return
  }
  submitting.value = true
  try {
    await auth.login(email.value, password.value)
    toast.success(t('auth.login.success'))
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (err) {
    toast.error(errorMessage(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <div class="pointer-events-none fixed inset-0 bg-[radial-gradient(ellipse_at_top,rgba(99,102,241,0.15),transparent_60%)]" />
    <Card class="relative w-full max-w-md">
      <CardHeader class="text-center">
        <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 to-violet-500 text-xl font-bold text-white shadow-lg shadow-indigo-500/30">
          G
        </div>
        <CardTitle class="text-2xl">{{ APP_TITLE }}</CardTitle>
        <CardDescription>{{ t('auth.login.title') }}</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="onSubmit">
          <div class="space-y-2">
            <Label for="email">{{ t('common.email') }}</Label>
            <Input id="email" v-model="email" type="email" placeholder="admin@example.com" autocomplete="email" required />
          </div>
          <div class="space-y-2">
            <Label for="password">{{ t('common.password') }}</Label>
            <Input id="password" v-model="password" type="password" placeholder="••••••••" autocomplete="current-password" required />
          </div>
          <Button type="submit" class="w-full" variant="gradient" :disabled="submitting">
            {{ submitting ? t('auth.login.submitting') : t('auth.login.submit') }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
