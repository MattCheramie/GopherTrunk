import { useEffect, useMemo, useState } from "react";
import { useStore } from "./store/shared";
import { ParamField } from "./components/ParamField";
import { ResultView } from "./components/ResultView";
import { RecipeBuilder } from "./components/RecipeBuilder";
import { ConsoleNav } from "./ConsoleNav";

export function App() {
  const [page, setPage] = useState<"tools" | "recipe">("tools");
  const loadSchema = useStore((s) => s.loadSchema);
  const schema = useStore((s) => s.schema);
  const loadingSchema = useStore((s) => s.loadingSchema);
  const schemaError = useStore((s) => s.schemaError);
  const tool = useStore((s) => s.tool);
  const mode = useStore((s) => s.mode);
  const select = useStore((s) => s.select);
  const selected = useStore((s) => s.selected());
  const startRun = useStore((s) => s.startRun);
  const run = useStore((s) => s.run);
  const uploading = useStore((s) => s.uploading);
  const files = useStore((s) => s.files);

  useEffect(() => {
    void loadSchema();
  }, [loadSchema]);

  const tools = useMemo(() => {
    const m = new Map<string, { mode: string; synopsis: string }[]>();
    for (const s of schema) {
      const arr = m.get(s.tool) ?? [];
      arr.push({ mode: s.mode, synopsis: s.synopsis });
      m.set(s.tool, arr);
    }
    return [...m.entries()];
  }, [schema]);

  const missingRequiredFile = (selected?.params ?? []).some(
    (p) => p.kind === "file" && p.required && !files[p.name],
  );
  const anyUploading = Object.values(uploading).some(Boolean);
  const canRun = !!selected && !missingRequiredFile && !anyUploading && run.state !== "running";

  return (
    <div className="flex h-full flex-col">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3">
        <div className="flex items-center gap-2">
          <img src="./favicon.svg" alt="" className="h-6 w-6" />
          <h1 className="text-base font-semibold">GopherTrunk Crypto Lab</h1>
          {run.state === "running" ? (
            <span className="text-xs text-accent animate-pulse">working…</span>
          ) : null}
        </div>
        <nav className="flex items-center gap-1">
          {(["tools", "recipe"] as const).map((p) => (
            <button
              key={p}
              onClick={() => setPage(p)}
              className={`rounded px-2.5 py-1 text-sm ${
                page === p ? "bg-accent/20 text-fg" : "text-muted hover:bg-panel"
              }`}
            >
              {p === "tools" ? "Tools" : "Recipe Builder"}
            </button>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-3 text-xs text-muted">
          <ConsoleNav self="cryptolab" />
          <span>offline cryptographic research</span>
        </div>
      </header>

      {page === "recipe" ? (
        <main className="min-h-0 flex-1 overflow-y-auto p-4">
          <RecipeBuilder />
        </main>
      ) : (
      <div className="flex min-h-0 flex-1">
        <nav className="w-56 shrink-0 overflow-y-auto border-r border-line p-2">
          {tools.length === 0 ? (
            <p className="help p-2">{loadingSchema ? "Loading tools…" : "No tools."}</p>
          ) : (
            tools.map(([t, modes]) => (
              <div key={t} className="mb-2">
                <div className="px-2 py-1 text-xs uppercase tracking-wide text-muted">{t}</div>
                {modes.map((m) => (
                  <button
                    key={m.mode}
                    onClick={() => select(t, m.mode)}
                    title={m.synopsis}
                    className={`mb-0.5 block w-full truncate rounded px-2 py-1.5 text-left text-sm ${
                      tool === t && mode === m.mode
                        ? "bg-accent/20 text-fg"
                        : "text-muted hover:bg-panel"
                    }`}
                  >
                    {m.mode}
                  </button>
                ))}
              </div>
            ))
          )}
        </nav>

        <main className="min-w-0 flex-1 space-y-4 overflow-y-auto p-4">
          {schemaError ? (
            <div className="rounded-md border border-err/40 bg-err/10 px-3 py-2 text-sm text-err">
              Failed to load tools: {schemaError}
            </div>
          ) : null}

          {selected ? (
            <>
              <div>
                <h2 className="text-lg font-semibold">
                  {selected.tool} <span className="text-muted">/ {selected.mode}</span>
                </h2>
                <p className="help mt-1 max-w-2xl">{selected.synopsis}</p>
              </div>

              <div className="card space-y-3">
                {(selected.params ?? []).length === 0 ? (
                  <p className="help">This mode takes no inputs.</p>
                ) : (
                  (selected.params ?? []).map((p) => <ParamField key={p.name} param={p} />)
                )}
                <div className="flex items-center gap-3 pt-1">
                  <button className="btn" disabled={!canRun} onClick={() => void startRun()}>
                    {run.state === "running" ? "Running…" : "Run"}
                  </button>
                  {missingRequiredFile ? (
                    <span className="help">Upload the required file(s) to run.</span>
                  ) : null}
                </div>
              </div>

              <ResultView />
            </>
          ) : loadingSchema ? (
            <p className="help">Loading tools…</p>
          ) : (
            <p className="help">Select a tool on the left to begin.</p>
          )}
        </main>
      </div>
      )}
    </div>
  );
}
