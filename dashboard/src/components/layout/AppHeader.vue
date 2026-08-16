<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { LogOut, Settings } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import { Button } from "@tiktok-live/ui/button"
import { Avatar, AvatarFallback, AvatarImage } from "@tiktok-live/ui/avatar"
import { DropdownMenu, DropdownMenuTrigger } from "@tiktok-live/ui/dropdown-menu"
import { useAuthStore } from '@/stores/auth'
import { APP_TITLE } from '@/lib/app'

const { t } = useI18n()
const auth = useAuthStore()
const router = useRouter()

const initials = computed(() => {
  const name = auth.user?.full_name || auth.user?.email || '?'
  return name.slice(0, 2).toUpperCase()
})

function onLogout() {
  auth.logout()
  toast.success(t('common.loggedOut'))
  router.push({ name: 'login' })
}
</script>

<template>
  <header class="sticky top-0 z-20 flex h-14 items-center justify-between gap-2 border-b bg-background/80 px-4 backdrop-blur lg:px-6">
    <div class="flex items-center gap-2">
      <!-- Slot cho SidebarTrigger (mobile + desktop toggle) -->
      <slot name="trigger" />
      <div class="flex items-center gap-2 lg:hidden">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-indigo-500 to-violet-500 text-sm font-bold text-white">
          G
        </div>
        <span class="text-sm font-semibold">{{ APP_TITLE }}</span>
      </div>
      <div class="hidden text-sm text-muted-foreground lg:block">
        Xin chào, <span class="font-medium text-foreground">{{ auth.user?.full_name || auth.user?.email }}</span>
      </div>
    </div>
      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="ghost" size="icon" class="rounded-full">
            <Avatar class="h-8 w-8">
              <AvatarImage :src="auth.user?.avatar_url || ''" />
              <AvatarFallback>{{ initials }}</AvatarFallback>
            </Avatar>
          </Button>
        </DropdownMenuTrigger>
        <div class="px-2 py-1.5 text-sm">
          <div class="font-medium">{{ auth.user?.full_name || auth.user?.email }}</div>
          <div class="text-xs text-muted-foreground">{{ auth.user?.email }}</div>
        </div>
        <div class="h-px my-1 bg-border" />
        <button
          class="relative flex w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground"
          @click="router.push({ name: 'settings' })"
        >
          <Settings class="h-4 w-4" /> Cài đặt
        </button>
        <div class="h-px my-1 bg-border" />
        <button
          class="relative flex w-full cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-destructive outline-none hover:bg-destructive/10"
          @click="onLogout"
        >
          <LogOut class="h-4 w-4" /> Đăng xuất
        </button>
      </DropdownMenu>
  </header>
</template>
