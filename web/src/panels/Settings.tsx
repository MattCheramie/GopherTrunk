import { useState } from "react";
import { prefs, type Theme } from "../store/prefs";
import { selectCanMutate, useShared } from "../store/shared";

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
