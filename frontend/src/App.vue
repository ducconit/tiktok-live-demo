<script setup lang="ts">
import { computed } from "vue";
import { useLiveSocket } from "@/composables/useLiveSocket";
import type { RoomInfo } from "@/types";
import ConnectBar from "@/components/ConnectBar.vue";
import RoomCard from "@/components/RoomCard.vue";
import EventFeed from "@/components/EventFeed.vue";

const { status, events, viewerCount, connectRoom, disconnectRoom } = useLiveSocket();

const connected = computed(() => status.value.state === "connected");
const connecting = computed(() => status.value.state === "connecting");

const roomInfo = computed<RoomInfo | null>(() => {
  if (status.value.state === "connected") {
    return (status.value.roomInfo as RoomInfo | null) ?? null;
  }
  return null;
});
</script>

<template>
  <div class="min-h-screen bg-background text-zinc-100">
    <header class="sticky top-0 z-10 border-b border-border bg-background/90 backdrop-blur">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <div class="flex items-center gap-2">
          <span
            class="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-ttcyan to-ttred text-lg font-bold text-white"
          >
            T
          </span>
          <div>
            <h1 class="text-lg font-bold leading-none">TikTok Bar</h1>
            <p class="text-xs text-zinc-500">LIVE event monitor · v1.1</p>
          </div>
        </div>
        <ConnectBar
          :connected="connected"
          :connecting="connecting"
          @connect="connectRoom"
          @disconnect="disconnectRoom"
        />
      </div>
    </header>

    <main class="mx-auto grid max-w-6xl grid-cols-1 gap-4 p-4 lg:grid-cols-3">
      <RoomCard :status="status" :room-info="roomInfo" :viewer-count="viewerCount" />
      <EventFeed :events="events" :status="status" />
    </main>
  </div>
</template>
