// signalQuality — the recovered-symbol quality math shared by the Histogram
// panel and the Plots-hub verdict. A clean 4-level (C4FM / CQPSK) control
// channel is scrambled, so each level sits near 25%; a collapsed slicer skews
// the distribution. The C4FM soft track additionally yields a per-level
// mean/spread and an MER-like SNR (a "level meter").

export const WINDOW_SYMBOLS = 4000;
export const CARDINALITY = 4;

export interface Quality {
  pct: number[]; // % per dibit bin
  total: number;
  balanceDev: number; // max |bin% − ideal%|
  snrDb: number | null; // MER-like estimate (C4FM soft only)
  levels: { mean: number; std: number }[] | null;
}

export function computeQuality(dibits: number[], soft: number[]): Quality {
  const cnt = new Array(CARDINALITY).fill(0);
  for (const d of dibits) if (d >= 0 && d < CARDINALITY) cnt[d]++;
  const total = dibits.length;
  const pct = cnt.map((c) => (total > 0 ? (100 * c) / total : 0));
  const ideal = 100 / CARDINALITY;
  const balanceDev = pct.reduce((m, p) => Math.max(m, Math.abs(p - ideal)), 0);

  // Per-level soft stats + MER-like SNR, when the soft track is aligned.
  let snrDb: number | null = null;
  let levels: { mean: number; std: number }[] | null = null;
  if (soft.length > 0 && soft.length === dibits.length) {
    const sum = new Array(CARDINALITY).fill(0);
    const sumsq = new Array(CARDINALITY).fill(0);
    for (let i = 0; i < dibits.length; i++) {
      const d = dibits[i];
      if (d < 0 || d >= CARDINALITY) continue;
      sum[d] += soft[i];
      sumsq[d] += soft[i] * soft[i];
    }
    levels = [];
    let grandSum = 0;
    let noise = 0;
    let signal = 0;
    for (let d = 0; d < CARDINALITY; d++) {
      const c = cnt[d];
      const mean = c > 0 ? sum[d] / c : 0;
      const varr = c > 0 ? Math.max(0, sumsq[d] / c - mean * mean) : 0;
      levels.push({ mean, std: Math.sqrt(varr) });
      grandSum += sum[d];
      noise += c * varr;
    }
    const mu = total > 0 ? grandSum / total : 0;
    for (let d = 0; d < CARDINALITY; d++) {
      signal += cnt[d] * (levels[d].mean - mu) * (levels[d].mean - mu);
    }
    if (total > 0 && noise > 0) {
      snrDb = 10 * Math.log10(signal / noise);
    }
  }
  return { pct, total, balanceDev, snrDb, levels };
}

export type Verdict = "clean" | "marginal" | "poor" | "unknown";

// qualityVerdict buckets a Quality into a clean/marginal/poor pill for the
// at-a-glance readout, preferring the MER-like SNR (C4FM soft) when present and
// otherwise falling back to how evenly the four levels balance (a collapsed
// slicer skews it far off the 25% ideal). Returns "unknown" until enough
// symbols have accumulated to be meaningful.
export function qualityVerdict(q: Quality | null): Verdict {
  if (!q || q.total < 200) return "unknown";
  if (q.snrDb != null) {
    if (q.snrDb >= 18) return "clean";
    if (q.snrDb >= 10) return "marginal";
    return "poor";
  }
  // No soft SNR (e.g. CQPSK): judge by level balance. A healthy scrambled
  // channel stays within a few points of 25%; a closing eye collapses it.
  if (q.balanceDev <= 6) return "clean";
  if (q.balanceDev <= 12) return "marginal";
  return "poor";
}
