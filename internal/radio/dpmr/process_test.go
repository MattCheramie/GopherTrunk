package dpmr

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestProcessDecodesCSBKAfterSync builds a dibit stream that
// contains a 24-dibit FS3 sync followed by a 40-dibit / 80-bit
// VoiceServiceAllocation CSBK, runs it through Process, and
// confirms a `trunking.Grant` event lands on the bus with the
// expected payload fields.
func TestProcessDecodesCSBKAfterSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "TestSys",
		FrequencyHz: 446_006_250,
	})

	csbk := CSBK{
		Type:     MsgVoiceServiceAllocation,
		Flags:    FlagGroupCall,
		SourceID: 0x123456,
		DestID:   0x654321,
		Extra:    7, // channel number
	}
	bits := CSBKBits(csbk)
	csbkDibits := framing.BitsToDibits(bits)
	if len(csbkDibits) != 40 {
		t.Fatalf("CSBK dibits = %d, want 40", len(csbkDibits))
	}

	// Build the dibit stream: 30-dibit padding (so the
	// SyncDetector primes before the FS3 starts) + 24-dibit FS3
	// sync + 40-dibit CSBK.
	stream := make([]uint8, 30)
	stream = append(stream, FS3Dibits()...)
	stream = append(stream, csbkDibits...)

	cc.Process(stream, 0)

	// Drain the bus and look for a Grant event.
	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Fatalf("Grant payload type = %T, want trunking.Grant", ev.Payload)
				}
				if g.System != "TestSys" {
					t.Errorf("Grant.System = %q, want TestSys", g.System)
				}
				if g.Protocol != "dpmr" {
					t.Errorf("Grant.Protocol = %q, want dpmr", g.Protocol)
				}
				if g.GroupID != 0x654321 {
					t.Errorf("Grant.GroupID = %d, want %d", g.GroupID, 0x654321)
				}
				if g.SourceID != 0x123456 {
					t.Errorf("Grant.SourceID = %d, want %d", g.SourceID, 0x123456)
				}
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("Process did not publish a KindGrant event for a valid CSBK")
			}
			return
		}
	}
}

// TestProcessIgnoresGarbageWithoutSync confirms a dibit stream that
// never contains the FS3 pattern produces no Ingest calls — the
// state machine stays silent.
func TestProcessIgnoresGarbageWithoutSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	garbage := make([]uint8, 1000)
	for i := range garbage {
		garbage[i] = uint8(i % 4) // 0,1,2,3,0,1,2,3,... — definitely no FS3
	}
	cc.Process(garbage, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event from garbage stream: %v", ev.Kind)
	default:
	}
}

// TestProcessHandlesSyncSpanningCalls confirms a CSBK whose sync
// arrives in one Process call and whose payload arrives in the
// next is still decoded correctly. This is the cross-call
// continuation the connector relies on.
func TestProcessHandlesSyncSpanningCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	csbk := CSBK{
		Type:     MsgVoiceServiceAllocation,
		Flags:    FlagGroupCall,
		SourceID: 0xAAAA,
		DestID:   0xBBBB,
	}
	bits := CSBKBits(csbk)
	csbkDibits := framing.BitsToDibits(bits)

	// Chunk 1 = 30-dibit padding + FS3 sync (the priming + sync
	// match closes at the end of this chunk). Chunk 2 = CSBK
	// dibits only.
	chunk1 := make([]uint8, 30)
	chunk1 = append(chunk1, FS3Dibits()...)
	cc.Process(chunk1, 0)
	cc.Process(csbkDibits, len(chunk1))

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("Process did not publish a KindGrant across the chunk boundary")
			}
			return
		}
	}
}

// TestSyncDetectorReset confirms the helper Reset clears history
// so a stale match doesn't fire after a stream re-sync.
func TestSyncDetectorReset(t *testing.T) {
	det := NewSyncDetector(FS3Dibits(), 0)
	// Prime with junk so the detector's primed counter is full.
	junk := make([]uint8, 100)
	det.Process(nil, junk, 0)
	det.Reset()
	if det.primed != 0 {
		t.Errorf("post-Reset primed = %d, want 0", det.primed)
	}
	if det.pos != 0 {
		t.Errorf("post-Reset pos = %d, want 0", det.pos)
	}
}
