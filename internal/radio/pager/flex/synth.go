package flex

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// Frame synthesis, for tests and signal fixtures. Everything the decoder
// reads is laid out as the reference decoders read it. The second sync, which
// the decoder only counts through, follows the 1600 symbols/s on-air frames
// (bit sync 1010, 0xED84, then both inverted); at 3200 symbols/s its bit sync
// is stretched to fill the same 25 ms, which no capture has confirmed.

// FrameSymbols returns the symbols (0..3) of one complete 1.875 s frame in
// the given mode: 144 symbols at 1600 symbols/s (bit sync, frame sync, frame
// information word), then the second sync and the data at the mode's rate.
// phases holds the information words of each phase in Mode.Phases() order;
// missing phases are sent idle.
func FrameSymbols(mode Mode, cycle, frame int, phases ...[PhaseWords]uint32) []byte {
	out := make([]byte, 0, FrameSymbolCount(mode))
	// Sync-sense bits: 1 is the low deviation.
	syncBits := func(v uint64, n int) {
		for i := n - 1; i >= 0; i-- {
			if (v>>uint(i))&1 != 0 {
				out = append(out, 0)
			} else {
				out = append(out, 3)
			}
		}
	}
	// Data-sense bits: 1 is the high deviation.
	dataBit := func(b uint32) {
		if b != 0 {
			out = append(out, 3)
		} else {
			out = append(out, 0)
		}
	}

	syncBits(0xAAAAAAAA, 32) // bit sync
	syncBits(uint64(mode.Code)<<48|uint64(SyncMarker)<<16|uint64(^mode.Code), 64)
	syncBits(uint64(syncTail), fiwSkipSymbols)

	fiw := framing.FLEXBCHEncode21(withChecksum(uint32(cycle&0xF)<<4 | uint32(frame&0x7F)<<8))
	for i := 0; i < 32; i++ {
		dataBit((fiw >> uint(i)) & 1)
	}

	// Second sync: bit sync, a 16-bit pattern, and both again inverted.
	half := mode.Baud * sync2Millis / 1000 / 2
	for inv := uint32(0); inv < 2; inv++ {
		for i := 0; i < half-16; i++ {
			dataBit(uint32(1-i&1) ^ inv)
		}
		for i := 15; i >= 0; i-- {
			dataBit(((sync2C >> uint(i)) & 1) ^ inv)
		}
	}

	var cw [4][PhaseWords]uint32
	order := mode.Phases()
	for k, ph := range order {
		words := idlePhase()
		if k < len(phases) {
			words = phases[k]
		}
		for i, w := range words {
			cw[ph-'A'][i] = framing.FLEXBCHEncode21(w)
		}
	}
	gray := func(a, b uint32) byte {
		if mode.Levels == 2 {
			b = 0
		}
		switch {
		case a == 1 && b == 0:
			return 3
		case a == 1:
			return 2
		case b == 1:
			return 1
		}
		return 0
	}
	for c := 0; c < PhaseWords*32; c++ {
		idx := ((c >> 5) &^ 7) | (c & 7)
		pos := uint((c >> 3) & 31)
		out = append(out, gray((cw[0][idx]>>pos)&1, (cw[1][idx]>>pos)&1))
		if mode.Baud == 3200 {
			out = append(out, gray((cw[2][idx]>>pos)&1, (cw[3][idx]>>pos)&1))
		}
	}
	return out
}

// syncTail is the 16 sync-sense bits between the frame sync and the frame
// information word: the complement of the marker's first half, which is what
// the on-air frames in testdata carry there (TestRealAirFrames).
const syncTail uint16 = ^uint16(SyncMarker >> 16)

// sync2C is the second sync's 16-bit pattern (multimon-ng FLEX_S2_C).
const sync2C uint32 = 0xED84

// FrameSymbolCount is the number of symbols FrameSymbols returns.
func FrameSymbolCount(mode Mode) int {
	return 32 + 64 + fiwSymbols + mode.Baud*(sync2Millis+dataMillis)/1000
}

// idlePhase is a phase with no pages: a block information word with an empty
// address field, then idle words.
func idlePhase() [PhaseWords]uint32 {
	var w [PhaseWords]uint32
	w[0] = withChecksum(1 << 10) // addresses [1,1): none
	for i := 1; i < PhaseWords; i++ {
		if i&1 == 1 {
			w[i] = idleOnes
		}
	}
	return w
}

// Page is one page for PhaseWith: an alphanumeric message, or a numeric one
// when Numeric is set (digits, space, '-', '[', ']', 'U').
type Page struct {
	Capcode uint32 // short address, 1..1933312
	Text    string
	Numeric bool
}

// PhaseWith lays pages out in one phase: block information word, the
// addresses, their vectors, then the message words. It returns false if the
// pages do not fit in the phase's 88 words.
func PhaseWith(pages ...Page) ([PhaseWords]uint32, bool) {
	w := idlePhase()
	n := len(pages)
	const aoffset = 1
	voffset := aoffset + n
	next := voffset + n // first free message word
	if next > PhaseWords {
		return w, false
	}
	w[0] = withChecksum(uint32(voffset) << 10)
	for k, p := range pages {
		w[aoffset+k] = (p.Capcode + 0x8000) & 0x1FFFFF
		var body []uint32
		var vec uint32
		if p.Numeric {
			body = packNumeric(p.Text)
			vec = uint32(PageNumeric)<<4 | uint32(next)<<7 | uint32(len(body)-1)<<14
		} else {
			// Header word: fragment 3 (first), no continuation.
			body = append([]uint32{3 << 11}, packAlpha(p.Text)...)
			vec = uint32(PageAlphanumeric)<<4 | uint32(next)<<7 | uint32(len(body))<<14
		}
		if next+len(body) > PhaseWords || (p.Numeric && len(body) > 8) {
			return w, false
		}
		w[voffset+k] = withChecksum(vec)
		copy(w[next:], body)
		next += len(body)
	}
	return w, true
}

// packAlpha packs 7-bit characters three to a word, leaving the first slot
// (the message signature) empty and padding with ETX.
func packAlpha(text string) []uint32 {
	chars := append([]byte{0}, text...)
	for len(chars)%3 != 0 {
		chars = append(chars, 0x03)
	}
	out := make([]uint32, 0, len(chars)/3)
	for i := 0; i < len(chars); i += 3 {
		out = append(out, uint32(chars[i]&0x7F)|uint32(chars[i+1]&0x7F)<<7|uint32(chars[i+2]&0x7F)<<14)
	}
	return out
}

// packNumeric packs a numeric message: a 2-bit header, then 4-bit digits LSB
// first across the words, padded with the space digit.
func packNumeric(text string) []uint32 {
	var stream []byte // one bit each
	stream = append(stream, 0, 0)
	for i := 0; i < len(text); i++ {
		d := 0x0C
		for v := 0; v < len(flexBCD); v++ {
			if flexBCD[v] == text[i] {
				d = v
				break
			}
		}
		for b := 0; b < 4; b++ {
			stream = append(stream, byte(d>>uint(b))&1)
		}
	}
	for len(stream)%21 != 0 {
		// pad with space digits, bit by bit (0xC = 0,0,1,1 LSB first)
		pad := []byte{0, 0, 1, 1}
		stream = append(stream, pad[(len(stream)-2)%4])
	}
	out := make([]uint32, 0, len(stream)/21)
	for i := 0; i < len(stream); i += 21 {
		var w uint32
		for b := 0; b < 21; b++ {
			w |= uint32(stream[i+b]) << uint(b)
		}
		out = append(out, w)
	}
	return out
}
