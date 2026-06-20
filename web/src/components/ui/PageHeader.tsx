import type { ReactNode } from "react";

interface Props {
  title: ReactNode;
  /** Optional muted subtitle/description under the title. */
  subtitle?: ReactNode;
  /** Right-aligned actions (filters, buttons, status chips, counts). */
  actions?: ReactNode;
  className?: string;
}

// PageHeader standardizes the per-panel `<header><h2>…</h2>…</header>`
// that every panel hand-rolled, so titles, counts, and actions line up
// consistently across the app.
export function PageHeader({ title, subtitle, actions, className = "" }: Props) {
  return (
    <header className={`flex items-start justify-between gap-3 flex-wrap ${className}`}>
      <div className="min-w-0">
        <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
        {subtitle && <p className="text-sm text-muted">{subtitle}</p>}
      </div>
      {actions && (
        <div className="flex items-center gap-2 shrink-0">{actions}</div>
      )}
    </header>
  );
}
