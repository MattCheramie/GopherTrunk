package phase2

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// physicalDibitRotation returns the dibit the receiver emits for canonical
// dibit c when a residual carrier rotates every differential by k·π/2. It is
// derived through the real slicer rather than assumed: the canonical on-air
// differential phases (sync.go) plus the rotation, through demod.DQPSK with
// the Phase 2 π/8 constellation offset, then the receiver's [0,1,3,2]
// canonicalisation. This is what makes rotationPerm a measurement of the
// demod rather than a restatement of the decoder's own assumption.
func physicalDibitRotation(c uint8, k int) uint8 {
	phase := [4]float64{math.Pi / 4, 3 * math.Pi / 4, -math.Pi / 4, -3 * math.Pi / 4}
	remap := [4]uint8{0, 1, 3, 2}
	rot := phase[c&3] + float64(k)*math.Pi/2
	d := demod.NewDQPSK()
	d.SetRotation(math.Pi / 8)
	s0 := complex64(complex(1, 0))
	s1 := complex64(complex(float32(math.Cos(rot)), float32(math.Sin(rot))))
	raw := d.Decode(nil, []complex64{s0, s1})
	return remap[raw[1]&3]
}

// TestRotationPermutationsMatchDecoder pins rotationPerm to what the demod
// actually produces. The canonical dibit convention is not in angular order,
// so "rotate by a quarter turn" is not "add one mod 4" — the previous
// detectors assumed it was and so could never match a genuinely rotated
// stream.
func TestRotationPermutationsMatchDecoder(t *testing.T) {
	for k := 0; k < 4; k++ {
		for c := uint8(0); c < 4; c++ {
			want := physicalDibitRotation(c, k)
			if got := rotationPerm[k][c]; got != want {
				t.Errorf("rotationPerm[%d][%d] = %d, demod produces %d", k, c, got, want)
			}
			if back := rotationInv[k][want]; back != c {
				t.Errorf("rotationInv[%d][%d] = %d, want %d", k, want, back, c)
			}
		}
	}
	// The old model, for the record: it agrees with the demod at k=0 only.
	for k := 1; k < 4; k++ {
		agree := true
		for c := uint8(0); c < 4; c++ {
			if (c+uint8(k))&3 != physicalDibitRotation(c, k) {
				agree = false
			}
		}
		if agree {
			t.Errorf("k=%d: (d+k)&3 unexpectedly matches the demod; the table derivation is wrong", k)
		}
	}
}

// TestSuperframeDecoderLocksUnderDibitRotation is the issue-#915 regression:
// real-air P25 Phase 2 is differentially decoded H-DQPSK, so a residual carrier
// offset near an odd multiple of ±1500 Hz (a quarter of the 6000-baud symbol
// rate) rotates every recovered differential by a constant k·π/2, which the
// slicer turns into a fixed permutation of the dibit alphabet. The
// SuperframeDecoder searches all four rotations and de-rotates the sliced
// superframe back to canonical, so a rotated stream locks and its MAC payload
// decodes byte-identically to the un-rotated one.
//
// The rotated stream is built with physicalDibitRotation — the real slicer —
// not with (d+k)&3. An earlier version of this test used the latter, which is
// not a rotation the demod can produce, so it exercised detectors that no
// real carrier offset would ever trigger.
func TestSuperframeDecoderLocksUnderDibitRotation(t *testing.T) {
	// A superframe carrying a known grant in sub-frame 0, voice elsewhere.
	grant := grantPDU(0x1234, 0x00ABCD, 0x1, 0x005)
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

	for k := 0; k < 4; k++ {
		stream := make([]uint8, len(base))
		for i, d := range base {
			stream[i] = physicalDibitRotation(d, k)
		}
		got := NewSuperframeDecoder().Process(stream, 0)
		if len(got) != 1 {
			t.Fatalf("k=%d: expected 1 superframe, got %d", k, len(got))
		}
		pdus := DecodeSuperframeMACPDUs(got[0], cfg)
		if len(pdus) == 0 {
			t.Fatalf("k=%d: no MAC PDU decoded from the locked superframe", k)
		}
		g, ok := pdus[0].AsGroupVoiceChannelGrant()
		if !ok {
			t.Fatalf("k=%d: decoded PDU is not a group voice grant (opcode %v)", k, pdus[0].Opcode)
		}
		if g.GroupAddress != 0x1234 || g.SourceID != 0x00ABCD {
			t.Errorf("k=%d: grant = tg 0x%X src 0x%X, want tg 0x1234 src 0xABCD",
				k, g.GroupAddress, g.SourceID)
		}
	}
}

// TestSuperframeDecoderDerotatesSoftConsistently checks the soft path: after
// de-rotation the soft differential must agree with the de-rotated hard dibit
// under the diagonal-frame convention (b0 = Re<0, b1 = Im<0, value = 2·b1+b0).
func TestSuperframeDecoderDerotatesSoftConsistently(t *testing.T) {
	grant := grantPDU(0x1234, 0x00ABCD, 0x1, 0x005)
	var subs [SubframesPerSuperframe][]uint8
	for i := range subs {
		if i == 0 {
			subs[i] = EncodeMACSubframe(SlotTypeMACSignaling, uint8(i), grant, TrellisOn, InterleaveOff)
		} else {
			subs[i] = EncodeVoiceSubframe(SlotTypeVoice4V, uint8(i), voicePayloads(Voice4VFrameCount))
		}
	}
	base := append(make([]uint8, 50), EncodeSuperframe(subs)...)

	// Ideal diagonal-frame soft sample for a canonical dibit: the on-air
	// differential phase rotated by +π/8 lands on a diagonal.
	softOf := func(c uint8, k int) complex64 {
		phase := [4]float64{math.Pi / 4, 3 * math.Pi / 4, -math.Pi / 4, -3 * math.Pi / 4}
		a := phase[c&3] + float64(k)*math.Pi/2 + math.Pi/8
		return complex(float32(math.Cos(a)), float32(math.Sin(a)))
	}
	for k := 0; k < 4; k++ {
		stream := make([]uint8, len(base))
		soft := make([]complex64, len(base))
		for i, d := range base {
			stream[i] = physicalDibitRotation(d, k)
			soft[i] = softOf(d, k)
		}
		got := NewSuperframeDecoder().ProcessSoft(stream, soft, 0)
		if len(got) != 1 {
			t.Fatalf("k=%d: expected 1 superframe, got %d", k, len(got))
		}
		for _, sub := range got[0].Subframes {
			if len(sub.Soft) != len(sub.Dibits) {
				t.Fatalf("k=%d: soft/dibit length mismatch", k)
			}
			for j, z := range sub.Soft {
				var b0, b1 uint8
				if real(z) < 0 {
					b0 = 1
				}
				if imag(z) < 0 {
					b1 = 1
				}
				if v := 2*b1 + b0; v != sub.Dibits[j] {
					t.Fatalf("k=%d sub %d dibit %d: soft decodes to %d, hard dibit is %d",
						k, sub.Index, j, v, sub.Dibits[j])
				}
			}
		}
	}
}
