import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { LiveSocket } from "./socket";
import type { LiveEvent, RoomInfo, StatusData } from "./types";
import { ConnectBar } from "./components/ConnectBar";
import { RoomCard } from "./components/RoomCard";
import { EventFeed } from "./components/EventFeed";

export function App() {
  const [status, setStatus] = useState<StatusData>({ state: "idle" });
  const [events, setEvents] = useState<LiveEvent[]>([]);
  const [viewerCount, setViewerCount] = useState<number | null>(null);
  const socketRef = useRef<LiveSocket | null>(null);

  const connected = status.state === "connected";
  const connecting = status.state === "connecting";

  useEffect(() => {
    const socket = new LiveSocket();
    socketRef.current = socket;
    socket.connect();

    const unsubscribe = socket.onEvent((event) => {
      if (event.type === "status") {
        setStatus(event.data as unknown as StatusData);
      } else if (event.type === "roomUser") {
        const vc = (event.data as { viewerCount?: number }).viewerCount;
        if (typeof vc === "number") setViewerCount(vc);
      } else {
        setEvents((prev) => [...prev.slice(-499), event]);
      }
    });

    return () => {
      unsubscribe();
      socket.disconnect();
      socketRef.current = null;
    };
  }, []);

  const connectRoom = useCallback((username: string) => {
    setEvents([]);
    setViewerCount(null);
    setStatus({ state: "connecting", username });
    socketRef.current?.connectRoom(username);
  }, []);

  const disconnectRoom = useCallback(() => {
    setEvents([]);
    setViewerCount(null);
    setStatus({ state: "idle" });
    socketRef.current?.disconnectRoom();
  }, []);

  const roomInfo = useMemo<RoomInfo | null>(() => {
    if (status.state === "connected") {
      return (status.roomInfo as RoomInfo | null) ?? null;
    }
    return null;
  }, [status]);

  return (
    <div className="min-h-screen bg-ink text-zinc-100">
      <header className="sticky top-0 z-10 border-b border-edge bg-ink/90 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-2">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-ttcyan to-ttred text-lg font-bold text-white">
              T
            </span>
            <div>
              <h1 className="text-lg font-bold leading-none">TikTok Bar</h1>
              <p className="text-xs text-zinc-500">LIVE event monitor</p>
            </div>
          </div>
          <ConnectBar
            connected={connected}
            connecting={connecting}
            onConnect={connectRoom}
            onDisconnect={disconnectRoom}
          />
        </div>
      </header>

      <main className="mx-auto grid max-w-6xl grid-cols-1 gap-4 p-4 lg:grid-cols-3">
        <RoomCard status={status} roomInfo={roomInfo} viewerCount={viewerCount} />
        <EventFeed events={events} status={status} />
      </main>
    </div>
  );
}
