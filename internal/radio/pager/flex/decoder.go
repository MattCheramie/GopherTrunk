package flex

import (
	"math/bits"
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// SyncMarker is the 32-bit centre of the 64-bit FLEX frame sync
// (code : marker : ^code), in the sync bit sense (see PushSymbol).
const SyncMarker uint32 = 0xA6C6AAAA

// SyncSymbolRate is the rate of the frame's opening section (bit sync, frame
// sync, frame information word), whatever the frame's mode.
const SyncSymbolRate = 1600

// PhaseWords is the number of 32-bit words in one phase of a frame.
const PhaseWords = 88

// Frame section lengths.
const (
	fiwSkipSymbols = 16   // the tail of the sync, between ^code and the FIW
	fiwSymbols     = 48   // skip + 32-bit frame information word
	sync2Millis    = 25   // second sync, at the frame's own symbol rate
	dataMillis     = 1760 // 11 blocks of 160 ms
)

// Mode is one FLEX transmission mode: the symbol rate and level count of the
// frame after its frame information word.
type Mode struct {
	Code   uint16 // 16-bit sync code
	Baud   int    // symbols per second: 1600 or 3200
	Levels int    // FSK levels: 2 or 4
}

// BitRate is the frame's bit rate: 1600, 3200 or 6400 bps.
func (m Mode) BitRate() int { return m.Baud * m.Levels / 2 }

// Phases lists the phases the mode carries, in decode order.
func (m Mode) Phases() []byte {
	switch {
	case m.Baud == 1600 && m.Levels == 2:
		return []byte{'A'}
	case m.Baud == 1600:
		return []byte{'A', 'B'}
	case m.Levels == 2:
		return []byte{'A', 'C'}
	}
	return []byte{'A', 'B', 'C', 'D'}
}

// The FLEX modes (multimon-ng flex_modes; ARIB STD-43A table 3.2-5).
var (
	Mode1600x2 = Mode{Code: 0x870C, Baud: 1600, Levels: 2}
	Mode3200x2 = Mode{Code: 0x7B18, Baud: 3200, Levels: 2}
	Mode1600x4 = Mode{Code: 0xB068, Baud: 1600, Levels: 4}
	Mode3200x4 = Mode{Code: 0xDEA0, Baud: 3200, Levels: 4}
	ModeReFLEX = Mode{Code: 0x4C7C, Baud: 3200, Levels: 4}

	// Modes is every mode the decoder follows.
	Modes = []Mode{Mode1600x2, Mode1600x4, Mode3200x2, Mode3200x4, ModeReFLEX}
)

// MatchSync tests the last 64 sync-sense bits for a frame sync: the marker
// within 3 bit errors, the two copies of the code (true and complemented)
// within 3 of each other, and the code within 3 of a known mode.
func MatchSync(reg uint64) (Mode, bool) {
	marker := uint32(reg >> 16)
	codeHigh := uint16(reg >> 48)
	codeLow := ^uint16(reg)
	if bits.OnesCount32(marker^SyncMarker) >= 4 || bits.OnesCount16(codeHigh^codeLow) >= 4 {
		return Mode{}, false
	}
	for _, m := range Modes {
		if bits.OnesCount16(m.Code^codeHigh) < 4 {
			return m, true
		}
	}
	return Mode{}, false
}

type decState uint8

const (
	stHunt  decState = iota // looking for the frame sync
	stFIW                   // sync tail + frame information word
	stSync2                 // second sync, at the frame's symbol rate
	stData                  // the interleaved phases
)

// Decoder turns a FLEX symbol stream into pages. Symbols are the four FSK
// levels 0..3, lowest frequency deviation first (a 2-level section uses 0
// and 3). Not safe for concurrent use — one Decoder per channel.
type Decoder struct {
	st       decState
	reg      uint64 // hunt: the last 64 sync-sense bits
	inverted bool   // the stream's frequency sense is inverted
	mode     Mode

	count  int    // symbols consumed in the current section
	fiwRaw uint32 // frame information word, first bit in the LSB
	frame  int
	cycle  int

	dataBit int  // interleave counter within the data section
	toggle  bool // 3200 baud: the next symbol belongs to phases C / D
	phase   [4][PhaseWords]uint32

	partial map[uint64]*partialMsg // alphanumeric fragments awaiting the rest
	absFrm  int                    // frames decoded, for fragment expiry

	framesSynced   atomic.Uint64
	fiwErrors      atomic.Uint64
	pagesEmitted   atomic.Uint64
	wordsFailed    atomic.Uint64
	framesByBitRte [3]atomic.Uint64 // 1600, 3200, 6400 bps
}

type partialMsg struct {
	msg  Message
	seen int // absFrm of the last fragment
}

// NewDecoder returns a Decoder hunting for a frame sync.
func NewDecoder() *Decoder {
	return &Decoder{partial: make(map[uint64]*partialMsg)}
}

// Hunting reports whether the decoder is between frames, looking for a sync.
func (d *Decoder) Hunting() bool { return d.st == stHunt }

// SymbolRate is the rate the NEXT symbol is sent at: 1600 through the frame
// information word, then the frame's own rate until the frame ends.
func (d *Decoder) SymbolRate() int {
	if d.st == stSync2 || d.st == stData {
		return d.mode.Baud
	}
	return SyncSymbolRate
}

// FrameMode is the mode of the frame being (or last) decoded.
func (d *Decoder) FrameMode() Mode { return d.mode }

// Inverted reports the frequency sense the last sync locked on.
func (d *Decoder) Inverted() bool { return d.inverted }

// Begin starts a frame whose sync a caller found itself (the receiver hunts
// at several timing phases at once): the next symbol pushed is the first
// after the 64 sync bits.
func (d *Decoder) Begin(mode Mode, inverted bool) {
	d.mode = mode
	d.inverted = inverted
	d.st = stFIW
	d.count = 0
	d.fiwRaw = 0
	d.reg = 0
}

// PushSymbol feeds one symbol and returns any pages the symbol completed
// (pages come out at the end of a frame).
//
// Bit senses, from the reference: in the sync a 1 is the LOW deviation
// (symbol < 2), sent MSB first; in the frame information word and the data a
// 1 is the HIGH deviation (symbol > 1), sent LSB first.
func (d *Decoder) PushSymbol(sym byte) []Message {
	sym &= 3
	switch d.st {
	case stHunt:
		d.reg <<= 1
		if sym < 2 {
			d.reg |= 1
		}
		if m, ok := MatchSync(d.reg); ok {
			d.Begin(m, false)
		} else if m, ok := MatchSync(^d.reg); ok {
			d.Begin(m, true)
		}
		return nil
	}
	if d.inverted {
		sym = 3 - sym
	}
	switch d.st {
	case stFIW:
		d.count++
		if d.count > fiwSkipSymbols {
			d.fiwRaw >>= 1
			if sym > 1 {
				d.fiwRaw |= 0x80000000
			}
		}
		if d.count == fiwSymbols {
			fiw, errs := framing.FLEXBCHDecode32(d.fiwRaw)
			if errs < 0 || !wordChecksumOK(fiw) {
				d.fiwErrors.Add(1)
				d.st = stHunt
				return nil
			}
			d.cycle = int((fiw >> 4) & 0x0F)
			d.frame = int((fiw >> 8) & 0x7F)
			d.framesSynced.Add(1)
			d.framesByBitRte[bitRateIndex(d.mode)].Add(1)
			d.st = stSync2
			d.count = 0
		}
	case stSync2:
		d.count++
		if d.count == d.mode.Baud*sync2Millis/1000 {
			d.st = stData
			d.count = 0
			d.dataBit = 0
			d.toggle = false
			d.phase = [4][PhaseWords]uint32{}
		}
	case stData:
		d.placeSymbol(sym)
		d.count++
		if d.count == d.mode.Baud*dataMillis/1000 {
			d.st = stHunt
			d.reg = 0
			return d.decodeFrame()
		}
	}
	return nil
}

// placeSymbol hands one data symbol's bits to their phases. A 4-level symbol
// carries two bits, Gray coded: the first (phase A or C) is the sign of the
// deviation, the second (phase B or D) is set on the two inner levels. At
// 3200 symbols/s alternate symbols belong to A/B and C/D.
//
// Each phase is block interleaved: within a block of eight words the bits go
// out column by column, so bit counter c lands in word
// ((c>>5)&^7)|(c&7), filling it from its LSB.
func (d *Decoder) placeSymbol(sym byte) {
	var a, b uint32
	if sym > 1 {
		a = 0x80000000
	}
	if d.mode.Levels == 4 && (sym == 1 || sym == 2) {
		b = 0x80000000
	}
	idx := ((d.dataBit >> 5) &^ 7) | (d.dataBit & 7)
	if idx >= PhaseWords {
		return
	}
	first := 0 // phases A, B
	if d.mode.Baud == 3200 && d.toggle {
		first = 2 // phases C, D
	}
	d.phase[first][idx] = d.phase[first][idx]>>1 | a
	d.phase[first+1][idx] = d.phase[first+1][idx]>>1 | b
	if d.mode.Baud == 3200 {
		d.toggle = !d.toggle
		if d.toggle {
			return // the C/D symbol of this bit is still to come
		}
	}
	d.dataBit++
}

// decodeFrame BCH-decodes each active phase and parses its pages.
func (d *Decoder) decodeFrame() []Message {
	d.absFrm++
	var out []Message
	var words [PhaseWords]uint32
	var errs [PhaseWords]int
	for _, ph := range d.mode.Phases() {
		buf := &d.phase[ph-'A']
		for i := range buf {
			words[i], errs[i] = framing.FLEXBCHDecode32(buf[i])
			if errs[i] < 0 {
				d.wordsFailed.Add(1)
			}
		}
		for _, f := range decodePhase(words[:], errs[:]) {
			f.msg.Mode = d.mode
			f.msg.Phase = ph
			f.msg.Cycle = d.cycle
			f.msg.Frame = d.frame
			if m, ok := d.reassemble(f); ok {
				out = append(out, m)
			}
		}
	}
	d.expirePartials()
	d.pagesEmitted.Add(uint64(len(out)))
	return out
}

// Fragment bookkeeping bounds: a message's fragments arrive within a few
// frames of each other; anything older is a message whose end was lost.
const (
	partialMaxAge = 64 // frames (two minutes)
	partialMax    = 256
)

// reassemble joins an alphanumeric message sent as several fragments (the
// header's continuation flag is set on all but the last). It returns the
// page to emit, or false while fragments are still outstanding.
func (d *Decoder) reassemble(f fragment) (Message, bool) {
	if f.msg.Type != PageAlphanumeric {
		return f.msg, true
	}
	key := f.msg.Capcode
	p := d.partial[key]
	if f.frag == 3 {
		p = nil // a first fragment starts over, whatever was pending
		delete(d.partial, key)
	}
	if p != nil {
		p.msg.Text += f.msg.Text
		p.msg.Corrected += f.msg.Corrected
		p.msg.Phase, p.msg.Cycle, p.msg.Frame = f.msg.Phase, f.msg.Cycle, f.msg.Frame
		p.seen = d.absFrm
		if f.cont == 0 {
			delete(d.partial, key)
			return p.msg, true
		}
		return Message{}, false
	}
	if f.cont == 0 {
		return f.msg, true // complete, or a tail whose head was missed
	}
	if len(d.partial) < partialMax {
		d.partial[key] = &partialMsg{msg: f.msg, seen: d.absFrm}
	}
	return Message{}, false
}

func (d *Decoder) expirePartials() {
	for k, p := range d.partial {
		if d.absFrm-p.seen > partialMaxAge {
			delete(d.partial, k)
		}
	}
}

func bitRateIndex(m Mode) int {
	switch m.BitRate() {
	case 3200:
		return 1
	case 6400:
		return 2
	}
	return 0
}

// Stats reports cumulative counters.
type Stats struct {
	FramesSynced uint64 // frames whose frame information word decoded
	FIWErrors    uint64 // syncs whose frame information word did not
	PagesEmitted uint64
	WordsFailed  uint64 // uncorrectable words in decoded frames
	Frames1600   uint64 // frames by bit rate
	Frames3200   uint64
	Frames6400   uint64
}

func (d *Decoder) Stats() Stats {
	return Stats{
		FramesSynced: d.framesSynced.Load(),
		FIWErrors:    d.fiwErrors.Load(),
		PagesEmitted: d.pagesEmitted.Load(),
		WordsFailed:  d.wordsFailed.Load(),
		Frames1600:   d.framesByBitRte[0].Load(),
		Frames3200:   d.framesByBitRte[1].Load(),
		Frames6400:   d.framesByBitRte[2].Load(),
	}
}
