package flex

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildPhaseWords constructs the 88 BCH-decoded info values for a phase
// carrying one alphanumeric page "HI" to a short capcode.
func buildAlphaPhase(capcode uint32) [PhaseWords]uint32 {
	var w [PhaseWords]uint32
	// One address at word 1, vector at word 2, message at words 3-4.
	// BIW: voffset=2, aoffset=1.
	w[0] = (2 << 10) | (0 << 8) // biw
	w[1] = capcode + 0x8000     // short address word
	// VIW: type 5 (alpha), mw1=3, len=2.
	w[2] = (2 << 14) | (3 << 7) | (5 << 4)
	// Message words: first char of the first word is a skipped header.
	hdr, h, i := uint32(0x40), uint32('H'), uint32('I')
	w[3] = hdr | (h << 7) | (i << 14)
	w[4] = 0x03 // ETX in the first character slot ends the message
	return w
}

// encodeFrameBits turns a phase's info words into the full wire bit
// stream: dotting, the 32-bit sync marker, the 16-bit mode code, the
// FIW, then the BCH-encoded codewords interleaved per the FLEX block
// scheme.
func encodeFrameBits(words [PhaseWords]uint32, frame, cycle int) []byte {
	var bits []byte
	push := func(v uint32, n int) {
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, byte((v>>uint(i))&1))
		}
	}
	for i := 0; i < 64; i++ { // dotting / bit sync
		bits = append(bits, byte(i&1))
	}
	push(SyncMarker, 32)
	push(uint32(speed1600x2), 16)
	fiw := uint32((cycle&0xF)<<4) | uint32((frame&0x7F)<<8)
	push(fiw, 32)

	var cw [PhaseWords]uint32
	for i := range words {
		cw[i] = framing.FLEXBCHEncode21(words[i])
	}
	for c := 0; c < PhaseBits; c++ {
		word := ((c >> 5) & 0xFFF8) | (c & 7)
		pos := (c >> 3) & 31
		bits = append(bits, byte((cw[word]>>uint(pos))&1))
	}
	return bits
}

func TestDecodeAlphanumericFrame(t *testing.T) {
	const capcode = 0x12345
	bits := encodeFrameBits(buildAlphaPhase(capcode), 7, 3)

	d := NewDecoder()
	var got []Message
	for _, b := range bits {
		got = append(got, d.Push(b)...)
	}
	if len(got) != 1 {
		t.Fatalf("decoded %d messages, want 1 (stats %+v)", len(got), d.Stats())
	}
	m := got[0]
	if m.Capcode != capcode {
		t.Errorf("Capcode = %#x, want %#x", m.Capcode, capcode)
	}
	if m.Type != PageAlphanumeric {
		t.Errorf("Type = %v, want alphanumeric", m.Type)
	}
	if m.Text != "HI" {
		t.Errorf("Text = %q, want %q", m.Text, "HI")
	}
	if m.Frame != 7 || m.Cycle != 3 {
		t.Errorf("Frame/Cycle = %d/%d, want 7/3", m.Frame, m.Cycle)
	}
}

// TestDecodeInvertedPolarity feeds the same frame with every bit
// complemented; the marker hunt must lock on the inverted sense and
// recover the true page.
func TestDecodeInvertedPolarity(t *testing.T) {
	const capcode = 0x0ABCD
	bits := encodeFrameBits(buildAlphaPhase(capcode), 1, 1)

	d := NewDecoder()
	var got []Message
	for _, b := range bits {
		got = append(got, d.Push(b^1)...)
	}
	if len(got) != 1 {
		t.Fatalf("decoded %d messages, want 1", len(got))
	}
	if got[0].Capcode != capcode {
		t.Errorf("Capcode = %#x, want %#x", got[0].Capcode, capcode)
	}
	if got[0].Text != "HI" {
		t.Errorf("Text = %q, want HI", got[0].Text)
	}
}

// TestBCHCorrectsBitErrors confirms the FLEX BCH layer corrects a
// couple of flipped bits in a codeword.
func TestBCHCorrectsBitErrors(t *testing.T) {
	info := uint32(0x15ABC)
	cw := framing.FLEXBCHEncode21(info)
	cw ^= 1 << 3  // flip a data bit
	cw ^= 1 << 25 // flip a parity bit
	got, errs := framing.FLEXBCHDecode32(cw)
	if got != info {
		t.Errorf("decoded info = %#x, want %#x", got, info)
	}
	if errs != 2 {
		t.Errorf("corrected %d errors, want 2", errs)
	}
}
