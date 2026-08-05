import { describe, it, expect } from "vitest";
import {
  envelopeTarget,
  nextEnvelopeGain,
  rampStepForFrames,
} from "./fadeEnvelope";

describe("envelopeTarget", () => {
  it("is 0 while not draining (primed-silent / post-underrun)", () => {
    expect(envelopeTarget(false, 10_000, 240)).toBe(0);
  });

  it("is 1 during healthy playback (available deep enough)", () => {
    expect(envelopeTarget(true, 10_000, 240)).toBe(1);
  });

  it("drops to 0 once within the release window of empty (tail fade)", () => {
    expect(envelopeTarget(true, 240, 240)).toBe(0);
    expect(envelopeTarget(true, 100, 240)).toBe(0);
    expect(envelopeTarget(true, 241, 240)).toBe(1);
  });
});

describe("nextEnvelopeGain", () => {
  it("steps up toward the target without overshooting", () => {
    expect(nextEnvelopeGain(0, 1, 0.25)).toBeCloseTo(0.25);
    expect(nextEnvelopeGain(0.9, 1, 0.25)).toBe(1); // clamps at target
  });

  it("steps down toward the target without undershooting", () => {
    expect(nextEnvelopeGain(1, 0, 0.25)).toBeCloseTo(0.75);
    expect(nextEnvelopeGain(0.1, 0, 0.25)).toBe(0); // clamps at target
  });

  it("holds when already at target", () => {
    expect(nextEnvelopeGain(1, 1, 0.25)).toBe(1);
    expect(nextEnvelopeGain(0, 0, 0.25)).toBe(0);
  });
});

describe("rampStepForFrames", () => {
  it("converts a frame count to a per-frame step", () => {
    expect(rampStepForFrames(4)).toBe(0.25);
  });
  it("is instant for a non-positive length", () => {
    expect(rampStepForFrames(0)).toBe(1);
    expect(rampStepForFrames(-5)).toBe(1);
  });
});

// Integration: replay the worklet's per-sample envelope math (which mirrors
// these helpers) and assert the audible property — a boundary that begins and
// ends at exactly zero, with a monotone ramp, i.e. no click.
describe("envelope over a call boundary", () => {
  const step = rampStepForFrames(8); // 8-frame ramp for a readable trace

  function run(
    frames: { draining: boolean; available: number }[],
    release: number,
  ): number[] {
    let env = 0;
    const gains: number[] = [];
    for (const f of frames) {
      const target = envelopeTarget(f.draining, f.available, release);
      env = nextEnvelopeGain(env, target, step);
      gains.push(env);
    }
    return gains;
  }

  it("fades IN from zero on prime (no startup click)", () => {
    // Draining with plenty buffered: env climbs from 0 up toward 1.
    const gains = run(
      Array.from({ length: 8 }, () => ({ draining: true, available: 1000 })),
      100,
    );
    expect(gains[0]).toBeGreaterThan(0);
    expect(gains[0]).toBeLessThan(1);
    // Monotonically non-decreasing and reaches unity.
    for (let i = 1; i < gains.length; i++) {
      expect(gains[i]).toBeGreaterThanOrEqual(gains[i - 1]);
    }
    expect(gains[gains.length - 1]).toBe(1);
  });

  it("fades OUT to zero as the ring empties (no PTT-out click)", () => {
    // Start at unity, then the tail drains inside the release window.
    const frames = [
      { draining: true, available: 1000 }, // healthy
      ...Array.from({ length: 8 }, () => ({ draining: true, available: 50 })),
      { draining: false, available: 0 }, // underran
    ];
    const gains = run(frames, 100);
    // Ends fully faded out.
    expect(gains[gains.length - 1]).toBe(0);
    // Once the tail window is entered, the envelope is monotonically
    // non-increasing down to zero.
    const tail = gains.slice(1);
    for (let i = 1; i < tail.length; i++) {
      expect(tail[i]).toBeLessThanOrEqual(tail[i - 1]);
    }
  });
});
