package tetra

import (
	"reflect"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// idealDiffsFromDibits synthesizes the noiseless (sigma 0) per-symbol π/4-DQPSK
// differentials parallel to a dibit stream, using the same rotation-0 polarity the
// soft decoder reads (Im→b1, Re→b2, +⇒bit 0 — see dmoNoisyDNB). Feeding these to the
// streaming extractor exercises the soft TCH/S path deterministically.
func idealDiffsFromDibits(dibits []uint8) []complex64 {
	bits := TetraDibitsToBits(dibits) // 2 bits per dibit, [b1(Im), b2(Re)]
	amp := func(b byte) float32 {
		if b&1 == 0 {
			return 1 // bit 0 ⇒ +
		}
		return -1 // bit 1 ⇒ -
	}
	diffs := make([]complex64, len(dibits))
	for i := range dibits {
		diffs[i] = complex(amp(bits[2*i+1]), amp(bits[2*i])) // (Re=b2, Im=b1)
	}
	return diffs
}

// buildDMOStream lays out a realistic multi-burst DMO transmission: a leading DSB
// (SCH/S + SCH/H) followed by several TCH/S DNBs, separated by inter-burst filler,
// so the streaming extractor is exercised over bursts at a range of absolute
// offsets. Returns the dibit stream plus the encoded speech frames per DNB so the
// caller can assert decode fidelity end to end.
func buildDMOStream(t *testing.T, colour uint32, nDNB int) (dibits []uint8, wantFrames [][2][]byte) {
	t.Helper()
	// DSB.
	schs := EncodeBSCH(seqBits(1, 60))
	schh := EncodeSCHHD(seqBits(2, 124), 0)
	dibits = append(dibits, ramp(5, 50)...) // leading noise/preamble before the first burst
	dibits = append(dibits, buildDSB(schs, schh)...)

	for i := 0; i < nDNB; i++ {
		frameA := seqBits(7+i, tchSpeechFrameBits)
		frameB := seqBits(9+i, tchSpeechFrameBits)
		type4 := framing.UnpackBitsMSB(EncodeTCHS(frameA, frameB), tchType3Bits)
		onair := framing.ScrambleTetra(type4, colour)
		block1 := onair[:dmBlockDibits*2]
		block2 := onair[dmBlockDibits*2:]
		dibits = append(dibits, ramp(11+i, 30)...) // inter-burst filler
		dibits = append(dibits, buildDNB(block1, block2)...)
		wantFrames = append(wantFrames, [2][]byte{framing.PackBitsMSB(frameA), framing.PackBitsMSB(frameB)})
	}
	dibits = append(dibits, ramp(99, 400)...) // trailing filler so the last burst is fully inside the window
	return dibits, wantFrames
}

// TestDMStreamExtractorMatchesWholeSlice feeds a multi-burst DMO stream through the
// streaming extractor in small chunks and asserts it emits exactly the bursts a
// single whole-slice ExtractDMBursts finds — same leads, same kinds, in ascending
// lead order, with no duplicates. This is the streaming adapter's contract: the
// sliding window must not drop, duplicate, or reorder bursts relative to the
// offline path the replay harness validates.
func TestDMStreamExtractorMatchesWholeSlice(t *testing.T) {
	dibits, _ := buildDMOStream(t, 0, 4)

	// Reference: whole-slice extraction (what the offline replay harness runs).
	ref := ExtractDMBursts(dibits, 0)
	wantLeads := map[int]DMBurstKind{}
	for _, b := range ref {
		wantLeads[b.Lead] = b.Kind
	}
	if len(wantLeads) < 5 { // 1 DSB + 4 DNB
		t.Fatalf("reference extraction found only %d distinct bursts, want >=5", len(wantLeads))
	}

	for _, chunk := range []int{1, 7, 64, 250, 4096} {
		var got []DMBurst
		e := NewDMStreamExtractor(func(b DMBurst) { got = append(got, b) })
		for i := 0; i < len(dibits); i += chunk {
			end := i + chunk
			if end > len(dibits) {
				end = len(dibits)
			}
			// Hard stream (nil diffs) for this contract test.
			e.Process(dibits[i:end], nil, i)
		}
		e.Flush()

		// Ordering + no duplicates.
		seen := map[int]struct{}{}
		lastLead := -1
		for _, b := range got {
			if _, dup := seen[b.Lead]; dup {
				t.Errorf("chunk=%d: burst lead %d emitted twice", chunk, b.Lead)
			}
			seen[b.Lead] = struct{}{}
			if b.Lead <= lastLead {
				t.Errorf("chunk=%d: bursts not in ascending lead order (%d after %d)", chunk, b.Lead, lastLead)
			}
			lastLead = b.Lead
			if k, ok := wantLeads[b.Lead]; !ok {
				t.Errorf("chunk=%d: emitted a burst at lead %d not in the reference set", chunk, b.Lead)
			} else if k != b.Kind {
				t.Errorf("chunk=%d: lead %d kind=%v, reference kind=%v", chunk, b.Lead, b.Kind, k)
			}
		}
		// Every reference burst must be emitted.
		for lead := range wantLeads {
			if _, ok := seen[lead]; !ok {
				t.Errorf("chunk=%d: reference burst at lead %d was never emitted", chunk, lead)
			}
		}
	}
}

// TestDMStreamExtractorSoftDecodes checks the streaming path carries the soft
// differentials through to a CRC-valid TCH/S soft decode: a DNB fed with its
// parallel differentials must yield the two speech frames DMBurstTCHSpeechSoft
// recovers, matching what was encoded.
func TestDMStreamExtractorSoftDecodes(t *testing.T) {
	const colour = 0x0AB1F
	dibits, wantFrames := buildDMOStream(t, colour, 3)
	// Synthesize a parallel soft-differential stream from the ideal dibits so the
	// soft extraction path is exercised (the receiver would supply real
	// differentials; here the ideal π/4 phasors make the soft decode deterministic).
	diffs := idealDiffsFromDibits(dibits)

	var speech [][]byte
	e := NewDMStreamExtractor(func(b DMBurst) {
		if b.Kind != DMBurstNormal {
			return
		}
		if fr := DMBurstTCHSpeechSoft(b, colour); len(fr) == 2 {
			speech = append(speech, fr...)
		} else if fr := DMBurstTCHSpeech(b, colour); len(fr) == 2 {
			speech = append(speech, fr...)
		}
	})
	const chunk = 512
	for i := 0; i < len(dibits); i += chunk {
		end := i + chunk
		if end > len(dibits) {
			end = len(dibits)
		}
		e.Process(dibits[i:end], diffs[i:end], i)
	}
	e.Flush()

	if len(speech) != 2*len(wantFrames) {
		t.Fatalf("recovered %d speech frames, want %d", len(speech), 2*len(wantFrames))
	}
	for i, want := range wantFrames {
		if !reflect.DeepEqual(speech[2*i], want[0]) || !reflect.DeepEqual(speech[2*i+1], want[1]) {
			t.Errorf("DNB %d: speech frame mismatch", i)
		}
	}
}
