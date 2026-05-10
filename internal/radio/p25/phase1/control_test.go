package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// buildLockedStream constructs a synthetic dibit window that places the
// FSW at the given offset, followed by a properly BCH-encoded NID for
// the supplied NAC and DUID, and a valid trellis-encoded + interleaved
// TSBK channel block (single Last-Block TSBK with the supplied opcode).
// Tail dibits are zero-padded.
func buildLockedStream(offset int, nac uint16, duid DUID, op Opcode) []uint8 {
	out := make([]uint8, offset+24+32+98+16)
	copy(out[offset:], FrameSyncWord[:])
	bits := EncodeNIDBits(nac, duid)
	for i := 0; i < 32; i++ {
		out[offset+24+i] = (bits[2*i] << 1) | bits[2*i+1]
	}
	tsbk := TSBK{LB: true, Opcode: op}
	channel := EncodeTSBKChannel(AssembleTSBK(tsbk))
	copy(out[offset+24+32:], channel)
	return out
}

func TestControlChannelEmitsLockOnTSDU(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	stream := buildLockedStream(10, 0x293, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Errorf("kind = %s, want cc.locked", ev.Kind)
		}
		ls, ok := ev.Payload.(LockState)
		if !ok {
			t.Fatalf("payload type = %T, want LockState", ev.Payload)
		}
		if ls.NAC != 0x293 || ls.DUID != DUIDTrunkingSignaling {
			t.Errorf("payload = %+v", ls)
		}
	case <-time.After(time.Second):
		t.Fatal("no event published")
	}
}

func TestControlChannelIgnoresNonTSDU(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	stream := buildLockedStream(10, 0x123, DUIDLogicalLink1, OpRFSSStatusBroadcast)
	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("unexpected event: %s", ev.Kind)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestControlChannelMarkLost(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := NewControlChannel(bus, nil, 851_000_000)
	cc.Process(buildLockedStream(10, 0x456, DUIDTrunkingSignaling, OpRFSSStatusBroadcast), 0)
	<-sub.C // CCLocked

	cc.MarkLost()
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLost {
			t.Errorf("kind = %s, want cc.lost", ev.Kind)
		}
	case <-time.After(time.Second):
		t.Fatal("no CCLost event")
	}
}

func TestControlChannelPublishesDecodeErrorOnUncorrectableNID(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Stream with FSW followed by 32 dibits of garbage (not a valid NID).
	stream := make([]uint8, 10+24+32+98)
	copy(stream[10:], FrameSyncWord[:])
	for i := 0; i < 32; i++ {
		stream[10+24+i] = uint8(i*7) & 0x3
	}

	cc := NewControlChannel(bus, nil, 851_000_000)
	cc.Process(stream, 0)

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindDecodeError {
				continue
			}
			de, ok := ev.Payload.(events.DecodeError)
			if !ok {
				t.Fatalf("payload type = %T, want DecodeError", ev.Payload)
			}
			if de.Protocol != "p25" || de.Stage != "nid-bch" {
				t.Errorf("DecodeError = %+v", de)
			}
			return
		case <-deadline:
			t.Fatal("no decode-error event published")
		}
	}
}

func TestControlChannelPublishesDecodeErrorOnCorruptTSBK(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Valid FSW + valid NID + corrupted TSBK channel block.
	stream := buildLockedStream(10, 0x111, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
	tsbkStart := 10 + 24 + 32
	// Flip every dibit in the TSBK block — well beyond the Viterbi
	// correction radius, so the CRC trailer will fail.
	for i := tsbkStart; i < tsbkStart+98; i++ {
		stream[i] = (^stream[i]) & 0x3
	}

	cc := NewControlChannel(bus, nil, 851_000_000)
	cc.Process(stream, 0)

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindDecodeError {
				continue
			}
			de, ok := ev.Payload.(events.DecodeError)
			if !ok {
				t.Fatalf("payload type = %T", ev.Payload)
			}
			if de.Protocol != "p25" {
				t.Errorf("protocol = %s, want p25", de.Protocol)
			}
			if de.Stage != "tsbk-crc" && de.Stage != "tsbk-trellis" {
				t.Errorf("stage = %s, want tsbk-crc or tsbk-trellis", de.Stage)
			}
			return
		case <-deadline:
			t.Fatal("no decode-error event published for corrupt TSBK")
		}
	}
}
