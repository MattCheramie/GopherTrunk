package nxdn

import (
	"encoding/binary"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestProcessDecodesSiteInfoFrameAfterSync builds a dibit stream
// that contains:
//
//   - 30 padding dibits (so SyncDetector primes before FSW starts)
//   - 8 outbound FSW dibits
//   - 8 LICH wire dibits (encoding RFCh = Control, parity OK)
//   - 32 SACCH dibits (junk; SACCH not consumed by adapter)
//   - 44 dibits whose raw bits decode to a valid RCCHSITEINFO
//     CACMessage (no FEC encoding — this exercises the adapter
//     wiring rather than the FEC path)
//
// Runs the stream through Process and confirms a KindCCLocked
// event lands on the bus.
//
// Production NXDN traffic carries the 88 CAC information bits
// across the 288-wire-bit Info field via a K=5 ½-rate Viterbi +
// interleaver + puncture chain that this adapter doesn't yet
// reverse. Until that FEC layer ships, the adapter sync-locks
// on the FSW but typically fails the CAC CRC on real signals.
func TestProcessDecodesSiteInfoFrameAfterSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, slog.Default(), 851_062_500, Rate9600)

	// LICH: RFCh = Control, FCT / Option / Direction zero, parity
	// computed by AssembleLICH. Wire-encode it (each info bit
	// doubled) into 16 bits = 8 dibits.
	lichInfo := AssembleLICH(LICH{RFCh: RFChControl})
	lichWire := EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)
	if len(lichDibits) != 8 {
		t.Fatalf("LICH wire dibits = %d, want 8", len(lichDibits))
	}

	// CAC: SITEINFO with non-trivial site / system IDs so the
	// state machine has something to publish + assert on.
	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA) // LocationID
	binary.BigEndian.PutUint16(payload[2:4], 0x1234) // SiteID
	binary.BigEndian.PutUint16(payload[4:6], 0x5678) // SystemID
	cacBytes := AssembleCAC(CACMessage{
		Type:    RCCHSITEINFO,
		Payload: payload,
	})
	// AssembleCAC produces 11 bytes (88 bits + CRC). Unpack MSB-
	// first into 88 bits, then to 44 dibits.
	cacBits := framing.UnpackBitsMSB(cacBytes, 88)
	cacDibits := framing.BitsToDibits(cacBits)
	if len(cacDibits) != 44 {
		t.Fatalf("CAC dibits = %d, want 44", len(cacDibits))
	}

	// Assemble the dibit stream: padding + FSW + LICH + SACCH
	// (junk) + CAC (raw).
	stream := make([]uint8, 30)
	stream = append(stream, FSWDibitsOutbound...)
	stream = append(stream, lichDibits...)
	stream = append(stream, make([]uint8, 32)...) // SACCH junk
	stream = append(stream, cacDibits...)

	cc.Process(stream, 0)

	var sawLock bool
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				ls, ok := ev.Payload.(LockState)
				if !ok {
					t.Fatalf("CCLocked payload type = %T, want LockState", ev.Payload)
				}
				if ls.FrequencyHz != 851_062_500 {
					t.Errorf("LockState.FrequencyHz = %d, want 851062500", ls.FrequencyHz)
				}
				if ls.SiteID != 0x1234 {
					t.Errorf("LockState.SiteID = %#x, want 0x1234", ls.SiteID)
				}
				if ls.SystemID != 0x5678 {
					t.Errorf("LockState.SystemID = %#x, want 0x5678", ls.SystemID)
				}
				sawLock = true
			}
		default:
			if !sawLock {
				t.Errorf("Process did not publish a KindCCLocked for a valid frame")
			}
			return
		}
	}
}

// TestProcessIgnoresGarbageWithoutSync confirms a dibit stream
// that never contains the outbound FSW produces no events.
func TestProcessIgnoresGarbageWithoutSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, slog.Default(), 0, Rate9600)

	garbage := make([]uint8, 1000)
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

// TestSyncDetectorReset confirms the helper Reset clears history.
func TestSyncDetectorReset(t *testing.T) {
	det := NewSyncDetector([][]uint8{FSWDibitsOutbound}, 0)
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
