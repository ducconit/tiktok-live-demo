import { ref, onBeforeUnmount } from "vue";
import { useSockudo } from "@sockudo/client/vue";
import type { Channel } from "@sockudo/client";
import { connectRoom, disconnectRoom } from "@/services/realtime";
import type { LiveEvent, StatusData } from "@/types";

export function useRealtime() {
  const sock = useSockudo();
  const status = ref<StatusData>({ state: "idle" });
  const events = ref<LiveEvent[]>([]);
  const viewerCount = ref<number | null>(null);

  let channel: Channel | null = null;
  let channelName = "";
  let currentUser = "";

  function onEvent(payload: unknown) {
    const ev = payload as LiveEvent;
    if (!ev || typeof ev.type !== "string") return;
    if (ev.type === "status") {
      status.value = ev.data as unknown as StatusData;
    } else if (ev.type === "roomUser") {
      const vc = (ev.data as { viewerCount?: number }).viewerCount;
      if (typeof vc === "number") viewerCount.value = vc;
    } else {
      events.value = [...events.value.slice(-499), ev];
    }
  }

  async function connect(username: string) {
    events.value = [];
    viewerCount.value = null;
    status.value = { state: "connecting", username };

    const res = await connectRoom(username);
    if (!res.connected) {
      status.value = { state: "error", message: res.error ?? "Lỗi kết nối" };
      return;
    }

    // Subscribe channel "user_<username>" trên Sockudo để nhận events realtime.
    currentUser = username;
    channelName = `user_${username}`;
    channel = sock.subscribe(channelName);
    channel.bind("event", onEvent);

    status.value = {
      state: "connected",
      roomId: res.roomId,
      roomInfo: res.roomInfo ?? null,
    };
  }

  async function disconnect() {
    if (channel) {
      try {
        sock.unsubscribe(channelName);
      } catch {
        // ignore
      }
      channel = null;
      channelName = "";
    }
    if (currentUser) {
      await disconnectRoom(currentUser);
      currentUser = "";
    }
    events.value = [];
    viewerCount.value = null;
    status.value = { state: "idle" };
  }

  onBeforeUnmount(() => {
    if (channel) {
      try {
        sock.unsubscribe(channelName);
      } catch {
        // ignore
      }
    }
    if (currentUser) {
      disconnectRoom(currentUser);
    }
  });

  return { status, events, viewerCount, connect, disconnect };
}
