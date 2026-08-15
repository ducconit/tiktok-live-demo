import type { LiveEvent } from "./types";

export class LiveSocket {
  private ws: WebSocket | null = null;
  private listeners = new Set<(event: LiveEvent) => void>();
  private url: string;
  private queue: unknown[] = [];

  constructor() {
    const base =
      import.meta.env.VITE_SERVER_URL ??
      (typeof window !== "undefined" ? window.location.origin : "http://localhost:3001");
    const host = base.replace(/^https?:\/\//, "");
    const proto = base.startsWith("https") ? "wss" : "ws";
    this.url = `${proto}://${host}/ws`;
  }

  private open(): void {
    if (this.ws) return;
    const ws = new WebSocket(this.url);
    this.ws = ws;

    ws.onopen = () => {
      while (this.queue.length > 0) {
        ws.send(JSON.stringify(this.queue.shift()));
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
    };
  }

  onEvent(fn: (event: LiveEvent) => void): () => void {
    this.listeners.add(fn);
    return () => {
      this.listeners.delete(fn);
    };
  }

  connectRoom(username: string): void {
    this.open();
    this.send({ action: "connect", username });
  }

  disconnectRoom(): void {
    // Send the disconnect command and let the server stop the tracker.
    // Do NOT close the socket here — closing immediately after send can drop
    // the message before it flushes, so the server never sees "disconnect".
    this.send({ action: "disconnect" });
  }

  disconnect(): void {
    this.queue = [];
    this.ws?.close();
    this.ws = null;
  }

  private send(obj: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(obj));
      return;
    }
    this.queue.push(obj);
  }
}
