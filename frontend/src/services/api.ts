import axios from "axios";

// Envelope chuẩn backend: { code, msg, data, meta } — code "0" = thành công.
export interface Envelope<T> {
  code: string;
  msg: string;
  data: T;
  meta: Record<string, unknown>;
}

export interface RoomPreview {
  live: boolean;
  roomId?: string;
  title?: string;
  userCount?: number;
  owner?: { uniqueId?: string; nickname?: string; profilePictureUrl?: string };
}

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? "",
  timeout: 10_000,
});

export { api };

// Room preview — GET /api/v1/public/live/{username} (không cần connect).
export async function fetchRoomPreview(username: string): Promise<RoomPreview> {
  const { data } = await api.get<Envelope<RoomPreview>>(
    `/api/v1/public/live/${encodeURIComponent(username)}`,
  );
  return data.data;
}
