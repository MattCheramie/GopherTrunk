package voice

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
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

func TestRewindMIInvertsAdvance(t *testing.T) {
	for _, mi := range []uint32{0, 1, 0xDEADBEEF, 0x3B77735F, 0xFFFFFFFF, 0x80000000} {
		if got := RewindMI(AdvanceMI(mi)); got != mi {
			t.Errorf("RewindMI(AdvanceMI(%08X)) = %08X", mi, got)
		}
		if got := AdvanceMI(RewindMI(mi)); got != mi {
			t.Errorf("AdvanceMI(RewindMI(%08X)) = %08X", mi, got)
		}
	}
}

// The embedded IV a superframe carries is the NEXT superframe's MI (see the
// package comment; TestEPCaptureIssue1187 pins it on air). The tracker must
// therefore decode the current superframe on what it already knew and use
// the IV only to steer the following one.
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
	// Second superframe: its MI is AdvanceMI(DEADBEEF) = D65FAFEB, and the IV
	// it embeds is the THIRD superframe's, AdvanceMI(D65FAFEB) = 8DFFBAFA
	// (corrections are fine when it agrees with the chain).
	mi, ok = tr.Next(0x8DFFBAFA, true, 2)
	if !ok || mi != 0xD65FAFEB || tr.Mismatches != 0 || tr.Predicted != 1 {
		t.Fatalf("second superframe: %08X/%v mismatches=%d predicted=%d", mi, ok, tr.Mismatches, tr.Predicted)
	}
	// Third: a corrected IV that disagrees is NOT trusted over the chain —
	// the current superframe decodes on the confirmed 8DFFBAFA and the
	// prediction carries.
	mi, ok = tr.Next(0x12345678, true, 1)
	if !ok || mi != 0x8DFFBAFA || tr.Mismatches != 1 || tr.Predicted != 2 {
		t.Fatalf("corrected disagreeing IV: %08X/%v mismatches=%d predicted=%d", mi, ok, tr.Mismatches, tr.Predicted)
	}
	// Fourth: a CLEAN verified IV that disagrees means the chain broke — the
	// current superframe is re-derived by rewinding it, and it steers the next.
	mi, ok = tr.Next(0x12345678, true, 0)
	if !ok || mi != RewindMI(0x12345678) || tr.Mismatches != 2 {
		t.Fatalf("clean disagreeing IV: %08X/%v mismatches=%d", mi, ok, tr.Mismatches)
	}
	if mi, _ := tr.Next(0, false, 0); mi != 0x12345678 {
		t.Fatalf("superframe after the override = %08X, want 12345678", mi)
	}
	tr.Reset()
	if tr.Known() {
		t.Fatal("Reset left the tracker known")
	}
	// Late entry: no PI header. The first verified embedded IV names the
	// NEXT superframe, so the one carrying it decodes on its rewind.
	if mi, ok := tr.Next(0x45145144, true, 3); !ok || mi != RewindMI(0x45145144) {
		t.Fatalf("late entry: %08X/%v", mi, ok)
	}
	if mi, _ := tr.Next(0, false, 0); mi != 0x45145144 {
		t.Fatalf("late-entry follow-on = %08X, want 45145144", mi)
	}
}

// epCaptureFixture is testdata/ep_issue1187_ptt1.json: the PI header and the
// first three voice superframes of one Enhanced Privacy transmission from the
// #1187 reporter's known-key capture, as sliced on air.
type epCaptureFixture struct {
	KeyHex   string `json:"key_hex"`
	PIHeader struct {
		MI string `json:"mi"`
	} `json:"pi_header"`
	Superframes []struct {
		EmbeddedIV string   `json:"embedded_iv"`
		OnAir      []string `json:"on_air"`
	} `json:"superframes"`
}

func loadEPCaptureFixture(t *testing.T) (key []byte, headerMI uint32, frames [][FramesPerSuperframe][]byte, ivs []uint32) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "ep_issue1187_ptt1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx epCaptureFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	key, err = hex.DecodeString(fx.KeyHex)
	if err != nil {
		t.Fatal(err)
	}
	mi, err := strconv.ParseUint(fx.PIHeader.MI, 16, 32)
	if err != nil {
		t.Fatal(err)
	}
	headerMI = uint32(mi)
	for _, sf := range fx.Superframes {
		var fr [FramesPerSuperframe][]byte
		for i, h := range sf.OnAir {
			b, err := hex.DecodeString(h)
			if err != nil {
				t.Fatal(err)
			}
			f := make([]byte, AMBEFrameBits)
			for k := range f {
				f[k] = (b[k>>3] >> uint(7-(k&7))) & 1
			}
			fr[i] = f
		}
		frames = append(frames, fr)
		iv, err := strconv.ParseUint(sf.EmbeddedIV, 16, 32)
		if err != nil {
			t.Fatal(err)
		}
		ivs = append(ivs, uint32(iv))
	}
	return key, headerMI, frames, ivs
}

// ambe2450B0 is the AMBE+2 3600x2450 fundamental index: payload bits 0..3
// and 37..39 (mbelib ambe3600x2450.c), NOT the first seven bits.
func ambe2450B0(f []byte) int {
	return int(f[0]&1)<<6 | int(f[1]&1)<<5 | int(f[2]&1)<<4 | int(f[3]&1)<<3 | int(f[37]&1)<<2 | int(f[38]&1)<<1 | int(f[39]&1)
}

// b0Continuity is the fraction of consecutive frames whose fundamental
// index moves by at most 10 (speech ≳ 0.5; random ≈ 0.16).
func b0Continuity(frames [][]byte) float64 {
	var pairs, close int
	prev := -1
	for _, f := range frames {
		if f == nil {
			continue
		}
		p := ambe2450B0(f)
		if p >= 120 {
			prev = -1
			continue
		}
		if prev >= 0 {
			pairs++
			if d := p - prev; d <= 10 && d >= -10 {
				close++
			}
		}
		prev = p
	}
	if pairs == 0 {
		return 0
	}
	return float64(close) / float64(pairs)
}

// TestEPCaptureIssue1187 is the on-air pin (#1187): literal frames from the
// reporter's known-key capture. It fixes two things a synthetic round-trip
// cannot: (1) each superframe's embedded IV equals AdvanceMI of the MI before
// it — the PI header's for the first — i.e. it names the NEXT superframe;
// (2) descrambling on that chain yields speech (b0 continuity), while
// applying each embedded IV to its OWN superframe (the previous tracker
// semantics) stays at ciphertext level.
func TestEPCaptureIssue1187(t *testing.T) {
	key, headerMI, frames, ivs := loadEPCaptureFixture(t)
	prev := headerMI
	for n, fr := range frames {
		iv, corrected, ok := ExtractEmbeddedIV(fr)
		if !ok || corrected != 0 || iv != ivs[n] {
			t.Fatalf("superframe %d embedded IV = %08X ok=%v corrected=%d, fixture %08X", n, iv, ok, corrected, ivs[n])
		}
		if want := AdvanceMI(prev); iv != want {
			t.Fatalf("superframe %d embedded IV %08X != AdvanceMI(previous %08X) = %08X", n, iv, prev, want)
		}
		prev = iv
	}
	decode := func(fr [FramesPerSuperframe][]byte) [][]byte {
		infos := make([][]byte, FramesPerSuperframe)
		for i := range fr {
			info, _, err := DecodeAMBEFrame(fr[i])
			if err != nil {
				t.Fatal(err)
			}
			infos[i] = info
		}
		return infos
	}
	// The production chain: PI header seeds the tracker, IVs steer it.
	var tr EPTracker
	tr.SetHeaderMI(headerMI)
	var chain [][]byte
	wantMI := []uint32{headerMI, ivs[0], ivs[1]}
	for n, fr := range frames {
		iv, corrected, ok := ExtractEmbeddedIV(fr)
		mi, known := tr.Next(iv, ok, corrected)
		if !known || mi != wantMI[n] {
			t.Fatalf("tracker MI for superframe %d = %08X (known=%v), want %08X", n, mi, known, wantMI[n])
		}
		infos := decode(fr)
		if _, _, err := DescrambleSuperframe(key, mi, infos); err != nil {
			t.Fatal(err)
		}
		chain = append(chain, infos...)
	}
	if c := b0Continuity(chain); c < 0.55 {
		t.Fatalf("descrambled on the header→IV chain: b0 continuity %.2f, want speech (≥ 0.55)", c)
	}
	// Late entry without the header: the first superframe decodes on the
	// rewind of its own IV, identically.
	var late EPTracker
	iv0, corrected0, ok0 := ExtractEmbeddedIV(frames[0])
	if mi, known := late.Next(iv0, ok0, corrected0); !known || mi != headerMI {
		t.Fatalf("late entry MI = %08X, want the header's %08X", mi, headerMI)
	}
	// The old reading — the embedded IV is THIS superframe's MI — is ciphertext.
	var own [][]byte
	for n, fr := range frames {
		infos := decode(fr)
		if _, _, err := DescrambleSuperframe(key, ivs[n], infos); err != nil {
			t.Fatal(err)
		}
		own = append(own, infos...)
	}
	if c := b0Continuity(own); c > 0.35 {
		t.Fatalf("descrambled with each superframe's own embedded IV: b0 continuity %.2f — should be ciphertext-like", c)
	}
	if tr.Mismatches != 0 || tr.Predicted != 0 {
		t.Fatalf("clean chain reported mismatches=%d predicted=%d", tr.Mismatches, tr.Predicted)
	}
}
