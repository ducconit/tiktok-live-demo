import type { RoomInfo, StatusData } from "../types";

interface Props {
  status: StatusData;
  roomInfo: RoomInfo | null;
  viewerCount: number | null;
}

const STATE_META: Record<string, { label: string; dot: string }> = {
  idle: { label: "Chưa kết nối", dot: "bg-zinc-500" },
  connecting: { label: "Đang kết nối…", dot: "bg-amber-400 animate-pulse" },
  connected: { label: "Đang theo dõi LIVE", dot: "bg-emerald-400 animate-pulse" },
  disconnected: { label: "Đã ngắt kết nối", dot: "bg-zinc-500" },
  error: { label: "Lỗi", dot: "bg-ttred" },
  ended: { label: "Stream đã kết thúc", dot: "bg-zinc-500" },
};

export function RoomCard({ status, roomInfo, viewerCount }: Props) {
  const meta = STATE_META[status.state] ?? STATE_META.idle;
  const owner = roomInfo?.owner;
  const avatar = owner?.profilePictureUrl;
  const nickname = owner?.nickname ?? owner?.uniqueId ?? "—";
  const title = roomInfo?.title ?? "Chưa có thông tin phòng";
  const resolvedViewers = viewerCount ?? roomInfo?.viewerCount ?? roomInfo?.userCount ?? null;

  return (
    <aside className="flex flex-col gap-4 lg:col-span-1">
      <section className="rounded-xl border border-edge bg-panel p-5">
        <div className="flex items-center gap-3">
          <div className="relative">
            {avatar ? (
              <img
                src={avatar}
                alt={nickname}
                className="h-16 w-16 rounded-full border border-edge object-cover"
              />
            ) : (
              <div className="flex h-16 w-16 items-center justify-center rounded-full border border-edge bg-zinc-800 text-2xl text-zinc-500">
                @
              </div>
            )}
            <span
              className={`absolute bottom-0 right-0 h-4 w-4 rounded-full border-2 border-panel ${meta.dot}`}
            />
          </div>
          <div className="min-w-0">
            <h2 className="truncate text-lg font-semibold">{nickname}</h2>
            {owner?.uniqueId && (
              <p className="truncate text-sm text-zinc-500">@{owner.uniqueId}</p>
            )}
          </div>
        </div>

        <div className="mt-4 space-y-2 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-zinc-500">Tiêu đề</span>
            <span className="max-w-[60%] truncate text-zinc-200">{title}</span>
          </div>
          {resolvedViewers !== null && (
            <div className="flex items-center justify-between">
              <span className="text-zinc-500">Người xem</span>
              <span className="font-medium text-ttcyan">{resolvedViewers.toLocaleString()}</span>
            </div>
          )}
          {status.roomId && (
            <div className="flex items-center justify-between">
              <span className="text-zinc-500">Room ID</span>
              <span className="truncate text-zinc-400">{status.roomId}</span>
            </div>
          )}
        </div>

        <div className="mt-4 flex items-center gap-2 rounded-lg border border-edge bg-ink/60 px-3 py-2">
          <span className={`h-2 w-2 rounded-full ${meta.dot}`} />
          <span className="text-xs text-zinc-300">{meta.label}</span>
        </div>
        {status.message && (
          <p className="mt-3 text-xs leading-relaxed text-ttred">{status.message}</p>
        )}
      </section>

      <section className="rounded-xl border border-edge bg-panel p-5 text-sm">
        <h3 className="mb-3 font-semibold text-zinc-300">Cách dùng</h3>
        <ol className="list-decimal space-y-2 pl-5 text-zinc-400">
          <li>Nhập @username của streamer đang LIVE.</li>
          <li>Bấm Kết nối để nhận event real-time.</li>
          <li>Gift, comment, lượt tham gia sẽ hiển thị ở cột bên phải.</li>
        </ol>
      </section>
    </aside>
  );
}
