import { describe, it, expect } from "vitest";
import {
  defaultSymbolDevice,
  parentSerial,
  type SpectrumDevice,
} from "./spectrum";

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

// The Plots panels follow the active call by matching its device_serial to
// the selected SDR. On a wideband SDR the call runs on a virtual voice tap
// ("wb:<parent>:tap-N") while the selector lists the parent, so the match
// only works once the tap is resolved back to its parent — otherwise the view
// is stuck on the control channel forever.
describe("parentSerial", () => {
  it("resolves a wideband voice-tap serial to its parent", () => {
    expect(parentSerial("wb:soapy-10_110_162_1_23313-00:tap-0")).toBe(
      "soapy-10_110_162_1_23313-00",
    );
    expect(parentSerial("wb:soapy-10_110_162_1_23313-00:tap-3")).toBe(
      "soapy-10_110_162_1_23313-00",
    );
  });

  it("passes a direct (non-tap) dongle serial through unchanged", () => {
    expect(parentSerial("rtlsdr-00000001")).toBe("rtlsdr-00000001");
    expect(parentSerial("soapy-10_110_162_1_23313-00")).toBe(
      "soapy-10_110_162_1_23313-00",
    );
  });
});
