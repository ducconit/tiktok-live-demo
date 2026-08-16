<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const { t } = useI18n()
const props = defineProps<{
  open: boolean
  submitting: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  submit: [body: { email: string; password: string; full_name: string }]
}>()

const form = reactive({ email: '', password: '', full_name: '' })
const errors = reactive<Record<string, string>>({})

watch(
  () => props.open,
  (open) => {
    if (open) {
      form.email = ''
      form.password = ''
      form.full_name = ''
      Object.keys(errors).forEach((k) => delete errors[k])
    }
  },
)

function onSubmit() {
  errors.email = form.email && /.+@.+\..+/.test(form.email) ? '' : t('validation.email')
  errors.password = form.password.length >= 8 ? '' : t('validation.min8')
  errors.full_name = form.full_name.length <= 128 ? '' : t('validation.max128')
  if (Object.values(errors).some(Boolean)) {
    toast.error(t('auth.login.failed'))
    return
  }
  emit('submit', { email: form.email, password: form.password, full_name: form.full_name })
}
</script>

<template>
  <Dialog :open="props.open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t('users.addDialogTitle') }}</DialogTitle>
        <DialogDescription>{{ t('users.addDialogDesc') }}</DialogDescription>
      </DialogHeader>
      <form class="space-y-4" @submit.prevent="onSubmit">
        <div class="space-y-2">
          <Label for="u-email">{{ t('common.email') }}</Label>
          <Input id="u-email" v-model="form.email" type="email" placeholder="user@example.com" />
          <p v-if="errors.email" class="text-xs text-destructive">{{ errors.email }}</p>
        </div>
        <div class="space-y-2">
          <Label for="u-name">{{ t('users.fullName') }}</Label>
          <Input id="u-name" v-model="form.full_name" placeholder="Nguyễn Văn A" />
          <p v-if="errors.full_name" class="text-xs text-destructive">{{ errors.full_name }}</p>
        </div>
        <div class="space-y-2">
          <Label for="u-pass">{{ t('common.password') }}</Label>
          <Input id="u-pass" v-model="form.password" type="password" placeholder="Tối thiểu 8 ký tự" />
          <p v-if="errors.password" class="text-xs text-destructive">{{ errors.password }}</p>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" @click="emit('update:open', false)">{{ t('common.cancel') }}</Button>
          <Button type="submit" variant="gradient" :disabled="props.submitting">
            {{ props.submitting ? t('settings.saving') : t('users.add') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
