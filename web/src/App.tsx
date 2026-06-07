import { useEffect, useMemo } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { api, HTTPError } from "./api/client";
import { openEventStream } from "./api/events";
import { ConnectScreen } from "./components/ConnectScreen";
import { TabBar, type Tab } from "./components/TabBar";
import { AudioPlayer } from "./components/AudioPlayer";
import { InstallPrompt } from "./components/InstallPrompt";
import { Active } from "./panels/Active";
import { Bookmarks } from "./panels/Bookmarks";
import { CCActivity } from "./panels/CCActivity";
import { Constellation } from "./panels/Constellation";
import { SymbolScope } from "./panels/SymbolScope";
import { Dashboard } from "./panels/Dashboard";
import { Devices } from "./panels/Devices";
import { Events } from "./panels/Events";
import { History } from "./panels/History";
import { Import } from "./panels/Import";
import { APRS } from "./panels/APRS";
import { AIS } from "./panels/AIS";
import { DSC } from "./panels/DSC";
import { ADSB } from "./panels/ADSB";
import { MDC1200 } from "./panels/MDC1200";
import { Metrics } from "./panels/Metrics";
import { Pagers } from "./panels/Pagers";
import { RadioIDs } from "./panels/RadioIDs";
import { Scanner } from "./panels/Scanner";
import { Hunt } from "./panels/Hunt";
import { Settings } from "./panels/Settings";
import { Spectrum } from "./panels/Spectrum";
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
  { to: "/hunt", label: "Hunt", icon: "🔍" },
  { to: "/systems", label: "Systems", icon: "❖" },
  { to: "/talkgroups", label: "Talkgroups", icon: "☷" },
  { to: "/rids", label: "Radio IDs", icon: "⌖" },
  { to: "/history", label: "History", icon: "↺" },
  { to: "/events", label: "Events", icon: "≣" },
  { to: "/cc", label: "CC Activity", icon: "⌁" },
  { to: "/tones", label: "Tones", icon: "♪" },
  { to: "/pagers", label: "Pagers", icon: "✉" },
  { to: "/aprs", label: "APRS", icon: "⛯" },
  { to: "/ais", label: "AIS", icon: "⚓" },
  { to: "/dsc", label: "DSC", icon: "📡" },
  { to: "/adsb", label: "ADS-B", icon: "✈" },
  { to: "/mdc1200", label: "MDC1200", icon: "📻" },
  { to: "/spectrum", label: "Spectrum", icon: "≈" },
  { to: "/constellation", label: "Constellation", icon: "✦" },
  { to: "/symbols", label: "Symbol scope", icon: "⩘" },
  { to: "/bookmarks", label: "Bookmarks", icon: "★" },
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
  const hiddenTabs = useShared((s) => s.hiddenTabs);
  const setHiddenTabs = useShared((s) => s.setHiddenTabs);
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

    // Pull the runtime snapshot once to learn which nav tabs the
    // operator turned off via web.tabs in config. On failure we leave
    // the nav untouched (everything visible) rather than blank it.
    api
      .runtime(cfg)
      .then((rt) => setHiddenTabs(rt.hidden_tabs ?? []))
      .catch(() => setHiddenTabs([]));

    const stream = openEventStream(cfg, {
      onEvents: appendEvents,
      onStatus: setWSStatus,
    });
    return () => {
      stream.close();
    };
  }, [
    connected,
    baseURL,
    token,
    appendEvents,
    setMutations,
    setWSStatus,
    setHiddenTabs,
  ]);

  // A hidden tab is dropped from both the main strip and the desktop
  // overflow row. The key is the route path minus its leading slash —
  // the same key the daemon emits in hidden_tabs. Routes stay mounted
  // (nav-only hiding), so a hidden panel is still reachable by URL.
  const hidden = useMemo(() => new Set(hiddenTabs), [hiddenTabs]);
  const tabKey = (t: Tab) => String(t.to).replace(/^\//, "");
  const visibleTabs = useMemo(
    () => TABS.filter((t) => !hidden.has(tabKey(t))),
    [hidden],
  );
  const visibleExtraTabs = useMemo(
    () => EXTRA_TABS.filter((t) => !hidden.has(tabKey(t))),
    [hidden],
  );

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
        {visibleExtraTabs.map((t) => (
          <button
            key={String(t.to)}
            onClick={() => navigate(t.to)}
            className="px-2 py-1 rounded text-muted hover:text-fg hover:bg-panel"
          >
            {t.icon} {t.label}
          </button>
        ))}
        {/* The Config Builder is a separate SPA the daemon serves at
            /config/. Open it in a new tab so editing config doesn't tear
            down the live operator session. */}
        <button
          onClick={() => window.open("/config/", "_blank", "noopener")}
          className="px-2 py-1 rounded text-muted hover:text-fg hover:bg-panel"
          title="Open the Config Builder/Editor in a new tab"
        >
          🛠 Config Builder ↗
        </button>
      </div>

      <main className="flex-1 p-3 sm:p-4 pb-20 sm:pb-4">
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/active" element={<Active />} />
          <Route path="/scanner" element={<Scanner />} />
          <Route path="/hunt" element={<Hunt />} />
          <Route path="/spectrum" element={<Spectrum />} />
          <Route path="/constellation" element={<Constellation />} />
          <Route path="/symbols" element={<SymbolScope />} />
          <Route path="/bookmarks" element={<Bookmarks />} />
          <Route path="/systems" element={<Systems />} />
          <Route path="/talkgroups" element={<Talkgroups />} />
          <Route path="/rids" element={<RadioIDs />} />
          <Route path="/history" element={<History />} />
          <Route path="/events" element={<Events />} />
          <Route path="/cc" element={<CCActivity />} />
          <Route path="/tones" element={<Tones />} />
          <Route path="/pagers" element={<Pagers />} />
          <Route path="/aprs" element={<APRS />} />
          <Route path="/ais" element={<AIS />} />
          <Route path="/dsc" element={<DSC />} />
          <Route path="/adsb" element={<ADSB />} />
          <Route path="/mdc1200" element={<MDC1200 />} />
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
