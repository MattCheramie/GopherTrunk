package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// scrambleMACRegionAt scrambles the MAC payload region of a MAC sub-frame in
// place at an arbitrary channel-bit phase, standing in for a real downlink
// whose intra-slot PN44 offset differs from GopherTrunk's superframe-grid
// assumption (slotChannelPN44Offset). It mirrors EncodeMACSubframeScrambled
// but takes the phase explicitly. XOR is symmetric, so descrambleAtOffset
// scrambles here.
func scrambleMACRegionAt(sub []uint8, seed uint64, phase int) {
	region := sub[MACPayloadOffset : MACPayloadOffset+macPDUDibitsTrellis]
	bits := framing.DibitsToBits(region)
	descrambleAtOffset(bits, seed, phase)
	copy(region, framing.BitsToDibits(bits))
}

// TestScramblerProbePinsUnknownIntraSlotOffset is the issue-#915 regression
// for the self-aligning probe (the second half of Finding B). A real
// Victorian MMR (WACN 0xBEE00) downlink scrambles the MAC channel bits at
// slot·360 + 20 (the reference decoders' intra-slot offset), but
// GopherTrunk's grid assumes slot·360 + 64 (MACPayloadOffset·2). The
// slot-base-only probe (DecodeSuperframeMACPDUsWithSlot) holds that +64 fixed
// while it sweeps the 12 slot bases, so it can never descramble a channel
// whose true intra-slot offset is +20 — which is the mac_rs_valid=0 the
// reporter measured. The pinned probe sweeps the full sequence phase,
// discovers +20 from the RS gate, and pins it for the rest of the call.
//
// This is a synthetic round-trip (GopherTrunk is receive-only and the
// reporter's raw IQ capture is not reachable from this build environment): it
// proves the probe now resolves the intra-slot phase degree of freedom that
// the fixed-offset descramble cannot. The exact real-air offset is confirmed
// separately by the reporter's live mac_rs_valid metric, per the
// verify-before-close policy.
func TestScramblerProbePinsUnknownIntraSlotOffset(t *testing.T) {
	const (
		wacn      = 0xBEE00
		sysID     = 0x164
		nac       = 0x161
		wantSrc   = 0x0A1B2C
		realIntra = 20 // reference intra-slot channel-bit offset, != MACPayloadOffset*2
	)
	if MACPayloadOffset*2 == realIntra {
		t.Fatal("test premise broken: realIntra must differ from MACPayloadOffset*2 so the fixed offset is wrong")
	}
	seed := framing.PN44SeedFromIdentity(wacn, sysID, nac)
	user := GroupVoiceChannelUser{ServiceOptions: 0x40, GroupAddress: 32202, SourceID: wantSrc}
	pdu := EncodeMACPDURS(EncodeGroupVoiceChannelUser(user, false))

	// A MAC sub-frame for the given slot, scrambled at the *reference*
	// intra-slot offset (slot·360 + realIntra), not GopherTrunk's assumed one.
	build := func(slot int) []uint8 {
		sub := EncodeMACSubframe(SlotTypeMACSignaling, uint8(slot), pdu, TrellisOn, InterleaveOff)
		scrambleMACRegionAt(sub, seed, slot*DibitsPerSubframe*2+realIntra)
		return sub
	}

	var subs [SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = build(0)
		} else {
			subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i), voicePayloads(Voice4VFrameCount))
		}
	}
	sf := decodeOneSuperframe(t, subs)
	// ScramblerProbe gates on RS to pick the true phase, so it is used with
	// RSOn (the composer warns when it is not); with RSOn the slot-base probe
	// does its real RS-gated sweep instead of degrading to a permissive parse.
	cfg := MACDecodeConfig{Trellis: TrellisOn, RS: RSOn, Scrambler: ScramblerProbe, Seed: seed}

	// Fails-first: the slot-base-only probe (fixed +64 intra-slot) is blind to
	// the intra-slot degree of freedom and recovers nothing.
	if got := DecodeSuperframeMACPDUsWithSlot(sf, cfg); len(got) != 0 {
		t.Fatalf("slot-base probe unexpectedly decoded %d PDUs; it should be blind to the intra-slot offset", len(got))
	}

	// The fix: the pinned full-phase probe self-aligns to +20.
	pin := &ScramblerPin{}
	got := DecodeSuperframeMACPDUsWithSlotPinned(sf, cfg, pin)
	if len(got) != 1 {
		t.Fatalf("pinned probe decoded %d PDUs, want 1", len(got))
	}
	if !got[0].RSValid {
		t.Error("pinned probe returned a non-RS-valid PDU; probe must only surface RS-verified PDUs")
	}
	u, ok := got[0].PDU.AsGroupVoiceChannelUser()
	if !ok || u.SourceID != wantSrc {
		t.Fatalf("recovered PDU = %+v ok=%v, want SourceID=%#x", u, ok, wantSrc)
	}
	if !pin.have {
		t.Fatal("probe did not pin the discovered channel phase")
	}

	// Cross-slot generalisation: a later sub-frame on a *different* slot with
	// the same constant offset decodes via the pinned phase — the pin applies
	// slot·360 + phase across the superframe's 12 slots.
	var subs2 [SubframesPerSuperframe][]uint8
	for i := range subs2 {
		if i == 4 {
			subs2[i] = build(4)
		} else {
			subs2[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i), voicePayloads(Voice4VFrameCount))
		}
	}
	got2 := DecodeSuperframeMACPDUsWithSlotPinned(decodeOneSuperframe(t, subs2), cfg, pin)
	if len(got2) != 1 || !got2[0].RSValid {
		t.Fatalf("pinned probe failed to decode a different slot via the pinned phase: got %d PDUs", len(got2))
	}
	if u2, ok := got2[0].PDU.AsGroupVoiceChannelUser(); !ok || u2.SourceID != wantSrc {
		t.Fatalf("cross-slot recovered PDU = %+v ok=%v, want SourceID=%#x", u2, ok, wantSrc)
	}
}
