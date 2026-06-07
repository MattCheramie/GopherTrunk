import { describe, expect, it } from "vitest";
import { hzToDisplay, parseFreqList } from "./fields";

describe("parseFreqList", () => {
  it("parses MHz and Hz, mixed separators, ignoring junk", () => {
    expect(parseFreqList("851.0375\n852.2625")).toEqual([851037500, 852262500]);
    expect(parseFreqList("851037500, 852262500")).toEqual([851037500, 852262500]);
    expect(parseFreqList("851.0375 MHz; 0; abc")).toEqual([851037500]);
  });
  it("returns empty for blank input", () => {
    expect(parseFreqList("")).toEqual([]);
    expect(parseFreqList("   \n  ")).toEqual([]);
  });
});

describe("hzToDisplay", () => {
  it("renders Hz as trimmed MHz", () => {
    expect(hzToDisplay(851037500)).toBe("851.0375 MHz");
    expect(hzToDisplay(852000000)).toBe("852 MHz");
    expect(hzToDisplay(0)).toBe("0");
  });
  it("round-trips through parseFreqList", () => {
    const hz = [851037500, 854012500];
    const text = hz.map(hzToDisplay).join("\n");
    expect(parseFreqList(text)).toEqual(hz);
  });
});
