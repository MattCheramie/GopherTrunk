import { useEffect, useState } from "react";
import { FileBar } from "./components/FileBar";
import { YamlPreview } from "./components/YamlPreview";
import { SECTIONS } from "./sections";
import { useStore } from "./store/shared";

export function App() {
  const init = useStore((s) => s.init);
  const config = useStore((s) => s.config);
  const validation = useStore((s) => s.validation);
  const validateAll = useStore((s) => s.validateAll);
  const lastError = useStore((s) => s.lastError);
  const lastSaved = useStore((s) => s.lastSaved);
  const setError = useStore((s) => s.setError);
  const token = useStore((s) => s.token);
  const setTok = useStore((s) => s.setToken);

  const [active, setActive] = useState("trunking");
  const [showPreview, setShowPreview] = useState(true);

  useEffect(() => {
    init();
  }, [init]);

  // Re-validate (whole config, one error per section) shortly after edits so
  // the nav badges + section banners stay live.
  useEffect(() => {
    if (!config) return;
    const t = setTimeout(() => validateAll(), 400);
    return () => clearTimeout(t);
  }, [config, validateAll]);

  const errorSections = new Set((validation?.errors ?? []).map((e) => e.section));
  const current = SECTIONS.find((s) => s.key === active) ?? SECTIONS[0];

  return (
    <div className="flex h-full flex-col">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
        <div className="flex items-center gap-2">
          <img src="./favicon.svg" alt="" className="h-6 w-6" />
          <h1 className="text-base font-semibold">GopherTrunk Config Builder</h1>
        </div>
        <div className="flex items-center gap-2">
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
          {SECTIONS.map((s) => (
            <button
              key={s.key}
              onClick={() => setActive(s.key)}
              className={`mb-0.5 flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-sm ${
                active === s.key ? "bg-accent/20 text-fg" : "text-muted hover:bg-white/5"
              }`}
            >
              <span>{s.label}</span>
              {errorSections.has(s.key) ? <span className="text-err">●</span> : null}
            </button>
          ))}
        </nav>

        {/* Editor */}
        <main className="min-w-0 flex-1 overflow-y-auto p-4">
          {config ? current.render() : <p className="help">Loading defaults…</p>}
        </main>

        {/* Preview */}
        {showPreview ? (
          <aside className="hidden w-96 shrink-0 overflow-hidden border-l border-white/10 p-4 lg:block">
            <YamlPreview />
          </aside>
        ) : null}
      </div>

      {/* Toasts */}
      {lastSaved ? (
        <div className="fixed bottom-3 right-3 z-40 rounded-md border border-ok/40 bg-ok/15 px-3 py-2 text-sm text-ok">
          {lastSaved}
        </div>
      ) : null}
      {lastError ? (
        <div className="fixed bottom-3 left-3 right-3 z-40 mx-auto max-w-md rounded-md border border-err/40 bg-err/15 px-3 py-2 text-sm text-err flex items-start gap-2">
          <span className="flex-1">{lastError}</span>
          <button className="underline" onClick={() => setError(null)}>
            dismiss
          </button>
        </div>
      ) : null}
    </div>
  );
}
