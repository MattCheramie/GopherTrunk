package acelp

import (
	"encoding/binary"
	"os"
	"testing"
)

// TestETSIReferenceConformance is the authoritative bit-exact conformance check
// for the clean-room TETRA ACELP decoder against the ETSI EN 300 395-2 reference
// C codec. It is skipped unless GT_ETSI_SERIAL and GT_ETSI_REF point at files
// produced by the reference tools:
//
//	scoder   in.pcm     serial.bin  synth_local.pcm   # ETSI reference encoder
//	sdecoder serial.bin ref_out.pcm                   # ETSI reference decoder
//
// GT_ETSI_SERIAL = serial.bin : shared bitstream, int16 LE, 138 words/frame
//
//	(word 0 = BFI, words 1..137 = the 137 coded speech bits 0/1) — exactly the
//	format ETSI sdecoder reads (Bits2prm_Tetra) and GT's Decoder.Decode consumes.
//
// GT_ETSI_REF = ref_out.pcm : reference decoder output, int16 LE, 240 samples/
//
//	frame, INCLUDING the reference Post_Process (saturating x2 after Decod_Tetra).
//	GT omits that x2, so the harness applies postProcessX2 to GT's output first.
//
// The decoder is fixed-point integer maths, so a faithful port matches the
// reference sample-for-sample (0 mismatches). The ETSI sources/vectors are
// copyrighted and are NOT committed.
//
// Reference build note (64-bit hosts): the ETSI basic-op saturation assumes a
// 32-bit Word32 (`typedef long`, 32-bit under ILP32). On LP64 `long` is 64-bit
// and every saturating op returns garbage — build with Word32 as 32-bit int
// (edit the typedef, or -m32).
func TestETSIReferenceConformance(t *testing.T) {
	serialPath := os.Getenv("GT_ETSI_SERIAL")
	refPath := os.Getenv("GT_ETSI_REF")
	if serialPath == "" || refPath == "" {
		t.Skip("set GT_ETSI_SERIAL and GT_ETSI_REF to the ETSI reference vectors to run the conformance check")
	}

	serial := readI16LE(t, serialPath)
	ref := readI16LE(t, refPath)

	const serialPerFrame = 138 // 1 BFI + 137 bits
	const pcmPerFrame = 240
	if len(serial)%serialPerFrame != 0 {
		t.Fatalf("serial length %d is not a multiple of %d", len(serial), serialPerFrame)
	}
	nFrames := len(serial) / serialPerFrame
	if got := len(ref) / pcmPerFrame; got != nFrames {
		t.Fatalf("frame count mismatch: serial has %d frames, ref has %d", nFrames, got)
	}

	dec := NewDecoder()
	bits := make([]byte, speechFrameBits)

	mismatches, maxAbs, bfiFrames := 0, 0, 0
	firstBadFrame := -1
	for f := 0; f < nFrames; f++ {
		base := f * serialPerFrame
		bfi := serial[base] != 0
		if bfi {
			bfiFrames++
		}
		for i := 0; i < speechFrameBits; i++ {
			bits[i] = byte(serial[base+1+i] & 1)
		}
		out := dec.Decode(bits, bfi)
		if len(out) != pcmPerFrame {
			t.Fatalf("frame %d: decoder returned %d samples, want %d", f, len(out), pcmPerFrame)
		}
		for i, s := range out {
			got := postProcessX2(s)
			want := ref[f*pcmPerFrame+i]
			if got != want {
				mismatches++
				if d := absDiff(int(got), int(want)); d > maxAbs {
					maxAbs = d
				}
				if firstBadFrame < 0 {
					firstBadFrame = f
				}
			}
		}
	}

	t.Logf("frames=%d (bfi=%d) samples=%d mismatches=%d maxAbsDiff=%d firstBadFrame=%d",
		nFrames, bfiFrames, nFrames*pcmPerFrame, mismatches, maxAbs, firstBadFrame)
	if mismatches != 0 {
		t.Errorf("ACELP decode diverges from the ETSI reference: %d/%d samples differ (max abs %d, first at frame %d)",
			mismatches, nFrames*pcmPerFrame, maxAbs, firstBadFrame)
	}
}

// postProcessX2 mirrors the reference Post_Process: a saturating multiply-by-two
// (add(x, x)) applied to each output sample after Decod_Tetra.
func postProcessX2(s int16) int16 {
	v := int32(s) * 2
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return int16(v)
}

func absDiff(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}

func readI16LE(t *testing.T, path string) []int16 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw)%2 != 0 {
		t.Fatalf("%s: odd byte length %d", path, len(raw))
	}
	out := make([]int16, len(raw)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(raw[i*2:]))
	}
	return out
}
