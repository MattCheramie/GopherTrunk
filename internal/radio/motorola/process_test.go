package motorola

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestProcessDecodesGroupVoiceGrantAfterSync builds a bit stream
// containing 30 bits of padding (so the SyncDetector primes), the
// 24-bit outbound sync, and a 32-bit OSW carrying a Group Voice
// Channel Grant. Process must publish a KindGrant on the bus.
func TestProcessDecodesGroupVoiceGrantAfterSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "Sys",
		FrequencyHz: 851_012_500,
	})

	// Command = (opcode << 4) | LCN. Group Voice Channel Grant on
	// LCN 5.
	cmd := (uint16(OpGroupVoiceChannelGrant) << 4) | 0x5
	osw := OSW{Address: 0xCAFE, Command: cmd}
	oswBits := OSWBits(osw)

	stream := make([]byte, 30)
	stream = append(stream, OutboundSyncBits()...)
	stream = append(stream, oswBits...)

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
				if g.Protocol != "motorola" {
					t.Errorf("Grant.Protocol = %q, want motorola", g.Protocol)
				}
				if g.GroupID != 0xCAFE {
					t.Errorf("Grant.GroupID = %#x, want 0xCAFE", g.GroupID)
				}
				sawGrant = true
			}
		default:
			if !sawGrant {
				t.Errorf("Process did not publish a KindGrant for a valid OSW")
			}
			return
		}
	}
}

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

func TestProcessHandlesSyncSpanningCalls(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})

	cmd := (uint16(OpGroupVoiceChannelGrant) << 4) | 0x3
	osw := OSW{Address: 0xBEEF, Command: cmd}
	oswBits := OSWBits(osw)

	chunk1 := make([]byte, 30)
	chunk1 = append(chunk1, OutboundSyncBits()...)
	cc.Process(chunk1, 0)
	cc.Process(oswBits, len(chunk1))

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
