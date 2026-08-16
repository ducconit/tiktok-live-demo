import { useQuery } from "@tanstack/vue-query";
import { fetchRoomPreview, type RoomPreview } from "@/services/api";

export function useRoomPreview(username: () => string | null) {
  return useQuery<RoomPreview, Error>({
    queryKey: ["room-preview", username],
    queryFn: () => fetchRoomPreview(username() ?? ""),
    enabled: () => !!username() && !username()!.includes(" "),
    staleTime: 30_000,
    retry: 1,
  });
}
