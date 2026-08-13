import { Badge } from "./ui/Badge";
import { qualityVerdict, type Quality } from "../lib/signalQuality";

// SignalHealth — shared signal-quality readouts so an operator sees signal
// health in context (on the Scanner cockpit, in a live call, in call
// history) instead of having to open a separate DSP scope. Every trunking
// UI shows an inline "is it healthy" cue next to the call/channel; these are
// GopherTrunk's.
//
// Two things get called "signal quality" and they measure DIFFERENT axes, so
// each chip says which one it is (an operator was rightly confused seeing the
// Scanner read "clean" and the Plots hub read "poor" for the same carrier):
//   - SignalQualityChip — DECODE health: the decoder's live TSBK (P25) / BSCH
//     (TETRA) frame-error rate. A locked, well-decoding control channel reads
//     "decode: clean" even at a modest raw SNR.
//   - SymbolQualityChip — SYMBOL quality: the raw recovered-symbol SNR / eye
//     from the live symbol stream. An ISI-smeared constellation reads
//     "symbol: poor" even while the control channel still decodes cleanly.
//   - SignalLevelBar    — raw front-end level (dBFS), a third orthogonal axis.
//   - SignalSummary     — a combined banner showing all of the above together.
//   - CallHealth        — a per-call badge row (SNR / EVM / level).

// SignalQualityChip is the DECODE-health indicator: a clean/marginal/poor pill
// (green/amber/red), with the measured carrier offset in its tooltip. Backed by
// the decoder's live TSBK (P25) / BSCH (TETRA) frame-error rate, so it works
// across protocols — not just TETRA. This is decode SUCCESS, a different axis
// from the raw symbol SNR (SymbolQualityChip) and the front-end level.
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
    <span
      title={`control-channel decode health (frame-error rate): ${quality}${offset} — a different axis from raw symbol SNR`}
    >
      <Badge tone={tone}>decode: {quality}</Badge>
    </span>
  );
}

// SymbolQualityChip is the SYMBOL-quality indicator: the clean/marginal/poor
// verdict from the raw recovered-symbol distribution (MER-like SNR on the C4FM
// soft track, level balance otherwise). It reads the constellation/eye the DSP
// actually sees, so it can say "poor" while the control channel still decodes
// cleanly (decode: clean) — the two are deliberately distinct axes.
export function SymbolQualityChip({ quality }: { quality: Quality | null }) {
  const verdict = qualityVerdict(quality);
  const tone =
    verdict === "clean"
      ? "ok"
      : verdict === "marginal"
        ? "warn"
        : verdict === "poor"
          ? "err"
          : "neutral";
  // Say WHICH measurement produced the verdict. The MER-like SNR only exists on
  // a 4-level (C4FM) soft track; on TETRA / π-4-DQPSK there is no such track and
  // the verdict comes from level balance instead. Naming it avoids the confusion
  // of a "poor" chip with no visible reason next to a clean decode.
  const detail =
    quality?.snrDb != null
      ? ` from level SNR (MER) ${quality.snrDb.toFixed(1)} dB`
      : quality
        ? ` from level balance ±${quality.balanceDev.toFixed(1)}% (no 4-level soft track on this modulation)`
        : "";
  return (
    <span
      title={`recovered-symbol quality${detail} — a different axis from control-channel decode health`}
    >
      <Badge tone={tone}>
        symbol: {verdict === "unknown" ? "acquiring…" : verdict}
      </Badge>
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

// SignalSummary is the combined at-a-glance banner: DECODE health, SYMBOL
// quality, and front-end LEVEL side by side and clearly labelled, so the same
// carrier reads consistently across the Dashboard, Scanner, and Plots hub
// instead of one page saying "clean" and another "poor" for what are actually
// two different measurements. Feed it the locked system's decode fields (from
// GET /api/v1/scanner) and the live symbol Quality (useSignalQuality).
export function SignalSummary({
  decodeQuality,
  offsetHz,
  signalDbfs,
  symbolQuality,
  status,
}: {
  decodeQuality?: string | null;
  offsetHz?: number;
  signalDbfs?: number | null;
  symbolQuality: Quality | null;
  status?: "connecting" | "open" | "closed" | "idle";
}) {
  const hasLevel = signalDbfs != null && Number.isFinite(signalDbfs);
  return (
    <div className="panel flex flex-wrap items-center gap-3 px-3 py-2 text-sm">
      <span className="text-muted text-xs uppercase tracking-wider">Signal</span>
      {decodeQuality ? (
        <SignalQualityChip quality={decodeQuality} offsetHz={offsetHz} />
      ) : (
        <span title="No locked control channel decoding yet">
          <Badge tone="neutral">decode: —</Badge>
        </span>
      )}
      <SymbolQualityChip quality={symbolQuality} />
      {symbolQuality?.snrDb != null && (
        <span className="font-mono text-xs text-muted">
          SNR {symbolQuality.snrDb.toFixed(1)} dB
        </span>
      )}
      {symbolQuality && (
        <span className="font-mono text-xs text-muted">
          balance ±{symbolQuality.balanceDev.toFixed(1)}%
        </span>
      )}
      {hasLevel && <SignalLevelBar dbfs={signalDbfs!} />}
      {status && (
        <span className="ml-auto text-xs text-muted">
          {status === "open"
            ? "live"
            : status === "connecting"
              ? "connecting…"
              : status === "closed"
                ? "offline"
                : "idle"}
        </span>
      )}
    </div>
  );
}
