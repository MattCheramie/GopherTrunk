import { useEffect, useRef } from "react";

interface Props {
  title: string;
  subtitle?: string;
  onClose: () => void;
  children: React.ReactNode;
}

// DetailModal renders a right-side sheet on desktop and a bottom-sheet
// on phones. Mirrors internal/tui/detail.go in spirit: read-only deep
// view for a single row.
export function DetailModal({ title, subtitle, onClose, children }: Props) {
  const dialogRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  useEffect(() => {
    dialogRef.current?.focus();
  }, []);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={title}
      className="fixed inset-0 z-50 flex items-end sm:items-stretch sm:justify-end bg-black/50 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        tabIndex={-1}
        onClick={(e) => e.stopPropagation()}
        className="panel w-full sm:max-w-md sm:h-full bg-bg p-4 sm:p-5 overflow-auto rounded-t-lg sm:rounded-none sm:rounded-l-lg outline-none"
      >
        <header className="flex items-start gap-3 mb-4">
          <div className="flex-1">
            <h3 className="text-lg font-semibold leading-tight">{title}</h3>
            {subtitle && (
              <p className="text-xs text-muted mt-0.5">{subtitle}</p>
            )}
          </div>
          <button
            className="btn-ghost !min-h-0 !p-1.5 text-xs"
            onClick={onClose}
            aria-label="Close detail"
          >
            ✕
          </button>
        </header>

        <div className="space-y-3">{children}</div>
      </div>
    </div>
  );
}

// DetailField renders one label/value pair inside a DetailModal.
// Pass `emptyHint` to replace the default em-dash with context-specific
// copy (e.g. "Awaiting status broadcasts") when the value is missing.
export function DetailField({
  label,
  value,
  mono,
  emptyHint,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  emptyHint?: React.ReactNode;
}) {
  const empty = value === null || value === undefined || value === "";
  return (
    <div>
      <p className="text-xs uppercase tracking-wider text-muted">{label}</p>
      <p className={`text-sm mt-0.5 ${mono && !empty ? "font-mono" : ""}`}>
        {empty ? (
          emptyHint != null ? (
            <span className="text-muted italic">{emptyHint}</span>
          ) : (
            <span className="text-muted">—</span>
          )
        ) : (
          value
        )}
      </p>
    </div>
  );
}
