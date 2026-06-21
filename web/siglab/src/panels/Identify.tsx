import { useState } from "react";
import { Bar } from "react-chartjs-2";
import { useStore } from "../store/shared";
import { ensureChart } from "../viz/chartSetup";
import type { IdentifyResult } from "../api/types";

ensureChart();

// Identify scans a capture against all candidate protocols and ranks them by
// the engine's composite score (identify.go).
export function Identify() {
  const captures = useStore((s) => s.captures);
  const identify = useStore((s) => s.identify);
  const [captureId, setCaptureId] = useState("");
  const [result, setResult] = useState<IdentifyResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function onIdentify() {
    if (!captureId) return;
    setBusy(true);
    setErr(null);
    try {
      setResult(await identify(captureId));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <div className="card flex flex-wrap items-end gap-3">
        <div className="flex-1">
          <label className="label">Capture</label>
          <select className="input" value={captureId} onChange={(e) => setCaptureId(e.target.value)}>
            <option value="">— select —</option>
            {captures.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>
        <button className="btn" disabled={busy || !captureId} onClick={onIdentify}>
          {busy ? "Scanning…" : "Identify"}
        </button>
      </div>
      {err && <div className="card text-xs text-err">{err}</div>}

      {result && (
        <div className="card">
          <p className="mb-2 text-sm">
            Winner: <span className="font-semibold text-accent">{result.winner}</span> · confidence{" "}
            {(result.confidence * 100).toFixed(0)}%
            {result.inconclusive && <span className="text-warn"> (inconclusive)</span>}
          </p>
          <Bar
            data={{
              labels: result.candidates.map((c) => c.protocol),
              datasets: [
                {
                  label: "score",
                  data: result.candidates.map((c) => c.score),
                  backgroundColor: result.candidates.map((c) =>
                    c.protocol === result.winner ? "rgba(34,197,94,0.7)" : "rgba(56,189,248,0.5)",
                  ),
                },
              ],
            }}
            options={{
              indexAxis: "y" as const,
              plugins: { legend: { display: false } },
              scales: { x: { beginAtZero: true } },
            }}
          />
          <table className="mt-3 w-full text-left text-xs">
            <thead className="text-muted">
              <tr>
                <th className="pr-3">Protocol</th>
                <th className="pr-3">Locked</th>
                <th className="pr-3">Sync hits</th>
                <th className="pr-3">FEC pass</th>
                <th>Score</th>
              </tr>
            </thead>
            <tbody>
              {result.candidates.map((c) => (
                <tr key={c.protocol} className="border-t border-line">
                  <td className="py-1 pr-3">{c.protocol}</td>
                  <td className="pr-3">{c.locked ? "✓" : ""}</td>
                  <td className="pr-3">{c.sync_hits}</td>
                  <td className="pr-3">{(c.fec_pass_rate * 100).toFixed(0)}%</td>
                  <td>{c.score.toFixed(3)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
