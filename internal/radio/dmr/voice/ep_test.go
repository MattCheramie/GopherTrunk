package voice

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/crypto/rc4"
)

// The literal vectors below were produced by independent implementations of
// the references' stated algorithms (DSD-FME dmr_pi.c / dmr_le.c / dsd_mbe.c,
// SDRTrunk VoiceSuperFrameProcessor / Golay24) — not by this package. A
// round-trip through Go code alone would pass with a wrong construction on
// both sides; these cannot.

func TestAdvanceMIReferenceVectors(t *testing.T) {
	for _, c := range []struct{ in, want uint32 }{
		{0xDEADBEEF, 0xD65FAFEB},
		{0x00000001, 0x45145144},
		{0x12345678, 0xB468E067},
		{0xFFFFFFFF, 0xFFFFFFFF},
	} {
		if got := AdvanceMI(c.in); got != c.want {
			t.Errorf("AdvanceMI(%08X) = %08X, want %08X", c.in, got, c.want)
		}
	}
}

func TestCRC4ReferenceVectors(t *testing.T) {
	// DSD-FME's division-then-invert and SDRTrunk's 0xF-initial-fill
	// formulations both produce these.
	for _, c := range []struct {
		iv   uint32
		want uint8
	}{
		{0xDEADBEEF, 0x5}, {0x00000000, 0xF}, {0x12345678, 0x9}, {0x80000001, 0xA},
	} {
		if got := crc4(c.iv); got != c.want {
			t.Errorf("crc4(%08X) = %X, want %X", c.iv, got, c.want)
		}
	}
}

func TestEPKeystreamReferenceVector(t *testing.T) {
	key := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	ks, err := EPKeystream(key, 0xDEADBEEF, 14)
	if err != nil {
		t.Fatal(err)
	}
	// Independent RC4 (KSA/PRGA written from the algorithm), key‖MI, bytes
	// 256..269 of the keystream.
	if got := hex.EncodeToString(ks); got != "28e71dc649ae90d466864165d1c5" {
		t.Fatalf("keystream = %s", got)
	}
	// The same construction WITHOUT the 256-byte drop is a different stream;
	// a descrambler that skipped the drop would decode nothing.
	if got := hex.EncodeToString(ks); got == "4fa6162c45a5b6f2fa0479825a00" {
		t.Fatalf("keystream was not positioned past the warm-up")
	}
	if _, err := EPKeystream(nil, 1, 1); err == nil {
		t.Fatal("empty key accepted")
	}
}

func TestGolay2412ReferenceLiterals(t *testing.T) {
	for _, c := range []struct {
		data uint16
		cw   uint32
	}{
		{0xABC, 0xABC23C}, {0x000, 0x000000}, {0xFFF, 0xFFFFFF}, {0x5A5, 0x5A56E4},
	} {
		if got := golayEncode2412(c.data); got != c.cw {
			t.Errorf("golayEncode2412(%03X) = %06X, want %06X", c.data, got, c.cw)
		}
		if d, ok := golayDecode2412(c.cw); !ok || d != c.data {
			t.Errorf("golayDecode2412(%06X) = %03X/%v", c.cw, d, ok)
		}
	}
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 2000; trial++ {
		data := uint16(rng.Intn(4096))
		cw := golayEncode2412(data)
		nerr := 1 + rng.Intn(4) // 1..4 flipped bits
		flipped := cw
		var used [24]bool
		for n := 0; n < nerr; {
			p := rng.Intn(24)
			if used[p] {
				continue
			}
			used[p] = true
			flipped ^= 1 << uint(p)
			n++
		}
		d, ok := golayDecode2412(flipped)
		switch {
		case nerr <= 3 && (!ok || d != data):
			t.Fatalf("%d errors on %06X: got %03X/%v, want %03X", nerr, cw, d, ok, data)
		case nerr == 4 && ok:
			t.Fatalf("4 errors on %06X accepted as %03X", cw, d)
		}
	}
}

// The embedded-IV nibble positions: SDRTrunk names on-air bits 71/67/63/59
// of each frame; DSD-FME reads C3[0..3] after deinterleaving. Through this
// package's own encoder (mbelib/dsd tables) those must be the same bits —
// C3[j] is payload bit 48-j.
func TestEmbeddedIVNibbleMatchesBothReferences(t *testing.T) {
	for nib := 0; nib < 16; nib++ {
		info := make([]byte, ambeInfoBits)
		info[48] = byte(nib >> 3 & 1) // C3[0]
		info[47] = byte(nib >> 2 & 1) // C3[1]
		info[46] = byte(nib >> 1 & 1) // C3[2]
		info[45] = byte(nib & 1)      // C3[3]
		frame, err := EncodeAMBEFrame(info)
		if err != nil {
			t.Fatal(err)
		}
		if got := ivNibble(frame); got != uint8(nib) {
			t.Fatalf("nibble %X read back as %X", nib, got)
		}
	}
}

func TestExtractEmbeddedIVRoundTripAndRejects(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	mk := func() [FramesPerSuperframe][]byte {
		var frames [FramesPerSuperframe][]byte
		for i := range frames {
			info := make([]byte, ambeInfoBits)
			for b := range info {
				info[b] = byte(rng.Intn(2))
			}
			f, _ := EncodeAMBEFrame(info)
			frames[i] = f
		}
		return frames
	}
	for _, iv := range []uint32{0xDEADBEEF, 0, 0xFFFFFFFF, 0x12345678} {
		frames := mk()
		EmbedIV(frames, iv)
		got, corrected, ok := ExtractEmbeddedIV(frames)
		if !ok || got != iv || corrected != 0 {
			t.Fatalf("iv %08X: got %08X/%d/%v", iv, got, corrected, ok)
		}
		// Up to three damaged nibble bits per codeword are corrected.
		frames[0][71] ^= 1
		frames[4][67] ^= 1
		frames[9][63] ^= 1
		if got, corrected, ok := ExtractEmbeddedIV(frames); !ok || got != iv || corrected != 3 {
			t.Fatalf("iv %08X with 3 bit errors: got %08X/%d/%v", iv, got, corrected, ok)
		}
	}
	// Vocoder data in C3 (a clear call) verifies by chance about 1% of the
	// time (radius-3 Golay spheres cover ~57% of the space, cubed, times the
	// CRC-4) — measured, and the reason the voice chain gates the embedded
	// IV on other evidence of encryption. Pin the order of magnitude so a
	// regression that loosens the gate (e.g. dropping the overall-parity
	// check or the CRC) is caught.
	accepted := 0
	for trial := 0; trial < 2000; trial++ {
		if _, _, ok := ExtractEmbeddedIV(mk()); ok {
			accepted++
		}
	}
	if accepted > 60 {
		t.Fatalf("%d/2000 random superframes verified as carrying an IV (expect ~1%%)", accepted)
	}
}

func TestDescrambleSuperframeRoundTrip(t *testing.T) {
	key := []byte{0xA1, 0xB2, 0xC3, 0xD4, 0xE5}
	const mi = 0x0BADF00D
	rng := rand.New(rand.NewSource(11))
	clear := make([][]byte, FramesPerSuperframe)
	for i := range clear {
		clear[i] = make([]byte, ambeInfoBits)
		for b := range clear[i] {
			clear[i][b] = byte(rng.Intn(2))
		}
	}
	// Frame 5 is the silence vector (transmitted in clear on air).
	copy(clear[5], ambeSilence[:])

	// Scramble with the construction spelled out directly — RC4(key‖MI),
	// skip 256, 7 bytes per frame — rather than through the code under test.
	c, _ := rc4.NewCipher(append(append([]byte{}, key...), 0x0B, 0xAD, 0xF0, 0x0D))
	c.KeyStream(256)
	ks := c.KeyStream(EPSuperframeBytes)
	scrambled := make([][]byte, FramesPerSuperframe)
	for i := range clear {
		scrambled[i] = append([]byte(nil), clear[i]...)
		if i == 5 {
			continue
		}
		for b := 0; b < ambeInfoBits; b++ {
			scrambled[i][b] ^= (ks[i*7+b/8] >> uint(7-b%8)) & 1
		}
	}
	// A frame that failed FEC is nil; the later frames must still line up.
	scrambled[2] = nil
	n, silence, err := DescrambleSuperframe(key, mi, scrambled)
	if err != nil {
		t.Fatal(err)
	}
	if n != FramesPerSuperframe-2 || silence != 1 {
		t.Fatalf("descrambled=%d silence=%d, want 16/1", n, silence)
	}
	for i := range clear {
		if i == 2 {
			continue
		}
		for b := range clear[i] {
			if scrambled[i][b] != clear[i][b] {
				t.Fatalf("frame %d bit %d differs after descramble", i, b)
			}
		}
	}
	// The wrong MI yields garbage — the chain must never call this a success.
	again := append([]byte(nil), clear[0]...)
	for b := 0; b < ambeInfoBits; b++ {
		again[b] ^= (ks[b/8] >> uint(7-b%8)) & 1
	}
	if _, _, err := DescrambleSuperframe(key, mi+1, [][]byte{again}); err != nil {
		t.Fatal(err)
	}
	same := 0
	for b := range again {
		if again[b] == clear[0][b] {
			same++
		}
	}
	if same == ambeInfoBits {
		t.Fatal("wrong MI reproduced the clear frame")
	}
}

func TestEPTrackerFollowsChain(t *testing.T) {
	var tr EPTracker
	if _, ok := tr.Next(0, false, 0); ok {
		t.Fatal("unknown tracker produced an MI")
	}
	tr.SetHeaderMI(0xDEADBEEF)
	mi, ok := tr.Next(0, false, 0) // embedded IV damaged: prediction carries
	if !ok || mi != 0xDEADBEEF || tr.Predicted != 1 {
		t.Fatalf("first superframe: %08X/%v predicted=%d", mi, ok, tr.Predicted)
	}
	mi, ok = tr.Next(0xD65FAFEB, true, 2) // embedded IV agrees with the LFSR (corrections are fine)
	if !ok || mi != 0xD65FAFEB || tr.Mismatches != 0 {
		t.Fatalf("second superframe: %08X/%v mismatches=%d", mi, ok, tr.Mismatches)
	}
	// A corrected IV that disagrees is NOT trusted over the chain.
	mi, ok = tr.Next(0x12345678, true, 1)
	if !ok || mi != 0x8DFFBAFA || tr.Mismatches != 1 || tr.Predicted != 2 {
		t.Fatalf("corrected disagreeing IV: %08X/%v mismatches=%d predicted=%d", mi, ok, tr.Mismatches, tr.Predicted)
	}
	// A CLEAN verified IV that disagrees overrides the prediction.
	mi, ok = tr.Next(0x12345678, true, 0)
	if !ok || mi != 0x12345678 || tr.Mismatches != 2 {
		t.Fatalf("clean disagreeing IV: %08X/%v mismatches=%d", mi, ok, tr.Mismatches)
	}
	if mi, _ := tr.Next(0, false, 0); mi != 0xB468E067 {
		t.Fatalf("prediction after override = %08X, want B468E067", mi)
	}
	tr.Reset()
	if tr.Known() {
		t.Fatal("Reset left the tracker known")
	}
	// Late entry: no PI header, the first verified embedded IV seeds the chain.
	if mi, ok := tr.Next(0x00000001, true, 3); !ok || mi != 1 {
		t.Fatalf("late entry: %08X/%v", mi, ok)
	}
	if mi, _ := tr.Next(0, false, 0); mi != 0x45145144 {
		t.Fatalf("late-entry prediction = %08X", mi)
	}
}
