import { describe, expect, it } from "vitest";
import { formatCommaList, parseCommaList } from "./csvList";

describe("comma list", () => {
  it("formats null/array", () => {
    expect(formatCommaList(null)).toBe("");
    expect(formatCommaList(["a", "b"])).toBe("a, b");
  });
  it("parses, trimming and dropping blanks", () => {
    expect(parseCommaList("a, b ,, c")).toEqual(["a", "b", "c"]);
    expect(parseCommaList("  ")).toBeNull();
    expect(parseCommaList("")).toBeNull();
  });
  it("round-trips", () => {
    expect(parseCommaList(formatCommaList(["x", "y"]))).toEqual(["x", "y"]);
  });
});
