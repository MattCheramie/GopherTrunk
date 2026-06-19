import { NavLink } from "react-router-dom";
import { useVisibleNavGroups, type NavItem } from "../../nav/registry";

interface Props {
  collapsed: boolean;
  onToggleCollapse: () => void;
}

// Sidebar: the desktop (sm+) primary navigation. A persistent, grouped,
// collapsible rail driven by the nav registry. Collapsed, it shows
// icon-only buttons with the label as a hover/focus tooltip; expanded,
// it shows grouped sections with text labels.
export function Sidebar({ collapsed, onToggleCollapse }: Props) {
  const groups = useVisibleNavGroups();

  return (
    <nav
      aria-label="Primary"
      className={[
        "hidden sm:flex flex-col shrink-0 border-r border-panel bg-bg/95",
        "sticky top-0 h-screen overflow-y-auto pt-safe-top",
        collapsed ? "w-16" : "w-56",
      ].join(" ")}
    >
      <div
        className={[
          "flex items-center h-12 px-3 shrink-0",
          collapsed ? "justify-center" : "justify-between",
        ].join(" ")}
      >
        {!collapsed && (
          <span className="text-sm font-semibold tracking-tight">
            GopherTrunk
          </span>
        )}
        <button
          type="button"
          onClick={onToggleCollapse}
          className="text-muted hover:text-fg p-1 rounded"
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-pressed={collapsed}
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          <span aria-hidden>{collapsed ? "»" : "«"}</span>
        </button>
      </div>

      <div className="flex-1 px-2 pb-4 space-y-4">
        {groups.map((g) => (
          <div key={g.id}>
            {!collapsed && (
              <p className="px-2 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted">
                {g.label}
              </p>
            )}
            <ul className="space-y-0.5">
              {g.items.map((item) => (
                <li key={item.to}>
                  <SidebarLink item={item} collapsed={collapsed} />
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </nav>
  );
}

function SidebarLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  const base = [
    "flex items-center gap-3 rounded-md px-2 py-2 text-sm font-medium transition-colors",
    collapsed ? "justify-center" : "",
  ].join(" ");

  const content = (
    <>
      <span className="text-base leading-none" aria-hidden>
        {item.icon}
      </span>
      {!collapsed && <span className="truncate">{item.label}</span>}
      {!collapsed && item.external && (
        <span className="ml-auto text-xs text-muted" aria-hidden>
          ↗
        </span>
      )}
    </>
  );

  // The Config Builder is a separate SPA; open it in a new tab rather
  // than routing so the live operator session isn't torn down.
  if (item.external) {
    return (
      <a
        href={item.to}
        target="_blank"
        rel="noopener"
        className={`${base} text-muted hover:text-fg hover:bg-panel`}
        title={collapsed ? `${item.label} (opens new tab)` : undefined}
        aria-label={collapsed ? `${item.label} (opens new tab)` : undefined}
      >
        {content}
      </a>
    );
  }

  return (
    <NavLink
      to={item.to}
      className={({ isActive }) =>
        [
          base,
          isActive
            ? "bg-accent/15 text-accent"
            : "text-muted hover:text-fg hover:bg-panel",
        ].join(" ")
      }
      title={collapsed ? item.label : undefined}
      aria-label={collapsed ? item.label : undefined}
    >
      {content}
    </NavLink>
  );
}
