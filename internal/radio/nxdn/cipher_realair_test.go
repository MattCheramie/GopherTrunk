package nxdn

import (
	"encoding/json"
	"os"
	"testing"
)

// scramblerFixture is a dump of every LICH-valid traffic frame the
// production receiver + TrafficChannel recovered from one of the issue
// #1261 reporter's known-key captures (447.100 MHz NXDN96, Kenwood radios,
// scanner voice_iq_debug cs16 at 2.4 MS/s): the SACCH channel bits and the
// FEC-decoded 49-bit AMBE+2 payloads per voice slot, exactly what
// TrafficChannel.OnFrame delivers.
type scramblerFixture struct {
	Key    uint16 `json:"key"`
	Frames []struct {
		Index    uint64   `json:"index"`
		LICH     uint8    `json:"lich"`
		SACCH    string   `json:"sacch"`
		Slots    []int    `json:"slots"`
		Payloads []string `json:"payloads"`
		Errors   []int    `json:"errors"`
	} `json:"frames"`
}

func loadScramblerFixture(t *testing.T, key int) (scramblerFixture, []TrafficFrame) {
	t.Helper()
	raw, err := os.ReadFile("testdata/scrambler_key" + itoa(key) + "_frames.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx scramblerFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	frames := make([]TrafficFrame, 0, len(fx.Frames))
	for _, f := range fx.Frames {
		tf := TrafficFrame{Index: f.Index, LICH: ParseAirLICH(lichDibits(f.LICH)), SACCH: bitString(f.SACCH)}
		for i, p := range f.Payloads {
			tf.Voice = append(tf.Voice, VCHFrame{Slot: f.Slots[i], Payload: bitString(p), Errors: f.Errors[i]})
		}
		frames = append(frames, tf)
	}
	return fx, frames
}

func bitString(s string) []byte {
	out := make([]byte, len(s))
	for i := range s {
		out[i] = s[i] - '0'
	}
	return out
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}

// ambeB0 is the AMBE+2 3600x2450 fundamental-frequency index: payload bits
// 0..3 and 37..39 (mbelib ambe3600x2450.c).
func ambeB0(p []byte) int {
	if len(p) != VoiceSlotBits {
		return -1
	}
	v := 0
	for _, i := range []int{0, 1, 2, 3, 37, 38, 39} {
		v = v<<1 | int(p[i]&1)
	}
	return v
}

// pitchContinuity is the fraction of consecutive voice frames whose b0
// differ by at most 10: speech ≳ 0.45, random bits / ciphertext ≈ 0.16.
func pitchContinuity(frames [][]byte) (float64, int) {
	pairs, cont := 0, 0
	for i := 1; i < len(frames); i++ {
		a, b := ambeB0(frames[i-1]), ambeB0(frames[i])
		if a < 0 || b < 0 {
			continue
		}
		pairs++
		if d := a - b; d <= 10 && d >= -10 {
			cont++
		}
	}
	if pairs == 0 {
		return 0, 0
	}
	return float64(cont) / float64(pairs), pairs
}

func copyFrames(in []TrafficFrame) []TrafficFrame {
	out := make([]TrafficFrame, len(in))
	for i, f := range in {
		out[i] = f
		out[i].Voice = make([]VCHFrame, len(f.Voice))
		for j, v := range f.Voice {
			out[i].Voice[j] = v
			out[i].Voice[j].Payload = append([]byte(nil), v.Payload...)
		}
	}
	return out
}

func runTracker(frames []TrafficFrame, tr *VoiceCipherTracker) [][]byte {
	var out [][]byte
	for _, f := range frames {
		for _, v := range tr.Frame(f) {
			out = append(out, v.Payload)
		}
	}
	for _, v := range tr.Flush() {
		out = append(out, v.Payload)
	}
	return out
}

// TestVoiceCipherTrackerDecodesRealAirScrambledCalls is the capture pin for
// the whole scrambler path: SACCH CRC, superframe assembly, VCALL fields,
// LFSR, reseed schedule and slot mapping. With the radios' programmed key
// the descrambled AMBE+2 frames have speech-like pitch continuity; with no
// key, or a neighbouring key, they sit at the random-bits floor. The old
// CRC-6 verified none of these SACCH blocks and the old register decoded
// nothing, so every assertion here fails against the previous code.
func TestVoiceCipherTrackerDecodesRealAirScrambledCalls(t *testing.T) {
	for _, c := range []struct {
		key   int
		keyID uint8
	}{{32767, 3}, {12345, 1}} {
		fx, frames := loadScramblerFixture(t, c.key)
		if fx.Key != uint16(c.key) {
			t.Fatalf("fixture key %d, want %d", fx.Key, c.key)
		}
		var raw [][]byte
		for _, f := range frames {
			for _, v := range f.Voice {
				raw = append(raw, v.Payload)
			}
		}
		rawCont, pairs := pitchContinuity(raw)

		// Key installed on VCALL, as the voice chain does.
		var vcalls []VCALLMessage
		tr := &VoiceCipherTracker{Hold: true}
		tr.OnVCALL = func(v VCALLMessage) {
			vcalls = append(vcalls, v)
			if v.Cipher == CipherScrambler && v.KeyID == c.keyID {
				tr.SetKey(uint16(c.key))
			}
		}
		dec := runTracker(copyFrames(frames), tr)
		cont, _ := pitchContinuity(dec)

		wrong := &VoiceCipherTracker{Hold: true}
		wrong.OnVCALL = func(VCALLMessage) { wrong.SetKey(uint16(c.key) ^ 1) }
		wrongCont, _ := pitchContinuity(runTracker(copyFrames(frames), wrong))

		t.Logf("key %d: frames=%d voice=%d sacch_ok=%d sacch_fail=%d superframes=%d vcalls=%d descrambled=%d ciphertext=%d unaligned=%d continuity raw=%.2f wrong-key=%.2f key=%.2f over %d pairs vcall=%+v",
			c.key, len(frames), len(raw), tr.SACCHOK, tr.SACCHFail, tr.Superframes, tr.VCALLs, tr.Descrambled, tr.Ciphertext, tr.Unaligned, rawCont, wrongCont, cont, pairs, tr.VCALL)

		if len(dec) != len(raw) {
			t.Fatalf("key %d: released %d voice frames, want every one of %d", c.key, len(dec), len(raw))
		}
		if tr.SACCHOK < 40 {
			t.Errorf("key %d: only %d CRC-valid SACCH blocks of %d frames (reference CRC-6 verifies ~48/65)", c.key, tr.SACCHOK, len(frames))
		}
		if len(vcalls) != 1 {
			t.Fatalf("key %d: %d distinct VCALLs, want 1: %+v", c.key, len(vcalls), vcalls)
		}
		v := vcalls[0]
		if v.Cipher != CipherScrambler || v.KeyID != c.keyID || v.Source != 8814 || v.Destination != 0 {
			t.Errorf("key %d: VCALL = %+v, want cipher scrambler key id %d src 8814 dst 0", c.key, v, c.keyID)
		}
		if cont < 0.55 {
			t.Errorf("key %d: descrambled pitch continuity %.2f, want ≥ 0.55 (speech)", c.key, cont)
		}
		if rawCont > 0.35 || wrongCont > 0.35 {
			t.Errorf("key %d: ciphertext continuity raw %.2f / wrong key %.2f should sit near the 0.16 floor", c.key, rawCont, wrongCont)
		}
		if tr.Descrambled < uint64(len(raw))*9/10 {
			t.Errorf("key %d: descrambled %d of %d voice frames", c.key, tr.Descrambled, len(raw))
		}
	}
}

// TestVoiceCipherTrackerNoKeyPassesCiphertext: with no key the frames come
// out exactly as they went in, and the hold still releases them all.
func TestVoiceCipherTrackerNoKeyPassesCiphertext(t *testing.T) {
	_, frames := loadScramblerFixture(t, 32767)
	var raw [][]byte
	for _, f := range frames {
		for _, v := range f.Voice {
			raw = append(raw, v.Payload)
		}
	}
	tr := &VoiceCipherTracker{Hold: true}
	out := runTracker(copyFrames(frames), tr)
	if len(out) != len(raw) {
		t.Fatalf("released %d frames, want %d", len(out), len(raw))
	}
	for i := range out {
		for j := range out[i] {
			if out[i][j] != raw[i][j] {
				t.Fatalf("frame %d bit %d changed with no key", i, j)
			}
		}
	}
	if tr.Ciphertext == 0 || tr.Descrambled != 0 {
		t.Errorf("counters: ciphertext=%d descrambled=%d", tr.Ciphertext, tr.Descrambled)
	}
}

// TestSACCHCRC6RealAir: the reference CRC verifies the capture's SACCH
// blocks (the earlier register verified only all-zero blocks) and does NOT
// verify an all-zero block — which the earlier register, with its zero
// preset, passed.
func TestSACCHCRC6RealAir(t *testing.T) {
	_, frames := loadScramblerFixture(t, 32767)
	ok := 0
	for _, f := range frames {
		if _, _, err := DecodeSACCH(f.SACCH); err == nil {
			ok++
		}
	}
	if ok < 40 {
		t.Fatalf("%d of %d real-air SACCH blocks verify, want ≥ 40", ok, len(frames))
	}
	zero := make([]byte, SACCHInfoBits)
	if VerifySACCHCRC6(zero) {
		t.Fatal("an all-zero SACCH info block must not verify (preset 0x3F)")
	}
	// MMDVMHost CNXDNCRC::createCRC6 on 26 zero bits.
	if got := SACCHCRC6(zero, sacchPayloadBits); got != 0x3F^0 && got == 0 {
		t.Fatalf("CRC-6 of 26 zero bits is %#x, must be non-zero", got)
	}
}
