package phase1

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestADPVoiceFrameOffsetsMatchOP25 pins the keystream layout against the
// literal offsets OP25's adp_process computes (267 + 11·i, +2 at i = 8,
// +101 for LDU2) — the only kind of test that catches constant drift.
func TestADPVoiceFrameOffsetsMatchOP25(t *testing.T) {
	want1 := []int{267, 278, 289, 300, 311, 322, 333, 344, 357}
	want2 := []int{368, 379, 390, 401, 412, 423, 434, 445, 458}
	for i := 0; i < LDUVoiceSubframeCount; i++ {
		if got, ok := ADPVoiceFrameOffset(DUIDLogicalLink1, i); !ok || got != want1[i] {
			t.Errorf("LDU1 u%d offset = %d,%v want %d", i, got, ok, want1[i])
		}
		if got, ok := ADPVoiceFrameOffset(DUIDLogicalLink2, i); !ok || got != want2[i] {
			t.Errorf("LDU2 u%d offset = %d,%v want %d", i, got, ok, want2[i])
		}
	}
	if _, ok := ADPVoiceFrameOffset(DUIDLogicalLink1, 9); ok {
		t.Error("subframe 9 accepted")
	}
	if _, ok := ADPVoiceFrameOffset(DUIDTerminator, 0); ok {
		t.Error("TDU accepted")
	}
	// The last LDU2 frame ends exactly at the superframe span.
	if last, _ := ADPVoiceFrameOffset(DUIDLogicalLink2, 8); last+11 != adpSuperframeKeystreamBytes {
		t.Errorf("superframe span = %d, want %d", last+11, adpSuperframeKeystreamBytes)
	}
}

// adpFixtureLDU is one LDU of the #1187 reporter's ADP call as the
// production Phase 1 receiver decoded it (every frame FEC-clean).
type adpFixtureLDU struct {
	DUID   int      `json:"duid"`
	MI     string   `json:"mi,omitempty"`
	AlgID  int      `json:"algid,omitempty"`
	KeyID  int      `json:"keyid,omitempty"`
	Frames []string `json:"frames"`
}

func loadADPFixture(t *testing.T) []adpFixtureLDU {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "adp_issue1187_ldus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ldus []adpFixtureLDU
	if err := json.Unmarshal(raw, &ldus); err != nil {
		t.Fatal(err)
	}
	if len(ldus) < 20 {
		t.Fatalf("fixture has %d LDUs", len(ldus))
	}
	return ldus
}

func fixtureMI(t *testing.T, s string) [9]byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 9 {
		t.Fatalf("bad MI %q", s)
	}
	var mi [9]byte
	copy(mi[:], b)
	return mi
}

// TestAdvanceMIMatchesCapturedSequence pins the MI LFSR against the fifteen
// consecutive Message Indicators the reporter's call announced: every
// LDU2's ES MI is AdvanceMI of the previous one, and RewindMI walks back.
func TestAdvanceMIMatchesCapturedSequence(t *testing.T) {
	// Two literal pairs from the capture, independent of the fixture file.
	first := fixtureMI(t, "17ceec55310a747500")
	second := fixtureMI(t, "64c1f7d686f6c64500")
	if got := AdvanceMI(first); got != second {
		t.Fatalf("AdvanceMI(%x) = %x, want %x", first, got, second)
	}
	if got := RewindMI(second); got != first {
		t.Fatalf("RewindMI(%x) = %x, want %x", second, got, first)
	}
	var prev [9]byte
	have := false
	pairs := 0
	for _, l := range loadADPFixture(t) {
		if l.MI == "" {
			continue
		}
		mi := fixtureMI(t, l.MI)
		if have {
			pairs++
			if got := AdvanceMI(prev); got != mi {
				t.Errorf("AdvanceMI(%x) = %x, capture says %x", prev, got, mi)
			}
			if got := RewindMI(mi); got != prev {
				t.Errorf("RewindMI(%x) = %x, capture says %x", mi, got, prev)
			}
		}
		prev, have = mi, true
	}
	if pairs < 10 {
		t.Fatalf("only %d consecutive MI pairs in the fixture", pairs)
	}
	rng := rand.New(rand.NewSource(1187))
	for i := 0; i < 1000; i++ {
		var mi [9]byte
		rng.Read(mi[:])
		if got := RewindMI(AdvanceMI(mi)); got != mi {
			t.Fatalf("RewindMI(AdvanceMI(%x)) = %x", mi, got)
		}
	}
}

// imbeB0 reads the IMBE fundamental-frequency parameter b0 from a packed
// 11-byte frame: info bits 0..5 (the top six bits of byte 0) and bits 85,
// 86 (bits 2 and 1 of byte 10), MSB-first — imbe.b0FromInfo on packed bytes.
func imbeB0(f []byte) int {
	return int(f[0]>>2)<<2 | int((f[10]>>2)&1)<<1 | int((f[10]>>1)&1)
}

// imbePitchContinuity is the fraction of consecutive voiced frame pairs
// whose b0 moves by at most 10 steps. Real speech holds its fundamental
// between 20 ms frames (≳ 0.5); ciphertext, or a wrong key or layout,
// gives an independent random b0 every frame (≈ 0.1). Silence-window and
// invalid b0 values (> 207) break the chain.
func imbePitchContinuity(frames [][]byte) (float64, int) {
	var pairs, close int
	prev := -1
	for _, f := range frames {
		if f == nil {
			prev = -1
			continue
		}
		b := imbeB0(f)
		if b > 207 {
			prev = -1
			continue
		}
		if prev >= 0 {
			pairs++
			if d := b - prev; d >= -10 && d <= 10 {
				close++
			}
		}
		prev = b
	}
	if pairs == 0 {
		return 0, 0
	}
	return float64(close) / float64(pairs), pairs
}

// adpFixtureDescramble replays the fixture through the production
// primitives under a chosen convention and returns the resulting frames.
//   - nextRule: the ES MI names the NEXT superframe (production; the first
//     superframe's MI is RewindMI of the first ES) — false applies it to the
//     superframe the LDU2 belongs to.
//   - skipOffset: shifts every keystream offset (0 = the OP25 layout).
func adpFixtureDescramble(t *testing.T, ldus []adpFixtureLDU, key []byte, nextRule bool, skipOffset int) [][]byte {
	t.Helper()
	var out [][]byte
	var cur [9]byte
	haveCur := false
	for i, l := range ldus {
		duid := DUID(l.DUID)
		var frames [LDUVoiceSubframeCount][]byte
		for k, h := range l.Frames {
			b, err := hex.DecodeString(h)
			if err != nil {
				t.Fatal(err)
			}
			frames[k] = b
		}
		if duid == DUIDLogicalLink2 && l.MI != "" {
			mi := fixtureMI(t, l.MI)
			if nextRule {
				if !haveCur {
					cur, haveCur = RewindMI(mi), true
				}
			} else {
				cur, haveCur = mi, true
				// The current-superframe reading also covers the LDU1 before
				// this LDU2, which was already emitted: re-descramble it.
				if i > 0 && DUID(ldus[i-1].DUID) == DUIDLogicalLink1 && len(out) >= LDUVoiceSubframeCount {
					var prevFrames [LDUVoiceSubframeCount][]byte
					copy(prevFrames[:], out[len(out)-LDUVoiceSubframeCount:])
					ks, err := ADPSuperframeKeystream(key, cur)
					if err != nil {
						t.Fatal(err)
					}
					ksShift := shiftKeystream(ks, skipOffset)
					if _, err := ADPDescrambleVoiceFrames(ksShift, DUIDLogicalLink1, &prevFrames); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		if haveCur {
			ks, err := ADPSuperframeKeystream(key, cur)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ADPDescrambleVoiceFrames(shiftKeystream(ks, skipOffset), duid, &frames); err != nil {
				t.Fatal(err)
			}
		}
		out = append(out, frames[:]...)
		if duid == DUIDLogicalLink2 && haveCur {
			if nextRule && l.MI != "" {
				cur = fixtureMI(t, l.MI)
			} else {
				cur = AdvanceMI(cur)
			}
		}
	}
	return out
}

// shiftKeystream returns ks with every offset moved by delta (a negative
// delta reads keystream that the OP25 layout skips; padded with zeros).
func shiftKeystream(ks []byte, delta int) []byte {
	if delta == 0 {
		return ks
	}
	out := make([]byte, len(ks))
	for i := range out {
		if j := i + delta; j >= 0 && j < len(ks) {
			out[i] = ks[j]
		}
	}
	return out
}

// TestADPDescrambleRecoversSpeechFromCapture is the on-air pin for the
// whole ADP construction (issue #1187): the reporter's 6.2 s ADP call
// (NAC 239, TG 1, key ID 1, key 1234567890, 15 LDU1 + 15 LDU2, all
// FEC-clean) descrambles to a speech-continuous IMBE pitch track under the
// OP25 layout + next-superframe MI rule, and stays ciphertext-random under
// every alternative: raw, wrong key, MI applied to the current superframe,
// or the 11-byte skip omitted.
func TestADPDescrambleRecoversSpeechFromCapture(t *testing.T) {
	ldus := loadADPFixture(t)
	key, _ := hex.DecodeString("1234567890")
	for _, l := range ldus {
		if l.MI != "" && (l.AlgID != 0xAA || l.KeyID != 1) {
			t.Fatalf("fixture LDU2 alg/kid = 0x%02X/%d", l.AlgID, l.KeyID)
		}
	}
	var raw [][]byte
	for _, l := range ldus {
		for _, h := range l.Frames {
			b, _ := hex.DecodeString(h)
			raw = append(raw, b)
		}
	}
	rawC, rawPairs := imbePitchContinuity(raw)
	good, goodPairs := imbePitchContinuity(adpFixtureDescramble(t, ldus, key, true, 0))
	wrongKey, _ := imbePitchContinuity(adpFixtureDescramble(t, ldus, []byte{0, 0, 0, 0, 0}, true, 0))
	current, _ := imbePitchContinuity(adpFixtureDescramble(t, ldus, key, false, 0))
	noSkip, _ := imbePitchContinuity(adpFixtureDescramble(t, ldus, key, true, -11))
	t.Logf("pitch continuity: raw %.2f (%d pairs) | production %.2f (%d pairs) | wrong key %.2f | current-superframe MI %.2f | no 11-byte skip %.2f",
		rawC, rawPairs, good, goodPairs, wrongKey, current, noSkip)
	if goodPairs < 200 {
		t.Fatalf("only %d voiced pairs after descramble", goodPairs)
	}
	if good < 0.5 {
		t.Fatalf("production descramble continuity %.2f < 0.5 — the layout or MI rule is wrong", good)
	}
	for name, c := range map[string]float64{"raw": rawC, "wrong key": wrongKey, "current-superframe MI": current, "no skip": noSkip} {
		if c > 0.25 {
			t.Errorf("%s continuity %.2f > 0.25 — the verdict cannot separate it from the real decode", name, c)
		}
	}
	// The first LDU of the capture is an LDU2 (the chain entered mid-
	// superframe), so its own frames are decoded under RewindMI of its ES.
	// Nine frames are too few for the continuity statistic to assert on
	// (this LDU scores 0.25 under the rewound MI and 0.20 under its own),
	// so the rewind is pinned by the 14-pair LFSR chain above and the
	// per-LDU figure is only reported here.
	first := adpFixtureDescramble(t, ldus, key, true, 0)[:LDUVoiceSubframeCount]
	c, n := imbePitchContinuity(first)
	t.Logf("first (rewound) LDU2: continuity %.2f over %d pairs", c, n)
	// Split by LDU type: both halves of the superframe must decode, or the
	// overall figure could be carried by one of them alone (LDU1 and LDU2
	// have different keystream offsets).
	all := adpFixtureDescramble(t, ldus, key, true, 0)
	var ldu1, ldu2 [][]byte
	invalid := [2]int{}
	for i, l := range ldus {
		fr := all[i*LDUVoiceSubframeCount : (i+1)*LDUVoiceSubframeCount]
		for _, f := range fr {
			if b := imbeB0(f); b > 207 && (b < 216 || b > 219) {
				invalid[l.DUID/10]++ // 5 → 0, 10 → 1
			}
		}
		if DUID(l.DUID) == DUIDLogicalLink1 {
			ldu1 = append(ldu1, fr...)
		} else {
			ldu2 = append(ldu2, fr...)
		}
	}
	c1, n1 := imbePitchContinuity(ldu1)
	c2, n2 := imbePitchContinuity(ldu2)
	t.Logf("by LDU type: LDU1 %.2f over %d pairs (%d invalid b0) | LDU2 %.2f over %d pairs (%d invalid b0)", c1, n1, invalid[0], c2, n2, invalid[1])
	if c1 < 0.35 || c2 < 0.35 {
		t.Errorf("one LDU type stays ciphertext-like: LDU1 %.2f, LDU2 %.2f", c1, c2)
	}
	// A wrong keystream leaves ~17 % of frames with an out-of-range b0;
	// the right one leaves none on this FEC-clean capture.
	if invalid[0] > 3 || invalid[1] > 3 {
		t.Errorf("invalid b0 values after descramble: LDU1 %d, LDU2 %d", invalid[0], invalid[1])
	}
}

// TestADPDescrambleSkipsFECFailedFrames: a nil frame keeps its slot so the
// frames after it still line up with their keystream bytes.
func TestADPDescrambleSkipsFECFailedFrames(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5}
	mi := [9]byte{9, 8, 7, 6, 5, 4, 3, 2, 1}
	ks, err := ADPSuperframeKeystream(key, mi)
	if err != nil {
		t.Fatal(err)
	}
	if len(ks) != ADPSuperframeKeystreamBytes {
		t.Fatalf("keystream = %d bytes", len(ks))
	}
	var frames [LDUVoiceSubframeCount][]byte
	for i := range frames {
		if i == 3 {
			continue
		}
		frames[i] = make([]byte, 11)
	}
	n, err := ADPDescrambleVoiceFrames(ks, DUIDLogicalLink2, &frames)
	if err != nil || n != 8 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	off, _ := ADPVoiceFrameOffset(DUIDLogicalLink2, 8)
	for j := 0; j < 11; j++ {
		if frames[8][j] != ks[off-256+j] {
			t.Fatalf("u8 byte %d = %02x, want keystream %02x", j, frames[8][j], ks[off-256+j])
		}
	}
	if _, err := ADPSuperframeKeystream([]byte{1, 2}, mi); err == nil {
		t.Error("short key accepted")
	}
	if _, err := ADPDescrambleVoiceFrames(ks, DUIDTerminator, &frames); err == nil {
		t.Error("TDU accepted")
	}
}
