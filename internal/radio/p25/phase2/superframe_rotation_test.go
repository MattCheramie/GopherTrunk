package phase2

import "testing"

// TestSuperframeDecoderLocksUnderDibitRotation is the issue-#915 regression:
// real-air P25 Phase 2 is differentially decoded H-DQPSK, so a residual carrier
// offset near an odd multiple of ±1500 Hz (a quarter of the 6000-baud symbol
// rate) rotates every recovered dibit by a constant 0..3. Before the fix the
// SuperframeDecoder matched the outbound sync under a single rotation only, so
// any rotated stream failed to lock and no MAC PDU (hence no source RID) was
// ever recovered. The decoder now searches all four rotations and de-rotates
// the sliced superframe back to canonical, so a rotated stream locks and its
// MAC payload decodes byte-identically to the un-rotated one.
//
// This was confirmed against a real Victorian MMR (WACN 0xBEE00) Phase 2 voice
// capture: the demodulated dibit stream was rotated by +3, and the
// SuperframeDecoder recovered 0 superframes at rotation 0 but 437 once the
// stream was rotation-aligned.
func TestSuperframeDecoderLocksUnderDibitRotation(t *testing.T) {
	// A superframe carrying a known grant in sub-frame 0, voice elsewhere.
	grant := grantPDU(0x1234, 0x00ABCD, 0x1, 0x005)
	grant.Payload = append(grant.Payload, make([]byte, 17-len(grant.Payload))...)
	var subs [SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = EncodeMACSubframe(SlotTypeMACSignaling, uint8(i), grant,
				TrellisOn, InterleaveOff)
		} else {
			subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i),
				voicePayloads(Voice4VFrameCount))
		}
	}
	const warmup = 50
	base := append(make([]uint8, warmup), EncodeSuperframe(subs)...)

	cfg := MACDecodeConfig{Trellis: TrellisOn, RS: RSOff, Interleave: InterleaveOff, Scrambler: ScramblerOff}

	for rot := uint8(0); rot < 4; rot++ {
		stream := make([]uint8, len(base))
		for i, d := range base {
			stream[i] = (d + rot) & 3
		}
		got := NewSuperframeDecoder().Process(stream, 0)
		if len(got) != 1 {
			t.Fatalf("rot=%d: expected 1 superframe, got %d", rot, len(got))
		}
		// The MAC sub-frame must decode back to the original grant, proving the
		// decoder de-rotated the payload to canonical dibits.
		pdus := DecodeSuperframeMACPDUs(got[0], cfg)
		if len(pdus) == 0 {
			t.Fatalf("rot=%d: no MAC PDU decoded from the locked superframe", rot)
		}
		g, ok := pdus[0].AsGroupVoiceChannelGrant()
		if !ok {
			t.Fatalf("rot=%d: decoded PDU is not a group voice grant (opcode %v)", rot, pdus[0].Opcode)
		}
		if g.GroupAddress != 0x1234 || g.SourceID != 0x00ABCD {
			t.Errorf("rot=%d: grant = tg 0x%X src 0x%X, want tg 0x1234 src 0xABCD",
				rot, g.GroupAddress, g.SourceID)
		}
	}
}
