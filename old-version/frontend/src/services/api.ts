import axios from "axios";

export interface RoomPreview {
  live: boolean;
  roomId?: string;
  title?: string;
  userCount?: number;
  owner?: { uniqueId?: string; nickname?: string; profilePictureUrl?: string };
}

const api = axios.create({
  baseURL: import.meta.env.VITE_SERVER_URL ?? "",
  timeout: 10_000,
});

export { api };

export async function fetchRoomPreview(username: string): Promise<RoomPreview> {
  const { data } = await api.get<RoomPreview>(`/api/room/${encodeURIComponent(username)}`);
  return data;
}
