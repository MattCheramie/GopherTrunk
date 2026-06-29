import { Link, useParams } from "react-router-dom";
import { useStore } from "../store/shared";
import { api } from "../api/client";
import type { Detail, Result } from "../api/types";
import { SymbolHistogram } from "../viz/SymbolHistogram";
import { Constellation } from "../viz/Constellation";
import { PSD } from "../viz/PSD";
import { Spectrogram } from "../viz/Spectrogram";
import { EyeDiagram } from "../viz/EyeDiagram";
import { SymbolScope } from "../viz/SymbolScope";
import { RotationTracker } from "../viz/RotationTracker";
import { VSAMetrics } from "../viz/VSAMetrics";
import { SyncLandscape } from "../viz/SyncLandscape";
import { ReceiverStates } from "../viz/ReceiverStates";
import { GrantsTable, EventTimeline } from "../viz/Tables";

export function Results() {
  const { captureId = "" } = useParams();
  const analysis = useStore((s) => s.analyses[captureId]);
  const capture = useStore((s) => s.captures.find((c) => c.id === captureId));
  const config = useStore((s) => s.config);

  if (!analysis) {
    return (
      <div className="card text-sm text-muted">
        No analysis for this capture.{" "}
        <Link to="/captures" className="text-accent">
          Back to captures
        </Link>
      </div>
    );
  }

  const r = analysis.result;
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold">{capture?.name ?? "Capture"}</h2>
        <StateBadge state={analysis.state} />
        {analysis.jobId && r && (
          <div className="ml-auto flex gap-2 text-sm">
            {["json", "jsonl", "yaml", "csv"].map((f) => (
              <a
                key={f}
                className="btn-ghost"
                href={api.exportURL(config, analysis.jobId!, f)}
              >
                {f}
              </a>
            ))}
          </div>
        )}
      </div>

      {analysis.state === "error" && (
        <div className="card text-sm text-err">Run failed: {analysis.error}</div>
      )}

      {analysis.state === "running" && (
        <div className="card">
          <h3 className="mb-2 text-sm font-semibold">Live decode events</h3>
          <div className="max-h-72 overflow-y-auto font-mono text-xs">
            {analysis.events.slice(-300).map((e) => (
              <div key={e.seq}>
                <span className="text-muted">{e.offset_sec.toFixed(2)}s</span>{" "}
                <span className="text-accent">{e.kind}</span>
              </div>
            ))}
            {analysis.events.length === 0 && (
              <span className="text-muted">decoding…</span>
            )}
          </div>
        </div>
      )}

      {r && <Summary r={r} />}
      {r && <VizGrid r={r} iq={analysis.iqTaps} jobId={analysis.jobId} />}
    </div>
  );
}

function StateBadge({ state }: { state: string }) {
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

function Summary({ r }: { r: Result }) {
  const cells: [string, string][] = [
    ["Protocol", r.protocol],
    ["Duration", `${r.duration_sec.toFixed(2)} s @ ${r.sample_rate_hz} Hz`],
    ["Samples", r.total_samples.toLocaleString()],
    [
      "Symbols",
      `${r.symbols.toLocaleString()} (${r.effective_baud.toFixed(0)} baud, ${r.baud_deviation_pct >= 0 ? "+" : ""}${r.baud_deviation_pct.toFixed(1)}%)`,
    ],
    [
      "Lock",
      r.locked
        ? `LOCKED${r.lock ? ` @ ${r.lock.frequency_hz} Hz` : ""} (${r.lock_latency_sec.toFixed(2)}s)`
        : "NOT LOCKED",
    ],
    ["Grants", String(r.grants.length)],
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
      {cells.map(([k, v]) => (
        <div key={k} className="card">
          <div className="text-xs uppercase tracking-wide text-muted">{k}</div>
          <div className={`mt-1 text-sm ${k === "Lock" && r.locked ? "text-ok" : ""}`}>{v}</div>
        </div>
      ))}
    </div>
  );
}

function VizGrid({ r, iq, jobId }: { r: Result; iq?: Result["iq_taps"]; jobId?: string }) {
  const detail = (r.detail ?? {}) as Detail;
  const soft = iq?.soft_samples ?? [];
  const hasIQ = !!(iq && iq.decimated_iq.length > 0);
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {r.signal && <SymbolHistogram signal={r.signal} />}
      {r.signal?.demod && <VSAMetrics demod={r.signal.demod} />}
      {hasIQ && <Constellation points={iq!.decimated_iq} />}
      {hasIQ && jobId && <PSD jobId={jobId} />}
      {hasIQ && jobId && <Spectrogram jobId={jobId} />}
      {(soft.length > 1 || detail.soft_eye) && (
        <EyeDiagram soft={soft} eye={detail.soft_eye} />
      )}
      {iq && (iq.symbol_dibits?.length ?? 0) > 0 && (
        <SymbolScope
          soft={iq.symbol_soft ?? []}
          dibits={iq.symbol_dibits ?? []}
          cardinality={iq.symbol_cardinality}
        />
      )}
      {iq && (iq.diff_phase?.length ?? 0) > 0 && (
        <RotationTracker phase={iq.diff_phase ?? []} />
      )}
      {detail.sync && <SyncLandscape sync={detail.sync} />}
      {detail.receiver_states && detail.receiver_states.length > 0 && (
        <ReceiverStates states={detail.receiver_states} />
      )}
      <GrantsTable grants={r.grants} />
      <EventTimeline events={r.events} />
    </div>
  );
}
