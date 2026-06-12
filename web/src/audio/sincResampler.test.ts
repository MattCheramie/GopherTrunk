import { describe, it, expect } from "vitest";
import { SincResampler } from "./sincResampler";

// Resample `input` in one shot at the given rates (helper for the continuity
// assertions below).
function once(
  input: number[],
  inRate: number,
  outRate: number,
): number[] {
  return Array.from(
    new SincResampler(inRate, outRate).process(Float32Array.from(input)),
  );
}

// goertzelMag returns the magnitude of frequency `freq` (Hz) in `samples`
// taken at `rate` Hz — a single-bin DFT, enough to compare in-band signal
// energy against out-of-band image energy.
function goertzelMag(samples: number[], rate: number, freq: number): number {
  const w = (2 * Math.PI * freq) / rate;
  const cos = Math.cos(w);
  const coeff = 2 * cos;
  let s0 = 0;
  let s1 = 0;
  let s2 = 0;
  for (const x of samples) {
    s0 = x + coeff * s1 - s2;
    s2 = s1;
    s1 = s0;
  }
  const real = s1 - s2 * cos;
  const imag = s2 * Math.sin(w);
  return Math.sqrt(real * real + imag * imag) / samples.length;
}

describe("SincResampler", () => {
  it("rejects non-positive rates", () => {
    expect(() => new SincResampler(0, 8000)).toThrow(/positive/);
    expect(() => new SincResampler(8000, -1)).toThrow(/positive/);
  });

  it("is a zero-copy passthrough when the rates match", () => {
    const r = new SincResampler(8000, 8000);
    expect(r.passthrough).toBe(true);
    const input = Float32Array.from([0.1, -0.2, 0.3]);
    // Same reference back: no work, no allocation.
    expect(r.process(input)).toBe(input);
  });

  it("preserves a DC level (per-phase unity-gain normalization)", () => {
    const out = once(new Array(256).fill(0.5), 8000, 48000);
    // Skip the leading samples that still blend in the zero-padded history:
    // the kernel reaches full real-sample support only past output ~96
    // (input position >= HALF_WIDTH). Past that, every phase row sums to 1.
    for (const v of out.slice(120)) expect(v).toBeCloseTo(0.5, 4);
  });

  it("joins chunks seamlessly — split feed equals one-shot feed", () => {
    // The load-bearing #629 property: continuous state across chunk boundaries.
    // Use an exact 2x ratio (like the LinearResampler test) so the assertion is
    // bit-exact and not muddied by floating-point drift at the chunk boundary.
    const input = Array.from({ length: 600 }, (_, i) =>
      Math.sin((2 * Math.PI * 1000 * i) / 8000),
    );
    const whole = once(input, 8000, 16000);

    const r = new SincResampler(8000, 16000);
    const split: number[] = [];
    for (let i = 0; i < input.length; i += 37) {
      split.push(
        ...r.process(Float32Array.from(input.slice(i, i + 37))),
      );
    }

    expect(split.length).toBe(whole.length);
    for (let i = 0; i < whole.length; i++) {
      expect(split[i]).toBe(whole[i]);
    }
  });

  it("handles empty and short chunks without emitting garbage", () => {
    const r = new SincResampler(8000, 16000);
    expect(Array.from(r.process(new Float32Array(0)))).toEqual([]);
    // A lone first sample has no right-side kernel support yet, so nothing is
    // emitted until enough lookahead arrives — then output stays continuous.
    expect(Array.from(r.process(Float32Array.from([1])))).toEqual([]);
    expect(() =>
      r.process(Float32Array.from(new Array(64).fill(1))),
    ).not.toThrow();
  });

  it("band-limits an 8k->48k upsample (suppresses images linear interp leaks)", () => {
    // A 3.4 kHz tone is in-band (below the 4 kHz input Nyquist). After ideal
    // band-limited upsampling to 48 kHz its only spectral images sit at
    // 8k +/- 3.4k = 4.6k / 11.4k etc. — all above 4 kHz and must be heavily
    // attenuated. Linear interpolation leaks the 4.6 kHz image badly; sinc
    // does not.
    const n = 4096;
    const tone = Array.from({ length: n }, (_, i) =>
      Math.sin((2 * Math.PI * 3400 * i) / 8000),
    );
    const out = once(tone, 8000, 48000).slice(64); // drop the warm-up

    const signal = goertzelMag(out, 48000, 3400); // in-band tone
    const image = goertzelMag(out, 48000, 4600); // first alias image
    // Image at least 40 dB below the passband tone.
    expect(20 * Math.log10(image / signal)).toBeLessThan(-40);
  });
});
