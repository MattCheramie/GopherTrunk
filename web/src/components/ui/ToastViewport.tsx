import { useShared, type ToastKind } from "../../store/shared";

// ToastViewport renders the stack of transient notifications. It sits
// once at the app root (in AppShell) and replaces the old single
// `lastError` strip. Success/info use a polite live region; errors and
// warnings are assertive so screen readers announce them promptly.
const KIND_CLASS: Record<ToastKind, string> = {
  success: "bg-ok/15 border-ok/40 text-ok",
  info: "bg-accent/15 border-accent/40 text-accent",
  warning: "bg-warn/15 border-warn/40 text-warn",
  error: "bg-err/15 border-err/40 text-err",
};

const KIND_ICON: Record<ToastKind, string> = {
  success: "✓",
  info: "ℹ",
  warning: "!",
  error: "✕",
};

export function ToastViewport() {
  const toasts = useShared((s) => s.toasts);
  const dismissToast = useShared((s) => s.dismissToast);

  if (toasts.length === 0) return null;

  return (
    <div
      className="fixed z-[80] bottom-20 sm:bottom-3 left-3 right-3 sm:left-auto sm:max-w-sm flex flex-col gap-2 pointer-events-none"
      aria-live="polite"
    >
      {toasts.map((t) => (
        <div
          key={t.id}
          role={t.kind === "error" || t.kind === "warning" ? "alert" : "status"}
          className={`panel border p-3 text-sm flex items-start gap-2 pointer-events-auto ${KIND_CLASS[t.kind]}`}
        >
          <span aria-hidden className="font-semibold leading-5">
            {KIND_ICON[t.kind]}
          </span>
          <span className="flex-1 break-words">{t.message}</span>
          <button
            type="button"
            className="text-xs underline shrink-0"
            onClick={() => dismissToast(t.id)}
            aria-label="Dismiss notification"
          >
            dismiss
          </button>
        </div>
      ))}
    </div>
  );
}
