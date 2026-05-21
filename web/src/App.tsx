import { useEffect, useMemo } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { api, HTTPError } from "./api/client";
import { openEventStream } from "./api/events";
import { ConnectScreen } from "./components/ConnectScreen";
import { TabBar, type Tab } from "./components/TabBar";
import { AudioPlayer } from "./components/AudioPlayer";
import { InstallPrompt } from "./components/InstallPrompt";
import { Active } from "./panels/Active";
import { Dashboard } from "./panels/Dashboard";
import { Devices } from "./panels/Devices";
import { Events } from "./panels/Events";
import { History } from "./panels/History";
import { Import } from "./panels/Import";
import { Metrics } from "./panels/Metrics";
import { Scanner } from "./panels/Scanner";
import { Settings } from "./panels/Settings";
import { Systems } from "./panels/Systems";
import { Talkgroups } from "./panels/Talkgroups";
import { Tones } from "./panels/Tones";
import { useShared } from "./store/shared";

const TABS: Tab[] = [
  { to: "/dashboard", label: "Dashboard", icon: "▤" },
  { to: "/active", label: "Active", icon: "●" },
  { to: "/scanner", label: "Scanner", icon: "≋" },
  { to: "/settings", label: "Settings", icon: "⚙" },
];
const EXTRA_TABS: Tab[] = [
  { to: "/systems", label: "Systems", icon: "❖" },
  { to: "/talkgroups", label: "Talkgroups", icon: "☷" },
  { to: "/history", label: "History", icon: "↺" },
  { to: "/events", label: "Events", icon: "≣" },
  { to: "/tones", label: "Tones", icon: "♪" },
  { to: "/metrics", label: "Metrics", icon: "▰" },
  { to: "/devices", label: "Devices", icon: "⌗" },
  { to: "/import", label: "Import", icon: "↗" },
];

export function App() {
  // Subscribe to the connection identity as primitives, not a derived
  // object. Effects keyed on `baseURL`/`token` re-run only when the
  // server actually changes — they cannot be re-fired by an unstable
  // selector reference (the failure mode behind issue #290).
  const baseURL = useShared((s) => s.serverURL ?? "");
  const token = useShared((s) => s.token);
  const connected = useShared((s) => s.connected);
  const setConnected = useShared((s) => s.setConnected);
  const setMutations = useShared((s) => s.setMutations);
  const setWSStatus = useShared((s) => s.setWSStatus);
  const appendEvents = useShared((s) => s.appendEvents);
  const lastError = useShared((s) => s.lastError);
  const setError = useShared((s) => s.setError);
  const navigate = useNavigate();

  // On mount, if we already have a server URL stored, try to validate
  // and skip the connect screen.
  useEffect(() => {
    if (!baseURL) return;
    if (connected) return;
    const cfg = { baseURL, token };
    let cancel = false;
    (async () => {
      try {
        await api.health(cfg);
        if (!cancel) setConnected(true);
      } catch (e) {
        if (!cancel) {
          if (e instanceof HTTPError && e.status === 401) {
            setError("Daemon requires a token — please re-enter it.");
          }
          setConnected(false);
        }
      }
    })();
    return () => {
      cancel = true;
    };
  }, [baseURL, token, connected, setConnected, setError]);

  // Once connected, open the WS event stream + bootstrap the mutations
  // capability gate. The polling for per-panel data happens inside each
  // panel so the data isn't fetched until a user actually looks at it.
  useEffect(() => {
    if (!connected || !baseURL) return;
    const cfg = { baseURL, token };
    api
      .mutations(cfg)
      .then(setMutations)
      .catch(() => setMutations(null));

    const stream = openEventStream(cfg, {
      onEvents: appendEvents,
      onStatus: setWSStatus,
    });
    return () => {
      stream.close();
    };
  }, [connected, baseURL, token, appendEvents, setMutations, setWSStatus]);

  const visibleTabs = useMemo(() => TABS, []);

  if (!baseURL || !connected) {
    return <ConnectScreen />;
  }

  return (
    <div className="min-h-full flex flex-col">
      <TabBar tabs={visibleTabs} />

      {/* Desktop overflow tabs sit beneath the main strip so the
          bottom-nav-friendly four-tab limit still leaves room for
          everything else. */}
      <div className="hidden sm:flex gap-1 px-3 py-1 border-b border-panel text-xs overflow-x-auto">
        {EXTRA_TABS.map((t) => (
          <button
            key={String(t.to)}
            onClick={() => navigate(t.to)}
            className="px-2 py-1 rounded text-muted hover:text-fg hover:bg-panel"
          >
            {t.icon} {t.label}
          </button>
        ))}
      </div>

      <main className="flex-1 p-3 sm:p-4 pb-20 sm:pb-4">
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/active" element={<Active />} />
          <Route path="/scanner" element={<Scanner />} />
          <Route path="/systems" element={<Systems />} />
          <Route path="/talkgroups" element={<Talkgroups />} />
          <Route path="/history" element={<History />} />
          <Route path="/events" element={<Events />} />
          <Route path="/tones" element={<Tones />} />
          <Route path="/metrics" element={<Metrics />} />
          <Route path="/devices" element={<Devices />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/import" element={<Import />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </main>

      <AudioPlayer />
      <InstallPrompt />

      {lastError && (
        <div
          role="alert"
          className="fixed bottom-20 sm:bottom-3 left-3 right-3 sm:left-auto sm:max-w-sm z-40 panel bg-err/15 border-err/40 text-err p-3 text-sm flex items-start gap-2"
        >
          <span className="flex-1">{lastError}</span>
          <button
            className="text-xs underline"
            onClick={() => setError(null)}
            aria-label="Dismiss error"
          >
            dismiss
          </button>
        </div>
      )}
    </div>
  );
}
