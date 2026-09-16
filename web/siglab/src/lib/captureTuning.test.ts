import { describe, expect, it } from "vitest";
import { parseCaptureTuning } from "./captureTuning";

describe("parseCaptureTuning", () => {
  it("sends a plain centre + bandwidth as a narrowband slice", () => {
    expect(parseCaptureTuning("442.8125", "880")).toEqual({
      ok: true,
      tuning: { center_hz: 442_812_500, bandwidth_hz: 880_000 },
    });
  });

  it("blank fields mean a full-band grab at the tuner centre", () => {
    expect(parseCaptureTuning("", "")).toEqual({ ok: true, tuning: {} });
    expect(parseCaptureTuning("  ", "\t")).toEqual({ ok: true, tuning: {} });
  });

  it("a bandwidth alone slices around the tuner centre", () => {
    expect(parseCaptureTuning("", "25")).toEqual({ ok: true, tuning: { bandwidth_hz: 25_000 } });
  });

  it("refuses a centre that is not a plain decimal instead of dropping it (15 Sep)", () => {
    // A decimal comma, a stray zero-width character from a paste, an exponent:
    // Number() turned each into NaN (or something else) and the old form
    // silently fell back to the tuner centre.
    for (const bad of ["442,8125", "442.8125​", "4.428125e2", "442.8125 MHz", "abc"]) {
      const r = parseCaptureTuning(bad, "880");
      expect(r.ok, bad).toBe(false);
      if (!r.ok) expect(r.error).toMatch(/Center MHz .* is not a number/);
    }
  });

  it("refuses a centre without a bandwidth instead of dropping it", () => {
    const r = parseCaptureTuning("442.8125", "");
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error).toMatch(/needs a Bandwidth kHz/);
  });

  it("refuses a non-positive or malformed bandwidth", () => {
    expect(parseCaptureTuning("", "0").ok).toBe(false);
    expect(parseCaptureTuning("", "-25").ok).toBe(false);
    expect(parseCaptureTuning("", "25k").ok).toBe(false);
  });
});
