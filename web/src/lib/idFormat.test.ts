import { describe, expect, it } from "vitest";
import { formatIdNumber } from "./idFormat";

describe("formatIdNumber", () => {
  it("renders hex with decimal in parentheses by default", () => {
    expect(formatIdNumber(706, "hex")).toBe("0x2C2 (706)");
    expect(formatIdNumber(781824, "hex")).toBe("0xBEE00 (781824)");
  });

  it("renders plain decimal in dec mode", () => {
    expect(formatIdNumber(706, "dec")).toBe("706");
  });

  it("returns null for null/undefined so callers can show a hint", () => {
    expect(formatIdNumber(null, "hex")).toBeNull();
    expect(formatIdNumber(undefined, "dec")).toBeNull();
  });

  it("renders zero (not treated as missing)", () => {
    expect(formatIdNumber(0, "hex")).toBe("0x0 (0)");
    expect(formatIdNumber(0, "dec")).toBe("0");
  });
});
