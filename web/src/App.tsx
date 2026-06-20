import { useEffect, useState } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { api, HTTPError } from "./api/client";
import { openEventStream } from "./api/events";
import { ConnectScreen } from "./components/ConnectScreen";
import { AppShell } from "./components/shell/AppShell";
import { AudioPlayer } from "./components/AudioPlayer";
import { InstallPrompt } from "./components/InstallPrompt";
import { Active } from "./panels/Active";
import { Bookmarks } from "./panels/Bookmarks";
import { CCActivity } from "./panels/CCActivity";
import { Constellation } from "./panels/Constellation";
import { SymbolScope } from "./panels/SymbolScope";
import { EyeDiagram } from "./panels/EyeDiagram";
import { Mixer } from "./panels/Mixer";
import { Tuning } from "./panels/Tuning";
import { Histogram } from "./panels/Histogram";
import { Plots } from "./panels/Plots";
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
import { LoRa } from "./panels/LoRa";
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
import { prefs } from "./store/prefs";

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
  const setHiddenTabs = useShared((s) => s.setHiddenTabs);
  const setConfigPath = useShared((s) => s.setConfigPath);
  const configPath = useShared((s) => s.configPath);
  const appendEvents = useShared((s) => s.appendEvents);
  const setError = useShared((s) => s.setError);

  // No-config nudge: when the daemon runs without a -config file
  // (config_path === ""), offer the Config Builder. Dismissal is
  // per-tab so it doesn't nag after the operator waves it off.
  const [noConfigDismissed, setNoConfigDismissed] = useState(
    prefs.noConfigDismissed(),
  );

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
    // operator turned off via web.tabs in config, and whether a config
    // file backs the daemon (empty config_path → offer the Config
    // Builder). On failure we leave the nav untouched (everything
    // visible) rather than blank it.
    api
      .runtime(cfg)
      .then((rt) => {
        setHiddenTabs(rt.hidden_tabs ?? []);
        setConfigPath(rt.config_path ?? "");
      })
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
    setConfigPath,
  ]);

  if (!baseURL || !connected) {
    return <ConnectScreen />;
  }

  return (
    <AppShell>
      {configPath === "" && !noConfigDismissed && (
        <div className="mb-3 flex items-start gap-2 rounded-md border border-warn/40 bg-warn/15 px-4 py-2 text-sm text-warn">
          <span className="flex-1">
            No config file is loaded — the daemon is running on built-in
            defaults. Open the Config Builder to create or load one.
          </span>
          <button
            className="btn-ghost shrink-0 text-xs"
            onClick={() => window.open("/config/", "_blank", "noopener")}
          >
            Open Config Builder ↗
          </button>
          <button
            className="shrink-0 text-xs underline"
            onClick={() => {
              prefs.setNoConfigDismissed(true);
              setNoConfigDismissed(true);
            }}
          >
            dismiss
          </button>
        </div>
      )}

      <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/active" element={<Active />} />
          <Route path="/scanner" element={<Scanner />} />
          <Route path="/hunt" element={<Hunt />} />
          <Route path="/spectrum" element={<Spectrum />} />
          <Route path="/plots" element={<Plots />} />
          <Route path="/plots/:tab" element={<Plots />} />
          <Route path="/constellation" element={<Constellation />} />
          <Route path="/symbols" element={<SymbolScope />} />
          <Route path="/eye" element={<EyeDiagram />} />
          <Route path="/mixer" element={<Mixer />} />
          <Route path="/tuning" element={<Tuning />} />
          <Route path="/histogram" element={<Histogram />} />
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
          <Route path="/lora" element={<LoRa />} />
          <Route path="/metrics" element={<Metrics />} />
          <Route path="/devices" element={<Devices />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="/import" element={<Import />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>

      <AudioPlayer />
      <InstallPrompt />
    </AppShell>
  );
}
