import { useEffect, useRef, useState } from "react";

interface Props {
  title: string;
  message: string;
  confirmLabel?: string;
  destructive?: boolean;
  onConfirm: () => Promise<void> | void;
  onCancel: () => void;
}

// ConfirmModal mirrors internal/tui/modal.go: a one-shot confirm
// dialog that wraps a destructive mutation. The caller's onConfirm
// can return a Promise so the modal stays open until the request
// resolves and surfaces failures inline.
export function ConfirmModal({
  title,
  message,
  confirmLabel = "Confirm",
  destructive,
  onConfirm,
  onCancel,
}: Props) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const dialogRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape" && !busy) onCancel();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [busy, onCancel]);

  useEffect(() => {
    dialogRef.current?.focus();
  }, []);

  async function commit() {
    setBusy(true);
    setError(null);
    try {
      await onConfirm();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "request failed");
      setBusy(false);
      return;
    }
    setBusy(false);
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={title}
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={busy ? undefined : onCancel}
    >
      <div
        ref={dialogRef}
        tabIndex={-1}
        onClick={(e) => e.stopPropagation()}
        className="panel w-full max-w-sm bg-bg p-5 outline-none"
      >
        <h3 className="text-lg font-semibold mb-1">{title}</h3>
        <p className="text-sm text-muted mb-4">{message}</p>
        {error && (
          <p className="text-sm text-err mb-3" role="alert">
            {error}
          </p>
        )}
        <div className="flex justify-end gap-2">
          <button
            className="btn-ghost"
            onClick={onCancel}
            disabled={busy}
          >
            Cancel
          </button>
          <button
            className={destructive ? "btn-danger" : "btn-primary"}
            onClick={commit}
            disabled={busy}
          >
            {busy ? "Working…" : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
