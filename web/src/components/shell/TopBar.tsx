import { useLocation } from "react-router-dom";
import { NAV_ITEMS } from "../../nav/registry";
import { useShared } from "../../store/shared";

interface Props {
  onOpenMore: () => void;
  onOpenPalette: () => void;
}

// TopBar: the sticky application header. On mobile it carries the
// hamburger that opens the nav drawer; on every device it shows the
// current page title, a live connection indicator, and the command
// palette trigger.
export function TopBar({ onOpenMore, onOpenPalette }: Props) {
  const { pathname } = useLocation();
  const wsStatus = useShared((s) => s.wsStatus);

  const current =
    NAV_ITEMS.find((i) => !i.external && i.to === pathname) ??
    NAV_ITEMS.find((i) => !i.external && pathname.startsWith(i.to + "/"));
  const title = current?.label ?? "GopherTrunk";

  const live = wsStatus === "open";
  const statusLabel = live
    ? "Live"
    : wsStatus === "connecting"
      ? "Connecting"
      : "Offline";

  return (
    <header className="sticky top-0 z-20 flex items-center gap-2 h-12 px-3 border-b border-panel bg-bg/95 backdrop-blur pt-safe-top">
      <button
        type="button"
        onClick={onOpenMore}
        className="sm:hidden text-muted hover:text-fg p-1 -ml-1 rounded"
        aria-label="Open navigation menu"
      >
        <span className="text-xl leading-none" aria-hidden>
          ☰
        </span>
      </button>

      <h1 className="flex-1 text-base font-semibold tracking-tight truncate">
        {title}
      </h1>

      <span
        className="flex items-center gap-1.5 text-xs text-muted"
        aria-live="polite"
        title={`Event stream: ${statusLabel}`}
      >
        <span
          aria-hidden
          className={[
            "inline-block h-2 w-2 rounded-full",
            live ? "bg-ok" : wsStatus === "connecting" ? "bg-warn" : "bg-err",
          ].join(" ")}
        />
        <span className="hidden sm:inline">{statusLabel}</span>
      </span>

      <button
        type="button"
        onClick={onOpenPalette}
        className="flex items-center gap-2 rounded-md border border-panel px-2 py-1 text-xs text-muted hover:text-fg"
        aria-label="Open command palette"
      >
        <span aria-hidden>🔎</span>
        <span className="hidden sm:inline">Jump to…</span>
        <kbd className="hidden sm:inline text-[10px] border border-panel rounded px-1">
          ⌘K
        </kbd>
      </button>
    </header>
  );
}
