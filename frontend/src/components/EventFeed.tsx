import { useEffect, useRef } from "react";
import type { LiveEvent, StatusData } from "../types";
import { EventRow } from "./EventRow";

interface Props {
  events: LiveEvent[];
  status: StatusData;
}

export function EventFeed({ events, status }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [events.length]);

  return (
    <section className="flex flex-col rounded-xl border border-edge bg-panel lg:col-span-2">
      <header className="flex items-center justify-between border-b border-edge px-5 py-3">
        <h2 className="font-semibold">Event feed</h2>
        <span className="rounded-full bg-ink/60 px-2.5 py-0.5 text-xs text-zinc-500">
          {events.length} sự kiện
        </span>
      </header>

      <div className="h-[60vh] overflow-y-auto p-4 lg:h-[70vh]">
        {events.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-zinc-600">
            <span className="text-4xl">🎁</span>
            <p className="max-w-xs text-sm">
              Chưa có sự kiện. Kết nối một phòng LIVE để nhận gift, comment và lượt tham gia.
            </p>
          </div>
        ) : (
          <ul className="space-y-2">
            {events.map((event, index) => (
              <EventRow key={`${event.ts}-${index}`} event={event} />
            ))}
          </ul>
        )}
        <div ref={bottomRef} />
      </div>

      {status.state === "ended" && (
        <footer className="border-t border-edge px-5 py-3 text-center text-sm text-zinc-500">
          Stream đã kết thúc.
        </footer>
      )}
    </section>
  );
}
