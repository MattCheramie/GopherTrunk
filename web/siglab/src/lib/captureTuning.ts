// parseCaptureTuning turns the capture form's "Center MHz" / "Bandwidth kHz"
// text into the request's center_hz / bandwidth_hz, or an error the operator
// can act on.
//
// It exists because the form used to coerce with Number() and silently fall
// back: anything that was not a plain decimal (a stray character pasted with
// the value, a decimal comma) became NaN, the centre was dropped, and the
// daemon carved the slice at the TUNER centre under the name the operator
// typed — a "442.8125 MHz" grab that neither repeater was in (15 Sep). A
// centre without a bandwidth was dropped the same way (the tuner is not
// retuned, so a centre only applies to a narrowband slice). Both are errors
// now, and the only silent path left is the one that was asked for: blank
// fields ⇒ a full-band grab at the tuner centre.
//
// Several centres ("442.3875, 443.2375" — comma, semicolon or whitespace
// separated) become centers_hz: the daemon records every slice from the same
// live stream, so the staged files are sample-synchronous.
export interface CaptureTuning {
  center_hz?: number;
  bandwidth_hz?: number;
  centers_hz?: number[];
}

export type CaptureTuningResult =
  | { ok: true; tuning: CaptureTuning }
  | { ok: false; error: string };

// A plain decimal with a dot: what every frequency in the console uses.
const decimal = /^[+-]?(\d+\.?\d*|\.\d+)$/;

function parsePositive(label: string, raw: string, unit: string): number | string {
  const text = raw.trim();
  if (!decimal.test(text)) {
    return `${label} "${raw}" is not a number — use a plain decimal like 442.3875 (${unit}); clear it for the default`;
  }
  const v = Number(text);
  if (!(v > 0)) {
    return `${label} must be greater than 0 ${unit}`;
  }
  return v;
}

// splitCentres tokenises the Center MHz field into one entry per centre.
// Whitespace and semicolons always separate; a comma separates only when
// every piece around it carries a decimal point ("442.3875,443.2375"), so a
// DECIMAL comma ("442,8125") is left as one token and rejected as not-a-number
// by the caller instead of being read as two centres (the 15 Sep guard).
function splitCentres(raw: string): string[] {
  const out: string[] = [];
  for (const tok of raw.split(/[;\s]+/)) {
    if (!tok) continue;
    if (tok.includes(",")) {
      const pieces = tok.split(",").filter((p) => p.length > 0);
      if (pieces.length > 0 && pieces.every((p) => p.includes("."))) {
        out.push(...pieces);
        continue;
      }
    }
    out.push(tok);
  }
  return out;
}

export function parseCaptureTuning(centerMHz: string, bandwidthKHz: string): CaptureTuningResult {
  const tuning: CaptureTuning = {};
  if (bandwidthKHz.trim()) {
    const v = parsePositive("Bandwidth kHz", bandwidthKHz, "kHz");
    if (typeof v === "string") return { ok: false, error: v };
    tuning.bandwidth_hz = Math.round(v * 1e3);
  }
  const parts = splitCentres(centerMHz);
  if (parts.length > 0) {
    const centers: number[] = [];
    for (const part of parts) {
      const v = parsePositive("Center MHz", part, "MHz");
      if (typeof v === "string") return { ok: false, error: v };
      centers.push(Math.round(v * 1e6));
    }
    if (tuning.bandwidth_hz === undefined) {
      return {
        ok: false,
        error:
          "Center MHz needs a Bandwidth kHz: the tuner is not retuned, so a centre only applies to a " +
          "narrowband slice carved around it — set a bandwidth, or clear the centre for a full-band " +
          "grab at the tuner centre",
      };
    }
    if (new Set(centers).size !== centers.length) {
      return { ok: false, error: "Center MHz lists the same frequency twice" };
    }
    if (centers.length === 1) {
      tuning.center_hz = centers[0];
    } else {
      tuning.centers_hz = centers;
    }
  }
  return { ok: true, tuning };
}
