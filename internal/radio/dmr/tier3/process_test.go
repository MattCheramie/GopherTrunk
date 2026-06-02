package tier3

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildAlohaBurst builds a 132-dibit DMR burst carrying an Aloha
// CSBK at the requested ColorCode + SystemID. The payload goes
// through full BPTC(196,96) encoding so DecodeBPTC + ParseCSBK
// (inside IngestBurst) recover the original.
func buildAlohaBurst(t *testing.T, cc uint8, systemID uint16) []uint8 {
	t.Helper()
	csbk := CSBK{Opcode: OpAloha, LB: true}
	// Aloha payload layout (see payloads.go ParseAloha):
	//   bytes [0]=ServiceOptions, [1]=AlohaService,
	//   bytes [2..3]=SystemID, ...
	csbk.Payload[2] = byte(systemID >> 8)
	csbk.Payload[3] = byte(systemID & 0xFF)
	csbkBytes := AssembleCSBK(csbk)

	infoBits := framing.UnpackBitsMSB(csbkBytes, 96)
	channelBits := framing.EncodeBPTC196_96(infoBits)

	// Pack the 196 channel bits into 98 dibits.
	dibits := framing.BitsToDibits(channelBits)
	if len(dibits) != 98 {
		t.Fatalf("BPTC produced %d dibits, want 98", len(dibits))
	}
	// Burst layout: payload first half (49 dibits) + slot type
	// before (5 dibits) + sync (24 dibits) + slot type after
	// (5 dibits) + payload second half (49 dibits) = 132 dibits.
	slotBits := dmr.AssembleSlotType(dmr.SlotType{
		ColorCode: cc,
		DataType:  dmr.DTCSBK,
	})
	slotDibits := framing.BitsToDibits(slotBits)
	if len(slotDibits) != 10 {
		t.Fatalf("slot-type dibits = %d, want 10", len(slotDibits))
	}

	out := make([]uint8, 0, dmr.BurstDibits)
	out = append(out, dibits[:dmr.HalfPayloadDibits]...)
	out = append(out, slotDibits[:dmr.SlotTypeDibits]...)
	out = append(out, dmr.BSData.Dibits[:]...)
	out = append(out, slotDibits[dmr.SlotTypeDibits:]...)
	out = append(out, dibits[dmr.HalfPayloadDibits:]...)
	if len(out) != dmr.BurstDibits {
		t.Fatalf("built burst length = %d, want %d", len(out), dmr.BurstDibits)
	}
	return out
}

// TestProcessLocksOnAlohaBurst: feed Process a stream of (padding,
// Aloha CSBK burst, trailing padding), confirm a KindCCLocked
// event lands on the bus with the ColorCode + SystemID we encoded.
func TestProcessLocksOnAlohaBurst(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 460_000_000,
	})

	burst := buildAlohaBurst(t, 0xA, 0x1234)

	// 200 dibits of padding for sync-detector priming + adapter
	// lookback + Aloha burst + 64 trailing dibits.
	stream := make([]uint8, 200)
	stream = append(stream, burst...)
	stream = append(stream, make([]uint8, 64)...)

	cc.Process(stream, 0)

	var sawLock bool
	var ls LockState
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				ls, _ = ev.Payload.(LockState)
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("Process did not publish a KindCCLocked")
				return
			}
			if ls.ColorCode != 0xA {
				t.Errorf("LockState.ColorCode = %d, want 10", ls.ColorCode)
			}
			if ls.SystemID != 0x1234 {
				t.Errorf("LockState.SystemID = %#x, want 0x1234", ls.SystemID)
			}
			return
		}
	}
}

// TestProcessLocksOnPolarityFlippedBurst is the issue #264 regression:
// an RTL-SDR Blog V4 / R828D presenting conjugated (spectrum-inverted)
// IQ flips the FM-discriminator sign, so every dibit arrives as
// (dibit + 2) mod 4. The adapter must detect the rotation-2 sync and
// de-rotate the burst so the same ColorCode + SystemID are recovered
// as from the canonical burst — i.e. the color code is stable, not
// "changing constantly".
func TestProcessLocksOnPolarityFlippedBurst(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 460_000_000,
	})

	burst := buildAlohaBurst(t, 0xA, 0x1234)

	stream := make([]uint8, 200)
	stream = append(stream, burst...)
	stream = append(stream, make([]uint8, 64)...)
	// Simulate I/Q-reversed / spectrum-inverted reception: every dibit
	// rotated by +2 mod 4 (the discriminator-polarity-flip rotation).
	for i := range stream {
		stream[i] = (stream[i] + 2) & 3
	}

	cc.Process(stream, 0)

	var sawLock bool
	var ls LockState
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				ls, _ = ev.Payload.(LockState)
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("Process did not lock on a polarity-flipped burst")
				return
			}
			if ls.ColorCode != 0xA {
				t.Errorf("LockState.ColorCode = %d, want 10", ls.ColorCode)
			}
			if ls.SystemID != 0x1234 {
				t.Errorf("LockState.SystemID = %#x, want 0x1234", ls.SystemID)
			}
			return
		}
	}
}

// TestProcessHandlesBurstSpanningCalls: a burst whose dibits
// arrive across two Process calls still drives IngestBurst.
func TestProcessHandlesBurstSpanningCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	burst := buildAlohaBurst(t, 0x5, 0xCAFE)
	stream := make([]uint8, 200)
	stream = append(stream, burst...)
	stream = append(stream, make([]uint8, 64)...)

	// Split right in the middle of the burst.
	mid := len(stream) / 2
	cc.Process(stream[:mid], 0)
	cc.Process(stream[mid:], mid)

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("Process did not publish a KindCCLocked across the chunk boundary")
			}
			return
		}
	}
}

// TestProcessIgnoresGarbage confirms a dibit stream without any
// DMR sync pattern produces no events.
func TestProcessIgnoresGarbage(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	garbage := make([]uint8, 2000)
	for i := range garbage {
		garbage[i] = uint8(i % 4)
	}
	cc.Process(garbage, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event from garbage stream: %v", ev.Kind)
	default:
	}
}
