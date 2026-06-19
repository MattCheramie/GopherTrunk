import { NavLink } from "react-router-dom";
import { PRIMARY_ITEMS, tabKey } from "../../nav/registry";
import { useShared } from "../../store/shared";

interface Props {
  onOpenMore: () => void;
}

// BottomNav: mobile-only (< sm) primary navigation. Four registry
// primaries plus a fifth "More" button that opens the full drawer, so
// every panel is reachable on a phone (the pre-overhaul nav only
// rendered the first four tabs and stranded the other 21).
export function BottomNav({ onOpenMore }: Props) {
  const hiddenTabs = useShared((s) => s.hiddenTabs);
  const hidden = new Set(hiddenTabs);
  const items = PRIMARY_ITEMS.filter((i) => !hidden.has(tabKey(i)));

  return (
    <nav
      aria-label="Primary"
      className="sm:hidden fixed bottom-0 inset-x-0 z-20 bg-bg/95 backdrop-blur border-t border-panel pb-safe-bottom"
    >
      <ul className="grid grid-cols-5">
        {items.map((item) => (
          <li key={item.to}>
            <NavLink
              to={item.to}
              className={({ isActive }) =>
                [
                  "flex flex-col items-center gap-0.5 py-2 text-xs font-medium",
                  isActive ? "text-accent" : "text-muted",
                ].join(" ")
              }
            >
              <span className="text-xl leading-none" aria-hidden>
                {item.icon}
              </span>
              <span>{item.label}</span>
            </NavLink>
          </li>
        ))}
        <li>
          <button
            type="button"
            onClick={onOpenMore}
            className="w-full flex flex-col items-center gap-0.5 py-2 text-xs font-medium text-muted"
            aria-label="More — open navigation menu"
          >
            <span className="text-xl leading-none" aria-hidden>
              ☰
            </span>
            <span>More</span>
          </button>
        </li>
      </ul>
    </nav>
  );
}
