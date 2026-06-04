// Focused stdlib packages ship handwritten types that don't always resolve
// cleanly under bundler module resolution; declare them as untyped so the
// build stays green while we wrap them in narrow local facades (see
// ../dsp/stats.ts). These provide the SciPy-like numeric routines.
declare module "@stdlib/stats-base-mean" {
  const mean: (N: number, x: ArrayLike<number>, stride: number) => number;
  export default mean;
}
declare module "@stdlib/stats-base-variance" {
  const variance: (
    N: number,
    correction: number,
    x: ArrayLike<number>,
    stride: number,
  ) => number;
  export default variance;
}
