import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useVisibleNavItems, type NavItem } from "../nav/registry";
import { useFocusTrap } from "../hooks/useFocusTrap";

interface Props {
  open: boolean;
  onClose: () => void;
}

// CommandPalette: a ⌘K / Ctrl-K fuzzy quick-switcher over every nav
// item. The fastest path to any panel on any device, and a discovery
// safety net for the deeper panels. Arrow keys move the selection,
// Enter activates it, Escape closes.
export function CommandPalette({ open, onClose }: Props) {
  const items = useVisibleNavItems();
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const panelRef = useFocusTrap<HTMLDivElement>(open);

  const results = useMemo(() => filterItems(items, query), [items, query]);

  // Reset query/selection each time the palette opens, and focus the
  // input so typing works immediately.
  useEffect(() => {
    if (!open) return;
    setQuery("");
    setActive(0);
    // Focus the input after the focus-trap's initial focus runs.
    const id = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => window.clearTimeout(id);
  }, [open]);

  useEffect(() => {
    setActive(0);
  }, [query]);

  if (!open) return null;

  function choose(item: NavItem | undefined) {
    if (!item) return;
    onClose();
    if (item.external) {
      window.open(item.to, "_blank", "noopener");
    } else {
      navigate(item.to);
    }
  }

  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => Math.min(a + 1, results.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => Math.max(a - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      choose(results[active]);
    }
  }

  return (
    <div
      className="fixed inset-0 z-[70] flex items-start justify-center bg-black/50 backdrop-blur-sm p-4 pt-[10vh]"
      onClick={onClose}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={onKeyDown}
        className="panel w-full max-w-lg bg-bg overflow-hidden outline-none"
      >
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Jump to a panel…"
          aria-label="Search panels"
          autoCapitalize="off"
          autoCorrect="off"
          spellCheck={false}
          className="w-full bg-transparent px-4 py-3 text-sm border-b border-panel focus:outline-none placeholder-muted"
        />
        <ul className="max-h-[50vh] overflow-y-auto py-1" role="listbox">
          {results.length === 0 && (
            <li className="px-4 py-3 text-sm text-muted">No panels match.</li>
          )}
          {results.map((item, i) => (
            <li key={item.to} role="option" aria-selected={i === active}>
              <button
                type="button"
                onClick={() => choose(item)}
                onMouseEnter={() => setActive(i)}
                className={[
                  "w-full flex items-center gap-3 px-4 py-2 text-sm text-left",
                  i === active ? "bg-accent/15 text-accent" : "text-fg",
                ].join(" ")}
              >
                <span className="text-base leading-none" aria-hidden>
                  {item.icon}
                </span>
                <span className="flex-1 truncate">{item.label}</span>
                {item.external && (
                  <span className="text-xs text-muted" aria-hidden>
                    ↗
                  </span>
                )}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

// filterItems ranks nav items against a query: substring matches on the
// label rank above keyword matches. Empty query returns everything in
// registry order.
function filterItems(items: NavItem[], query: string): NavItem[] {
  const q = query.trim().toLowerCase();
  if (!q) return items;
  const scored: { item: NavItem; score: number }[] = [];
  for (const item of items) {
    const label = item.label.toLowerCase();
    let score = -1;
    if (label.startsWith(q)) score = 3;
    else if (label.includes(q)) score = 2;
    else if (item.keywords?.some((k) => k.toLowerCase().includes(q))) score = 1;
    if (score >= 0) scored.push({ item, score });
  }
  scored.sort((a, b) => b.score - a.score);
  return scored.map((s) => s.item);
}
