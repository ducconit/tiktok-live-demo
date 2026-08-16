<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useRoomPreview } from "@/composables/useRoomQuery";

const props = defineProps<{ connected: boolean; connecting: boolean }>();
const emit = defineEmits<{ connect: [username: string]; disconnect: [] }>();

const value = ref("");
const previewUsername = ref<string | null>(null);
let debounceTimer: ReturnType<typeof setTimeout> | undefined;

watch(value, (v) => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    const t = v.trim();
    previewUsername.value = t && !t.includes(" ") ? t : null;
  }, 500);
});

onBeforeUnmount(() => clearTimeout(debounceTimer));

const preview = useRoomPreview(() => previewUsername.value);
const previewData = computed(() => preview.data.value ?? null);
const previewLoading = computed(() => preview.isPending.value);

function submit() {
  if (!value.value.trim() || props.connected || props.connecting) return;
  emit("connect", value.value.trim());
}
</script>

<template>
  <div class="flex w-full max-w-sm flex-col gap-1">
    <form @submit.prevent="submit" class="flex w-full items-center gap-2">
      <div class="relative flex-1">
        <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500">
          @
        </span>
        <Input
          v-model="value"
          placeholder="tiktok username"
          :disabled="connecting"
          class="w-full pl-7"
        />
      </div>
      <Button v-if="connected" type="button" variant="destructive" @click="emit('disconnect')">
        Dừng
      </Button>
      <Button v-else type="button" @click="submit" :disabled="connecting || !value.trim()">
        {{ connecting ? "Đang kết nối…" : "Kết nối" }}
      </Button>
    </form>
    <p v-if="previewUsername && !connected && !connecting" class="pl-1 text-xs">
      <template v-if="previewLoading">Đang kiểm tra…</template>
      <template v-else-if="previewData?.live">
        <span class="text-emerald-400">● Đang LIVE</span>
        <span v-if="previewData?.title" class="text-zinc-500"> — {{ previewData.title }}</span>
      </template>
      <template v-else><span class="text-zinc-500">Không live hoặc không tìm thấy</span></template>
    </p>
  </div>
</template>
