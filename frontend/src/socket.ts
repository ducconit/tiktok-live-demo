import type { LiveEvent } from "./types";

export class LiveSocket {
  private ws: WebSocket | null = null;
  private listeners = new Set<(event: LiveEvent) => void>();
  private url: string;
  private queue: unknown[] = [];
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closedByUser = false;

  constructor() {
    const base =
      import.meta.env.VITE_SERVER_URL ??
      (typeof window !== "undefined" ? window.location.origin : "http://localhost:3001");
    const host = base.replace(/^https?:\/\//, "");
    const proto = base.startsWith("https") ? "wss" : "ws";
    this.url = `${proto}://${host}/ws`;
  }

  connect(): void {
    this.closedByUser = false;
    this.open();
  }

  private open(): void {
    if (this.ws) return;
    const ws = new WebSocket(this.url);
    this.ws = ws;

    ws.onopen = () => {
      while (this.queue.length > 0) {
        const msg = this.queue.shift();
        if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg));
      }
    };

    ws.onmessage = (msg) => {
      try {
        const event = JSON.parse(msg.data as string) as LiveEvent;
        this.listeners.forEach((fn) => fn(event));
      } catch {
        // ignore malformed frames
      }
    };

    ws.onclose = () => {
      this.ws = null;
      if (!this.closedByUser) this.scheduleReconnect();
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.open();
    }, 1500);
  }

  onEvent(fn: (event: LiveEvent) => void): () => void {
    this.listeners.add(fn);
    return () => {
      this.listeners.delete(fn);
    };
  }

  connectRoom(username: string): void {
    this.send({ action: "connect", username });
  }

  disconnectRoom(): void {
    this.send({ action: "disconnect" });
  }

  disconnect(): void {
    this.closedByUser = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.ws?.close();
    this.ws = null;
  }

  private send(obj: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(obj));
      return;
    }
    this.queue.push(obj);
    if (!this.ws) this.open();
  }
}
