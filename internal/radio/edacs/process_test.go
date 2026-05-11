package edacs

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestProcessDecodesGroupVoiceGrantAfterSync builds a bit stream
// containing 30 bits of padding (so the SyncDetector primes before
// the sync starts), the 24-bit outbound sync, and a 40-bit
// GroupVoiceGrant CCW. Process must publish a KindGrant on the
// bus with the expected payload fields.
func TestProcessDecodesGroupVoiceGrantAfterSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 866_000_000,
	})

	ccw := CCW{
		Command: CmdGroupVoiceGrant,
		Status:  0,
		Address: 0xCAFE,
		LCN:     5,
		Aux:     0x123,
	}
	ccwBits := CCWBits(ccw)

	// 30 padding bits + 24 sync bits + 40 CCW bits.
	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, ccwBits...)

	cc.Process(stream, 0)

	var sawGrant bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Fatalf("Grant payload type = %T, want trunking.Grant", ev.Payload)
				}
				if g.System != "Sys" {
					t.Errorf("Grant.System = %q, want Sys", g.System)
				}
				if g.Protocol != "edacs" {
					t.Errorf("Grant.Protocol = %q, want edacs", g.Protocol)
				}
				if g.GroupID != 0xCAFE {
					t.Errorf("Grant.GroupID = %#x, want 0xCAFE", g.GroupID)
				}
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("Process did not publish a KindGrant for a valid CCW")
			}
			return
		}
	}
}

// TestProcessIgnoresGarbageWithoutSync confirms a stream that
// never contains the 24-bit sync produces no events.
func TestProcessIgnoresGarbageWithoutSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	garbage := make([]byte, 1000)
	for i := range garbage {
		garbage[i] = byte(i & 1)
	}
	cc.Process(garbage, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event from garbage stream: %v", ev.Kind)
	default:
	}
}

// TestProcessHandlesSyncSpanningCalls confirms a CCW whose sync
// arrives in one Process call and whose payload arrives in the
// next still decodes correctly.
func TestProcessHandlesSyncSpanningCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	ccw := CCW{
		Command: CmdGroupVoiceGrant,
		Address: 0xBEEF,
		LCN:     3,
	}
	ccwBits := CCWBits(ccw)

	chunk1 := make([]byte, 30)
	chunk1 = append(chunk1, OutboundSyncBits()...)
	cc.Process(chunk1, 0)
	cc.Process(ccwBits, len(chunk1))

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

func TestSyncDetectorReset(t *testing.T) {
	det := NewSyncDetector(OutboundSyncBits(), 0)
	junk := make([]byte, 100)
	det.Process(nil, junk, 0)
	det.Reset()
	if det.primed != 0 {
		t.Errorf("post-Reset primed = %d, want 0", det.primed)
	}
	if det.pos != 0 {
		t.Errorf("post-Reset pos = %d, want 0", det.pos)
	}
}
