import { useEffect, useState } from "react";
import { api, HTTPError } from "../api/client";
import { writes } from "../api/write";
import type {
  ConfigActivateResponse,
  ConfigFileInfo,
  RuntimeDTO,
  SettingsPatch,
} from "../api/types";
import { prefs, type Theme } from "../store/prefs";
import {
  selectCanMutate,
  selectClientConfig,
  useShared,
} from "../store/shared";

// Settings: theme, write mode, audio defaults, "forget device".
// Mutations panels are AND-gated with the daemon's allow_mutations
// flag, so the toggle is a UI affordance — flipping it on doesn't
// magically bypass server auth.
export function Settings() {
  const [theme, setTheme] = useState<Theme>(prefs.theme());
  const writeMode = useShared((s) => s.writeMode);
  const setWriteMode = useShared((s) => s.setWriteMode);
  const canMutate = useShared(selectCanMutate);
  const mutations = useShared((s) => s.mutations);
  const setCredentials = useShared((s) => s.setCredentials);
  const setConnected = useShared((s) => s.setConnected);
  const reset = useShared((s) => s.reset);
  const serverURL = useShared((s) => s.serverURL);

  const onTheme = (t: Theme) => {
    setTheme(t);
    prefs.setTheme(t);
    document.documentElement.dataset.theme = t === "monochrome" ? "mono" : "dark";
  };

  return (
    <div className="space-y-4 max-w-2xl">
      <header>
        <h2 className="text-xl font-semibold">Settings</h2>
        <p className="text-sm text-muted">
          Stored locally in your browser; nothing is sent to the daemon.
        </p>
      </header>

      <section className="panel p-4 space-y-3">
        <h3 className="panel-title">Server</h3>
        <p className="text-sm font-mono text-muted">{serverURL ?? "—"}</p>
        <button
          className="btn-ghost"
          onClick={() => {
            setCredentials(null, null, false);
            setConnected(false);
            reset();
          }}
        >
          Change server
        </button>
      </section>

      <section className="panel p-4 space-y-3">
        <h3 className="panel-title">Theme</h3>
        <div className="flex gap-2">
          {(["dark", "monochrome"] as const).map((t) => (
            <button
              key={t}
              className={
                theme === t
                  ? "btn-primary"
                  : "btn-ghost"
              }
              onClick={() => onTheme(t)}
            >
              {t}
            </button>
          ))}
        </div>
      </section>

      <section className="panel p-4 space-y-3">
        <h3 className="panel-title">Write mode</h3>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            className="h-5 w-5"
            checked={writeMode}
            onChange={(e) => setWriteMode(e.target.checked)}
          />
          <span>
            Allow mutations from this browser
            {mutations && !mutations.allow_mutations && (
              <span className="ml-2 text-xs text-warn">
                (daemon currently rejects mutations — set
                api.allow_mutations or api.auth.mode to enable)
              </span>
            )}
          </span>
        </label>
        <p className="text-xs text-muted">
          {canMutate
            ? "Mutations are enabled. Destructive actions still require confirmation."
            : "Mutations are read-only. Toggle on and confirm the daemon allows it."}
        </p>
      </section>

      <section className="panel p-4 space-y-3">
        <h3 className="panel-title">PWA</h3>
        <p className="text-sm text-muted">
          Add GopherTrunk to your phone's home screen for a full-
          screen, offline-cached app. On iOS use Safari's Share menu →
          Add to Home Screen. On Android the browser may prompt
          automatically; the prompt also appears in this app when
          available.
        </p>
      </section>

      <ConfigFileSection />

      <LiveConfigSection />

      <section className="panel p-4 space-y-3">
        <h3 className="panel-title text-err">Danger zone</h3>
        <button
          className="btn-danger"
          onClick={() => {
            prefs.clearAll();
            setCredentials(null, null, false);
            setConnected(false);
            reset();
          }}
        >
          Forget this device
        </button>
      </section>
    </div>
  );
}

// --- Live config editing ---
//
// The LiveConfigSection mirrors the TUI's editable settings rows. Each
// row pulls its current value from the runtime DTO and dispatches a
// PATCH /api/v1/settings on save. Restart-required fields get a
// yellow badge; the section is hidden entirely when the daemon was
// started without a -config file (PATCH would return 503).

interface FieldSpec {
  field: string;
  label: string;
  type: "string" | "number" | "bool";
  restart?: boolean;
  read: (r: RuntimeDTO) => string;
}

function liveConfigFields(): FieldSpec[] {
  return [
    {
      field: "log.level",
      label: "Log level",
      type: "string",
      read: (r) => r.log_level ?? "info",
    },
    {
      field: "log.format",
      label: "Log format",
      type: "string",
      restart: true,
      read: (r) => r.log_format ?? "text",
    },
    {
      field: "audio.volume",
      label: "Audio volume (0..1)",
      type: "number",
      read: (r) => String(r.audio?.volume ?? 0.8),
    },
    {
      field: "audio.muted",
      label: "Audio muted",
      type: "bool",
      read: (r) => String(r.audio?.muted ?? false),
    },
    {
      field: "scanner.scan_mode",
      label: "Scanner scan mode (all|list)",
      type: "string",
      read: (r) =>
        (r["scanner_scan_mode"] as string | undefined) ?? "all",
    },
    {
      field: "recordings.dir",
      label: "Recordings directory",
      type: "string",
      restart: true,
      read: (r) => (r["recording_dir"] as string | undefined) ?? "",
    },
    {
      field: "recordings.sample_rate",
      label: "Recordings sample rate (Hz)",
      type: "number",
      restart: true,
      read: (r) => String(r["recording_sample_rate"] ?? 8000),
    },
    {
      field: "sdr.sample_rate",
      label: "SDR sample rate (Hz)",
      type: "number",
      restart: true,
      read: (r) => String(r["sdr_sample_rate"] ?? 2_400_000),
    },
    {
      field: "metrics.enabled",
      label: "Metrics enabled",
      type: "bool",
      restart: true,
      read: (r) => String(r.metrics_enabled ?? false),
    },
  ];
}

function LiveConfigSection() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const [runtime, setRuntime] = useState<RuntimeDTO | null>(null);
  const [editing, setEditing] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancel = false;
    const load = async () => {
      try {
        const r = await api.runtime(cfg);
        if (!cancel) setRuntime(r);
      } catch {
        // ignore — the daemon may briefly be unavailable during
        // restart; the section will refresh on the next render.
      }
    };
    load();
    const t = window.setInterval(load, 5_000);
    return () => {
      cancel = true;
      window.clearInterval(t);
    };
  }, [cfg]);

  if (!runtime) {
    return (
      <section className="panel p-4 space-y-3">
        <h3 className="panel-title">Live config</h3>
        <p className="text-sm text-muted">Loading runtime…</p>
      </section>
    );
  }

  const cfgPath = (runtime["config_path"] as string | undefined) ?? "";
  if (!cfgPath) {
    return (
      <section className="panel p-4 space-y-3">
        <h3 className="panel-title">Live config</h3>
        <div className="text-sm panel bg-warn/15 border-warn/40 text-warn p-3">
          The daemon is running without a <code>-config</code> file, so
          <code> PATCH /api/v1/settings</code> returns 503. Restart with
          <code> -config /path/to/config.yaml</code> to enable inline
          edits.
        </div>
      </section>
    );
  }

  const fields = liveConfigFields();

  async function save(spec: FieldSpec, raw: string) {
    setError(null);
    setBusy(true);
    try {
      const patch = buildSettingsPatch(spec, raw);
      await writes.updateSettings(cfg, patch);
      // Refresh.
      const r = await api.runtime(cfg);
      setRuntime(r);
      setEditing(null);
    } catch (e) {
      if (e instanceof HTTPError) {
        setError(`${spec.field}: ${e.message}`);
      } else {
        setError(`${spec.field}: ${String(e)}`);
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel p-4 space-y-3">
      <h3 className="panel-title">Live config</h3>
      <p className="text-xs text-muted">
        Backed by <code>{cfgPath}</code>. Edits land in config.yaml with
        comments preserved; hot-reloadable knobs apply immediately,
        restart-required fields are flagged below.
      </p>
      {!canMutate && (
        <div className="text-sm panel bg-warn/15 border-warn/40 text-warn p-3">
          Mutations are disabled — daemon auth blocks writes.
        </div>
      )}
      {error && (
        <div
          role="alert"
          className="text-sm panel bg-err/15 border-err/40 text-err p-3"
        >
          {error}
        </div>
      )}
      <table className="text-sm w-full">
        <tbody>
          {fields.map((f) => (
            <tr key={f.field} className="border-t border-panel">
              <td className="py-2 pr-3 text-muted whitespace-nowrap">
                {f.label}
                {f.restart && (
                  <span className="ml-2 text-xs text-warn">
                    [restart]
                  </span>
                )}
              </td>
              <td className="py-2">
                {editing === f.field ? (
                  <input
                    autoFocus
                    className="input w-full"
                    value={draft}
                    onChange={(e) => setDraft(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") save(f, draft);
                      if (e.key === "Escape") setEditing(null);
                    }}
                    disabled={busy}
                  />
                ) : (
                  <code className="font-mono">{f.read(runtime) || "—"}</code>
                )}
              </td>
              <td className="py-2 pl-3 text-right whitespace-nowrap">
                {editing === f.field ? (
                  <>
                    <button
                      className="btn-primary mr-2"
                      onClick={() => save(f, draft)}
                      disabled={busy}
                    >
                      Save
                    </button>
                    <button
                      className="btn-ghost"
                      onClick={() => setEditing(null)}
                      disabled={busy}
                    >
                      Cancel
                    </button>
                  </>
                ) : (
                  <button
                    className="btn-ghost"
                    disabled={!canMutate}
                    onClick={() => {
                      setEditing(f.field);
                      setDraft(f.read(runtime));
                    }}
                  >
                    Edit
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

// buildSettingsPatch converts a field spec + raw string into the
// typed SettingsPatch shape the daemon expects.
function buildSettingsPatch(spec: FieldSpec, raw: string): SettingsPatch {
  const v = raw.trim();
  switch (spec.field) {
    case "log.level":
      return { log_level: v };
    case "log.format":
      return { log_format: v };
    case "audio.volume": {
      const n = Number(v);
      if (Number.isNaN(n) || n < 0 || n > 1) {
        throw new Error("audio.volume must be a float in [0, 1]");
      }
      return { audio_volume: n };
    }
    case "audio.muted":
      return { audio_muted: parseBool(v) };
    case "scanner.scan_mode":
      return { scanner_scan_mode: v };
    case "recordings.dir":
      return { recordings_dir: v };
    case "recordings.sample_rate": {
      const n = Number(v);
      if (!Number.isFinite(n) || n <= 0) {
        throw new Error("recordings.sample_rate must be a positive integer");
      }
      return { recordings_sample_rate: n };
    }
    case "sdr.sample_rate": {
      const n = Number(v);
      if (!Number.isFinite(n) || n <= 0) {
        throw new Error("sdr.sample_rate must be a positive integer");
      }
      return { sdr_sample_rate: n };
    }
    case "metrics.enabled":
      return { metrics_enabled: parseBool(v) };
  }
  throw new Error(`unknown field ${spec.field}`);
}

function parseBool(v: string): boolean {
  switch (v.toLowerCase()) {
    case "true":
    case "on":
    case "yes":
    case "1":
      return true;
    case "false":
    case "off":
    case "no":
    case "0":
      return false;
  }
  throw new Error("expected true/false (also accepts on/off, yes/no, 1/0)");
}

// --- Config file (load / hot-swap) ---
//
// ConfigFileSection lists the config files the daemon discovered and lets
// the operator switch the active one. "Reload" hot-applies what it can and
// flags the rest as restart-required; "Restart" re-execs the daemon into the
// new file (every field takes effect) — the connection drops, so we poll
// health and reload the page once it's back.
function ConfigFileSection() {
  const cfg = useShared(selectClientConfig);
  const canMutate = useShared(selectCanMutate);
  const currentPath = useShared((s) => s.configPath);

  const [files, setFiles] = useState<ConfigFileInfo[] | null>(null);
  const [unavailable, setUnavailable] = useState(false);
  const [pending, setPending] = useState<string | null>(null); // path awaiting reload/restart choice
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ConfigActivateResponse | null>(null);
  const [restarting, setRestarting] = useState(false);

  useEffect(() => {
    let cancel = false;
    (async () => {
      try {
        const r = await api.configFiles(cfg);
        if (!cancel) setFiles(r.files ?? []);
      } catch (e) {
        // 503 → the config-builder subsystem isn't wired (shouldn't happen
        // on the daemon, but be defensive); hide the section in that case.
        if (!cancel) {
          if (e instanceof HTTPError && e.status === 503) setUnavailable(true);
          else setError((e as Error).message);
          setFiles([]);
        }
      }
    })();
    return () => {
      cancel = true;
    };
  }, [cfg]);

  if (unavailable) return null;

  async function activate(path: string, mode: "reload" | "restart") {
    setError(null);
    setResult(null);
    setBusy(true);
    try {
      const r = await writes.activateConfig(cfg, path, mode);
      setPending(null);
      if (mode === "restart") {
        setRestarting(true);
        void waitForDaemonAndReload(cfg);
      } else {
        setResult(r);
      }
    } catch (e) {
      if (e instanceof HTTPError) setError(e.message);
      else setError(String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel p-4 space-y-3">
      <h3 className="panel-title">Config file</h3>
      <p className="text-xs text-muted">
        {currentPath
          ? <>Active: <code className="font-mono">{currentPath}</code></>
          : "The daemon is running on built-in defaults (no config file)."}
      </p>

      {!canMutate && (
        <div className="text-sm panel bg-warn/15 border-warn/40 text-warn p-3">
          Mutations are disabled — daemon auth blocks switching config files.
        </div>
      )}
      {error && (
        <div role="alert" className="text-sm panel bg-err/15 border-err/40 text-err p-3">
          {error}
        </div>
      )}
      {restarting && (
        <div className="text-sm panel bg-accent/15 border-accent/40 p-3">
          Daemon restarting into the new config… this page will reload once
          it's back.
        </div>
      )}
      {result && (
        <div className="text-sm panel bg-ok/15 border-ok/40 p-3 space-y-1">
          <div>
            Loaded <code className="font-mono">{result.path}</code>.
          </div>
          {result.applied && result.applied.length > 0 && (
            <div>Applied live: {result.applied.join(", ")}.</div>
          )}
          {result.restart_required && result.restart_required.length > 0 && (
            <div className="text-warn">
              Restart required for: {result.restart_required.join(", ")} — use
              “Restart” to apply these.
            </div>
          )}
        </div>
      )}

      {files === null ? (
        <p className="text-sm text-muted">Loading…</p>
      ) : files.length === 0 ? (
        <p className="text-sm text-muted">
          No config files found in the discovery directories.
        </p>
      ) : (
        <ul className="space-y-1">
          {files.map((f) => (
            <li
              key={f.path}
              className={`rounded border p-2 text-sm ${
                f.path === currentPath ? "border-accent/50 bg-accent/10" : "border-panel"
              }`}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="min-w-0">
                  <span className={f.valid ? "" : "text-warn"}>
                    {f.valid ? "" : "⚠ "}
                    {f.name}
                  </span>
                  {f.path === currentPath && (
                    <span className="ml-2 text-xs text-accent">active</span>
                  )}
                  <span className="block truncate text-xs text-muted">{f.dir}</span>
                </span>
                {f.path !== currentPath &&
                  (pending === f.path ? (
                    <span className="flex shrink-0 gap-1">
                      <button
                        className="btn-ghost text-xs"
                        disabled={busy || !f.valid}
                        title={f.valid ? "Reload live (hot-apply what it can)" : "Fix validation errors first"}
                        onClick={() => activate(f.path, "reload")}
                      >
                        Reload
                      </button>
                      <button
                        className="btn-ghost text-xs text-warn"
                        disabled={busy || !f.valid}
                        title="Restart the daemon into this config (drops the session briefly)"
                        onClick={() => activate(f.path, "restart")}
                      >
                        Restart
                      </button>
                      <button
                        className="text-xs underline"
                        disabled={busy}
                        onClick={() => setPending(null)}
                      >
                        cancel
                      </button>
                    </span>
                  ) : (
                    <button
                      className="btn-ghost shrink-0 text-xs"
                      disabled={!canMutate || busy}
                      onClick={() => {
                        setResult(null);
                        setError(null);
                        setPending(f.path);
                      }}
                    >
                      Load this config
                    </button>
                  ))}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

// waitForDaemonAndReload polls /api/v1/health until the restarted daemon is
// reachable again (or a cap is hit), then reloads the page so the SPA
// re-bootstraps cleanly against the new config.
async function waitForDaemonAndReload(cfg: { baseURL: string; token: string | null }) {
  const deadline = Date.now() + 60_000;
  // Give the daemon a moment to tear down before we start probing.
  await new Promise((r) => setTimeout(r, 1_500));
  while (Date.now() < deadline) {
    try {
      await api.health(cfg);
      window.location.reload();
      return;
    } catch {
      await new Promise((r) => setTimeout(r, 1_500));
    }
  }
  // Gave up waiting — reload anyway so the operator sees the connect screen
  // (or a recovered session) rather than a stuck "restarting…" banner.
  window.location.reload();
}
