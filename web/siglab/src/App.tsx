import { useEffect } from "react";
import { NavLink, Navigate, Route, Routes } from "react-router-dom";
import { useStore } from "./store/shared";
import { Captures } from "./panels/Captures";
import { Results } from "./panels/Results";
import { Synthesize } from "./panels/Synthesize";
import { Identify } from "./panels/Identify";
import { Wideband } from "./panels/Wideband";
import { Compare } from "./panels/Compare";

const tabs = [
  { to: "/captures", label: "Captures" },
  { to: "/synthesize", label: "Synthesize" },
  { to: "/identify", label: "Identify" },
  { to: "/wideband", label: "Wideband" },
  { to: "/compare", label: "Compare" },
];

export function App() {
  const loadProtocols = useStore((s) => s.loadProtocols);

  useEffect(() => {
    loadProtocols().catch(() => {
      /* surfaced lazily in the forms */
    });
  }, [loadProtocols]);

  return (
    <div className="flex min-h-full flex-col">
      <header className="flex items-center gap-4 border-b border-line bg-panel px-4 py-3">
        <div className="flex items-center gap-2 font-semibold">
          <span className="text-accent">◷</span> Signal Lab
        </div>
        <nav className="flex gap-1 text-sm">
          {tabs.map((t) => (
            <NavLink
              key={t.to}
              to={t.to}
              className={({ isActive }) =>
                `rounded-md px-3 py-1.5 ${
                  isActive ? "bg-accent/20 text-accent" : "text-muted hover:text-fg"
                }`
              }
            >
              {t.label}
            </NavLink>
          ))}
        </nav>
        <div className="ml-auto text-xs text-muted">offline signal analysis</div>
      </header>

      <main className="flex-1 p-4">
        <Routes>
          <Route path="/" element={<Navigate to="/captures" replace />} />
          <Route path="/captures" element={<Captures />} />
          <Route path="/results/:captureId" element={<Results />} />
          <Route path="/synthesize" element={<Synthesize />} />
          <Route path="/identify" element={<Identify />} />
          <Route path="/wideband" element={<Wideband />} />
          <Route path="/compare" element={<Compare />} />
          <Route path="*" element={<Navigate to="/captures" replace />} />
        </Routes>
      </main>
    </div>
  );
}
