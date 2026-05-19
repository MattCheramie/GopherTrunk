package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildLockedStream constructs a synthetic dibit window that places the
// FSW at the given offset, followed by a properly BCH-encoded NID for
// the supplied NAC and DUID, and a valid trellis-encoded + interleaved
// TSBK channel block (single Last-Block TSBK with the supplied opcode).
// Tail dibits are zero-padded.
func buildLockedStream(offset int, nac uint16, duid DUID, op Opcode) []uint8 {
	return buildLockedStreamWithTSBK(offset, nac, duid, TSBK{LB: true, Opcode: op})
}

// buildLockedStreamWithTSBK is the variant that takes a fully-formed
// TSBK so callers can carry a payload (used by the IdentifierUpdate +
// grant publication tests).
func buildLockedStreamWithTSBK(offset int, nac uint16, duid DUID, tsbk TSBK) []uint8 {
	out := make([]uint8, offset+24+32+98+16)
	copy(out[offset:], FrameSyncWord[:])
	bits := EncodeNIDBits(nac, duid)
	for i := 0; i < 32; i++ {
		out[offset+24+i] = (bits[2*i] << 1) | bits[2*i+1]
	}
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

func TestControlChannelAppliesIdentifierUpdateAndPublishesGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Build a stream with two TSBKs back-to-back: an IdentifierUpdate
	// for ChannelID 1 (12.5 kHz spacing, base 851 MHz) followed by a
	// GroupVoiceChannelGrant on (ID=1, Number=16) which should resolve
	// to 851.2 MHz on the bus.
	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdate}
	identTSBK.Payload = AssembleIdentifierUpdate(IdentifierUpdate{
		ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000,
	})

	grantPayload := [8]byte{
		0xC0,                  // service options: emergency + encrypted
		(1 << 4) | 0x00, 0x10, // channel = ID 1, number 0x010 (=16)
		0x12, 0x34, // group address 0x1234
		0xAB, 0xCD, 0xEF, // source ID 0xABCDEF
	}
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: grantPayload}

	stream1 := buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, identTSBK)
	stream2 := buildLockedStreamWithTSBK(0, 0x293, DUIDTrunkingSignaling, grantTSBK)

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	cc.Process(stream1, 0)
	cc.Process(stream2, len(stream1))

	// Drain events; assert exactly one Grant event lands and looks right.
	var got *trunking.Grant
	deadline := time.After(time.Second)
	for got == nil {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g := ev.Payload.(trunking.Grant)
				got = &g
			}
		case <-deadline:
			t.Fatal("no grant event published")
		}
	}
	if got.System != "TestSys" || got.Protocol != "p25" {
		t.Errorf("identity = %s/%s", got.System, got.Protocol)
	}
	if got.GroupID != 0x1234 || got.SourceID != 0xABCDEF {
		t.Errorf("group/source = %d/%d, want 4660/11259375", got.GroupID, got.SourceID)
	}
	if got.ChannelID != 1 || got.ChannelNum != 0x010 {
		t.Errorf("channel = %d.%d, want 1.16", got.ChannelID, got.ChannelNum)
	}
	if got.FrequencyHz != 851_200_000 {
		t.Errorf("freq = %d, want 851_200_000", got.FrequencyHz)
	}
	if !got.Encrypted || !got.Emergency {
		t.Errorf("flags = enc=%v emer=%v, want both", got.Encrypted, got.Emergency)
	}
	if got.At.Unix() != 1_700_000_000 {
		t.Errorf("At = %v, want injected Now", got.At)
	}
}

func TestControlChannelPublishesAffiliation(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Response = denied (2), AnnGroup = 0xAABB, Group = 0x1234,
	// TargetID = 0xABCDEF.
	payload := [8]byte{0x02, 0xAA, 0xBB, 0x12, 0x34, 0xAB, 0xCD, 0xEF}
	tsbk := TSBK{LB: true, Opcode: OpGroupAffiliationResponse, Payload: payload}
	stream := buildLockedStreamWithTSBK(10, 0x111, DUIDTrunkingSignaling, tsbk)

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Now:         func() time.Time { return time.Unix(1_700_000_001, 0).UTC() },
	})
	cc.Process(stream, 0)

	var got *trunking.Affiliation
	deadline := time.After(time.Second)
	for got == nil {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindAffiliation {
				a := ev.Payload.(trunking.Affiliation)
				got = &a
			}
		case <-deadline:
			t.Fatal("no affiliation event published")
		}
	}
	if got.System != "TestSys" || got.Protocol != "p25" {
		t.Errorf("identity = %s/%s", got.System, got.Protocol)
	}
	if got.SourceID != 0xABCDEF {
		t.Errorf("SourceID = %06X, want ABCDEF", got.SourceID)
	}
	if got.GroupID != 0x1234 {
		t.Errorf("GroupID = %04X, want 1234", got.GroupID)
	}
	if got.AnnouncementGroup != 0xAABB {
		t.Errorf("AnnouncementGroup = %04X, want AABB", got.AnnouncementGroup)
	}
	if got.Response != trunking.AffiliationDenied {
		t.Errorf("Response = %v, want denied", got.Response)
	}
	if got.At.Unix() != 1_700_000_001 {
		t.Errorf("At = %v, want injected Now", got.At)
	}
}

func TestControlChannelPublishesUnitRegistration(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Response = accepted (0), WACN = 0xBEE08, SystemID = 0x534,
	// SourceID = 0x112233.
	payload := [8]byte{0x00, 0xBE, 0xE0, 0x85, 0x34, 0x11, 0x22, 0x33}
	tsbk := TSBK{LB: true, Opcode: OpUnitRegistrationResponse, Payload: payload}
	stream := buildLockedStreamWithTSBK(10, 0x222, DUIDTrunkingSignaling, tsbk)

	cc := New(Options{
		Bus:         bus,
		SystemName:  "TestSys",
		FrequencyHz: 851_000_000,
		Now:         func() time.Time { return time.Unix(1_700_000_002, 0).UTC() },
	})
	cc.Process(stream, 0)

	var got *trunking.UnitRegistration
	deadline := time.After(time.Second)
	for got == nil {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindUnitRegistration {
				u := ev.Payload.(trunking.UnitRegistration)
				got = &u
			}
		case <-deadline:
			t.Fatal("no registration event published")
		}
	}
	if got.System != "TestSys" || got.Protocol != "p25" {
		t.Errorf("identity = %s/%s", got.System, got.Protocol)
	}
	if got.SourceID != 0x112233 {
		t.Errorf("SourceID = %06X, want 112233", got.SourceID)
	}
	if got.WACN != 0xBEE08 {
		t.Errorf("WACN = %05X, want BEE08", got.WACN)
	}
	if got.SystemID != 0x534 {
		t.Errorf("SystemID = %03X, want 534", got.SystemID)
	}
	if got.Response != trunking.RegistrationAccepted {
		t.Errorf("Response = %v, want accepted", got.Response)
	}
	if got.At.Unix() != 1_700_000_002 {
		t.Errorf("At = %v, want injected Now", got.At)
	}
}

func TestControlChannelGrantBeforeIdentifierUpdateEmitsDecodeError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	grantPayload := [8]byte{0x00, 0x20, 0x05, 0x00, 0x42, 0, 0, 0}
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: grantPayload}
	stream := buildLockedStreamWithTSBK(10, 0x111, DUIDTrunkingSignaling, grantTSBK)

	cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1})
	cc.Process(stream, 0)

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				t.Fatalf("unexpected grant emitted without band-plan: %+v", ev.Payload)
			}
			if ev.Kind != events.KindDecodeError {
				continue
			}
			de := ev.Payload.(events.DecodeError)
			if de.Protocol == "p25" && de.Stage == "no-bandplan" {
				return
			}
		case <-deadline:
			t.Fatal("no decode-error with stage=no-bandplan")
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
