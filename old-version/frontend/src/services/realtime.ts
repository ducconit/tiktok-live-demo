import axios from "axios";
import type { RoomInfo } from "@/types";

const api = axios.create({
  baseURL: import.meta.env.VITE_SERVER_URL ?? "",
  timeout: 20_000,
});

export { api };

export interface ConnectResult {
  connected: boolean;
  roomId?: string;
  roomInfo?: RoomInfo | null;
  error?: string;
}

// Bắt đầu track (Go server connect TikTok) — events sẽ được publish lên Sockudo.
export async function connectRoom(username: string): Promise<ConnectResult> {
  try {
    const { data } = await api.post<ConnectResult>("/api/connect", { username });
    return data;
  } catch (e) {
    const msg = (e as { response?: { data?: { error?: string } } }).response?.data?.error;
    return { connected: false, error: msg ?? "Lỗi kết nối" };
  }
}

// Dừng track.
export async function disconnectRoom(username: string): Promise<void> {
  try {
    await api.post("/api/disconnect", { username });
  } catch {
    // ignore
  }
}
