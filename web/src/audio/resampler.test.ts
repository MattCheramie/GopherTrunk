import { describe, it, expect } from "vitest";
import { LinearResampler } from "./resampler";

// Resample `input` in one shot at the given rates (helper for the
// continuity assertions below).
function once(
  input: number[],
  inRate: number,
  outRate: number,
): number[] {
  return Array.from(
    new LinearResampler(inRate, outRate).process(Float32Array.from(input)),
  );
}

describe("LinearResampler", () => {
  it("rejects non-positive rates", () => {
    expect(() => new LinearResampler(0, 8000)).toThrow(/positive/);
    expect(() => new LinearResampler(8000, -1)).toThrow(/positive/);
  });

  it("is a zero-copy passthrough when the rates match", () => {
    const r = new LinearResampler(8000, 8000);
    expect(r.passthrough).toBe(true);
    const input = Float32Array.from([0.1, -0.2, 0.3]);
    // Same reference back: no work, no allocation.
    expect(r.process(input)).toBe(input);
  });

  it("linearly interpolates when upsampling 2x", () => {
    // ratio = inRate/outRate = 0.5: one output every half input sample.
    expect(once([0, 10, 20, 30], 1, 2)).toEqual([0, 5, 10, 15, 20, 25]);
  });

  it("preserves a DC level (no boundary ripple)", () => {
    const out = once([0.5, 0.5, 0.5, 0.5], 8000, 48000);
    for (const v of out) expect(v).toBeCloseTo(0.5, 6);
  });

  it("joins chunks seamlessly — split feed equals one-shot feed", () => {
    // The whole point of #629: continuous state across chunk boundaries.
    const whole = once([0, 10, 20, 30, 40, 50], 1, 2);

    const r = new LinearResampler(1, 2);
    const split = [
      ...r.process(Float32Array.from([0, 10])),
      ...r.process(Float32Array.from([20, 30])),
      ...r.process(Float32Array.from([40, 50])),
    ];

    expect(split).toEqual(whole);
  });

  it("carries fractional position across a downsample (3:1)", () => {
    // Compare a split feed against the one-shot result for outRate<inRate too.
    const whole = once([0, 3, 6, 9, 12, 15, 18], 3, 1);

    const r = new LinearResampler(3, 1);
    const split = [
      ...r.process(Float32Array.from([0, 3, 6])),
      ...r.process(Float32Array.from([9, 12])),
      ...r.process(Float32Array.from([15, 18])),
    ];

    expect(split).toEqual(whole);
  });

  it("handles empty and single-sample chunks without emitting garbage", () => {
    const r = new LinearResampler(8000, 16000);
    expect(Array.from(r.process(new Float32Array(0)))).toEqual([]);
    // A lone first sample has no upper neighbour yet, so nothing is emitted
    // until the next sample arrives — then output is continuous.
    expect(Array.from(r.process(Float32Array.from([1])))).toEqual([]);
    const out = Array.from(r.process(Float32Array.from([1, 1])));
    for (const v of out) expect(v).toBeCloseTo(1, 6);
  });
});
