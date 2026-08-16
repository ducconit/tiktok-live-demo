import { describe, it, expect, beforeEach, vi } from "vitest";
import { LiveSocket } from "./socket";
import type { LiveEvent } from "@/types";

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState: number = MockWebSocket.CONNECTING;
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  sent: string[] = [];

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  }
}

beforeEach(() => {
  MockWebSocket.instances = [];
  (globalThis as unknown as { WebSocket: unknown }).WebSocket = MockWebSocket;
});

describe("LiveSocket", () => {
  it("queues the connect command until the socket is open", () => {
    const s = new LiveSocket();
    s.connectRoom("nhu2hand2");

    const ws = MockWebSocket.instances[0];
    expect(ws).toBeDefined();
    expect(ws.sent).toHaveLength(0); // queued, not sent yet

    ws.readyState = MockWebSocket.OPEN;
    ws.onopen?.();
    expect(ws.sent).toHaveLength(1);
    expect(JSON.parse(ws.sent[0])).toEqual({ action: "connect", username: "nhu2hand2" });
  });

  it("disconnectRoom flushes the command then closes the socket", () => {
    const s = new LiveSocket();
    s.connectRoom("user1");
    const ws = MockWebSocket.instances[0];
    ws.readyState = MockWebSocket.OPEN;
    ws.onopen?.();

    s.disconnectRoom();

    expect(ws.sent).toContain(JSON.stringify({ action: "disconnect" }));
    expect(ws.readyState).toBe(MockWebSocket.CLOSED);
  });

  it("delivers events to subscribers", () => {
    const s = new LiveSocket();
    const handler = vi.fn();
    s.onEvent(handler);

    s.connectRoom("user1");
    const ws = MockWebSocket.instances[0];
    ws.readyState = MockWebSocket.OPEN;
    ws.onmessage?.({ data: JSON.stringify({ type: "chat", data: { comment: "hi" }, ts: 1 }) });

    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({ type: "chat" }) as LiveEvent,
    );
  });

  it("disconnect() (unmount) clears queue and closes the socket", () => {
    const s = new LiveSocket();
    s.connectRoom("user1");
    const ws = MockWebSocket.instances[0];

    s.disconnect();
    expect(ws.readyState).toBe(MockWebSocket.CLOSED);
  });
});
