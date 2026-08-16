import axios from "axios";
import type { Envelope } from "@/services/api";
import type { RoomInfo } from "@/types";

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? "",
  timeout: 20_000,
});

export { api };

export interface ConnectResult {
  connected: boolean;
  roomId?: string;
  roomInfo?: RoomInfo | null;
  error?: string;
}

interface ConnectData {
  connected: true;
  roomId: string;
  roomInfo: RoomInfo | null;
}

// Bắt đầu track (backend connect TikTok) — events publish lên Sockudo
// channel "user_<username>". POST /api/v1/public/live/{username}/connect.
export async function connectRoom(username: string): Promise<ConnectResult> {
  try {
    const { data } = await api.post<Envelope<ConnectData>>(
      `/api/v1/public/live/${encodeURIComponent(username)}/connect`,
    );
    return data.data;
  } catch (e) {
    // Envelope lỗi: msg đã được backend dịch (i18n theo Accept-Language).
    const msg = (e as { response?: { data?: { msg?: string } } }).response?.data?.msg;
    return { connected: false, error: msg ?? "Lỗi kết nối" };
  }
}

// Dừng track — POST /api/v1/public/live/{username}/disconnect.
export async function disconnectRoom(username: string): Promise<void> {
  try {
    await api.post(`/api/v1/public/live/${encodeURIComponent(username)}/disconnect`);
  } catch {
    // ignore
  }
}
