package wmbus

// accessCode is the 16-chip synchronisation pattern that ends the preamble
// of both T and C mode frames: the tail of the 0101… preamble followed by
// T mode's "0000111101" sync word.
const accessCode = 0x543D

// After the access code a C mode frame sends a second 16-chip word naming
// its frame format; a T mode frame goes straight into 3-of-6 coded data.
const (
	cModeFormatA = 0x54CD
	cModeFormatB = 0x543D
)

// code3of6 maps a nibble to its 6-chip T mode code word (EN 13757-4): every
// word has exactly three ones, which keeps the chip stream DC-balanced.
var code3of6 = [16]uint8{
	0x16, 0x0D, 0x0E, 0x0B, 0x1C, 0x19, 0x1A, 0x13,
	0x2C, 0x25, 0x26, 0x23, 0x34, 0x31, 0x32, 0x29,
}

// decode3of6 maps a 6-chip word back to its nibble; invalid words are 0xFF.
var decode3of6 = func() (t [64]uint8) {
	for i := range t {
		t[i] = 0xFF
	}
	for n, c := range code3of6 {
		t[c] = uint8(n)
	}
	return t
}()

// Frame is one received link-layer frame.
type Frame struct {
	Mode   Mode
	Format Format
	// Raw is the frame as received: L-field first, CRC bytes included.
	Raw []byte
	// Data is Raw with the CRC bytes removed. Its L-field counts the bytes
	// after it, whichever the format.
	Data []byte
	// CRCOK reports whether every block CRC matched.
	CRCOK bool
}

type framerState uint8

const (
	stHunt     framerState = iota // looking for the access code
	stModeWord                    // collecting the 12 chips after it
	stCTrailer                    // C mode: the last 4 chips of the format word
	stCData                       // C mode: NRZ bytes
	stTData                       // T mode: 3-of-6 coded bytes
)

// FramerStats counts framing outcomes.
type FramerStats struct {
	Syncs       uint64 // access codes found
	Frames      uint64 // complete frames delivered (any CRC result)
	CRCOK       uint64 // of which every block CRC passed
	CodingError uint64 // T mode frames abandoned on an invalid 3-of-6 word
	BadLength   uint64 // frames abandoned on an impossible L-field
}

// Framer turns a chip stream into frames. It is not safe for concurrent
// use.
type Framer struct {
	onFrame func(Frame)

	state  framerState
	shift  uint32
	chips  int
	format Format
	need   int
	buf    []byte
	stats  FramerStats
}

// NewFramer returns a Framer that calls onFrame for every complete frame,
// whether or not its CRCs pass.
func NewFramer(onFrame func(Frame)) *Framer {
	return &Framer{onFrame: onFrame}
}

// Stats returns the framing counters.
func (f *Framer) Stats() FramerStats { return f.stats }

// Busy reports whether the framer is part-way through a frame.
func (f *Framer) Busy() bool { return f.state != stHunt }

// Reset abandons any frame in progress.
func (f *Framer) Reset() {
	f.state, f.shift, f.chips, f.buf = stHunt, 0, 0, f.buf[:0]
}

// Push feeds one chip (0 or 1).
func (f *Framer) Push(chip uint8) {
	f.shift = f.shift<<1 | uint32(chip&1)
	f.chips++
	switch f.state {
	case stHunt:
		if f.shift&0xFFFF == accessCode {
			f.stats.Syncs++
			f.state, f.chips = stModeWord, 0
		}
	case stModeWord:
		if f.chips < 12 {
			return
		}
		w := f.shift & 0xFFF
		hi, lo := decode3of6[w>>6], decode3of6[w&0x3F]
		switch {
		case hi != 0xFF && lo != 0xFF:
			f.beginData(ModeT1, FormatA, hi<<4|lo)
		case w == cModeFormatA>>4:
			f.format, f.state, f.chips = FormatA, stCTrailer, 0
		case w == cModeFormatB>>4:
			f.format, f.state, f.chips = FormatB, stCTrailer, 0
		default:
			f.Reset()
		}
	case stCTrailer:
		if f.chips < 4 {
			return
		}
		if f.shift&0xF != cModeFormatA&0xF {
			f.Reset()
			return
		}
		f.state, f.chips, f.need, f.buf = stCData, 0, 0, f.buf[:0]
	case stCData:
		if f.chips < 8 {
			return
		}
		b := byte(f.shift)
		f.chips = 0
		if f.need == 0 {
			f.beginData(ModeC1, f.format, b)
			return
		}
		f.appendByte(ModeC1, b)
	case stTData:
		if f.chips < 12 {
			return
		}
		w := f.shift & 0xFFF
		hi, lo := decode3of6[w>>6], decode3of6[w&0x3F]
		f.chips = 0
		if hi == 0xFF || lo == 0xFF {
			// A real frame with a coding error fails its CRC anyway;
			// abandoning it frees the framer for the next preamble
			// instead of swallowing noise for up to 3000 chips.
			f.stats.CodingError++
			f.Reset()
			return
		}
		f.appendByte(ModeT1, hi<<4|lo)
	}
}

// beginData starts collecting a frame whose L-field is l.
func (f *Framer) beginData(m Mode, format Format, l byte) {
	n := frameLength(l, format)
	if n == 0 {
		f.stats.BadLength++
		f.Reset()
		return
	}
	f.format, f.need, f.chips = format, n, 0
	f.buf = append(f.buf[:0], l)
	if m == ModeT1 {
		f.state = stTData
	} else {
		f.state = stCData
	}
}

func (f *Framer) appendByte(m Mode, b byte) {
	f.buf = append(f.buf, b)
	if len(f.buf) < f.need {
		return
	}
	raw := append([]byte(nil), f.buf...)
	format := f.format
	f.Reset()
	data, ok, err := stripCRC(raw, format)
	if err != nil {
		f.stats.BadLength++
		return
	}
	f.stats.Frames++
	if ok {
		f.stats.CRCOK++
	}
	if f.onFrame != nil {
		f.onFrame(Frame{Mode: m, Format: format, Raw: raw, Data: data, CRCOK: ok})
	}
}

// encodeFrameChips is the inverse of the framer, for tests and the
// synthesiser: preamble, access code, (C mode) format word, then the frame
// bytes 3-of-6 coded (T1) or NRZ (C1), MSB first, plus a short postamble.
func encodeFrameChips(m Mode, format Format, raw []byte, preambleChips int) []uint8 {
	var out []uint8
	bits := func(v uint32, n int) {
		for i := n - 1; i >= 0; i-- {
			out = append(out, uint8(v>>i&1))
		}
	}
	for i := range preambleChips {
		out = append(out, uint8(i&1))
	}
	bits(accessCode, 16)
	if m == ModeC1 {
		if format == FormatB {
			bits(cModeFormatB, 16)
		} else {
			bits(cModeFormatA, 16)
		}
		for _, b := range raw {
			bits(uint32(b), 8)
		}
	} else {
		for _, b := range raw {
			bits(uint32(code3of6[b>>4]), 6)
			bits(uint32(code3of6[b&0xF]), 6)
		}
	}
	bits(0b0101, 4)
	return out
}

// buildFrame assembles a raw on-air frame (CRCs included) from link-layer
// data whose first byte is the L-field in format A's convention (it counts
// the data bytes after it). For format B the L-field is rewritten to
// include the CRC bytes.
func buildFrame(data []byte, format Format) []byte {
	var raw []byte
	crc := func(block []byte) {
		c := CRC16(block)
		raw = append(raw, block...)
		raw = append(raw, byte(c>>8), byte(c))
	}
	if format == FormatA {
		crc(data[:10])
		for p := 10; p < len(data); p += 16 {
			crc(data[p:min(p+16, len(data))])
		}
		return raw
	}
	d := append([]byte(nil), data...)
	crcs := 2
	if len(d) > 126 {
		crcs = 4
	}
	d[0] = byte(len(d) - 1 + crcs)
	for p := 0; p < len(d); p += 126 {
		crc(d[p:min(p+126, len(d))])
	}
	return raw
}
