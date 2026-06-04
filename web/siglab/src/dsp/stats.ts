// SciPy-like numeric helpers. The summary statistics are delegated to stdlib
// (the numerically-careful stats-base routines); the window functions are the
// small DSP primitives the FFT path needs.

import stdlibMean from "@stdlib/stats-base-mean";
import stdlibVariance from "@stdlib/stats-base-variance";

// mean/std wrap stdlib's stats-base routines (SciPy's scipy.stats analogues).
export function mean(xs: ArrayLike<number>): number {
  if (xs.length === 0) return 0;
  return stdlibMean(xs.length, xs, 1);
}

export function std(xs: ArrayLike<number>): number {
  if (xs.length < 2) return 0;
  // correction = 1 → sample standard deviation (ddof=1, like numpy/SciPy).
  return Math.sqrt(stdlibVariance(xs.length, 1, xs, 1));
}

// hannWindow is the raised-cosine taper applied before each FFT to suppress
// spectral leakage (SciPy's scipy.signal.windows.hann).
export function hannWindow(n: number): Float32Array {
  const w = new Float32Array(n);
  if (n === 1) {
    w[0] = 1;
    return w;
  }
  for (let i = 0; i < n; i++) {
    w[i] = 0.5 - 0.5 * Math.cos((2 * Math.PI * i) / (n - 1));
  }
  return w;
}

// percentile returns the p-th (0..1) percentile of xs via linear interpolation.
export function percentile(xs: number[], p: number): number {
  if (xs.length === 0) return 0;
  const sorted = [...xs].sort((a, b) => a - b);
  const idx = p * (sorted.length - 1);
  const lo = Math.floor(idx);
  const hi = Math.ceil(idx);
  if (lo === hi) return sorted[lo];
  return sorted[lo] + (sorted[hi] - sorted[lo]) * (idx - lo);
}
