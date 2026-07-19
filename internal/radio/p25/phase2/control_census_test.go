package phase2

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestControlChannelMACCensus verifies the control-channel opcode census
// (issue #915) counts every MAC PDU by opcode — crucially including
// opcodes GT does NOT parse as a grant. The per-grant DEBUG line has
// survivorship bias (it only fires on opcodes already recognised as
// grants), so a RID-bearing grant arriving under a dropped/mis-mapped
// opcode is invisible there; the census is what surfaces it.
func TestControlChannelMACCensus(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	if got := cc.MACCensus(); len(got) != 0 {
		t.Fatalf("fresh MACCensus() = %v, want empty", got)
	}

	grant := []byte{0x00, 0x10, 0x07, 0xCA, 0xFE, 0x12, 0x34, 0x56}
	cc.Ingest(MACPDU{Opcode: OpGroupVoiceChannelGrant, Payload: grant})
	cc.Ingest(MACPDU{Opcode: OpGroupVoiceChannelGrant, Payload: grant})
	// 0x43 is not in GT's parsed-grant set: it must still be inventoried,
	// because "an opcode GT ignores" is exactly where a missing RID-bearing
	// grant would hide on a system whose source RIDs never populate.
	cc.Ingest(MACPDU{Opcode: Opcode(0x43), Payload: make([]byte, 8)})

	census := cc.MACCensus()
	if census[OpGroupVoiceChannelGrant] != 2 {
		t.Errorf("census[0x%02X] = %d, want 2", uint8(OpGroupVoiceChannelGrant), census[OpGroupVoiceChannelGrant])
	}
	if census[Opcode(0x43)] != 1 {
		t.Errorf("census[0x43] = %d, want 1 — an unparsed opcode must still be counted", census[Opcode(0x43)])
	}
	if len(census) != 2 {
		t.Errorf("census tracks %d opcodes, want 2: %v", len(census), census)
	}

	// The snapshot must be a copy — mutating it can't corrupt the channel's
	// running census.
	census[Opcode(0x43)] = 999
	if again := cc.MACCensus(); again[Opcode(0x43)] != 1 {
		t.Errorf("MACCensus() returned a live map, not a snapshot: %v", again)
	}
}
