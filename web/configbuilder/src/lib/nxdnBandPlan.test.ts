import { describe, expect, it } from "vitest";
import { nxdnBandPlanForMode, nxdnBandPlanMode } from "./nxdnBandPlan";

describe("nxdnBandPlanMode", () => {
  it("classifies null / linear / table", () => {
    expect(nxdnBandPlanMode(null)).toBe("none");
    expect(nxdnBandPlanMode(undefined)).toBe("none");
    expect(nxdnBandPlanMode({ Linear: { BaseHz: 1, SpacingHz: 1, Offset: 1 }, Table: null })).toBe(
      "linear",
    );
    expect(nxdnBandPlanMode({ Linear: null, Table: [{ Channel: 1, FreqHz: 1 }] })).toBe("table");
    expect(nxdnBandPlanMode({ Linear: null, Table: [] })).toBe("table");
  });
});

describe("nxdnBandPlanForMode", () => {
  it("none clears the plan", () => {
    expect(nxdnBandPlanForMode("none", { Linear: null, Table: [] })).toBeNull();
  });

  it("linear seeds a default and clears table", () => {
    expect(nxdnBandPlanForMode("linear", null)).toEqual({
      Linear: { BaseHz: 0, SpacingHz: 0, Offset: 1 },
      Table: null,
    });
  });

  it("table seeds an empty list and clears linear", () => {
    expect(nxdnBandPlanForMode("table", null)).toEqual({ Linear: null, Table: [] });
  });

  it("preserves the previous branch values when re-selecting it", () => {
    const prev = { Linear: null, Table: [{ Channel: 9, FreqHz: 453_325_000 }] };
    expect(nxdnBandPlanForMode("table", prev)).toEqual(prev);
  });
});
