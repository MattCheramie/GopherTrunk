import { NavLink, type To } from "react-router-dom";

export interface Tab {
  to: To;
  label: string;
  icon: string;
}

interface Props {
  tabs: Tab[];
}

// TabBar: horizontal strip on desktop, fixed bottom nav on mobile.
// The same component renders both — Tailwind's responsive variants
// switch the layout at the `sm` breakpoint.
export function TabBar({ tabs }: Props) {
  return (
    <>
      {/* Desktop / tablet: top tabs */}
      <nav
        aria-label="Primary"
        className="hidden sm:flex sticky top-0 z-20 bg-bg/95 backdrop-blur border-b border-panel px-3 pt-safe-top"
      >
        <ul className="flex gap-1 overflow-x-auto">
          {tabs.map((t) => (
            <li key={String(t.to)}>
              <NavLink
                to={t.to}
                className={({ isActive }) =>
                  [
                    "inline-flex items-center gap-2 px-3 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                    isActive
                      ? "text-accent border-accent"
                      : "text-muted border-transparent hover:text-fg",
                  ].join(" ")
                }
              >
                <span aria-hidden>{t.icon}</span>
                <span>{t.label}</span>
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      {/* Mobile: bottom nav */}
      <nav
        aria-label="Primary"
        className="sm:hidden fixed bottom-0 inset-x-0 z-20 bg-bg/95 backdrop-blur border-t border-panel pb-safe-bottom"
      >
        <ul className="grid grid-cols-4">
          {tabs.slice(0, 4).map((t) => (
            <li key={String(t.to)}>
              <NavLink
                to={t.to}
                className={({ isActive }) =>
                  [
                    "flex flex-col items-center gap-0.5 py-2 text-xs font-medium",
                    isActive ? "text-accent" : "text-muted",
                  ].join(" ")
                }
              >
                <span className="text-xl leading-none" aria-hidden>
                  {t.icon}
                </span>
                <span>{t.label}</span>
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
    </>
  );
}
