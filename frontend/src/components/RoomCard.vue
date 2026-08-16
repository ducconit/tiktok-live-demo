<script setup lang="ts">
import { computed } from "vue";
import { Card } from "@/components/ui/card";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import type { RoomInfo, StatusData } from "@/types";

const props = defineProps<{
  status: StatusData;
  roomInfo: RoomInfo | null;
  viewerCount: number | null;
}>();

const STATE_META: Record<string, { label: string; dot: string }> = {
  idle: { label: "Chưa kết nối", dot: "bg-zinc-500" },
  connecting: { label: "Đang kết nối…", dot: "bg-amber-400 animate-pulse" },
  connected: { label: "Đang theo dõi LIVE", dot: "bg-emerald-400 animate-pulse" },
  disconnected: { label: "Đã ngắt kết nối", dot: "bg-zinc-500" },
  error: { label: "Lỗi", dot: "bg-destructive" },
  ended: { label: "Stream đã kết thúc", dot: "bg-zinc-500" },
};

const meta = computed(() => STATE_META[props.status.state] ?? STATE_META.idle);
const owner = computed(() => props.roomInfo?.owner);
const avatar = computed(() => owner.value?.profilePictureUrl);
const nickname = computed(() => owner.value?.nickname ?? owner.value?.uniqueId ?? "—");
const title = computed(() => props.roomInfo?.title ?? "Chưa có thông tin phòng");
const resolvedViewers = computed(
  () => props.viewerCount ?? props.roomInfo?.viewerCount ?? props.roomInfo?.userCount ?? null,
);
</script>

<template>
  <aside class="flex flex-col gap-4 lg:col-span-1">
    <Card>
      <div class="p-5">
        <div class="flex items-center gap-3">
          <div class="relative">
            <Avatar class="h-16 w-16 border border-border">
              <AvatarImage v-if="avatar" :src="avatar" :alt="nickname" />
              <AvatarFallback class="text-2xl">@</AvatarFallback>
            </Avatar>
            <span
              class="absolute -bottom-0 -right-0 h-4 w-4 rounded-full border-2 border-card"
              :class="meta.dot"
            />
          </div>
          <div class="min-w-0">
            <h2 class="truncate text-lg font-semibold">{{ nickname }}</h2>
            <p v-if="owner?.uniqueId" class="truncate text-sm text-zinc-500">
              @{{ owner.uniqueId }}
            </p>
          </div>
        </div>

        <div class="mt-4 space-y-2 text-sm">
          <div class="flex items-center justify-between">
            <span class="text-zinc-500">Tiêu đề</span>
            <span class="max-w-[60%] truncate text-zinc-200">{{ title }}</span>
          </div>
          <div v-if="resolvedViewers !== null" class="flex items-center justify-between">
            <span class="text-zinc-500">Người xem</span>
            <span class="font-medium text-ttcyan">{{ resolvedViewers.toLocaleString() }}</span>
          </div>
          <div v-if="status.roomId" class="flex items-center justify-between">
            <span class="text-zinc-500">Room ID</span>
            <span class="truncate text-zinc-400">{{ status.roomId }}</span>
          </div>
        </div>

        <div class="mt-4 flex items-center gap-2 rounded-lg border border-border bg-ink/60 px-3 py-2">
          <span class="h-2 w-2 rounded-full" :class="meta.dot" />
          <span class="text-xs text-zinc-300">{{ meta.label }}</span>
        </div>
        <p v-if="status.message" class="mt-3 text-xs leading-relaxed text-destructive">
          {{ status.message }}
        </p>
      </div>
    </Card>

    <Card>
      <div class="p-5 text-sm">
        <h3 class="mb-3 font-semibold text-zinc-300">Cách dùng</h3>
        <ol class="list-decimal space-y-2 pl-5 text-zinc-400">
          <li>Nhập @username của streamer đang LIVE.</li>
          <li>Bấm Kết nối để nhận event real-time.</li>
          <li>Gift, comment, lượt tham gia sẽ hiển thị ở cột bên phải.</li>
        </ol>
      </div>
    </Card>
  </aside>
</template>
