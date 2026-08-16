import { ref, onBeforeUnmount } from "vue";
import { LiveSocket } from "@/services/socket";
import type { LiveEvent, StatusData } from "@/types";

export function useLiveSocket() {
  const socket = new LiveSocket();
  const status = ref<StatusData>({ state: "idle" });
  const events = ref<LiveEvent[]>([]);
  const viewerCount = ref<number | null>(null);

  const unsubscribe = socket.onEvent((event) => {
    if (event.type === "status") {
      status.value = event.data as unknown as StatusData;
    } else if (event.type === "roomUser") {
      const vc = (event.data as { viewerCount?: number }).viewerCount;
      if (typeof vc === "number") viewerCount.value = vc;
    } else {
      events.value = [...events.value.slice(-499), event];
    }
  });

  const connectRoom = (username: string) => {
    events.value = [];
    viewerCount.value = null;
    status.value = { state: "connecting", username };
    socket.connectRoom(username);
  };

  const disconnectRoom = () => {
    events.value = [];
    viewerCount.value = null;
    status.value = { state: "idle" };
    socket.disconnectRoom();
  };

  onBeforeUnmount(() => {
    unsubscribe();
    socket.disconnect();
  });

  return { status, events, viewerCount, connectRoom, disconnectRoom };
}
