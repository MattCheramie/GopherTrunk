package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

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
	pdu := EncodeGroupVoiceChannelUser(user, false)

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
