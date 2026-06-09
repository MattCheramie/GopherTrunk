import { describe, it, expect } from "vitest";
import { defaultSymbolDevice, type SpectrumDevice } from "./spectrum";

// Issue #402: the symbol-domain panels (Eye Diagram, Symbol Scope,
// Tuning, Histogram) must default to the control-role SDR so a panel
// opened during active control-channel decoding lands on the dongle that
// actually carries a decodable C4FM channel — not the enumeration's first
// entry, which on a multi-SDR rig is often an idle voice dongle.
function dev(serial: string, role: string): SpectrumDevice {
  return {
    serial,
    driver: "rtlsdr",
    role,
    center_hz: 420_012_500,
    sample_rate_hz: 2_400_000,
  };
}

describe("defaultSymbolDevice", () => {
  it("returns null for an empty list", () => {
    expect(defaultSymbolDevice([])).toBeNull();
  });

  it("prefers the control-role device over earlier entries", () => {
    const list = [dev("voice-1", "voice"), dev("ctrl-1", "control")];
    expect(defaultSymbolDevice(list)?.serial).toBe("ctrl-1");
  });

  it("falls back to the first device when no control role is present", () => {
    const list = [dev("voice-1", "voice"), dev("voice-2", "voice")];
    expect(defaultSymbolDevice(list)?.serial).toBe("voice-1");
  });

  it("returns the sole device regardless of role", () => {
    const list = [dev("only", "voice")];
    expect(defaultSymbolDevice(list)?.serial).toBe("only");
  });
});
