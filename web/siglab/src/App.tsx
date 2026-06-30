import { useEffect, useState } from "react";
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
  // Surface a Crypto Lab link only when the daemon advertises that console
  // (a -tags cryptolab build). Same-origin probe; fails closed when siglab
  // runs standalone or the daemon lacks the console.
  const [cryptolab, setCryptolab] = useState(false);

  useEffect(() => {
    loadProtocols().catch(() => {
      /* surfaced lazily in the forms */
    });
  }, [loadProtocols]);

  useEffect(() => {
    fetch("/api/v1/runtime")
      .then((r) => (r.ok ? r.json() : null))
      .then((rt) => {
        if (rt && rt.cryptolab_console === true) setCryptolab(true);
      })
      .catch(() => {
        /* no daemon / no console — leave the link hidden */
      });
  }, []);

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
        <div className="ml-auto flex items-center gap-3 text-xs text-muted">
          {cryptolab ? (
            <a
              href="/cryptolab/"
              title="Crypto Lab — cryptographic analysis & security testing"
              className="rounded-md px-2 py-1 hover:text-fg"
            >
              🔐 Crypto Lab
            </a>
          ) : null}
          <span>offline signal analysis</span>
        </div>
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
