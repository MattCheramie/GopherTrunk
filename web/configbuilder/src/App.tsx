import { useEffect, useMemo, useState } from "react";
import { FileBar } from "./components/FileBar";
import { YamlPreview } from "./components/YamlPreview";
import { SECTIONS } from "./sections";
import { useStore } from "./store/shared";

export function App() {
  const init = useStore((s) => s.init);
  const config = useStore((s) => s.config);
  const validation = useStore((s) => s.validation);
  const validateAll = useStore((s) => s.validateAll);
  const errors = useStore((s) => s.errors);
  const busy = useStore((s) => s.busy);
  const lastSaved = useStore((s) => s.lastSaved);
  const setError = useStore((s) => s.setError);
  const dismissError = useStore((s) => s.dismissError);
  const token = useStore((s) => s.token);
  const setTok = useStore((s) => s.setToken);

  const [active, setActive] = useState("trunking");
  const [showPreview, setShowPreview] = useState(true);

  useEffect(() => {
    init();
  }, [init]);

  // Re-validate shortly after edits so nav badges + section banners stay live.
  useEffect(() => {
    if (!config) return;
    const t = setTimeout(() => validateAll(), 400);
    return () => clearTimeout(t);
  }, [config, validateAll]);

  // Warn before leaving with unsaved changes.
  useEffect(() => {
    const h = (e: BeforeUnloadEvent) => {
      if (useStore.getState().dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", h);
    return () => window.removeEventListener("beforeunload", h);
  }, []);

  // Ctrl/Cmd+S → save (or ask the FileBar to open Save As when untitled).
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        e.preventDefault();
        const st = useStore.getState();
        if (st.path) void st.save(st.path, true);
        else window.dispatchEvent(new CustomEvent("configbuilder:saveas"));
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, []);

  const errList = validation?.errors ?? [];
  const errorCounts = useMemo(() => {
    const m: Record<string, number> = {};
    for (const e of errList) m[e.section] = (m[e.section] ?? 0) + 1;
    return m;
  }, [errList]);

  // Flag sections whose draft differs from a blank config so a from-scratch
  // build shows at a glance what's been touched.
  const defaults = useStore((s) => s.defaults);
  const configured = useMemo(() => {
    const set = new Set<string>();
    if (!config || !defaults) return set;
    for (const s of SECTIONS) {
      if (JSON.stringify(config[s.cfgKey]) !== JSON.stringify(defaults[s.cfgKey])) {
        set.add(s.key);
      }
    }
    return set;
  }, [config, defaults]);

  const current = SECTIONS.find((s) => s.key === active) ?? SECTIONS[0];

  const jumpToFirstError = () => {
    if (errList.length > 0) setActive(errList[0].section);
  };

  return (
    <div className="flex h-full flex-col">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
        <div className="flex items-center gap-2">
          <img src="./favicon.svg" alt="" className="h-6 w-6" />
          <h1 className="text-base font-semibold">GopherTrunk Config Builder</h1>
          {busy ? <span className="text-xs text-accent animate-pulse">working…</span> : null}
        </div>
        <div className="flex items-center gap-2">
          {validation ? (
            errList.length > 0 ? (
              <button
                className="rounded-md border border-err/40 bg-err/10 px-2 py-1 text-xs text-err"
                onClick={jumpToFirstError}
                title="Jump to the first error"
              >
                {errList.length} error{errList.length === 1 ? "" : "s"} in{" "}
                {Object.keys(errorCounts).length} section
                {Object.keys(errorCounts).length === 1 ? "" : "s"} ↗
              </button>
            ) : (
              <span className="rounded-md border border-ok/30 bg-ok/10 px-2 py-1 text-xs text-ok">
                ✓ valid
              </span>
            )
          ) : null}
          <input
            className="input w-48"
            type="password"
            placeholder="API token (if required)"
            value={token}
            onChange={(e) => setTok(e.target.value)}
          />
          <button className="btn-ghost" onClick={() => setShowPreview((p) => !p)}>
            {showPreview ? "Hide preview" : "Show preview"}
          </button>
        </div>
      </header>

      <div className="border-b border-white/10 px-4 py-2">
        <FileBar />
      </div>

      <div className="flex min-h-0 flex-1">
        {/* Section nav */}
        <nav className="w-44 shrink-0 overflow-y-auto border-r border-white/10 p-2">
          {SECTIONS.map((s) => {
            const n = errorCounts[s.key] ?? 0;
            return (
              <button
                key={s.key}
                onClick={() => setActive(s.key)}
                className={`mb-0.5 flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-sm ${
                  active === s.key ? "bg-accent/20 text-fg" : "text-muted hover:bg-white/5"
                }`}
              >
                <span className="flex items-center gap-1.5">
                  {configured.has(s.key) ? (
                    <span className="text-accent" title="Configured (differs from defaults)">
                      ●
                    </span>
                  ) : (
                    <span className="text-white/20" title="Empty (defaults)">
                      ○
                    </span>
                  )}
                  {s.label}
                </span>
                {n > 0 ? <span className="text-xs text-err">{n}</span> : null}
              </button>
            );
          })}
        </nav>

        {/* Editor + preview — side by side on lg, stacked on small screens */}
        <div className="flex min-w-0 flex-1 flex-col lg:flex-row">
          <main className="min-w-0 flex-1 overflow-y-auto p-4">
            {config ? current.render() : <p className="help">Loading defaults…</p>}
          </main>
          {showPreview ? (
            <aside className="max-h-72 w-full shrink-0 overflow-hidden border-t border-white/10 p-4 lg:max-h-none lg:w-96 lg:border-l lg:border-t-0">
              <YamlPreview />
            </aside>
          ) : null}
        </div>
      </div>

      {/* Toasts */}
      {lastSaved ? (
        <div className="fixed bottom-3 right-3 z-40 rounded-md border border-ok/40 bg-ok/15 px-3 py-2 text-sm text-ok">
          {lastSaved}
        </div>
      ) : null}
      {errors.length > 0 ? (
        <div className="fixed bottom-3 left-3 right-3 z-40 mx-auto flex max-w-md flex-col gap-2">
          {errors.map((msg, i) => (
            <div
              key={i}
              className="flex items-start gap-2 rounded-md border border-err/40 bg-err/15 px-3 py-2 text-sm text-err"
            >
              <span className="flex-1">{msg}</span>
              <button className="underline" onClick={() => dismissError(i)}>
                dismiss
              </button>
            </div>
          ))}
          {errors.length > 1 ? (
            <button className="self-end text-xs underline text-err" onClick={() => setError(null)}>
              dismiss all
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
