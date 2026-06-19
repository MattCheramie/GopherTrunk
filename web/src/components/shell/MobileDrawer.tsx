import { useEffect } from "react";
import { NavLink } from "react-router-dom";
import { useVisibleNavGroups, type NavItem } from "../../nav/registry";
import { useFocusTrap } from "../../hooks/useFocusTrap";

interface Props {
  open: boolean;
  onClose: () => void;
}

// MobileDrawer: the full navigation sheet behind the bottom-nav "More"
// button. Lists every group and item so all panels are reachable on a
// phone. Slides in from the left; Escape and a backdrop tap close it.
export function MobileDrawer({ open, onClose }: Props) {
  const groups = useVisibleNavGroups();
  const panelRef = useFocusTrap<HTMLDivElement>(open);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="sm:hidden fixed inset-0 z-50 bg-black/50 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label="Navigation"
        onClick={(e) => e.stopPropagation()}
        className="absolute inset-y-0 left-0 w-[min(20rem,85vw)] bg-bg border-r border-panel overflow-y-auto pt-safe-top pb-safe-bottom outline-none"
      >
        <header className="flex items-center justify-between h-12 px-4 border-b border-panel sticky top-0 bg-bg">
          <span className="text-sm font-semibold tracking-tight">
            GopherTrunk
          </span>
          <button
            type="button"
            onClick={onClose}
            className="btn-ghost !min-h-0 !p-1.5 text-xs"
            aria-label="Close navigation"
          >
            ✕
          </button>
        </header>

        <div className="px-3 py-3 space-y-4">
          {groups.map((g) => (
            <div key={g.id}>
              <p className="px-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted">
                {g.label}
              </p>
              <ul className="space-y-0.5">
                {g.items.map((item) => (
                  <li key={item.to}>
                    <DrawerLink item={item} onNavigate={onClose} />
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function DrawerLink({
  item,
  onNavigate,
}: {
  item: NavItem;
  onNavigate: () => void;
}) {
  const base =
    "flex items-center gap-3 rounded-md px-2 py-2 text-sm font-medium";

  if (item.external) {
    return (
      <a
        href={item.to}
        target="_blank"
        rel="noopener"
        onClick={onNavigate}
        className={`${base} text-muted hover:text-fg hover:bg-panel`}
      >
        <span className="text-base leading-none" aria-hidden>
          {item.icon}
        </span>
        <span className="truncate">{item.label}</span>
        <span className="ml-auto text-xs" aria-hidden>
          ↗
        </span>
      </a>
    );
  }

  return (
    <NavLink
      to={item.to}
      onClick={onNavigate}
      className={({ isActive }) =>
        [
          base,
          isActive
            ? "bg-accent/15 text-accent"
            : "text-muted hover:text-fg hover:bg-panel",
        ].join(" ")
      }
    >
      <span className="text-base leading-none" aria-hidden>
        {item.icon}
      </span>
      <span className="truncate">{item.label}</span>
    </NavLink>
  );
}
