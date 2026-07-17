import { useEffect, useMemo, useState } from "react";
import { useStore } from "./store/shared";
import { ParamField } from "./components/ParamField";
import { ResultView } from "./components/ResultView";
import { RecipeBuilder } from "./components/RecipeBuilder";
import { Bundle } from "./components/Bundle";
import { ConsoleNav } from "./ConsoleNav";

export function App() {
  const [page, setPage] = useState<"tools" | "recipe" | "bundle">("tools");
  const [bundleEnabled, setBundleEnabled] = useState(false);
  const [query, setQuery] = useState("");
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
    // Gate the Bundles page on the daemon advertising the bundle endpoints.
    // A standalone `cryptolab serve` 503s the runtime route, hiding the tab.
    let alive = true;
    fetch("/api/v1/runtime")
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        if (alive && j && typeof j === "object" && j.bundle === true) setBundleEnabled(true);
      })
      .catch(() => {
        /* standalone serve — leave the Bundles tab hidden */
      });
    return () => {
      alive = false;
    };
  }, [loadSchema]);

  // Group the (already category→tool→mode sorted) schema into workflow
  // categories so the sidebar reflects the triage → attack → verify pipeline
  // instead of an alphabetical tool list. A search box filters across
  // tool / mode / synopsis / category.
  const groups = useMemo(() => {
    const q = query.trim().toLowerCase();
    const match = (s: (typeof schema)[number]) =>
      !q ||
      s.tool.toLowerCase().includes(q) ||
      s.mode.toLowerCase().includes(q) ||
      (s.synopsis ?? "").toLowerCase().includes(q) ||
      (s.category ?? "").toLowerCase().includes(q);
    const cats: { category: string; items: { tool: string; mode: string; synopsis: string }[] }[] = [];
    for (const s of schema) {
      if (!match(s)) continue;
      const cat = s.category ?? "Other";
      let c = cats.find((x) => x.category === cat);
      if (!c) {
        c = { category: cat, items: [] };
        cats.push(c);
      }
      c.items.push({ tool: s.tool, mode: s.mode, synopsis: s.synopsis });
    }
    return cats;
  }, [schema, query]);

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
          {((bundleEnabled
            ? (["tools", "recipe", "bundle"] as const)
            : (["tools", "recipe"] as const)) as readonly ("tools" | "recipe" | "bundle")[]).map(
            (p) => (
              <button
                key={p}
                onClick={() => setPage(p)}
                className={`rounded px-2.5 py-1 text-sm ${
                  page === p ? "bg-accent/20 text-fg" : "text-muted hover:bg-panel"
                }`}
              >
                {p === "tools" ? "Tools" : p === "recipe" ? "Recipe Builder" : "Bundles"}
              </button>
            ),
          )}
        </nav>
        <div className="ml-auto flex items-center gap-3 text-xs text-muted">
          <ConsoleNav self="cryptolab" />
          <span>offline cryptographic research</span>
        </div>
      </header>

      {page === "bundle" ? (
        <main className="min-h-0 flex-1 overflow-y-auto p-4">
          <Bundle />
        </main>
      ) : page === "recipe" ? (
        <main className="min-h-0 flex-1 overflow-y-auto p-4">
          <RecipeBuilder />
        </main>
      ) : (
      <div className="flex min-h-0 flex-1">
        <nav className="flex w-60 shrink-0 flex-col overflow-hidden border-r border-line">
          <div className="border-b border-line p-2">
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search tools…"
              aria-label="Search tools"
              className="input w-full text-sm"
            />
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            {schema.length === 0 ? (
              <p className="help p-2">{loadingSchema ? "Loading tools…" : "No tools."}</p>
            ) : groups.length === 0 ? (
              <p className="help p-2">No tools match “{query}”.</p>
            ) : (
              groups.map((g) => (
                <div key={g.category} className="mb-3">
                  <div className="px-2 py-1 text-xs font-semibold uppercase tracking-wide text-accent">
                    {g.category}
                  </div>
                  {g.items.map((m) => (
                    <button
                      key={`${m.tool}/${m.mode}`}
                      onClick={() => select(m.tool, m.mode)}
                      title={m.synopsis}
                      className={`mb-0.5 block w-full truncate rounded px-2 py-1.5 text-left text-sm ${
                        tool === m.tool && mode === m.mode
                          ? "bg-accent/20 text-fg"
                          : "text-muted hover:bg-panel"
                      }`}
                    >
                      <span className="text-muted">{m.tool}</span>
                      <span className="text-fg"> {m.mode}</span>
                    </button>
                  ))}
                </div>
              ))
            )}
          </div>
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
                {selected.category ? (
                  <div className="text-xs font-semibold uppercase tracking-wide text-accent">
                    {selected.category}
                  </div>
                ) : null}
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
