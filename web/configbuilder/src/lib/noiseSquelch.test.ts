import { describe, expect, it } from "vitest";
import { isAnalogFM, noiseSquelchValue, parseNoiseSquelch } from "./noiseSquelch";

describe("noise squelch setting", () => {
  it("keeps unset apart from on and off", () => {
    expect(noiseSquelchValue(undefined)).toBe("");
    expect(noiseSquelchValue(null)).toBe("");
    expect(noiseSquelchValue(true)).toBe("on");
    expect(noiseSquelchValue(false)).toBe("off");
    expect(parseNoiseSquelch("")).toBeUndefined();
    expect(parseNoiseSquelch("on")).toBe(true);
    expect(parseNoiseSquelch("off")).toBe(false);
  });

  it("round-trips every value", () => {
    for (const v of [undefined, true, false]) {
      expect(parseNoiseSquelch(noiseSquelchValue(v))).toBe(v);
    }
  });

  it("belongs to fm and nfm channels only", () => {
    expect(isAnalogFM("")).toBe(true);
    expect(isAnalogFM(undefined)).toBe(true);
    expect(isAnalogFM("fm")).toBe(true);
    expect(isAnalogFM("nfm")).toBe(true);
    for (const m of ["am", "p25", "dmr", "nxdn", "nxdn48", "dstar", "ysf"]) {
      expect(isAnalogFM(m)).toBe(false);
    }
  });
});
