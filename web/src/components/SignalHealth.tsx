import { Badge } from "./ui/Badge";

// SignalHealth — shared signal-quality readouts so an operator sees signal
// health in context (on the Scanner cockpit, in a live call, in call
// history) instead of having to open a separate DSP scope. Every trunking
// UI shows an inline "is it healthy" cue next to the call/channel; these are
// GopherTrunk's.
//
// Three primitives:
//   - SignalQualityChip — the decoder's clean/marginal/poor verdict pill.
//   - SignalLevelBar    — raw front-end level (dBFS) as a bar + number.
//   - CallHealth        — a per-call badge row (SNR / EVM / level) built from
//                         the measured figures the daemon stamps on a call.

// SignalQualityChip is the simple control-channel signal indicator: a
// clean/marginal/poor pill (green/amber/red), with the measured carrier offset
// in its tooltip. Backed by the decoder's live TSBK (P25) / BSCH (TETRA)
// frame-error rate, so it works across protocols — not just TETRA.
export function SignalQualityChip({
  quality,
  offsetHz,
}: {
  quality: string;
  offsetHz?: number;
}) {
  const tone =
    quality === "clean" ? "ok" : quality === "marginal" ? "warn" : "err";
  const offset =
    offsetHz != null ? ` · offset ${offsetHz > 0 ? "+" : ""}${offsetHz} Hz` : "";
  return (
    <span title={`control-channel signal: ${quality}${offset}`}>
      <Badge tone={tone}>signal: {quality}</Badge>
    </span>
  );
}

// SignalLevelBar renders the locked carrier's raw front-end level (mean channel
// power in dBFS) as a numeric read-out plus a small level bar, so an operator can
// aim an antenna or trim LNA gain against a live number instead of just the
// clean/marginal/poor pill. This is signal STRENGTH, not decode quality — a strong
// bar with a red quality chip means "plenty of signal, but something else (offset,
// overload, wrong colour code) is hurting the decode".
//
// The bar maps the useful SDR range −90…−20 dBFS to 0…100%. Colour is by level:
// green ≥ −45 dBFS, amber −45…−60, red below −60.
export function SignalLevelBar({ dbfs }: { dbfs: number }) {
  const floorDb = -90;
  const ceilDb = -20;
  const pct = Math.max(
    0,
    Math.min(100, ((dbfs - floorDb) / (ceilDb - floorDb)) * 100),
  );
  const color = dbfs >= -45 ? "#34d399" : dbfs >= -60 ? "#fbbf24" : "#f87171";
  return (
    <span
      className="inline-flex items-center gap-1.5"
      title={`signal level ${dbfs.toFixed(1)} dBFS (mean channel power) — aim antenna / trim LNA gain to maximise`}
    >
      <span
        className="inline-block h-1.5 w-12 rounded-full bg-panel overflow-hidden"
        aria-hidden
      >
        <span
          className="block h-full rounded-full"
          style={{ width: `${pct}%`, backgroundColor: color }}
        />
      </span>
      <span className="font-mono text-xs text-muted">{dbfs.toFixed(1)} dBFS</span>
    </span>
  );
}

// CallHealth renders a compact per-call signal-health readout from the figures
// the daemon stamps on a call. SNR (dB) and EVM (%) are the true decode-quality
// numbers — the ones to compare against another decoder — and are measured only
// by the P25 Phase 1 chains today; signal level (dBFS) is front-end power. When a
// call carries none of them (most protocols, currently), renders an em dash so a
// history table cell is never blank.
//
// Tone thresholds match the C4FM regime GopherTrunk targets: SNR ≥18 dB clean /
// ≥10 marginal / below poor; EVM ≤10% clean / ≤20 marginal / above poor.
export function CallHealth({
  evmPct,
  snrDb,
  signalDbfs,
}: {
  evmPct?: number | null;
  snrDb?: number | null;
  signalDbfs?: number | null;
}) {
  const hasSNR = snrDb != null && Number.isFinite(snrDb);
  const hasEVM = evmPct != null && Number.isFinite(evmPct);
  const hasLevel = signalDbfs != null && Number.isFinite(signalDbfs);
  if (!hasSNR && !hasEVM && !hasLevel) {
    return <span className="text-muted text-xs">—</span>;
  }
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      {hasSNR && (
        <Badge tone={snrDb! >= 18 ? "ok" : snrDb! >= 10 ? "warn" : "err"}>
          SNR {snrDb!.toFixed(1)} dB
        </Badge>
      )}
      {hasEVM && (
        <Badge tone={evmPct! <= 10 ? "ok" : evmPct! <= 20 ? "warn" : "err"}>
          EVM {evmPct!.toFixed(1)}%
        </Badge>
      )}
      {hasLevel && <SignalLevelBar dbfs={signalDbfs!} />}
    </span>
  );
}
