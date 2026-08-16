<script setup lang="ts">
import { computed } from "vue";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import type { LiveEvent, User } from "@/types";

const props = defineProps<{ event: LiveEvent }>();

const BADGE: Record<string, { label: string; color: string; icon: string }> = {
  chat: { label: "Bình luận", color: "bg-zinc-700 text-zinc-200", icon: "💬" },
  gift: { label: "Quà", color: "bg-destructive/15 text-destructive", icon: "🎁" },
  member: { label: "Tham gia", color: "bg-ttcyan/15 text-ttcyan", icon: "👋" },
  like: { label: "Like", color: "bg-pink-500/15 text-pink-400", icon: "❤️" },
  follow: { label: "Follow", color: "bg-violet-500/15 text-violet-400", icon: "➕" },
  share: { label: "Chia sẻ", color: "bg-amber-500/15 text-amber-400", icon: "🔗" },
  social: { label: "Tương tác", color: "bg-amber-500/15 text-amber-400", icon: "🔔" },
  roomUser: { label: "Người xem", color: "bg-emerald-500/15 text-emerald-400", icon: "👥" },
  emote: { label: "Emote", color: "bg-zinc-700 text-zinc-200", icon: "😀" },
  envelope: { label: "Treasure", color: "bg-yellow-500/15 text-yellow-400", icon: "📦" },
  questionNew: { label: "Câu hỏi", color: "bg-sky-500/15 text-sky-400", icon: "❓" },
  liveIntro: { label: "Intro", color: "bg-zinc-700 text-zinc-200", icon: "🎬" },
  linkMicBattle: { label: "Battle", color: "bg-orange-500/15 text-orange-400", icon: "⚔️" },
  linkMicArmies: { label: "Battle điểm", color: "bg-orange-500/15 text-orange-400", icon: "🏆" },
  subNotify: { label: "Subscribe", color: "bg-purple-500/15 text-purple-400", icon: "⭐" },
};

function getUser(data: Record<string, unknown>): User | undefined {
  const user = data.user as User | undefined;
  return user && typeof user === "object" ? user : undefined;
}

function time(ts: number): string {
  return new Date(ts).toLocaleTimeString("vi-VN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function renderContent(event: LiveEvent): string {
  const data = event.data;
  switch (event.type) {
    case "chat":
      return String(data.comment ?? "");
    case "gift": {
      const name = data.giftName ?? data.giftPictureUrl ?? "";
      const count = Number(data.repeatCount ?? 1);
      const diamond = data.diamondCount != null ? ` · ${data.diamondCount} 💎` : "";
      const streak = data.giftType === 1 && !data.repeatEnd ? " (đang streak…)" : "";
      return `${count > 1 ? `x${count} ` : ""}${name}${diamond}${streak}`;
    }
    case "member": {
      const mc = data.memberCount;
      return mc != null ? `đã tham gia phòng (${mc} người xem)` : "đã tham gia phòng";
    }
    case "like":
      return `đã gửi ${data.likeCount ?? 0} like (tổng ${data.totalLikeCount ?? 0})`;
    case "follow":
      return "đã follow host";
    case "share":
      return "đã chia sẻ stream";
    case "social": {
      const action = data.action ? String(data.action) : "";
      return action || "đã tương tác";
    }
    case "roomUser":
      return `lượt xem hiện tại: ${(data.viewerCount ?? 0).toLocaleString()}`;
    case "emote":
      return "đã gửi emote";
    case "envelope":
      return "đã gửi treasure chest";
    case "questionNew": {
      const details = (data.details as Record<string, unknown> | undefined) ?? {};
      return String(details.questionText ?? "đã đặt câu hỏi");
    }
    case "liveIntro": {
      const host = (data.host as User | undefined) ?? {};
      return `host ${host.nickname ?? host.uniqueId ?? ""} bắt đầu intro`;
    }
    case "linkMicBattle":
      return "một trận battle bắt đầu";
    case "linkMicArmies":
      return `battle: gift x${data.giftCount ?? 1} · ${data.totalDiamondCount ?? 0} 💎`;
    case "subNotify":
      return "đã đăng ký kênh";
    default:
      return "";
  }
}

const meta = computed(
  () => BADGE[props.event.type] ?? { label: props.event.type, color: "bg-zinc-700 text-zinc-200", icon: "•" },
);
const user = computed(() => getUser(props.event.data));
const nickname = computed(() => user.value?.nickname ?? user.value?.uniqueId ?? "Ẩn danh");
const uniqueId = computed(() => user.value?.uniqueId);
</script>

<template>
  <li class="flex items-start gap-3 rounded-lg border border-border/60 bg-ink/40 px-3 py-2.5">
    <template v-if="user">
      <Avatar class="mt-0.5 h-9 w-9">
        <AvatarImage v-if="user.profilePictureUrl" :src="user.profilePictureUrl" :alt="nickname" />
        <AvatarFallback>{{ meta.icon }}</AvatarFallback>
      </Avatar>

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
          <span class="truncate font-medium text-zinc-100">{{ nickname }}</span>
          <span v-if="uniqueId" class="truncate text-xs text-zinc-500">@{{ uniqueId }}</span>
          <Badge :class="meta.color">{{ meta.label }}</Badge>
        </div>
        <p class="mt-0.5 break-words text-sm text-zinc-300">{{ renderContent(event) }}</p>
      </div>
    </template>
    <template v-else>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="text-sm">{{ meta.icon }}</span>
          <Badge :class="meta.color">{{ meta.label }}</Badge>
        </div>
        <p class="mt-0.5 break-words text-sm text-zinc-300">{{ renderContent(event) }}</p>
      </div>
    </template>

    <span class="shrink-0 pt-0.5 text-[10px] tabular-nums text-zinc-600">
      {{ time(event.ts) }}
    </span>
  </li>
</template>
