package flex

import (
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// SyncMarker is the 32-bit FLEX frame sync word ('A') that precedes the
// speed code at the head of every frame's Sync 1 (multimon
// FLEX_SYNC_MARKER).
const SyncMarker uint32 = 0xA6C6AAAA

// speed1600x2 is the 16-bit FLEX mode code for 1600 bps / 2-level FSK —
// the mode this decoder handles. Other modes (1600/4, 3200/2, 3200/4)
// are recognised enough to be skipped cleanly.
const speed1600x2 uint16 = 0x870C

// PhaseWords is the number of 32-bit codewords in one phase of a FLEX
// frame's data portion; PhaseBits is that in bits. At 1600/2 there is
// one phase (A).
const (
	PhaseWords = 88
	PhaseBits  = PhaseWords * 32 // 2816
)

type decState uint8

const (
	stHuntMarker decState = iota // scanning for the 32-bit sync marker
	stSpeed                      // reading the 16-bit mode code
	stFIW                        // reading the 32-bit frame info word
	stData                       // de-interleaving the 88-codeword phase
)

// Decoder turns a demodulated FLEX bit stream into pages. Push one wire
// bit at a time; completed pages are returned as they decode. Not safe
// for concurrent use — one Decoder per channel.
type Decoder struct {
	st       decState
	reg      uint32 // sliding 32-bit window for marker / FIW
	bitCnt   int    // bits collected in the current fixed-length field
	inverted bool   // locked on the complemented bit sense

	frame int
	cycle int

	c     int // data-bit counter within the phase (0..PhaseBits-1)
	words [PhaseWords]uint32

	// Counters for /metrics.
	framesSynced atomic.Uint64
	pagesEmitted atomic.Uint64
}

// NewDecoder returns a ready Decoder in the marker-hunt state.
func NewDecoder() *Decoder { return &Decoder{} }

// Push feeds one wire bit. Returns any pages completed by this bit.
func (d *Decoder) Push(bit byte) []Message {
	b := uint32(bit & 1)
	d.reg = (d.reg << 1) | b

	switch d.st {
	case stHuntMarker:
		if d.reg == SyncMarker {
			d.inverted = false
			d.enterSpeed()
		} else if (^d.reg) == SyncMarker {
			d.inverted = true
			d.enterSpeed()
		}
		return nil
	case stSpeed:
		d.bitCnt++
		if d.bitCnt < 16 {
			return nil
		}
		code := uint16(d.reg & 0xFFFF)
		if d.inverted {
			code = ^code
		}
		if code == speed1600x2 {
			d.st = stFIW
			d.bitCnt = 0
		} else {
			d.reset() // unsupported / mis-synced mode
		}
		return nil
	case stFIW:
		d.bitCnt++
		if d.bitCnt < 32 {
			return nil
		}
		fiw := d.reg
		if d.inverted {
			fiw = ^fiw
		}
		d.cycle = int((fiw >> 4) & 0x0F)
		d.frame = int((fiw >> 8) & 0x7F)
		d.framesSynced.Add(1)
		d.enterData()
		return nil
	case stData:
		d.placeDataBit(b)
		d.c++
		if d.c >= PhaseBits {
			msgs := d.decodePhase()
			d.reset()
			return msgs
		}
		return nil
	}
	return nil
}

func (d *Decoder) enterSpeed() {
	d.st = stSpeed
	d.bitCnt = 0
}

func (d *Decoder) enterData() {
	d.st = stData
	d.c = 0
	for i := range d.words {
		d.words[i] = 0
	}
}

// placeDataBit de-interleaves one data bit into its codeword. The FLEX
// block interleave (multimon): bit counter c maps to codeword index
// ((c>>5)&0xFFF8)|(c&7) and bit position (c>>3)&31 within that word.
func (d *Decoder) placeDataBit(b uint32) {
	bit := b
	if d.inverted {
		bit ^= 1
	}
	if bit == 0 {
		return
	}
	word := ((d.c >> 5) & 0xFFF8) | (d.c & 7)
	pos := (d.c >> 3) & 31
	if word < PhaseWords {
		d.words[word] |= 1 << uint(pos)
	}
}

// decodePhase BCH-decodes the assembled codewords and runs the logical
// FLEX parse, returning the pages in this phase.
func (d *Decoder) decodePhase() []Message {
	infos := make([]uint32, PhaseWords)
	corr := make([]int, PhaseWords)
	for i := 0; i < PhaseWords; i++ {
		info, errs := framing.FLEXBCHDecode32(d.words[i])
		infos[i] = info
		corr[i] = errs
	}
	msgs := DecodePhase(infos, corr, d.frame, d.cycle)
	d.pagesEmitted.Add(uint64(len(msgs)))
	return msgs
}

// reset returns the decoder to marker-hunt.
func (d *Decoder) reset() {
	d.st = stHuntMarker
	d.bitCnt = 0
	d.c = 0
}

// Stats reports cumulative counters.
type Stats struct {
	FramesSynced uint64
	PagesEmitted uint64
}

func (d *Decoder) Stats() Stats {
	return Stats{
		FramesSynced: d.framesSynced.Load(),
		PagesEmitted: d.pagesEmitted.Load(),
	}
}
