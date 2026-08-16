<script setup lang="ts">
import { ref, watch } from "vue";
import type { LiveEvent, StatusData } from "@/types";
import EventRow from "@/components/EventRow.vue";

const props = defineProps<{ events: LiveEvent[]; status: StatusData }>();
const bottomRef = ref<HTMLElement | null>(null);

watch(
  () => props.events.length,
  () => {
    bottomRef.value?.scrollIntoView({ behavior: "smooth" });
  },
);
</script>

<template>
  <section class="flex flex-col rounded-xl border border-border bg-card lg:col-span-2">
    <header class="flex items-center justify-between border-b border-border px-5 py-3">
      <h2 class="font-semibold">Event feed</h2>
      <span class="rounded-full bg-ink/60 px-2.5 py-0.5 text-xs text-zinc-500">
        {{ events.length }} sự kiện
      </span>
    </header>

    <div class="h-[60vh] overflow-y-auto p-4 lg:h-[70vh]">
      <div
        v-if="events.length === 0"
        class="flex h-full flex-col items-center justify-center gap-2 text-center text-zinc-600"
      >
        <span class="text-4xl">🎁</span>
        <p class="max-w-xs text-sm">
          Chưa có sự kiện. Kết nối một phòng LIVE để nhận gift, comment và lượt tham gia.
        </p>
      </div>
      <ul v-else class="space-y-2">
        <EventRow
          v-for="(event, index) in events"
          :key="`${event.ts}-${index}`"
          :event="event"
        />
      </ul>
      <div ref="bottomRef" />
    </div>

    <footer
      v-if="status.state === 'ended'"
      class="border-t border-border px-5 py-3 text-center text-sm text-zinc-500"
    >
      Stream đã kết thúc.
    </footer>
  </section>
</template>
