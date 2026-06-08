import { describe, expect, it } from "vitest";
import { dmrBandPlanForMode, dmrBandPlanMode } from "./dmrBandPlan";

describe("dmrBandPlanMode", () => {
  it("classifies null / linear / table", () => {
    expect(dmrBandPlanMode(null)).toBe("none");
    expect(dmrBandPlanMode(undefined)).toBe("none");
    expect(dmrBandPlanMode({ Linear: { BaseHz: 1, SpacingHz: 1, Offset: 1 }, Table: null })).toBe(
      "linear",
    );
    expect(dmrBandPlanMode({ Linear: null, Table: [{ LCN: 1, FreqHz: 1 }] })).toBe("table");
    expect(dmrBandPlanMode({ Linear: null, Table: [] })).toBe("table");
  });
});

describe("dmrBandPlanForMode", () => {
  it("none clears the plan", () => {
    expect(dmrBandPlanForMode("none", { Linear: null, Table: [] })).toBeNull();
  });

  it("linear seeds a default and clears table", () => {
    const bp = dmrBandPlanForMode("linear", null);
    expect(bp).toEqual({ Linear: { BaseHz: 0, SpacingHz: 0, Offset: 1 }, Table: null });
  });

  it("table seeds an empty list and clears linear", () => {
    const bp = dmrBandPlanForMode("table", null);
    expect(bp).toEqual({ Linear: null, Table: [] });
  });

  it("preserves the previous branch values when re-selecting it", () => {
    const prev = { Linear: { BaseHz: 5, SpacingHz: 6, Offset: 1 }, Table: null };
    expect(dmrBandPlanForMode("linear", prev)).toEqual({
      Linear: { BaseHz: 5, SpacingHz: 6, Offset: 1 },
      Table: null,
    });
  });
});
