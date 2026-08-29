package motorola

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// realAirStream renders OSWs back-to-back in the on-air frame format
// plus the trailing sync the bracket framer needs, preceded by an
// alternating warmup that can never frame (10101010… ≠ 10101100).
func realAirStream(warmup int, osws ...OSW) []byte {
	out := make([]byte, 0, warmup+len(osws)*FrameBits+SyncBits)
	for i := 0; i < warmup; i++ {
		out = append(out, byte(i&1))
	}
	for _, o := range osws {
		out = append(out, EncodeOSWFrame(o)...)
	}
	return append(out, OutboundSyncBits()...)
}

func drainEvents(sub *events.Subscription) (locked []LockState, grants []trunking.Grant) {
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				if ls, ok := ev.Payload.(LockState); ok {
					locked = append(locked, ls)
				}
			case events.KindGrant:
				if g, ok := ev.Payload.(trunking.Grant); ok {
					grants = append(grants, g)
				}
			}
		default:
			return locked, grants
		}
	}
}

// TestProcessDecodesRealAirFormat is the issue #1143 regression: a
// control-channel stream in the REAL SmartNet air format (OP25
// rx_smartnet framing) must lock and publish the grant. The
// pre-#1143 decoder (24-bit sync 0xA4D7AA + BCH(64,16,11), a framing
// no real system transmits) decodes NOTHING from this stream —
// verified failing-first against the old code.
func TestProcessDecodesRealAirFormat(t *testing.T) {
	bus := events.NewBus(256)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Real", FrequencyHz: 854_562_500})

	var seq []OSW
	for r := 0; r < 5; r++ {
		seq = append(seq,
			// Two-OSW system ID + CC broadcast (the lock signal).
			OSW{Address: 0x4567, Command: CmdFirstNormal},
			OSW{Address: 0x1F00, Command: 0x8E},
			// Two-OSW analog group voice grant: source RID 0x2E9A,
			// talkgroup 0xB010, channel 0x1A5 (861.5375 MHz).
			OSW{Address: 0x2E9A, Command: CmdFirstNormal},
			OSW{Address: 0xB010, Group: true, Command: 0x1A5},
			OSW{Address: 0x02F8, Command: CmdIdle},
		)
	}
	cc.Process(realAirStream(200, seq...), 0)

	locked, grants := drainEvents(sub)
	if len(locked) == 0 {
		t.Fatal("no cc.locked from a real-air-format stream")
	}
	if locked[0].SystemID != 0x4567 {
		t.Errorf("SystemID = %#x, want 0x4567", locked[0].SystemID)
	}
	if len(grants) == 0 {
		t.Fatal("no grant from a real-air-format stream")
	}
	g := grants[0]
	if g.GroupID != 0xB010 || g.SourceID != 0x2E9A {
		t.Errorf("grant tg/src = %#x/%#x, want 0xB010/0x2E9A", g.GroupID, g.SourceID)
	}
	if g.FrequencyHz != 861_537_500 {
		t.Errorf("grant freq = %d, want 861537500 (channel 0x1A5, 800_standard)", g.FrequencyHz)
	}
}

// TestProcessSurvivesChunkBoundaries feeds the same stream one bit at
// a time; framing state must carry across Process calls.
func TestProcessSurvivesChunkBoundaries(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "Chunked", FrequencyHz: 854_562_500})

	stream := realAirStream(31,
		OSW{Address: 0x4567, Command: CmdFirstNormal},
		OSW{Address: 0x1F00, Command: 0x8E},
		OSW{Address: 0x02F8, Command: CmdIdle},
	)
	idx := 0
	for _, b := range stream {
		cc.Process([]byte{b}, idx)
		idx++
	}
	locked, _ := drainEvents(sub)
	if len(locked) == 0 || locked[0].SystemID != 0x4567 {
		t.Fatalf("bit-at-a-time stream did not lock: %+v", locked)
	}
}

// TestProcessCorrectsWireBitError flips one info-position wire bit
// inside a frame; the ECC must repair it end-to-end through the
// framer.
func TestProcessCorrectsWireBitError(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "ECC", FrequencyHz: 854_562_500})

	stream := realAirStream(20,
		OSW{Address: 0x4567, Command: CmdFirstNormal},
		OSW{Address: 0x1F00, Command: 0x8E},
		OSW{Address: 0x02F8, Command: CmdIdle},
	)
	// First frame's payload starts after warmup+sync; corrupt an
	// info-position bit (wire offset 40 = seq position 10).
	stream[20+SyncBits+40] ^= 1
	cc.Process(stream, 0)
	locked, _ := drainEvents(sub)
	if len(locked) == 0 || locked[0].SystemID != 0x4567 {
		t.Fatalf("single wire bit error was not corrected: %+v", locked)
	}
}

// TestProcessRequiresBracketSync: a frame whose trailing sync is
// corrupted must be dropped (framing lost), not decoded — the
// double-sync bracket is what makes the 8-bit sync safe.
func TestProcessRequiresBracketSync(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := New(Options{Bus: bus, SystemName: "Bracket", FrequencyHz: 854_562_500})

	stream := realAirStream(20,
		OSW{Address: 0x4567, Command: CmdFirstNormal},
		OSW{Address: 0x1F00, Command: 0x8E},
	)
	// Corrupt the SECOND frame's sync (the first frame's bracket).
	stream[20+FrameBits] ^= 1
	cc.Process(stream, 0)
	locked, _ := drainEvents(sub)
	if len(locked) != 0 {
		t.Fatalf("frame without its bracket sync still decoded: %+v", locked)
	}
}
