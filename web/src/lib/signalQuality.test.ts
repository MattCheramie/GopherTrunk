import { describe, it, expect } from "vitest";
import { computeQuality, qualityVerdict } from "./signalQuality";

describe("computeQuality", () => {
  it("reports an even 4-level distribution as balanced", () => {
    const dibits = [0, 1, 2, 3, 0, 1, 2, 3];
    const q = computeQuality(dibits, []);
    expect(q.total).toBe(8);
    expect(q.balanceDev).toBeCloseTo(0, 5); // each level exactly 25%
    expect(q.snrDb).toBeNull(); // no soft track
  });

  it("flags a collapsed slicer as far off balance", () => {
    const dibits = new Array(100).fill(0); // all in one bin
    const q = computeQuality(dibits, []);
    expect(q.balanceDev).toBeCloseTo(75, 5); // 100% vs 25% ideal
  });

  it("computes an SNR from an aligned soft track", () => {
    // Four tight, well-separated levels → high SNR.
    const dibits: number[] = [];
    const soft: number[] = [];
    const means = [-3, -1, 1, 3];
    for (let i = 0; i < 400; i++) {
      const d = i % 4;
      dibits.push(d);
      // Small intra-level jitter (not parity-locked to d, so each level has a
      // real, non-zero spread → finite noise).
      soft.push(means[d] + ((i % 3) - 1) * 0.02);
    }
    const q = computeQuality(dibits, soft);
    expect(q.snrDb).not.toBeNull();
    expect(q.snrDb!).toBeGreaterThan(18);
  });

  it("ignores a misaligned soft track (no SNR)", () => {
    const q = computeQuality([0, 1, 2, 3], [0.1, 0.2]); // lengths differ
    expect(q.snrDb).toBeNull();
  });
});

describe("qualityVerdict", () => {
  const base = { pct: [], total: 1000, balanceDev: 0, snrDb: null, levels: null };

  it("is unknown until enough symbols accumulate", () => {
    expect(qualityVerdict(null)).toBe("unknown");
    expect(qualityVerdict({ ...base, total: 50 })).toBe("unknown");
  });

  it("buckets by SNR when a soft track is present", () => {
    expect(qualityVerdict({ ...base, snrDb: 20 })).toBe("clean");
    expect(qualityVerdict({ ...base, snrDb: 12 })).toBe("marginal");
    expect(qualityVerdict({ ...base, snrDb: 5 })).toBe("poor");
  });

  it("falls back to level balance when there is no SNR (e.g. CQPSK)", () => {
    expect(qualityVerdict({ ...base, balanceDev: 3 })).toBe("clean");
    expect(qualityVerdict({ ...base, balanceDev: 9 })).toBe("marginal");
    expect(qualityVerdict({ ...base, balanceDev: 30 })).toBe("poor");
  });
});
