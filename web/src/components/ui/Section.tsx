import { useId, useState, type ReactNode } from "react";
import { prefs } from "../../store/prefs";

interface Props {
  /** Stable id used to persist the open/closed state across visits. */
  id: string;
  title: ReactNode;
  /** Optional muted description under the title. */
  description?: ReactNode;
  /** Optional right-aligned header content (status, count). */
  actions?: ReactNode;
  defaultCollapsed?: boolean;
  children: ReactNode;
  className?: string;
}

// Section is a titled, collapsible group whose open/closed state
// persists per id. Used to break the dense Hunt/Scanner/Settings forms
// into scannable, foldable groups (Phase 5).
export function Section({
  id,
  title,
  description,
  actions,
  defaultCollapsed = false,
  children,
  className = "",
}: Props) {
  const [collapsed, setCollapsed] = useState(() =>
    prefs.sectionCollapsed(id, defaultCollapsed),
  );
  const bodyId = useId();

  function toggle() {
    setCollapsed((c) => {
      const next = !c;
      prefs.setSectionCollapsed(id, next);
      return next;
    });
  }

  return (
    <section className={`card ${className}`}>
      <header className="flex items-center gap-3 p-3 sm:p-4">
        <button
          type="button"
          onClick={toggle}
          aria-expanded={!collapsed}
          aria-controls={bodyId}
          className="flex items-center gap-2 flex-1 text-left"
        >
          <span
            aria-hidden
            className={`text-muted transition-transform ${collapsed ? "" : "rotate-90"}`}
          >
            ▶
          </span>
          <span className="min-w-0">
            <span className="block font-medium">{title}</span>
            {description && (
              <span className="block text-xs text-muted">{description}</span>
            )}
          </span>
        </button>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </header>
      {!collapsed && (
        <div id={bodyId} className="px-3 sm:px-4 pb-4 pt-0 space-y-3">
          {children}
        </div>
      )}
    </section>
  );
}
