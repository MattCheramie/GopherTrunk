import { useEffect, useMemo, useState } from "react";
import { useRecipe } from "../store/recipe";
import type { BuilderStep } from "../store/recipe";
import type { OpSpec, RecipeStepResult } from "../api/types";

// RecipeBuilder is the interactive, config-builder-style editor for the
// `recipe` pipeline: pick an input, assemble an ordered list of operations
// (transforms + analyses) with their parameters, run it server-side, and read
// the per-step report plus the final bytes. It calls the dedicated
// /api/v1/cryptolab/recipe endpoint (no job needed — the pipeline is fast).
export function RecipeBuilder() {
  const ops = useRecipe((s) => s.ops);
  const loadOps = useRecipe((s) => s.loadOps);
  const opsError = useRecipe((s) => s.opsError);
  const steps = useRecipe((s) => s.steps);
  const addStep = useRecipe((s) => s.addStep);
  const clearSteps = useRecipe((s) => s.clearSteps);
  const input = useRecipe((s) => s.input);
  const uploading = useRecipe((s) => s.uploading);
  const uploadInput = useRecipe((s) => s.uploadInput);
  const running = useRecipe((s) => s.running);
  const result = useRecipe((s) => s.result);
  const error = useRecipe((s) => s.error);
  const run = useRecipe((s) => s.run);

  const [pick, setPick] = useState("");

  useEffect(() => {
    void loadOps();
  }, [loadOps]);

  const grouped = useMemo(() => {
    // External ops shell out to a host program; they are CLI-only and the
    // recipe endpoint refuses them, so the web palette hides them.
    const web = ops.filter((o) => !o.external);
    return { transforms: web.filter((o) => o.transform), analyses: web.filter((o) => !o.transform) };
  }, [ops]);

  const canRun = !!input && steps.length > 0 && !running && !uploading;

  return (
    <div className="grid gap-4 lg:grid-cols-[18rem_minmax(0,1fr)]">
      {/* Op palette */}
      <aside className="space-y-3">
        <div className="card space-y-2">
          <div className="label">Operations</div>
          {opsError ? <p className="help text-err">{opsError}</p> : null}
          <OpPalette title="Transforms" ops={grouped.transforms} onAdd={addStep} />
          <OpPalette title="Analyses" ops={grouped.analyses} onAdd={addStep} />
        </div>
      </aside>

      {/* Pipeline + result */}
      <section className="space-y-4">
        {/* Input */}
        <div className="card space-y-2">
          <div className="label">Input</div>
          <input
            type="file"
            className="input"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) void uploadInput(f);
            }}
          />
          {uploading ? <p className="help text-accent">uploading…</p> : null}
          {input ? (
            <p className="help text-ok">
              ✓ {input.name} ({input.size.toLocaleString()} bytes)
            </p>
          ) : (
            <p className="help">Upload the payload to run the pipeline over.</p>
          )}
        </div>

        {/* Pipeline */}
        <div className="card space-y-3">
          <div className="flex items-center justify-between">
            <div className="label mb-0">Pipeline ({steps.length})</div>
            <div className="flex items-center gap-2">
              <select className="input w-auto py-1" value={pick} onChange={(e) => setPick(e.target.value)}>
                <option value="">add operation…</option>
                <optgroup label="Transforms">
                  {grouped.transforms.map((o) => (
                    <option key={o.name} value={o.name}>
                      {o.name}
                    </option>
                  ))}
                </optgroup>
                <optgroup label="Analyses">
                  {grouped.analyses.map((o) => (
                    <option key={o.name} value={o.name}>
                      {o.name}
                    </option>
                  ))}
                </optgroup>
              </select>
              <button
                className="btn-ghost"
                disabled={!pick}
                onClick={() => {
                  if (pick) addStep(pick);
                }}
              >
                + Add
              </button>
              {steps.length > 0 ? (
                <button className="btn-danger" onClick={() => clearSteps()}>
                  Clear
                </button>
              ) : null}
            </div>
          </div>

          {steps.length === 0 ? (
            <p className="help">
              Empty pipeline. Add operations from the palette; bytes flow from each step into the next.
            </p>
          ) : (
            <ol className="space-y-2">
              {steps.map((st, i) => (
                <StepCard key={st.id} step={st} index={i} count={steps.length} />
              ))}
            </ol>
          )}

          <div className="flex items-center gap-3 pt-1">
            <button className="btn" disabled={!canRun} onClick={() => void run()}>
              {running ? "Running…" : "Run pipeline"}
            </button>
            {!input ? <span className="help">Upload an input to run.</span> : null}
          </div>
          {error ? (
            <div className="rounded-md border border-err/40 bg-err/10 px-3 py-2 text-sm text-err">{error}</div>
          ) : null}
        </div>

        {result ? <RecipeResultView /> : null}
      </section>
    </div>
  );
}

function OpPalette({ title, ops, onAdd }: { title: string; ops: OpSpec[]; onAdd: (op: string) => void }) {
  return (
    <div>
      <div className="mb-1 text-xs uppercase tracking-wide text-muted">{title}</div>
      <div className="flex flex-wrap gap-1">
        {ops.map((o) => (
          <button
            key={o.name}
            title={o.synopsis}
            onClick={() => onAdd(o.name)}
            className="rounded border border-line px-2 py-1 text-xs hover:bg-panel"
          >
            {o.name}
          </button>
        ))}
      </div>
    </div>
  );
}

function StepCard({ step, index, count }: { step: BuilderStep; index: number; count: number }) {
  const spec = useRecipe((s) => s.opSpec(step.op));
  const moveStep = useRecipe((s) => s.moveStep);
  const removeStep = useRecipe((s) => s.removeStep);
  const duplicateStep = useRecipe((s) => s.duplicateStep);
  const setStepParam = useRecipe((s) => s.setStepParam);

  return (
    <li className="rounded-md border border-line p-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-baseline gap-2">
          <span className="text-xs text-muted">{index + 1}.</span>
          <span className="font-mono text-sm">{step.op}</span>
          <span
            className={`rounded px-1.5 py-0.5 text-[10px] uppercase ${
              spec?.transform ? "bg-accent/20 text-accent" : "bg-panel text-muted"
            }`}
          >
            {spec?.transform ? "transform" : "analysis"}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button className="btn-ghost px-2 py-1" disabled={index === 0} onClick={() => moveStep(step.id, -1)} title="Move up">
            ↑
          </button>
          <button
            className="btn-ghost px-2 py-1"
            disabled={index === count - 1}
            onClick={() => moveStep(step.id, 1)}
            title="Move down"
          >
            ↓
          </button>
          <button className="btn-ghost px-2 py-1" onClick={() => duplicateStep(step.id)} title="Duplicate">
            Dup
          </button>
          <button className="btn-danger px-2 py-1" onClick={() => removeStep(step.id)} title="Remove">
            ✕
          </button>
        </div>
      </div>
      {spec?.synopsis ? <p className="help mt-1">{spec.synopsis}</p> : null}
      {(spec?.params ?? []).length > 0 ? (
        <div className="mt-2 grid gap-2 sm:grid-cols-2">
          {(spec?.params ?? []).map((p) => (
            <label key={p.name}>
              <span className="label">{p.label}</span>
              <input
                className="input"
                type={p.kind === "int" || p.kind === "float" ? "number" : "text"}
                step={p.kind === "float" ? "any" : undefined}
                value={step.params[p.name] ?? ""}
                placeholder={p.kind === "hex" ? "hex" : ""}
                onChange={(e) => setStepParam(step.id, p.name, e.target.value)}
              />
              {p.help ? <p className="help">{p.help}</p> : null}
            </label>
          ))}
        </div>
      ) : null}
    </li>
  );
}

function RecipeResultView() {
  const result = useRecipe((s) => s.result)!;

  const download = () => {
    if (!result.output_b64) return;
    const bin = atob(result.output_b64);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    const url = URL.createObjectURL(new Blob([bytes], { type: "application/octet-stream" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = "recipe-output.bin";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-3">
      {result.error ? (
        <div className="rounded-md border border-err/40 bg-err/10 px-3 py-2 text-sm text-err">
          Pipeline stopped: {result.error}
        </div>
      ) : (
        <div className="rounded-md border border-ok/30 bg-ok/10 px-3 py-2 text-sm text-ok">
          ✓ Ran {result.steps.length} step(s): {result.input_bytes} → {result.final_bytes} bytes
        </div>
      )}

      <div className="card overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="text-xs uppercase text-muted">
            <tr>
              <th className="py-1 pr-3">#</th>
              <th className="py-1 pr-3">op</th>
              <th className="py-1 pr-3">bytes</th>
              <th className="py-1">note</th>
            </tr>
          </thead>
          <tbody>
            {result.steps.map((s, i) => (
              <StepRow key={i} i={i} s={s} />
            ))}
          </tbody>
        </table>
      </div>

      <div className="card space-y-2">
        <div className="flex items-center justify-between">
          <div className="label mb-0">Output</div>
          {result.output_b64 ? (
            <button className="btn-ghost" onClick={download}>
              Download ({result.final_bytes.toLocaleString()} bytes)
            </button>
          ) : null}
        </div>
        <div>
          <div className="help">ASCII (first 64)</div>
          <pre className="overflow-x-auto rounded bg-bg p-2 font-mono text-xs">{result.final_ascii || "—"}</pre>
        </div>
        <div>
          <div className="help">Hex (first 64)</div>
          <pre className="overflow-x-auto rounded bg-bg p-2 font-mono text-xs">{result.final_hex || "—"}</pre>
        </div>
      </div>
    </div>
  );
}

function StepRow({ i, s }: { i: number; s: RecipeStepResult }) {
  const info = s.info ? Object.entries(s.info).map(([k, v]) => `${k}=${typeof v === "object" ? JSON.stringify(v) : String(v)}`).join(" ") : "";
  return (
    <tr className="border-t border-line/60 align-top">
      <td className="py-1 pr-3 text-muted">{i + 1}</td>
      <td className="py-1 pr-3 font-mono">
        {s.op}
        <span className="ml-1 text-[10px] uppercase text-muted">{s.transform ? "" : "(analysis)"}</span>
      </td>
      <td className="py-1 pr-3 font-mono text-xs text-muted">
        {s.bytes_in}
        {s.transform ? `→${s.bytes_out}` : ""}
      </td>
      <td className="py-1 text-xs text-muted">{s.note || info || "—"}</td>
    </tr>
  );
}
