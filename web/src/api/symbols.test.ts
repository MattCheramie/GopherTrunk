import { describe, it, expect } from "vitest";
import {
  autoProtoFor,
  demodModeToProto,
  symbolProtoForProtocol,
  symbolTargetForSystem,
} from "./symbols";
import type { SpectrumDevice } from "./spectrum";
import type { SystemHuntStatusDTO } from "./types";

describe("autoProtoFor", () => {
  // The regression: p25_modulation is only ever populated for P25 Phase 1
  // systems, so a TETRA / TETRA DMO / DMR rig sent nothing and "Auto" fell back
  // to a P25 C4FM receiver. Demodulating a π/4-DQPSK carrier that way yields a
  // meaningless-but-non-empty 4-level soft track, from which the panels computed
  // an MER around 9 dB and latched "symbol: poor" forever — next to a correct
  // "decode: clean" from the frame-error rate.
  it("uses the daemon's protocol-aware selector for non-P25 systems", () => {
    expect(autoProtoFor({ symbol_proto: "tetra" })).toBe("tetra");
    expect(autoProtoFor({ symbol_proto: "dmr" })).toBe("dmr");
  });

  it("does not fall back to C4FM when the daemon knows the protocol", () => {
    // A TETRA device carries no p25_modulation; the old path returned p25-c4fm.
    expect(autoProtoFor({ symbol_proto: "tetra", p25_modulation: undefined }))
      .toBe("tetra");
    expect(demodModeToProto(undefined)).toBe("p25-c4fm"); // the old behaviour
  });

  it("prefers symbol_proto over p25_modulation when both are present", () => {
    expect(autoProtoFor({ symbol_proto: "p25-cqpsk", p25_modulation: "c4fm" }))
      .toBe("p25-cqpsk");
  });

  it("falls back to p25_modulation for a daemon too old to send symbol_proto", () => {
    expect(autoProtoFor({ p25_modulation: "cqpsk" })).toBe("p25-cqpsk");
    expect(autoProtoFor({ p25_modulation: "c4fm" })).toBe("p25-c4fm");
  });

  it("defaults to C4FM when nothing is known", () => {
    expect(autoProtoFor(null)).toBe("p25-c4fm");
    expect(autoProtoFor(undefined)).toBe("p25-c4fm");
    expect(autoProtoFor({})).toBe("p25-c4fm");
  });
});

function dev(p: Partial<SpectrumDevice>): SpectrumDevice {
  return {
    serial: "SDR",
    driver: "rtlsdr",
    role: "control",
    center_hz: 851_000_000,
    sample_rate_hz: 2_400_000,
    ...p,
  };
}

function sys(p: Partial<SystemHuntStatusDTO>): SystemHuntStatusDTO {
  return { name: "Sys", protocol: "p25", state: "locked", ...p };
}

describe("symbolProtoForProtocol", () => {
  it("maps each protocol to its /diag/symbols receiver", () => {
    expect(symbolProtoForProtocol("p25")).toBe("p25-c4fm");
    expect(symbolProtoForProtocol("P25")).toBe("p25-c4fm"); // case-insensitive
    expect(symbolProtoForProtocol("p25-phase2")).toBe("p25-phase2");
    expect(symbolProtoForProtocol("tetra")).toBe("tetra");
    expect(symbolProtoForProtocol("tetra-dmo")).toBe("tetra");
    expect(symbolProtoForProtocol("dmr")).toBe("dmr");
    expect(symbolProtoForProtocol("dmr-tier2")).toBe("dmr");
    expect(symbolProtoForProtocol("nxdn")).toBe("nxdn");
  });

  it("returns '' for protocols with no symbol-scope receiver", () => {
    expect(symbolProtoForProtocol("mpt1327")).toBe("");
    expect(symbolProtoForProtocol("edacs")).toBe("");
    expect(symbolProtoForProtocol("")).toBe("");
  });
});

describe("symbolTargetForSystem", () => {
  it("maps a locked system to its in-band SDR with the offset from centre", () => {
    const devices = [
      dev({ serial: "CTRL", center_hz: 851_012_500, symbol_proto: "p25-c4fm" }),
    ];
    const target = symbolTargetForSystem(
      devices,
      sys({ locked_freq_hz: 851_012_500 }),
    );
    expect(target).toEqual({ serial: "CTRL", proto: "p25-c4fm", offset: 0 });
  });

  it("computes the offset of an in-band CC away from the SDR centre", () => {
    const devices = [
      dev({ serial: "WB", center_hz: 851_000_000, sample_rate_hz: 6_000_000, symbol_proto: "p25-c4fm" }),
    ];
    const target = symbolTargetForSystem(
      devices,
      sys({ locked_freq_hz: 851_500_000 }),
    );
    expect(target).toEqual({ serial: "WB", proto: "p25-c4fm", offset: 500_000 });
  });

  it("opens no stream for a system that isn't locked", () => {
    const devices = [dev({ symbol_proto: "p25-c4fm" })];
    expect(
      symbolTargetForSystem(devices, sys({ state: "hunting", locked_freq_hz: 851_000_000 })),
    ).toBeNull();
  });

  it("opens no stream when the CC falls outside every SDR passband", () => {
    // 851.0 MHz centre, 2.4 MS/s → passband ±1.2 MHz; a 900 MHz CC is out.
    const devices = [dev({ symbol_proto: "p25-c4fm" })];
    expect(
      symbolTargetForSystem(devices, sys({ locked_freq_hz: 900_000_000 })),
    ).toBeNull();
  });

  it("picks the nearest-centre SDR when several span the CC", () => {
    const devices = [
      dev({ serial: "A", center_hz: 851_000_000, sample_rate_hz: 6_000_000, symbol_proto: "p25-c4fm" }),
      dev({ serial: "B", center_hz: 852_000_000, sample_rate_hz: 6_000_000, symbol_proto: "p25-c4fm" }),
    ];
    // 851.9 MHz is in both passbands but closer to B's centre.
    const target = symbolTargetForSystem(
      devices,
      sys({ locked_freq_hz: 851_900_000 }),
    );
    expect(target?.serial).toBe("B");
  });

  it("falls back to the system's protocol when the device reports no symbol_proto", () => {
    const devices = [dev({ serial: "T", center_hz: 467_912_500, symbol_proto: undefined })];
    const target = symbolTargetForSystem(
      devices,
      sys({ protocol: "tetra", locked_freq_hz: 467_912_500 }),
    );
    expect(target).toEqual({ serial: "T", proto: "tetra", offset: 0 });
  });

  it("gives two wideband-hosted systems distinct per-SDR targets", () => {
    // The concurrent case: two systems on two wideband taps → two meters, each
    // on its own serial, rather than one shared reading.
    const devices = [
      dev({ serial: "WB-A", center_hz: 851_000_000, sample_rate_hz: 6_000_000, symbol_proto: "p25-c4fm" }),
      dev({ serial: "WB-B", center_hz: 460_000_000, sample_rate_hz: 6_000_000, symbol_proto: "tetra" }),
    ];
    const a = symbolTargetForSystem(devices, sys({ name: "P25", protocol: "p25", locked_freq_hz: 851_100_000 }));
    const b = symbolTargetForSystem(devices, sys({ name: "TETRA", protocol: "tetra", locked_freq_hz: 460_200_000 }));
    expect(a?.serial).toBe("WB-A");
    expect(b?.serial).toBe("WB-B");
    expect(a?.serial).not.toBe(b?.serial);
  });
});
