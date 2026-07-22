package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// TestMACDescrambleChannelDomain is the issue-#915 "Finding B" regression:
// P25 Phase 2 PN44 scrambling (TIA-102.BBAC-1 §7.2.5) wraps the CODED channel
// bits between demodulation and FEC, so the descramble must run on the raw
// channel dibits BEFORE the trellis/deinterleave — not on the 144 information
// bits recovered AFTER the trellis, the way GopherTrunk did before this fix.
// The trellis code does not commute with the XOR, so descrambling in the info
// domain can never satisfy the outer RS(24,16,9) check on a genuinely
// scrambled burst: real Victorian MMR (WACN 0xBEE00) Phase 2 traffic decoded
// with mac_rs_valid=0 for every seed, which is exactly what this test pins.
//
// The fixture is built with the reporter's confirmed MMR identity (WACN
// 0xBEE00, System ID 0x164, NAC 0x161) and a channel-scrambled TrellisOn MAC
// sub-frame carrying a GROUP_VOICE_CHANNEL_USER with a known source RID.
//
// This is a synthetic round-trip (GopherTrunk is receive-only, and the
// reporter's raw IQ capture is not reachable from CI): it proves the
// descramble DOMAIN is correct and self-consistent with the transmit side.
// The exact per-slot channel offset onto real air is confirmed separately by
// the reporter's live mac_rs_valid metric.
func TestMACDescrambleChannelDomain(t *testing.T) {
	const (
		wacn    = 0xBEE00
		sysID   = 0x164
		nac     = 0x161
		slot    = 0
		wantSrc = 0x00ABCD
		wantTG  = 32202 // TG 32202 / "NE DISP 2", the reporter's capture channel.
	)
	seed := framing.PN44SeedFromIdentity(wacn, sysID, nac)
	offset := slotChannelPN44Offset(slot)

	user := GroupVoiceChannelUser{ServiceOptions: 0x40, GroupAddress: wantTG, SourceID: wantSrc}
	pdu := EncodeMACPDURS(EncodeGroupVoiceChannelUser(user, false))

	// A realistically channel-scrambled MAC sub-frame, then the exact 146-dibit
	// channel window the decoder lifts out of it.
	sub := EncodeMACSubframeScrambled(SlotTypeMACSignaling, uint8(slot), pdu,
		TrellisOn, InterleaveOff, slot, seed)
	macDibits := sub[MACPayloadOffset : MACPayloadOffset+macPDUDibitsTrellis]

	// 1. The fix: channel-domain descramble recovers the PDU and passes RS.
	got, ok := decodeMACPDUDibits(macDibits, TrellisOn, RSOn, InterleaveOff,
		ScramblerOn, seed, offset)
	if !ok {
		t.Fatal("channel-domain descramble failed to decode the scrambled MAC PDU")
	}
	u, ok := got.AsGroupVoiceChannelUser()
	if !ok || u.SourceID != wantSrc {
		t.Fatalf("recovered PDU = %+v ok=%v, want SourceID=%#x", u, ok, wantSrc)
	}

	// 2. The fixture really is scrambled: decoding with the scrambler off does
	// not satisfy RS, so passing in step 1 means the descramble did the work.
	if _, ok := decodeMACPDUDibits(macDibits, TrellisOn, RSOn, InterleaveOff,
		ScramblerOff, seed, offset); ok {
		t.Fatal("scrambled burst decoded with the scrambler OFF; fixture is not actually scrambled")
	}

	// 3. Domain guard (the regression itself): replicate the pre-#915 path —
	// trellis-decode first, then descramble the INFO bits — and confirm it can
	// NOT recover the PDU. This is the mac_rs_valid=0 failure the fix corrects;
	// if a future change moved the descramble back after the trellis, this
	// assertion would fail.
	infoDibits, _ := framing.DecodeP25Trellis(macDibits)
	infoBits := framing.DibitsToBits(infoDibits)
	descrambleAtOffset(infoBits, seed, offset)
	if info := framing.PackBitsMSB(infoBits); len(info) >= 18 && verifyMACPDURS(info[:18]) {
		t.Fatal("info-domain (post-trellis) descramble unexpectedly passed RS; the domain fix is untested")
	}

	// 4. A wrong seed must not spuriously validate.
	if _, ok := decodeMACPDUDibits(macDibits, TrellisOn, RSOn, InterleaveOff,
		ScramblerOn, seed^0xFFF, offset); ok {
		t.Fatal("channel-domain descramble validated under the wrong seed")
	}

	// 5. ScramblerProbe self-aligns: handed offset 0, the RS-gated sweep finds
	// the true channel phase on its own — the path a receiver uses when its
	// slot-to-sequence offset is not yet pinned.
	probed, ok := decodeMACPDUDibits(macDibits, TrellisOn, RSOn, InterleaveOff,
		ScramblerProbe, seed, 0)
	if !ok {
		t.Fatal("ScramblerProbe failed to self-align to the channel offset")
	}
	if pu, ok := probed.AsGroupVoiceChannelUser(); !ok || pu.SourceID != wantSrc {
		t.Fatalf("probe recovered wrong PDU: %+v ok=%v, want SourceID=%#x", pu, ok, wantSrc)
	}
}

// TestDecodeSuperframeMACPDUsChannelScrambled drives the whole
// superframe-structured path (slice → ISCH slot-type → channel descramble →
// FEC → RS) over a scrambled superframe, confirming the ControlChannel's own
// MACDecodeConfig (ScramblerOn + identity seed) recovers a source RID from a
// channel-scrambled MAC sub-frame end to end (issue #915).
func TestDecodeSuperframeMACPDUsChannelScrambled(t *testing.T) {
	const (
		wacn    = 0xBEE00
		sysID   = 0x164
		nac     = 0x161
		wantSrc = 315203
	)
	seed := framing.PN44SeedFromIdentity(wacn, sysID, nac)
	user := GroupVoiceChannelUser{ServiceOptions: 0x40, GroupAddress: 0x4EEA, SourceID: wantSrc}
	pdu := EncodeMACPDURS(EncodeGroupVoiceChannelUser(user, false))

	var subs [SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = EncodeMACSubframeScrambled(SlotTypeMACSignaling, uint8(i), pdu,
				TrellisOn, InterleaveOff, i, seed)
		} else {
			subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i), voicePayloads(Voice4VFrameCount))
		}
	}
	cfg := MACDecodeConfig{Trellis: TrellisOn, Scrambler: ScramblerOn, Seed: seed}
	got := DecodeSuperframeMACPDUsWithSlot(decodeOneSuperframe(t, subs), cfg)
	if len(got) != 1 {
		t.Fatalf("DecodeSuperframeMACPDUsWithSlot returned %d PDUs, want 1", len(got))
	}
	if !got[0].RSValid {
		t.Error("channel-scrambled source PDU decoded with RSValid=false")
	}
	u, ok := got[0].PDU.AsGroupVoiceChannelUser()
	if !ok || u.SourceID != wantSrc {
		t.Errorf("recovered PDU = %+v ok=%v, want SourceID=%d", u, ok, wantSrc)
	}
}
