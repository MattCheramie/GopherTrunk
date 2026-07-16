import { useStore } from "../store/shared";
import { api } from "../api/client";
import type { RunState } from "../store/shared";

function StateBadge({ state }: { state: RunState }) {
  const cls =
    state === "done"
      ? "bg-ok/20 text-ok"
      : state === "error"
        ? "bg-err/20 text-err"
        : state === "running"
          ? "bg-warn/20 text-warn"
          : "bg-panel text-muted";
  return <span className={`rounded px-2 py-0.5 text-xs ${cls}`}>{state}</span>;
}

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

// formatScalar renders a leaf value: integers stay integers, floats trim to 4
// significant decimals (so a count doesn't show 4.0000 and a ratio isn't a long
// tail), everything else stringifies.
function formatScalar(v: unknown): string {
  if (v === null || v === undefined) return "";
  if (typeof v === "number") {
    return Number.isInteger(v) ? String(v) : String(Number(v.toFixed(4)));
  }
  return String(v);
}

// KeyVals renders an object as readable `key=value` chips instead of a
// JSON one-liner, so a Field value or a Finding detail map is legible.
function KeyVals({ obj }: { obj: Record<string, unknown> }) {
  const entries = Object.entries(obj);
  if (entries.length === 0) return <span className="text-muted">—</span>;
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-0.5">
      {entries.map(([k, v]) => (
        <span key={k} className="whitespace-nowrap">
          <span className="text-muted">{k}=</span>
          <span>{isPlainObject(v) || Array.isArray(v) ? JSON.stringify(v) : formatScalar(v)}</span>
        </span>
      ))}
    </div>
  );
}

export function ResultView() {
  const run = useStore((s) => s.run);

  if (run.state === "idle") {
    return <div className="card help">Configure a tool and press Run to see results here.</div>;
  }

  const r = run.result;
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <h3 className="text-sm font-semibold">Result</h3>
        <StateBadge state={run.state} />
        {r ? (
          <span className="text-xs text-muted">
            {r.tool}/{r.mode}
          </span>
        ) : null}
      </div>

      {run.error ? <div className="card text-sm text-err">Run failed: {run.error}</div> : null}

      {r?.summary ? <div className="card text-sm">{r.summary}</div> : null}

      {r?.fields && r.fields.length > 0 ? (
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-4">
          {r.fields.map((f) => (
            <div key={f.key} className="card">
              <div className="text-xs uppercase tracking-wide text-muted">{f.key}</div>
              <div className="mt-1 break-words font-mono text-sm">
                {isPlainObject(f.value) ? <KeyVals obj={f.value} /> : formatScalar(f.value)}
              </div>
            </div>
          ))}
        </div>
      ) : null}

      {r?.findings && r.findings.length > 0 ? (
        <div className="card">
          <h4 className="mb-2 text-sm font-semibold">Findings ({r.findings.length})</h4>
          <table className="w-full text-left text-xs">
            <thead className="text-muted">
              <tr>
                <th className="pr-3">#</th>
                <th className="pr-3">Label</th>
                <th className="pr-3">Score</th>
                <th>Detail</th>
              </tr>
            </thead>
            <tbody>
              {r.findings.map((f, i) => (
                <tr key={i} className="border-t border-line align-top">
                  <td className="py-1 pr-3 text-muted">{i + 1}</td>
                  <td className="py-1 pr-3 font-mono">{f.label}</td>
                  <td className="py-1 pr-3 tabular-nums">{formatScalar(f.score)}</td>
                  <td className="py-1 font-mono text-muted">
                    {f.detail ? <KeyVals obj={f.detail} /> : ""}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {r?.notes && r.notes.length > 0 ? (
        <div className="card space-y-1">
          <h4 className="text-sm font-semibold">Notes</h4>
          {r.notes.map((n, i) => (
            <p key={i} className="help">
              {n}
            </p>
          ))}
        </div>
      ) : null}

      {run.artifacts.length > 0 && run.jobId ? (
        <div className="card">
          <h4 className="mb-2 text-sm font-semibold">Artifacts</h4>
          <ul className="space-y-1 text-sm">
            {run.artifacts.map((a) => (
              <li key={a.name} className="flex items-center justify-between gap-2">
                <span className="font-mono">{a.name}</span>
                <a
                  className="btn-ghost"
                  href={api.artifactURL(run.jobId!, a.name)}
                  download={a.name}
                >
                  Download ({a.size.toLocaleString()} B)
                </a>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="card">
        <h4 className="mb-2 text-sm font-semibold">Live log</h4>
        <div className="max-h-64 overflow-y-auto font-mono text-xs">
          {run.events.length === 0 ? (
            <span className="text-muted">{run.state === "running" ? "running…" : "no log output"}</span>
          ) : (
            run.events.map((e) => (
              <div key={e.seq}>
                <span className="text-muted">{e.time}</span>{" "}
                <span className="text-accent">{e.msg}</span>
                {e.attrs ? <span className="text-muted"> {e.attrs}</span> : null}
              </div>
            ))
          )}
        </div>
      </div>

      {r ? (
        <details className="card">
          <summary className="cursor-pointer select-none text-sm font-semibold">Raw JSON</summary>
          <pre className="mt-2 overflow-auto rounded-md border border-line bg-bg p-3 text-xs">
            {JSON.stringify(r, null, 2)}
          </pre>
        </details>
      ) : null}
    </div>
  );
}
