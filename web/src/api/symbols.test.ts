import { describe, expect, it } from "vitest";
import { autoProtoFor, demodModeToProto } from "./symbols";

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
