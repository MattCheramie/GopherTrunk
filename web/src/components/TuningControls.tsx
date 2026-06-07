// TuningControls — the shared offset / frequency / Hold / Centre block
// used by both the Constellation and Symbol scope panels. Pulling it into
// one component keeps the two panels' tuning UX identical and gives a
// single home for the fine-grained entry both panels need.
//
// Two equivalent ways to land on a channel:
//   - Offset (kHz, relative to the SDR centre), with 1 Hz numeric
//     resolution so channel grids like 6.25 / 12.5 kHz are reachable.
//   - Frequency (absolute, MHz) — type the channel directly; the offset
//     is derived from the SDR centre. Disabled until the device centre is
//     known. Both fields read off the same offset state, so they stay in
//     sync.
//
// The slider is for coarse dragging; the numeric fields are the precise
// path. Editing any control pins the view (Hold on) via onOffsetChange —
// the caller owns that side effect.

interface TuningControlsProps {
  // SDR centre in Hz, or null until the device is known. Gates the MHz
  // field (which needs an absolute reference) but not the offset field.
  centerHz: number | null;
  // Nyquist bound for the offset, in kHz (|offset| <= sample_rate/2).
  maxOffsetKHz: number;
  // Current (already clamped) offset in kHz.
  offsetKHz: number;
  // Set a new offset. The caller is expected to pin the view (Hold on).
  onOffsetChange: (khz: number) => void;
  hold: boolean;
  onHoldChange: (hold: boolean) => void;
  // True when, with Hold off, the view is tracking an active call.
  followingActive: boolean;
  // Recentre on the SDR centre (offset 0). Caller pins the view.
  onCentre: () => void;
}

// Trim a number to `digits` decimals without trailing zeros, so the
// controlled inputs show "6.25" rather than "6.250" yet still accept the
// fine values the reporter needs.
function trim(n: number, digits: number): number {
  return parseFloat(n.toFixed(digits));
}

export function TuningControls({
  centerHz,
  maxOffsetKHz,
  offsetKHz,
  onOffsetChange,
  hold,
  onHoldChange,
  followingActive,
  onCentre,
}: TuningControlsProps) {
  const freqMHz = centerHz != null ? (centerHz + offsetKHz * 1000) / 1e6 : null;

  return (
    <>
      <label className="flex items-center gap-2">
        <span className="text-muted">Offset</span>
        <input
          type="range"
          min={-maxOffsetKHz}
          max={maxOffsetKHz}
          step={0.05}
          value={offsetKHz}
          onChange={(e) => onOffsetChange(Number(e.target.value))}
          className="w-40 accent-sky-400"
          aria-label="View offset from SDR centre, kHz"
        />
        <input
          type="number"
          step={0.001}
          value={trim(offsetKHz, 3)}
          onChange={(e) => onOffsetChange(Number(e.target.value))}
          className="w-24 bg-surface border border-border rounded px-2 py-1 text-right font-mono"
          aria-label="View offset from SDR centre in kHz (numeric)"
        />
        <span className="text-muted">kHz</span>
      </label>

      <label className="flex items-center gap-2">
        <span className="text-muted">Freq</span>
        <input
          type="number"
          step={0.000001}
          value={freqMHz != null ? trim(freqMHz, 6) : ""}
          disabled={centerHz == null}
          onChange={(e) => {
            if (centerHz == null) return;
            const mhz = Number(e.target.value);
            if (!Number.isFinite(mhz)) return;
            onOffsetChange((mhz * 1e6 - centerHz) / 1000);
          }}
          className="w-32 bg-surface border border-border rounded px-2 py-1 text-right font-mono disabled:opacity-40"
          aria-label="Tuned frequency, MHz"
        />
        <span className="text-muted">MHz</span>
      </label>

      <button
        type="button"
        className="rounded border border-border px-2 py-1 hover:bg-surface disabled:opacity-40"
        disabled={offsetKHz === 0}
        onClick={onCentre}
        title="Recentre the view on the SDR centre frequency"
      >
        Centre
      </button>

      <label
        className="flex items-center gap-1.5"
        title="When off, the view follows the newest active call on this SDR"
      >
        <input
          type="checkbox"
          checked={hold}
          onChange={(e) => onHoldChange(e.target.checked)}
          className="accent-sky-400"
        />
        <span>
          Hold
          {!hold && followingActive && (
            <span className="text-accent"> · following call</span>
          )}
        </span>
      </label>
    </>
  );
}
