<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { KeyRound, LayoutDashboard, Moon, Settings, Shield, SlidersHorizontal, Sun, Users, Database, Languages } from 'lucide-vue-next'
import { useThemeStore } from '@/stores/theme'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/i18n'
import { APP_TITLE } from '@/lib/app'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar'

const route = useRoute()
const theme = useThemeStore()
const { t, locale } = useI18n()

// Menu phẳng — không sub-menu cho dashboard skeleton (mỗi item 1 route)
const navItems = computed(() => [
  { to: '/', label: t('nav.overview'), icon: LayoutDashboard, perm: 'dashboard.view' },
  { to: '/users', label: t('nav.users'), icon: Users, perm: 'users.read' },
  { to: '/roles', label: t('nav.roles'), icon: Shield, perm: 'roles.read' },
  { to: '/api-keys', label: t('nav.apiKeys'), icon: KeyRound, perm: 'api_keys.read' },
  { to: '/config', label: t('nav.config'), icon: SlidersHorizontal, perm: 'config.read' },
  { to: '/cache', label: t('nav.cache'), icon: Database, perm: 'cache.read' },
  { to: '/settings', label: t('nav.settings'), icon: Settings, perm: 'settings.read' },
])

function isActive(to: string) {
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}

function toggleLocale() {
  setLocale(locale.value === 'vi' ? 'en' : 'vi')
}
</script>

<template>
  <Sidebar collapsible="icon">
    <!-- Header: logo + tên app -->
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg" as-child>
            <RouterLink to="/">
              <div class="flex aspect-square size-8 items-center justify-center rounded-lg bg-gradient-to-br from-indigo-500 to-violet-500 text-sm font-bold text-white">
                G
              </div>
              <div class="grid flex-1 text-left text-sm leading-tight">
                <span class="truncate font-semibold">{{ APP_TITLE }}</span>
                <span class="truncate text-xs text-muted-foreground">Admin</span>
              </div>
            </RouterLink>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>

    <!-- Menu chính -->
    <SidebarContent>
      <SidebarGroup>
        <SidebarGroupLabel>Menu</SidebarGroupLabel>
        <SidebarMenu>
          <SidebarMenuItem v-for="item in navItems" :key="item.to">
            <SidebarMenuButton
              as-child
              :is-active="isActive(item.to)"
              :tooltip="item.label"
            >
              <RouterLink :to="item.to">
                <component :is="item.icon" />
                <span>{{ item.label }}</span>
              </RouterLink>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroup>
    </SidebarContent>

    <!-- Footer: theme + locale -->
    <SidebarFooter>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton :tooltip="theme.theme === 'dark' ? t('common.light') : t('common.dark')" @click="theme.toggle">
            <Sun v-if="theme.theme === 'dark'" />
            <Moon v-else />
            <span>{{ theme.theme === 'dark' ? t('common.dark') : t('common.light') }}</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
        <SidebarMenuItem>
          <SidebarMenuButton :tooltip="locale === 'vi' ? 'English' : 'Tiếng Việt'" @click="toggleLocale">
            <Languages />
            <span>{{ locale === 'vi' ? 'English' : 'Tiếng Việt' }}</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarFooter>

    <SidebarRail />
  </Sidebar>
</template>
