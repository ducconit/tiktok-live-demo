<script setup lang="ts">
import { RouterView } from 'vue-router'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@tiktok-live/ui/sidebar"
</script>

<template>
  <SidebarProvider>
    <AppSidebar />
    <SidebarInset>
      <AppHeader>
        <template #trigger>
          <SidebarTrigger class="-ml-1" />
        </template>
      </AppHeader>
      <main class="flex-1 p-4 lg:p-6">
        <RouterView />
      </main>
    </SidebarInset>
  </SidebarProvider>
</template>
