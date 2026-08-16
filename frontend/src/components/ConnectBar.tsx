import { useState, type FormEvent } from "react";

interface Props {
  connected: boolean;
  connecting: boolean;
  onConnect: (username: string) => void;
  onDisconnect: () => void;
}

export function ConnectBar({ connected, connecting, onConnect, onDisconnect }: Props) {
  const [value, setValue] = useState("");

  const submit = () => {
    if (!value.trim() || connected || connecting) return;
    onConnect(value.trim());
  };

  // Form submit is ONLY used for the Enter key (preventDefault stops the
  // native submission). Both buttons are type="button" so clicking Dừng never
  // triggers a form submission — otherwise React swapping the button's type to
  // "submit" mid-click makes the browser submit + auto-reconnect.
  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    submit();
  };

  return (
    <form onSubmit={handleSubmit} className="flex w-full max-w-sm items-center gap-2">
      <div className="relative flex-1">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500">
          @
        </span>
        <input
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="tiktok username"
          disabled={connecting}
          className="w-full rounded-lg border border-edge bg-panel py-2 pl-7 pr-3 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-ttcyan focus:outline-none focus:ring-1 focus:ring-ttcyan disabled:opacity-50"
        />
      </div>
      {connected ? (
        <button
          type="button"
          onClick={onDisconnect}
          className="shrink-0 rounded-lg bg-ttred px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90"
        >
          Dừng
        </button>
      ) : (
        <button
          type="button"
          onClick={submit}
          disabled={connecting || !value.trim()}
          className="shrink-0 rounded-lg bg-ttcyan px-4 py-2 text-sm font-medium text-black transition-opacity hover:opacity-90 disabled:opacity-40"
        >
          {connecting ? "Đang kết nối…" : "Kết nối"}
        </button>
      )}
    </form>
  );
}
