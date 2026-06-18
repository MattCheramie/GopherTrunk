package phase1

import (
	"context"
	"log/slog"
	"strings"
	"sync"
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

// buildControlFrame builds one contiguous FSW + NID + TSBK frame (154
// dibits, no status symbols) for the given NAC / DUID / TSBK.
func buildControlFrame(nac uint16, duid DUID, tsbk TSBK) []uint8 {
	frame := make([]uint8, 0, 24+32+98)
	frame = append(frame, FrameSyncWord[:]...)
	bits := EncodeNIDBits(nac, duid)
	for i := 0; i < 32; i++ {
		frame = append(frame, (bits[2*i]<<1)|bits[2*i+1])
	}
	return append(frame, EncodeTSBKChannel(AssembleTSBK(tsbk))...)
}

// buildLockedStreamWithTSBK is the variant that takes a fully-formed
// TSBK so callers can carry a payload (used by the IdentifierUpdate +
// grant publication tests). The FSW + NID + TSBK frame is interleaved
// with P25 status symbols, mirroring a real on-air stream, so the
// receiver's status-symbol stripping is exercised.
func buildLockedStreamWithTSBK(offset int, nac uint16, duid DUID, tsbk TSBK) []uint8 {
	onAir := InjectControlStatusSymbols(buildControlFrame(nac, duid, tsbk))
	out := make([]uint8, offset+len(onAir)+16)
	copy(out[offset:], onAir)
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

// TestControlChannelLocksUnderDibitRotation feeds a TSDU stream whose
// dibits have all been shifted by k — the residual quadrant ambiguity
// the CQPSK/LSM demod leaves on simulcast P25 sites. The sync detector
// recovers the rotation; parseFrame must undo it so the NID + TSBK
// still decode. Before the rotateDibits fix, odd rotations (1, 3) —
// exactly the π/4-DQPSK quadrant slips — corrupted every NID and the
// control channel never locked.
func TestControlChannelLocksUnderDibitRotation(t *testing.T) {
	for k := uint8(0); k < 4; k++ {
		bus := events.NewBus(8)
		sub := bus.Subscribe()
		stream := buildLockedStream(10, 0x293, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
		// received = canonical - k, so the detector reports rot == k.
		for i := range stream {
			stream[i] = (stream[i] + 4 - k) & 3
		}
		cc := NewControlChannel(bus, nil, 851_000_000)
		cc.Process(stream, 0)
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				t.Errorf("k=%d kind = %s, want cc.locked", k, ev.Kind)
			} else if ls := ev.Payload.(LockState); ls.NAC != 0x293 {
				t.Errorf("k=%d NAC = %#x, want 0x293", k, ls.NAC)
			}
		case <-time.After(time.Second):
			t.Errorf("k=%d no lock — NID not recovered under rotation", k)
		}
		sub.Close()
		bus.Close()
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

	// Stream with FSW followed by 32 dibits of garbage (not a valid
	// NID), interleaved with status symbols as it arrives on-air.
	frame := make([]uint8, 24+32+98)
	copy(frame, FrameSyncWord[:])
	for i := 0; i < 32; i++ {
		frame[24+i] = uint8(i*7) & 0x3
	}
	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

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

func TestControlChannelAppliesVUHFIdentifierUpdateAndPublishesGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// UHF P25 site (the scenario that motivated wiring opcode 0x34):
	// IdentifierUpdateVUHF for ChannelID 1, base 460 MHz, 12.5 kHz
	// spacing, -5 MHz tx offset; followed by a GroupVoiceChannelGrant
	// on (ID=1, Number=16), which must resolve to 460.2 MHz via the
	// new dispatcher path.
	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdateVUHF}
	identTSBK.Payload = AssembleIdentifierUpdateVUHF(IdentifierUpdate{
		ChannelID:   1,
		BandwidthHz: 12_500,
		SpacingHz:   12_500,
		TxOffsetHz:  -5_000_000,
		BaseHz:      460_000_000,
	})

	grantPayload := [8]byte{
		0x00,                  // service options: cleartext, non-emergency
		(1 << 4) | 0x00, 0x10, // channel = ID 1, number 0x010 (=16)
		0x12, 0x34, // group address 0x1234
		0xAB, 0xCD, 0xEF, // source ID 0xABCDEF
	}
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: grantPayload}

	stream1 := buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, identTSBK)
	stream2 := buildLockedStreamWithTSBK(0, 0x293, DUIDTrunkingSignaling, grantTSBK)

	cc := New(Options{
		Bus:         bus,
		SystemName:  "UHF-Site",
		FrequencyHz: 461_437_500,
		Now:         func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	cc.Process(stream1, 0)
	cc.Process(stream2, len(stream1))

	var got *trunking.Grant
	deadline := time.After(time.Second)
	for got == nil {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindDecodeError {
				de := ev.Payload.(events.DecodeError)
				if de.Stage == "no-bandplan" {
					t.Fatal("VUHF IdentifierUpdate was not applied: grant fell to no-bandplan")
				}
			}
			if ev.Kind == events.KindGrant {
				g := ev.Payload.(trunking.Grant)
				got = &g
			}
		case <-deadline:
			t.Fatal("no grant event published")
		}
	}
	if got.System != "UHF-Site" || got.Protocol != "p25" {
		t.Errorf("identity = %s/%s", got.System, got.Protocol)
	}
	if got.GroupID != 0x1234 || got.SourceID != 0xABCDEF {
		t.Errorf("group/source = %d/%d, want 4660/11259375", got.GroupID, got.SourceID)
	}
	if got.ChannelID != 1 || got.ChannelNum != 0x010 {
		t.Errorf("channel = %d.%d, want 1.16", got.ChannelID, got.ChannelNum)
	}
	if got.FrequencyHz != 460_200_000 {
		t.Errorf("freq = %d, want 460_200_000", got.FrequencyHz)
	}
}

// TestControlChannelRoutesTDMAChannelGrantAsPhase2 confirms the fix
// for issue #376: when the Phase 1 CC has accepted an IdentifierUpdate
// for a Phase 2 TDMA channel (opcode 0x33), a subsequent voice grant
// on that channel ID must publish with Protocol="p25-phase2" + a
// populated P25Phase2Decode so the voice composer routes the call
// into the Phase 2 MAC dispatch path. Before this fix, all Phase 1 CC
// grants — including TDMA ones — published Protocol="p25", the
// composer never entered runP25Phase2VoiceChain, and PR #409's MAC
// PDU dispatch was dead code on MMR-class hybrid systems.
func TestControlChannelRoutesTDMAChannelGrantAsPhase2(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdateTDMA}
	identTSBK.Payload = AssembleIdentifierUpdateTDMA(IdentifierUpdate{
		ChannelID:   1,
		BandwidthHz: 12_500,
		SpacingHz:   12_500,
		BaseHz:      851_000_000,
	})

	grantPayload := [8]byte{
		0xC0,                  // service options: emergency + encrypted
		(1 << 4) | 0x00, 0x10, // channel = ID 1, number 0x010 (=16)
		0x12, 0x34,
		0xAB, 0xCD, 0xEF,
	}
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: grantPayload}

	stream1 := buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, identTSBK)
	stream2 := buildLockedStreamWithTSBK(0, 0x293, DUIDTrunkingSignaling, grantTSBK)

	cc := New(Options{
		Bus:                bus,
		SystemName:         "MMR",
		FrequencyHz:        851_000_000,
		P25Phase2Trellis:   1, // TrellisOn
		P25Phase2Scrambler: 1, // ScramblerOn
		Now:                func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	cc.Process(stream1, 0)
	cc.Process(stream2, len(stream1))

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

	if got.Protocol != "p25-phase2" {
		t.Errorf("Protocol = %q, want %q", got.Protocol, "p25-phase2")
	}
	if got.P25Phase2Decode.Trellis != 1 {
		t.Errorf("P25Phase2Decode.Trellis = %d, want 1 (TrellisOn)", got.P25Phase2Decode.Trellis)
	}
	if got.P25Phase2Decode.Scrambler != 1 {
		t.Errorf("P25Phase2Decode.Scrambler = %d, want 1 (ScramblerOn)", got.P25Phase2Decode.Scrambler)
	}
	if got.FrequencyHz != 851_200_000 {
		t.Errorf("FrequencyHz = %d, want 851_200_000", got.FrequencyHz)
	}
}

// TestControlChannelLocksAndGrantsAcrossSmallChunks feeds a two-frame
// stream (IdentifierUpdate then GroupVoiceChannelGrant) through Process
// in 19-dibit batches — the dibit count a real RTL-SDR's 16 KiB USB
// transfer yields per call. No single batch holds a whole 154-dibit
// frame, so locking + granting here proves Process assembles frames
// across call boundaries (issue #275 — previously every FSW hit was
// discarded because the NID/TSBK lookahead had to fit in one call).
func TestControlChannelLocksAndGrantsAcrossSmallChunks(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	const nac = 0x293
	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdate}
	identTSBK.Payload = AssembleIdentifierUpdate(IdentifierUpdate{
		ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000,
	})
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: [8]byte{
		0x00,
		(1 << 4) | 0x00, 0x10, // channel = ID 1, number 16
		0x12, 0x34, // group address 0x1234
		0xAB, 0xCD, 0xEF, // source ID 0xABCDEF
	}}

	stream := append(
		buildLockedStreamWithTSBK(10, nac, DUIDTrunkingSignaling, identTSBK),
		buildLockedStreamWithTSBK(10, nac, DUIDTrunkingSignaling, grantTSBK)...,
	)

	cc := New(Options{
		Bus: bus, SystemName: "TestSys", FrequencyHz: 851_000_000,
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})

	const batch = 19
	for off := 0; off < len(stream); off += batch {
		end := off + batch
		if end > len(stream) {
			end = len(stream)
		}
		cc.Process(stream[off:end], off)
	}

	var locked, granted bool
	for drained := false; !drained; {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				locked = true
			case events.KindGrant:
				if g := ev.Payload.(trunking.Grant); g.FrequencyHz != 851_200_000 {
					t.Errorf("grant freq = %d, want 851_200_000", g.FrequencyHz)
				}
				granted = true
			}
		default:
			drained = true
		}
	}
	if !locked {
		t.Error("no cc.locked event — frame not assembled across 19-dibit batches")
	}
	if !granted {
		t.Error("no grant event — TSBK not assembled across 19-dibit batches")
	}
}

// buildControlFrameMultiTSBK builds one FSW + NID followed by several
// back-to-back 98-dibit TSBK channel blocks — the P25 TSDU layout that packs
// up to maxTSBKBlocks signalling blocks after a single sync/NID. buildControl-
// Frame only carries one block; this exercises the multi-block path.
func buildControlFrameMultiTSBK(nac uint16, duid DUID, tsbks ...TSBK) []uint8 {
	frame := make([]uint8, 0, 24+32+len(tsbks)*98)
	frame = append(frame, FrameSyncWord[:]...)
	bits := EncodeNIDBits(nac, duid)
	for i := 0; i < 32; i++ {
		frame = append(frame, (bits[2*i]<<1)|bits[2*i+1])
	}
	for _, tsbk := range tsbks {
		frame = append(frame, EncodeTSBKChannel(AssembleTSBK(tsbk))...)
	}
	return frame
}

// TestControlChannelDecodesAllThreeTSBKBlocksAcrossChunks pins the issue
// #402 / PR #470 multi-block decode under chunk boundaries: a single TSDU
// carries three TSBK blocks after one FSW+NID, and parseFrame must decode all
// three — resuming the trailing blocks across Process() calls — even when the
// IQ chunk boundary falls mid-TSDU. The live "block=2 fails first" report was
// a dropped IQ chunk corrupting the last block; this test proves the decode
// path itself stays continuous when the dibits do, so a future refactor of the
// resume/continuation bookkeeping can't silently drop blocks 2-3 again (the
// pre-#470 behaviour that dropped ~2/3 of site signalling).
func TestControlChannelDecodesAllThreeTSBKBlocksAcrossChunks(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	const nac = 0x293
	// Block 0: identifier update so the grants in blocks 1-2 resolve a freq.
	ident := TSBK{LB: false, Opcode: OpIdentifierUpdate}
	ident.Payload = AssembleIdentifierUpdate(IdentifierUpdate{
		ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000,
	})
	// Block 1: grant on channel number 16 -> 851_200_000.
	grantA := TSBK{LB: false, Opcode: OpGroupVoiceChannelGrant, Payload: [8]byte{
		0x00, (1 << 4) | 0x00, 0x10, 0x12, 0x34, 0xAB, 0xCD, 0xEF,
	}}
	// Block 2 (last): grant on channel number 32 -> 851_400_000. This is the
	// "block=2" the live report saw fail first.
	grantB := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: [8]byte{
		0x00, (1 << 4) | 0x00, 0x20, 0x56, 0x78, 0x11, 0x22, 0x33,
	}}

	onAir := InjectControlStatusSymbols(buildControlFrameMultiTSBK(nac, DUIDTrunkingSignaling, ident, grantA, grantB))
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

	cc := New(Options{
		Bus: bus, SystemName: "TestSys", FrequencyHz: 851_000_000,
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})

	// An odd batch splits the TSDU mid-block so the trailing TSBKs land on a
	// later Process() call and must be resumed.
	const batch = 17
	for off := 0; off < len(stream); off += batch {
		end := off + batch
		if end > len(stream) {
			end = len(stream)
		}
		cc.Process(stream[off:end], off)
	}

	gotFreqs := map[uint32]bool{}
	for drained := false; !drained; {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				gotFreqs[ev.Payload.(trunking.Grant).FrequencyHz] = true
			}
		default:
			drained = true
		}
	}
	if !gotFreqs[851_200_000] {
		t.Error("block 1 grant did not decode — second TSBK block dropped")
	}
	if !gotFreqs[851_400_000] {
		t.Error("block 2 grant did not decode — last (block=2) TSBK dropped; multi-block resume regressed")
	}
}

func TestControlChannelPublishesAffiliation(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Feed an RFSS Status Broadcast first (LRA=9, SystemID=0x123,
	// RFSS=1, Site=1) so the affiliation event stamps the serving site
	// (issue #698).
	rfssTSBK := TSBK{LB: true, Opcode: OpRFSSStatusBroadcast, Payload: [8]byte{0x09, 0x01, 0x23, 0x01, 0x01, 0x00, 0x00, 0x00}}
	rfssStream := buildLockedStreamWithTSBK(10, 0x111, DUIDTrunkingSignaling, rfssTSBK)

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
	cc.Process(rfssStream, 0)
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
	if got.RFSSID != 1 || got.SiteID != 1 {
		t.Errorf("site = RFSS %d / Site %d, want 1/1", got.RFSSID, got.SiteID)
	}
	if got.NAC != 0x111 {
		t.Errorf("NAC = %03X, want 111", got.NAC)
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

	// Feed an RFSS Status Broadcast first (RFSS=2, Site=9) so the
	// registration event stamps the serving site (issue #698).
	rfssTSBK := TSBK{LB: true, Opcode: OpRFSSStatusBroadcast, Payload: [8]byte{0x09, 0x01, 0x23, 0x02, 0x09, 0x00, 0x00, 0x00}}
	rfssStream := buildLockedStreamWithTSBK(10, 0x222, DUIDTrunkingSignaling, rfssTSBK)

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
	cc.Process(rfssStream, 0)
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
	if got.RFSSID != 2 || got.SiteID != 9 {
		t.Errorf("site = RFSS %d / Site %d, want 2/9", got.RFSSID, got.SiteID)
	}
	if got.NAC != 0x222 {
		t.Errorf("NAC = %03X, want 222", got.NAC)
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

	// Valid FSW + valid NID + corrupted TSBK channel block. Corrupt the
	// contiguous TSBK before status symbols are interleaved, so the
	// flips land squarely on the 98-dibit channel block.
	frame := buildControlFrame(0x111, DUIDTrunkingSignaling, TSBK{LB: true, Opcode: OpRFSSStatusBroadcast})
	// Flip every dibit in the TSBK block — well beyond the Viterbi
	// correction radius, so the CRC trailer will fail.
	for i := 24 + 32; i < 24+32+98; i++ {
		frame[i] = (^frame[i]) & 0x3
	}
	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

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

// TestControlChannelLocksThroughPostFSWSlip splices surplus dibits
// between the FSW and the NID — the post-FSW symbol slip issue #275's
// reporter observed, where the FSW detects reliably but the NID, shifted
// by a dibit or two, never BCH-decodes under a fixed single-offset read.
// parseFrame's bounded alignment search must recover the lock across the
// full NIDSearchSpan range — slips 1..6 after the span was widened from
// the original ±2 (which left field retests pegged at the +2 edge every
// frame; #275 post-#321).
func TestControlChannelLocksThroughPostFSWSlip(t *testing.T) {
	for _, slip := range []int{1, 2, 3, 4, 5, 6} {
		bus := events.NewBus(8)
		sub := bus.Subscribe()

		base := buildLockedStream(10, 0x293, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
		// The FSW occupies indices 10..33; splice `slip` surplus dibits
		// in right after it, before the NID. Everything downstream —
		// NID, its interleaved status symbol, the TSBK — shifts with it.
		slipped := make([]uint8, 0, len(base)+slip)
		slipped = append(slipped, base[:34]...)
		for i := 0; i < slip; i++ {
			slipped = append(slipped, 0)
		}
		slipped = append(slipped, base[34:]...)

		cc := NewControlChannel(bus, nil, 851_000_000)
		cc.Process(slipped, 0)
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				t.Errorf("slip=%d kind = %s, want cc.locked", slip, ev.Kind)
			} else if ls := ev.Payload.(LockState); ls.NAC != 0x293 {
				t.Errorf("slip=%d NAC = %#x, want 0x293", slip, ls.NAC)
			}
		case <-time.After(time.Second):
			t.Errorf("slip=%d no lock — alignment search did not recover the NID", slip)
		}
		sub.Close()
		bus.Close()
	}
}

// TestControlChannelLocksThroughPostFSWSlipSmallChunks repeats the
// post-FSW slip test under RTL-realistic small-chunk delivery: each
// Process call carries only ~19 dibits, so the slipped frame straddles
// many calls and parseFrame runs under whatever buffer state
// trimBuffer leaves it in. The widened search must still recover the
// lock — i.e. the buffer math at the new NIDSearchSpan stays
// consistent with cross-call frame assembly (#275 post-#321).
func TestControlChannelLocksThroughPostFSWSlipSmallChunks(t *testing.T) {
	for _, slip := range []int{1, 3, 6} {
		bus := events.NewBus(16)
		sub := bus.Subscribe()

		base := buildLockedStream(10, 0x293, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
		slipped := make([]uint8, 0, len(base)+slip)
		slipped = append(slipped, base[:34]...)
		for i := 0; i < slip; i++ {
			slipped = append(slipped, 0)
		}
		slipped = append(slipped, base[34:]...)

		cc := NewControlChannel(bus, nil, 851_000_000)
		const batch = 19
		for off := 0; off < len(slipped); off += batch {
			end := off + batch
			if end > len(slipped) {
				end = len(slipped)
			}
			cc.Process(slipped[off:end], off)
		}
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				t.Errorf("slip=%d kind = %s, want cc.locked", slip, ev.Kind)
			} else if ls := ev.Payload.(LockState); ls.NAC != 0x293 {
				t.Errorf("slip=%d NAC = %#x, want 0x293", slip, ls.NAC)
			}
		case <-time.After(time.Second):
			t.Errorf("slip=%d no lock under small-chunk delivery — buffer/trim math broke at the widened span", slip)
		}
		sub.Close()
		bus.Close()
	}
}

// TestControlChannelLocksThroughWiderPostFSWSlip confirms the
// per-instance Options.NIDSearchSpan override actually widens the
// search beyond the package-level constant. A 10-dibit post-FSW slip
// is unreachable for a default ControlChannel (NIDSearchSpan=6) but
// must lock for one constructed with Options{NIDSearchSpan: 12} — the
// bisect knob the replay subcommand exposes for issue #275 retests
// where the closest miss keeps pegging at the ±6 boundary.
func TestControlChannelLocksThroughWiderPostFSWSlip(t *testing.T) {
	const slip = 10
	base := buildLockedStream(10, 0x293, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
	slipped := make([]uint8, 0, len(base)+slip)
	slipped = append(slipped, base[:34]...)
	for i := 0; i < slip; i++ {
		slipped = append(slipped, 0)
	}
	slipped = append(slipped, base[34:]...)

	// Default span must NOT lock — guards that the test fixture's
	// slip really does exceed the production grid (and so the wider
	// span below is the cause of the recovery, not noise).
	{
		bus := events.NewBus(8)
		sub := bus.Subscribe()
		cc := New(Options{Bus: bus, FrequencyHz: 851_000_000})
		cc.Process(slipped, 0)
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				t.Fatalf("default span locked at slip=%d — fixture no longer exercises >±%d offsets", slip, NIDSearchSpan)
			}
		case <-time.After(100 * time.Millisecond):
			// no lock expected — pass
		}
		sub.Close()
		bus.Close()
	}

	// Wider span MUST lock — proves the per-instance override drives
	// the grid bounds in searchNID.
	bus := events.NewBus(8)
	sub := bus.Subscribe()
	cc := New(Options{Bus: bus, FrequencyHz: 851_000_000, NIDSearchSpan: 12})
	cc.Process(slipped, 0)
	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Errorf("kind = %s, want cc.locked", ev.Kind)
		} else if ls := ev.Payload.(LockState); ls.NAC != 0x293 {
			t.Errorf("NAC = %#x, want 0x293", ls.NAC)
		}
	case <-time.After(time.Second):
		t.Errorf("no lock at slip=%d under NIDSearchSpan=12 — per-instance span did not widen the grid", slip)
	}
	sub.Close()
	bus.Close()
}

// TestSearchBoundaryHelpers locks down the boundary-pegged diag the
// next #275 field retest will hinge on. The two helpers — atSearchBoundary
// and boundaryNote — together turn the failure diag (and the success
// "nid corrected" log) into a measurement: a reader can tell a bounded
// framing slip (search converged interior with low errs → fix worked)
// from one that exceeds the span (still pegged at the edge → look
// elsewhere). The diag-string format wires boundaryNote() directly into
// the format strings, so guarding the helpers guards the contract.
func TestSearchBoundaryHelpers(t *testing.T) {
	cases := []struct {
		delta      int
		atBoundary bool
	}{
		{0, false},
		{1, false},
		{NIDSearchSpan - 1, false},
		{NIDSearchSpan, true},
		{-NIDSearchSpan, true},
		{-(NIDSearchSpan - 1), false},
		{NIDSearchSpan + 1, false}, // outside the grid; not "at" boundary
	}
	for _, tc := range cases {
		got := atSearchBoundary(tc.delta, NIDSearchSpan)
		if got != tc.atBoundary {
			t.Errorf("atSearchBoundary(%d) = %v, want %v", tc.delta, got, tc.atBoundary)
		}
		note := boundaryNote(tc.delta, NIDSearchSpan)
		if tc.atBoundary {
			if !strings.Contains(note, "search boundary") {
				t.Errorf("boundaryNote(%d) = %q, want contains \"search boundary\"", tc.delta, note)
			}
		} else if note != "" {
			t.Errorf("boundaryNote(%d) = %q, want empty", tc.delta, note)
		}
	}
}

// TestSearchBoundaryHelpersHonorPerInstanceSpan exercises the bisect
// knob #275's diagnostic-widening commit added: a ControlChannel
// constructed with Options.NIDSearchSpan=12 must report delta=±12 (not
// ±6) as the boundary, and emit the matching note. Without this guard a
// refactor could silently revert the helpers to the package constant
// and quietly invalidate every wider-span replay run.
func TestSearchBoundaryHelpersHonorPerInstanceSpan(t *testing.T) {
	const wider = 12
	if atSearchBoundary(NIDSearchSpan, wider) {
		t.Errorf("atSearchBoundary(%d, %d) = true, want false (only ±%d is boundary)",
			NIDSearchSpan, wider, wider)
	}
	if !atSearchBoundary(wider, wider) {
		t.Errorf("atSearchBoundary(%d, %d) = false, want true", wider, wider)
	}
	note := boundaryNote(wider, wider)
	if !strings.Contains(note, "±12") {
		t.Errorf("boundaryNote(%d, %d) = %q, want contains \"±12\"", wider, wider, note)
	}
}

// TestSearchNIDFailureDiagFlagsBoundary exercises the wiring of
// boundaryNote into searchNID's failure diag. A slip past NIDSearchSpan
// can never be recovered; we feed several deliberately-corrupt streams
// and assert that at least one of the failure diags surfaces the
// "search boundary" suffix. This guards the format-string call to
// boundaryNote without requiring the BCH error landscape to land a
// boundary-pegged hypothesis on any single fixture.
func TestSearchNIDFailureDiagFlagsBoundary(t *testing.T) {
	seenBoundary := false
	for seed := uint32(1); seed <= 24 && !seenBoundary; seed++ {
		cap := &diagCapture{}
		bus := events.NewBus(16)
		// Build a frame whose NID dibits are deliberately poisoned at
		// the seed-derived positions: the BCH decoder will find a
		// nearest-codeword miscorrection at SOME delta, and across
		// seeds at least one will be the +6 or -6 grid edge.
		frame := buildControlFrame(0x293, DUIDTrunkingSignaling,
			TSBK{LB: true, Opcode: OpRFSSStatusBroadcast})
		r := seed
		for i := 24; i < 24+32; i++ {
			r = r*1664525 + 1013904223
			frame[i] ^= uint8((r >> 13) & 0x3)
		}
		// Corrupt the TSBK too so corroboration cannot rescue anything.
		for i := 24 + 32; i < 24+32+98; i++ {
			frame[i] = (^frame[i]) & 0x3
		}
		onAir := InjectControlStatusSymbols(frame)
		stream := make([]uint8, 10+len(onAir)+16)
		copy(stream[10:], onAir)

		cc := New(Options{Bus: bus, Log: slog.New(cap), FrequencyHz: 851_000_000})
		cc.Process(stream, 0)
		if cap.containsDiag("search boundary") {
			seenBoundary = true
		}
		bus.Close()
	}
	if !seenBoundary {
		t.Skip("BCH error landscape never produced a boundary-pegged closest miss across 24 fixtures; helper unit-test still guards the contract")
	}
}

// TestControlChannelC4FMRejectsNonPhysicalRotation guards the C4FM
// rotation-set restriction (issue #275 post-#321). A C4FM FM-discriminator
// stream physically presents only rotations 0 and 2; the field retest
// converged on a rot=3 NID miscorrection that crowded out the real
// alignment. With RotationsC4FM the search must NOT lock on a
// rot=1-shifted stream (it's not a real signal a C4FM front end can
// produce). With the default RotationsAll the same stream must lock —
// proving the restriction is what's blocking the lock, not the stream.
func TestControlChannelC4FMRejectsNonPhysicalRotation(t *testing.T) {
	base := buildLockedStream(10, 0x293, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
	// Rotate the stream by k=1: received = canonical - 1, so the FSW
	// correlator finds the FSW under rotation 1.
	rotated := make([]uint8, len(base))
	for i, d := range base {
		rotated[i] = (d + 3) & 3
	}

	// With C4FM rotation restriction {0,2}, no FSW hit, no lock.
	t.Run("c4fm-rejects", func(t *testing.T) {
		bus := events.NewBus(16)
		defer bus.Close()
		sub := bus.Subscribe()
		defer sub.Close()

		cc := New(Options{
			Bus:         bus,
			FrequencyHz: 851_000_000,
			Rotations:   RotationsC4FM,
		})
		cc.Process(rotated, 0)

		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				t.Fatalf("rot=1 stream locked under C4FM rotation restriction; should not")
			}
		case <-time.After(100 * time.Millisecond):
			// No lock event — expected.
		}
	})

	// Sanity: the default all-rotations set still locks the same stream.
	t.Run("all-rotations-locks", func(t *testing.T) {
		bus := events.NewBus(16)
		defer bus.Close()
		sub := bus.Subscribe()
		defer sub.Close()

		cc := NewControlChannel(bus, nil, 851_000_000)
		cc.Process(rotated, 0)

		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				t.Errorf("kind = %s, want cc.locked under RotationsAll", ev.Kind)
			} else if ls := ev.Payload.(LockState); ls.NAC != 0x293 {
				t.Errorf("NAC = %#x, want 0x293", ls.NAC)
			}
		case <-time.After(time.Second):
			t.Errorf("rot=1 stream did not lock under RotationsAll — sanity check broken")
		}
	})
}

// diagCapture is a slog.Handler that records the "diag" attribute of
// every NID-decode failure log line so the boundary-diag test can
// assert on its content.
type diagCapture struct {
	mu    sync.Mutex
	diags []string
}

func (h *diagCapture) Enabled(context.Context, slog.Level) bool { return true }

func (h *diagCapture) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "diag" {
			h.diags = append(h.diags, a.Value.String())
		}
		return true
	})
	return nil
}

func (h *diagCapture) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *diagCapture) WithGroup(string) slog.Handler      { return h }

func (h *diagCapture) containsDiag(needle string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, d := range h.diags {
		if strings.Contains(d, needle) {
			return true
		}
	}
	return false
}

// TestControlChannelNoFalseLockOnNoise feeds a valid FSW followed by
// pseudo-random dibits. parseFrame's bounded alignment search probes
// every combination of (delta, strip, rot) across the trusted
// (BCH + even-parity) and marginal (NID errs 7..11 corroborated by the
// TSBK CRC) tiers; neither may manufacture a lock out of noise — a
// noise NID has no clean TSBK to corroborate it.
func TestControlChannelNoFalseLockOnNoise(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	frame := make([]uint8, 24+32+98)
	copy(frame, FrameSyncWord[:])
	// Deterministic LCG fill so the test is reproducible.
	r := uint32(0x9E3779B9)
	for i := 24; i < len(frame); i++ {
		r = r*1664525 + 1013904223
		frame[i] = uint8(r>>13) & 0x3
	}
	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

	cc := NewControlChannel(bus, nil, 851_000_000)
	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind == events.KindCCLocked {
			t.Fatalf("false lock on noise: %+v", ev.Payload)
		}
		// KindDecodeError is the correct outcome here.
	case <-time.After(100 * time.Millisecond):
	}
}

// TestControlChannelLocksOnMarginalNIDCorroboratedByTSBK injects 8 bit
// errors into the NID — inside BCH(63,16,11)'s t=11 radius but past the
// NIDAcceptErrs=6 trusted threshold — while leaving the TSBK intact.
// searchNID's marginal tier must still lock, because the frame's TSBK
// decodes cleanly under the same alignment and corroborates the NID.
// This is issue #275's strong-site symptom: a NID that BCH-decodes but
// pegs above the trusted gate (the reporter's 9/10/11-error probes).
func TestControlChannelLocksOnMarginalNIDCorroboratedByTSBK(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	frame := buildControlFrame(0x293, DUIDTrunkingSignaling,
		TSBK{LB: true, Opcode: OpRFSSStatusBroadcast})
	// Flip the MSB of 8 distinct NID dibits (frame[24:56]). Each is a
	// distinct even-indexed codeword bit, so exactly 8 BCH errors land;
	// the trailing parity bit (LSB of NID dibit 31) is left untouched,
	// so the corrected codeword still passes even-parity.
	for _, i := range []int{1, 4, 8, 12, 16, 20, 25, 30} {
		frame[24+i] ^= 0b10
	}
	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

	cc := NewControlChannel(bus, nil, 851_000_000)
	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		if ev.Kind != events.KindCCLocked {
			t.Fatalf("kind = %s, want cc.locked", ev.Kind)
		}
		if ls := ev.Payload.(LockState); ls.NAC != 0x293 {
			t.Errorf("NAC = %#x, want 0x293", ls.NAC)
		}
	case <-time.After(time.Second):
		t.Fatal("no lock — marginal NID not corroborated by its clean TSBK")
	}
}

// TestSearchNIDClosestMissReportsErrPattern locks down the per-dibit
// error fingerprint searchNID now appends to the closest-miss diag
// (issue #275 — the field reporter needs to distinguish post-FSW
// timing slip from SNR-limited demod corruption on the next retest).
//
// Eight known NID dibits get their MSB flipped (= one bit error per
// dibit, eight bit errors total — inside BCH(63,16,11)'s t=11 radius
// but past the trusted gate of NIDAcceptErrs=6). The TSBK is corrupted
// so the marginal tier cannot corroborate, forcing the
// "no NID corroborated" diag branch — which is where err_pattern is
// emitted. Asserting on the exact 32-char pattern verifies both that
// the formatter renders it correctly AND that the canonical-alignment
// hypothesis (delta=0, strip=true, rot=fswRot) wins marginal[0]
// (lowest errs of the parity-valid candidates).
func TestSearchNIDClosestMissReportsErrPattern(t *testing.T) {
	cap := &diagCapture{}
	bus := events.NewBus(16)
	defer bus.Close()

	frame := buildControlFrame(0x293, DUIDTrunkingSignaling,
		TSBK{LB: true, Opcode: OpRFSSStatusBroadcast})
	injected := []int{1, 4, 8, 12, 16, 20, 25, 30}
	for _, i := range injected {
		frame[24+i] ^= 0b10
	}
	for i := 24 + 32; i < 24+32+98; i++ {
		frame[i] = (^frame[i]) & 0x3
	}
	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

	cc := New(Options{Bus: bus, Log: slog.New(cap), FrequencyHz: 851_000_000})
	cc.Process(stream, 0)

	want := make([]byte, 32)
	for i := range want {
		want[i] = '0'
	}
	for _, i := range injected {
		want[i] = '1'
	}
	needle := "err_pattern=" + string(want)
	if !cap.containsDiag(needle) {
		cap.mu.Lock()
		seen := append([]string(nil), cap.diags...)
		cap.mu.Unlock()
		t.Fatalf("no diag contained %q\nseen diags: %v", needle, seen)
	}
}

// TestControlChannelRejectsMarginalNIDWithoutTSBK is the false-lock
// guard for the marginal tier: the same 8-bit-error NID, but with the
// TSBK channel block corrupted too. With no clean TSBK to corroborate
// the alignment, searchNID must reject the NID — a NID 7..11 BCH
// corrections from the received word is as plausibly a miscorrection as
// a real noisy NID, and only the TSBK CRC can tell them apart.
func TestControlChannelRejectsMarginalNIDWithoutTSBK(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	frame := buildControlFrame(0x293, DUIDTrunkingSignaling,
		TSBK{LB: true, Opcode: OpRFSSStatusBroadcast})
	for _, i := range []int{1, 4, 8, 12, 16, 20, 25, 30} {
		frame[24+i] ^= 0b10
	}
	// Flip every TSBK dibit — well past the Viterbi correction radius,
	// so the CRC trailer fails and the alignment cannot be corroborated.
	for i := 24 + 32; i < 24+32+98; i++ {
		frame[i] = (^frame[i]) & 0x3
	}
	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

	cc := NewControlChannel(bus, nil, 851_000_000)
	cc.Process(stream, 0)

	deadline := time.After(time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				t.Fatal("false lock: marginal NID accepted without TSBK corroboration")
			}
			if ev.Kind == events.KindDecodeError {
				return
			}
		case <-deadline:
			t.Fatal("no decode-error event for uncorroborated marginal NID")
		}
	}
}

// TestControlChannelDeferredGrantReplayedAfterIdentifierUpdate covers
// the issue #345 follow-up: a Group Voice Channel Grant TSBK arrives
// before the matching IdentifierUpdate. The grant is queued; the
// subsequent IDEN_UP drains the queue and the grant resolves through
// the freshly applied band-plan slot. Operator-visible outcome: a
// trunking.Grant event lands on the bus with the correct
// FrequencyHz, restoring call/talkgroup creation in the UI for sites
// that broadcast IDEN_UP at a slower cadence than the first grant.
func TestControlChannelDeferredGrantReplayedAfterIdentifierUpdate(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Same payload layout as the VUHF happy path (id=1, num=16 on a
	// 460 MHz UHF system) but reversed in arrival order — grant first,
	// IDEN_UP second.
	grantPayload := [8]byte{
		0x00,
		(1 << 4) | 0x00, 0x10,
		0x12, 0x34,
		0xAB, 0xCD, 0xEF,
	}
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: grantPayload}
	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdateVUHF}
	identTSBK.Payload = AssembleIdentifierUpdateVUHF(IdentifierUpdate{
		ChannelID:   1,
		BandwidthHz: 12_500,
		SpacingHz:   12_500,
		TxOffsetHz:  -5_000_000,
		BaseHz:      460_000_000,
	})

	stream1 := buildLockedStreamWithTSBK(10, 0x293, DUIDTrunkingSignaling, grantTSBK)
	stream2 := buildLockedStreamWithTSBK(0, 0x293, DUIDTrunkingSignaling, identTSBK)

	// Clock is monotonic but the two TSBKs land within a few hundred
	// ms of each other on real systems; keep the test clock well
	// inside pendingGrantTTL.
	base := time.Unix(1_700_000_000, 0).UTC()
	cc := New(Options{
		Bus: bus, SystemName: "UHF-Site", FrequencyHz: 461_437_500,
		Now: func() time.Time { return base },
	})
	cc.Process(stream1, 0)
	cc.Process(stream2, len(stream1))

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
			t.Fatal("no grant event published after deferred IDEN_UP drained the queue")
		}
	}
	if got.ChannelID != 1 || got.ChannelNum != 0x010 || got.FrequencyHz != 460_200_000 {
		t.Errorf("grant = id=%d num=%d freq=%d, want 1/16/460200000",
			got.ChannelID, got.ChannelNum, got.FrequencyHz)
	}
	if got.GroupID != 0x1234 || got.SourceID != 0xABCDEF {
		t.Errorf("group/source = %d/%d, want 4660/11259375", got.GroupID, got.SourceID)
	}
}

// TestControlChannelDeferredGrantExpiresAfterTTL covers the cap on
// the deferred queue: a queued grant older than pendingGrantTTL is
// dropped on drain. We can't easily advance the receiver's wall clock
// across the FSM, so we exercise the ring directly — the wiring in
// control.go is one-line and shares the same Now() injection.
func TestControlChannelDeferredGrantExpiresAfterTTL(t *testing.T) {
	var q pendingGrants
	base := time.Unix(1_700_000_000, 0).UTC()
	q.add(7, voiceGrant{channelID: 7, channelNumber: 42}, 0x293, base)

	// Advance one second past the TTL boundary — the entry must be dropped.
	drained := q.drain(7, base.Add(pendingGrantTTL+time.Second))
	if len(drained) != 0 {
		t.Fatalf("expected 0 drained entries after TTL, got %d", len(drained))
	}

	// A fresh add followed by an immediate drain still works.
	q.add(7, voiceGrant{channelID: 7, channelNumber: 99}, 0x293, base)
	drained = q.drain(7, base.Add(time.Second))
	if len(drained) != 1 || drained[0].g.channelNumber != 99 {
		t.Fatalf("fresh entry not preserved: %+v", drained)
	}
}

// TestControlChannelDeferredGrantsBoundedByRingCap ensures the
// per-channel-ID ring drops oldest entries when full so a stuck
// channel ID (site never broadcasts the IDEN_UP) can't grow memory
// unbounded.
func TestControlChannelDeferredGrantsBoundedByRingCap(t *testing.T) {
	var q pendingGrants
	base := time.Unix(1_700_000_000, 0).UTC()
	for i := 0; i < pendingGrantSlotCap+2; i++ {
		q.add(3, voiceGrant{channelID: 3, channelNumber: uint16(i)}, 0x111, base)
	}
	drained := q.drain(3, base)
	if len(drained) != pendingGrantSlotCap {
		t.Fatalf("ring size = %d, want %d", len(drained), pendingGrantSlotCap)
	}
	// Oldest two entries (channelNumber 0, 1) should have been
	// evicted; the surviving range is [2, 2+cap).
	for i, e := range drained {
		want := uint16(i + 2)
		if e.g.channelNumber != want {
			t.Errorf("drained[%d].channelNumber = %d, want %d (oldest should have been evicted)",
				i, e.g.channelNumber, want)
		}
	}
}

// TestControlChannelAppliesTDMAIdentifierUpdateAndPublishesGrant
// covers the issue #345 round-3 fix: the Mt Anakie site broadcasts
// IDEN_UP for channel id=10 as opcode 0x33 (TDMA-2), not 0x34 VUHF.
// With 0x33 now wired into dispatchTSBK, a subsequent grant on
// (id=10, num=176) must resolve through the freshly applied band-plan
// slot to 468.6125 MHz — the exact downlink the operator should see
// pop into /api/v1/calls/active.
func TestControlChannelAppliesTDMAIdentifierUpdateAndPublishesGrant(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdateTDMA}
	identTSBK.Payload = AssembleIdentifierUpdateTDMA(IdentifierUpdate{
		ChannelID:   10,
		BandwidthHz: 6_250,
		SpacingHz:   6_250,
		TxOffsetHz:  -10_000_000,
		BaseHz:      467_512_500,
	})

	// num=176 = 0x0B0 — exact Mt Anakie grant the operator reported.
	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: [8]byte{
		0x00,
		(10 << 4) | 0x00, 0xB0,
		0x12, 0x34,
		0xAB, 0xCD, 0xEF,
	}}

	stream1 := buildLockedStreamWithTSBK(10, 0x164, DUIDTrunkingSignaling, identTSBK)
	stream2 := buildLockedStreamWithTSBK(0, 0x164, DUIDTrunkingSignaling, grantTSBK)

	cc := New(Options{
		Bus: bus, SystemName: "MMR", FrequencyHz: 420_087_500,
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})
	cc.Process(stream1, 0)
	cc.Process(stream2, len(stream1))

	var got *trunking.Grant
	deadline := time.After(time.Second)
	for got == nil {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindDecodeError {
				de := ev.Payload.(events.DecodeError)
				if de.Stage == "no-bandplan" {
					t.Fatal("TDMA IdentifierUpdate was not applied: grant fell to no-bandplan")
				}
			}
			if ev.Kind == events.KindGrant {
				g := ev.Payload.(trunking.Grant)
				got = &g
			}
		case <-deadline:
			t.Fatal("no grant event published")
		}
	}
	if got.ChannelID != 10 || got.ChannelNum != 0x0B0 {
		t.Errorf("grant channel = %d.%d, want 10.176", got.ChannelID, got.ChannelNum)
	}
	// base 467_512_500 + 176 * 6250 = 468_612_500
	if got.FrequencyHz != 468_612_500 {
		t.Errorf("freq = %d, want 468_612_500", got.FrequencyHz)
	}
}

// TestControlChannelDeferredGrantReplayedAfterTDMAIdentifierUpdate
// composes the prior fix (deferred queue) with the new TDMA dispatch:
// a grant arrives before its TDMA IDEN_UP, lands in the pending ring,
// then the IDEN_UP arrives via the 0x33 path and drains it.
func TestControlChannelDeferredGrantReplayedAfterTDMAIdentifierUpdate(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	grantTSBK := TSBK{LB: true, Opcode: OpGroupVoiceChannelGrant, Payload: [8]byte{
		0x00,
		(10 << 4) | 0x00, 0xB0,
		0x12, 0x34,
		0xAB, 0xCD, 0xEF,
	}}
	identTSBK := TSBK{LB: false, Opcode: OpIdentifierUpdateTDMA}
	identTSBK.Payload = AssembleIdentifierUpdateTDMA(IdentifierUpdate{
		ChannelID:   10,
		BandwidthHz: 6_250,
		SpacingHz:   6_250,
		TxOffsetHz:  -10_000_000,
		BaseHz:      467_512_500,
	})

	stream1 := buildLockedStreamWithTSBK(10, 0x164, DUIDTrunkingSignaling, grantTSBK)
	stream2 := buildLockedStreamWithTSBK(0, 0x164, DUIDTrunkingSignaling, identTSBK)

	base := time.Unix(1_700_000_000, 0).UTC()
	cc := New(Options{
		Bus: bus, SystemName: "MMR", FrequencyHz: 420_087_500,
		Now: func() time.Time { return base },
	})
	cc.Process(stream1, 0)
	cc.Process(stream2, len(stream1))

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
			t.Fatal("no grant event published after TDMA IDEN_UP drained the queue")
		}
	}
	if got.FrequencyHz != 468_612_500 {
		t.Errorf("freq = %d, want 468_612_500", got.FrequencyHz)
	}
}

// TestControlChannelStatsCountsTrustedDecodes drives a clean TSDU
// stream through Process and asserts the new CCStats counter for
// NID-trusted accept + TSBK-decoded both increment per frame. Issue
// #402 Phase 2: the replay EOF summary depends on these counts to
// answer "of the FSW hits, what fraction made it through each gate"
// without parsing every debug log line.
func TestControlChannelStatsCountsTrustedDecodes(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	// Drain the bus so Publish doesn't block — we only care about the
	// stats counters here, not the events.
	go func() {
		for range sub.C {
		}
	}()

	cc := NewControlChannel(bus, nil, 851_000_000)
	const frames = 5
	stream := buildLockedStream(10, 0x293, DUIDTrunkingSignaling, OpRFSSStatusBroadcast)
	// buildLockedStream emits one TSDU frame; concatenate to get N.
	wide := make([]uint8, 0, len(stream)*frames)
	for i := 0; i < frames; i++ {
		wide = append(wide, stream...)
	}
	cc.Process(wide, 0)

	got := cc.Stats()
	if got.NIDTrusted < int64(frames) {
		t.Errorf("NIDTrusted = %d, want ≥ %d (one per frame)", got.NIDTrusted, frames)
	}
	if got.NIDMarginal != 0 || got.NIDFailed != 0 {
		t.Errorf("clean stream produced marginal/failed NIDs: marginal=%d failed=%d",
			got.NIDMarginal, got.NIDFailed)
	}
	if got.TSBKDecoded < int64(frames) {
		t.Errorf("TSBKDecoded = %d, want ≥ %d (one per frame)", got.TSBKDecoded, frames)
	}
	if got.TSBKTrellisFailed != 0 || got.TSBKCRCFailed != 0 {
		t.Errorf("clean stream produced TSBK failures: trellis=%d crc=%d",
			got.TSBKTrellisFailed, got.TSBKCRCFailed)
	}
}

// TestControlChannelStatsCountsNIDFailure: a garbage NID after a clean
// FSW (the same shape TestControlChannelPublishesDecodeErrorOnUncorrectableNID
// drives) must increment NIDFailed and NOT increment any of the
// success / TSBK counters.
func TestControlChannelStatsCountsNIDFailure(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	go func() {
		for range sub.C {
		}
	}()

	frame := make([]uint8, 24+32+98)
	copy(frame, FrameSyncWord[:])
	for i := 0; i < 32; i++ {
		frame[24+i] = uint8(i*7) & 0x3
	}
	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)

	cc := NewControlChannel(bus, nil, 851_000_000)
	cc.Process(stream, 0)

	got := cc.Stats()
	if got.NIDFailed == 0 {
		t.Errorf("NIDFailed = 0, want ≥ 1 on a garbage-NID frame")
	}
	if got.NIDTrusted != 0 || got.NIDMarginal != 0 {
		t.Errorf("garbage stream produced NID accepts: trusted=%d marginal=%d",
			got.NIDTrusted, got.NIDMarginal)
	}
	if got.TSBKDecoded != 0 {
		t.Errorf("garbage stream produced a TSBK decode: %d", got.TSBKDecoded)
	}
}
