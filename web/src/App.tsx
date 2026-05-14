import { useEffect, useMemo } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { api, HTTPError } from "./api/client";
import { openEventStream } from "./api/events";
import { ConnectScreen } from "./components/ConnectScreen";
import { TabBar, type Tab } from "./components/TabBar";
import { AudioPlayer } from "./components/AudioPlayer";
import { InstallPrompt } from "./components/InstallPrompt";
import { Dashboard } from "./panels/Dashboard";
import { Settings } from "./panels/Settings";
import { Placeholder } from "./panels/Placeholder";
import {
  selectClientConfig,
  useShared,
} from "./store/shared";

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
];

export function App() {
  const cfg = useShared(selectClientConfig);
  const connected = useShared((s) => s.connected);
  const setConnected = useShared((s) => s.setConnected);
  const setMutations = useShared((s) => s.setMutations);
  const setWSStatus = useShared((s) => s.setWSStatus);
  const appendEvent = useShared((s) => s.appendEvent);
  const lastError = useShared((s) => s.lastError);
  const setError = useShared((s) => s.setError);
  const navigate = useNavigate();

  // On mount, if we already have a server URL stored, try to validate
  // and skip the connect screen.
  useEffect(() => {
    if (!cfg.baseURL) return;
    if (connected) return;
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
  }, [cfg, connected, setConnected, setError]);

  // Once connected, open the WS event stream + bootstrap the mutations
  // capability gate. The polling for per-panel data happens inside each
  // panel so the data isn't fetched until a user actually looks at it.
  useEffect(() => {
    if (!connected || !cfg.baseURL) return;
    api
      .mutations(cfg)
      .then(setMutations)
      .catch(() => setMutations(null));

    const stream = openEventStream(cfg, {
      onEvent: appendEvent,
      onStatus: setWSStatus,
    });
    return () => {
      stream.close();
    };
  }, [connected, cfg, appendEvent, setMutations, setWSStatus]);

  const visibleTabs = useMemo(() => TABS, []);

  if (!cfg.baseURL || !connected) {
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
          <Route
            path="/active"
            element={
              <Placeholder
                title="Active calls"
                hint="The dashboard already lists active calls. This panel will add per-call detail, end-call mutations, and audio-cockpit shortcuts."
              />
            }
          />
          <Route
            path="/scanner"
            element={
              <Placeholder
                title="Scanner cockpit"
                hint="Hold / resume / retune the CC hunter, lockout conventional channels, and manual-tune a VFO."
              />
            }
          />
          <Route
            path="/systems"
            element={<Placeholder title="Systems" hint="Trunked-system browser." />}
          />
          <Route
            path="/talkgroups"
            element={
              <Placeholder
                title="Talkgroups"
                hint="Priority / lockout / scan toggles per talkgroup."
              />
            }
          />
          <Route
            path="/history"
            element={<Placeholder title="History" hint="Call log explorer." />}
          />
          <Route
            path="/events"
            element={
              <Placeholder
                title="Events"
                hint="Live ring buffer of every event the daemon publishes."
              />
            }
          />
          <Route
            path="/tones"
            element={
              <Placeholder
                title="Tones"
                hint="Tone-out alert review and per-device resets."
              />
            }
          />
          <Route
            path="/metrics"
            element={
              <Placeholder
                title="Metrics"
                hint="Charted Prometheus counters via Chart.js."
              />
            }
          />
          <Route
            path="/devices"
            element={<Placeholder title="Devices" hint="SDR pool inspector." />}
          />
          <Route path="/settings" element={<Settings />} />
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
