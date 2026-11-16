// Package flex decodes Motorola FLEX paging — the high-throughput successor
// to POCSAG used by most modern paging networks (hospital / EMS dispatch,
// commercial carriers).
//
// A FLEX frame is 1.875 s long and always opens at 1600 symbols/s, 2-level
// FSK; the sync word names the mode the REST of the frame is sent in:
//
//	code    symbols/s  levels  bit rate  phases
//	0x870C    1600        2    1600 bps  A
//	0x7B18    3200        2    3200 bps  A C
//	0xB068    1600        4    3200 bps  A B
//	0xDEA0    3200        4    6400 bps  A B C D
//	0x4C7C    3200        4    6400 bps  A B C D   (ReFLEX)
//
// Each phase is an independent 1600 bps stream of 88 interleaved BCH(31,21)
// words with its own block information word, addresses, vectors and
// messages, so a 6400 bps frame carries four times the pages of a 1600 bps
// one. The mode is read per frame: a network that mixes rates is followed
// frame by frame with no configuration.
//
// This file is the logical layer (one phase's 88 words → pages); decoder.go
// is the frame layer (symbols → phases); synth.go builds frames for tests;
// the DSP front end is internal/radio/pager/flex/receiver.
//
// Reference: multimon-ng demod_flex.c / demod_flex_next.c (GPL — read for the
// protocol facts, nothing ported), whose constants cite ARIB STD-43A. The
// 1600 bps path is pinned on literal on-air frames (realair_test.go); the
// 3200 / 6400 bps paths follow the same reference and are synthetic-verified
// only — a capture of a multi-rate network is the on-air gate.
//
// Not decoded: binary / secure message bodies, group-call enrolment (short
// instruction vectors), the BIW system-information words (time, SSID).
package flex

import (
	"fmt"
	"strings"
)

// PageType is the FLEX vector type (vector word bits 4..6).
type PageType uint8

const (
	PageSecure        PageType = 0
	PageShortInstruct PageType = 1
	PageTone          PageType = 2 // "short message / tone only"
	PageNumeric       PageType = 3
	PageSpecialNum    PageType = 4
	PageAlphanumeric  PageType = 5
	PageBinary        PageType = 6
	PageNumberedNum   PageType = 7
	PageUnknown       PageType = 0xFF
)

// TypeName returns a stable lowercase label for a PageType — used as
// the pager_log encoding column value and the panel's display tag.
func (p PageType) TypeName() string {
	switch p {
	case PageNumeric, PageSpecialNum, PageNumberedNum:
		return "numeric"
	case PageAlphanumeric:
		return "alpha"
	case PageTone:
		return "tone"
	case PageShortInstruct:
		return "short"
	case PageBinary:
		return "binary"
	case PageSecure:
		return "secure"
	}
	return "unknown"
}

// Message is one decoded FLEX page.
type Message struct {
	Capcode   uint64   // pager address (long addresses exceed 32 bits)
	Type      PageType // vector type
	Text      string   // message body (alphanumeric / numeric)
	Mode      Mode     // the frame's transmission mode
	Phase     byte     // 'A'..'D'
	Cycle     int      // FIW cycle number (0..14)
	Frame     int      // FIW frame number (0..127)
	Corrected int      // BCH bit errors corrected across this page's words
}

// String is a one-line rendering in multimon's familiar shape.
func (m Message) String() string {
	return fmt.Sprintf("%d/%d %02d.%03d.%c [%09d] %s %s",
		m.Mode.Baud, m.Mode.Levels, m.Cycle, m.Frame, m.Phase, m.Capcode,
		m.Type.TypeName(), m.Text)
}

const idleOnes uint32 = 0x1FFFFF

// wordChecksumOK reports whether a 21-bit word's nibble checksum holds: the
// five nibbles plus bit 20 sum to 0xF modulo 16. The frame information word,
// the block information word and every vector word carry it.
func wordChecksumOK(w uint32) bool {
	sum := (w & 0xF) + ((w >> 4) & 0xF) + ((w >> 8) & 0xF) +
		((w >> 12) & 0xF) + ((w >> 16) & 0xF) + ((w >> 20) & 1)
	return sum&0xF == 0xF
}

// withChecksum fills bits 0..3 of w so that wordChecksumOK holds.
func withChecksum(w uint32) uint32 {
	w &^= 0xF
	sum := ((w >> 4) & 0xF) + ((w >> 8) & 0xF) + ((w >> 12) & 0xF) +
		((w >> 16) & 0xF) + ((w >> 20) & 1)
	return w | ((0xF - (sum & 0xF)) & 0xF)
}

// fragment is one page's parse before reassembly.
type fragment struct {
	msg Message
	// frag is the fragment number (3 = first / only); cont is set when more
	// fragments follow in a later frame. Only alphanumeric pages fragment.
	frag, cont int
}

// decodePhase walks one phase's 88 BCH-decoded information words. errs is the
// per-word BCH result (index-aligned, negative = uncorrectable).
//
// Layout: word 0 is the block information word; addresses occupy
// [aoffset, voffset); the vector for the address at i sits at
// voffset + (i - aoffset) and names the message words. A long address uses
// two address words and two vector words.
func decodePhase(words []uint32, errs []int) []fragment {
	if len(words) < PhaseWords || len(errs) < PhaseWords || errs[0] < 0 {
		return nil
	}
	biw := words[0]
	if biw == 0 || biw == idleOnes {
		return nil
	}
	aoffset := int((biw>>8)&0x03) + 1
	voffset := int((biw >> 10) & 0x3F)
	if voffset < aoffset || voffset+(voffset-aoffset) > PhaseWords {
		return nil
	}

	bad := func(i int) bool { return i < 0 || i >= PhaseWords || errs[i] < 0 }
	fixed := func(i int) int {
		if i >= 0 && i < PhaseWords && errs[i] > 0 {
			return errs[i]
		}
		return 0
	}

	var out []fragment
	for i := aoffset; i < voffset; i++ {
		j := voffset + (i - aoffset)
		if bad(i) {
			continue
		}
		aw := words[i]
		if aw == 0 || aw == idleOnes {
			continue
		}
		long := (aw >= 0x000001 && aw <= 0x008000) || (aw >= 0x1F7FFF && aw <= 0x1FFFFE)
		capcode := uint64(aw) - 0x8000
		corrected := fixed(i)
		if long {
			if i+1 >= voffset || bad(i+1) {
				i++
				continue
			}
			var ok bool
			capcode, ok = longCapcode(aw, words[i+1])
			corrected += fixed(i + 1)
			i++ // the second address word (and its vector slot) is consumed
			if !ok {
				continue
			}
		}
		if bad(j) {
			continue
		}
		viw := words[j]
		if !wordChecksumOK(viw) {
			// Every vector word carries the checksum; one that fails is a
			// mis-corrected word (or a tone-only address with no vector).
			continue
		}
		f := fragment{frag: 3}
		f.msg = Message{
			Capcode:   capcode,
			Type:      PageType((viw >> 4) & 0x07),
			Corrected: corrected + fixed(j),
		}
		switch f.msg.Type {
		case PageShortInstruct:
			continue // group enrolment — not a page
		case PageTone:
			f.msg.Text = decodeShortMessage(words, viw, j, long)
		case PageNumeric, PageSpecialNum, PageNumberedNum:
			text, ok := decodeNumeric(words, errs, viw, j, long, f.msg.Type)
			if !ok {
				continue
			}
			f.msg.Text = text
		default: // alphanumeric, secure, binary: a header word + a body
			mw1 := int((viw >> 7) & 0x7F)
			n := int((viw >> 14) & 0x7F)
			hdr := mw1
			if long {
				hdr = j + 1 // the header rides in the second vector word
			} else {
				mw1++
			}
			if n >= 1 {
				n--
			}
			if bad(hdr) || mw1 >= PhaseWords || (mw1 == 0 && n == 0) {
				continue
			}
			if mw1+n > PhaseWords {
				n = PhaseWords - mw1
			}
			f.msg.Corrected += fixed(hdr)
			for k := mw1; k < mw1+n; k++ {
				f.msg.Corrected += fixed(k)
			}
			if f.msg.Type == PageAlphanumeric {
				f.frag = int((words[hdr] >> 11) & 0x03)
				f.cont = int((words[hdr] >> 10) & 0x01)
				f.msg.Text = decodeAlpha(words[mw1:mw1+n], errs[mw1:mw1+n], f.frag == 3)
			}
		}
		out = append(out, f)
	}
	return out
}

// longCapcode combines the two words of a long address (the reference's
// three address sets, ARIB STD-43A table 3.8.1-1).
func longCapcode(w1, w2 uint32) (uint64, bool) {
	a, b := uint64(w1), uint64(w2)
	var c uint64
	switch {
	case a >= 1 && a <= 32768 && b >= 2064383 && b <= 2097150:
		c = a + (2097151-b)*32768 + 2068480
	case a >= 1 && a <= 32768 && b >= 1966081 && b <= 2031616:
		c = a + (b-1933312)*32768 + 2068480
	case a >= 2064383 && a <= 2097150 && b >= 1966081 && b <= 2031616:
		c = (a - 2064383) + (b-1867776)*32768 + 2068479
	default:
		return 0, false
	}
	return c, c <= maxCapcode
}

// maxCapcode is the largest address the protocol defines.
const maxCapcode = 4297068542

// decodeAlpha unpacks 7-bit characters, three per word, low group first. In a
// message's first fragment the first character slot of the first body word
// holds the message signature, not text. An uncorrectable word reads "???".
func decodeAlpha(words []uint32, errs []int, first bool) string {
	var sb strings.Builder
	for k, w := range words {
		if errs[k] < 0 {
			sb.WriteString("???")
			continue
		}
		for c := 0; c < 3; c++ {
			if k == 0 && c == 0 && first {
				continue
			}
			ch := byte((w >> (uint(c) * 7)) & 0x7F)
			switch {
			case ch == '\n' || ch == '\r' || ch == '\t':
				sb.WriteByte(' ')
			case ch >= 0x20 && ch < 0x7F:
				sb.WriteByte(ch)
			}
		}
	}
	return sb.String()
}

// flexBCD is the 4-bit numeric alphabet; 0xC is the space that also pads the
// last word.
const flexBCD = "0123456789 U -]["

// decodeNumeric reads a numeric page: a bit stream over up to eight words,
// LSB first, that opens with a 2-bit (10-bit for numbered numeric) header and
// then carries 4-bit digits. The first word is the message's first word for a
// short address and the second vector word for a long one.
func decodeNumeric(words []uint32, errs []int, viw uint32, j int, long bool, t PageType) (string, bool) {
	w1 := int((viw >> 7) & 0x7F)
	w2 := int((viw>>14)&0x07) + w1
	var dw uint32
	if long {
		if j+1 >= PhaseWords || errs[j+1] < 0 {
			return "", false
		}
		dw = words[j+1]
	} else {
		if w1 >= PhaseWords || errs[w1] < 0 {
			return "", false
		}
		dw = words[w1]
		w1++
		w2++
	}
	if w1 > PhaseWords || w2 > PhaseWords {
		return "", false
	}
	count := 4 + 2
	if t == PageNumberedNum {
		count = 4 + 10
	}
	var sb strings.Builder
	var digit byte
	for i := w1; i <= w2; i++ {
		for k := 0; k < 21; k++ {
			digit = (digit >> 1) & 0x0F
			if dw&1 != 0 {
				digit |= 0x08
			}
			dw >>= 1
			if count--; count == 0 {
				sb.WriteByte(flexBCD[digit])
				count = 4
			}
		}
		if i >= PhaseWords || errs[i] < 0 {
			break
		}
		dw = words[i]
	}
	return strings.TrimRight(sb.String(), " "), true
}

// decodeShortMessage reads the digits a short-message vector carries in
// place of a message pointer (three, plus five more in a long address's
// second vector word). Sub-type 0 is the numeric form; others are tone-only.
func decodeShortMessage(words []uint32, viw uint32, j int, long bool) string {
	if (viw>>7)&0x03 != 0 {
		return ""
	}
	var sb strings.Builder
	for i := uint(9); i <= 17; i += 4 {
		sb.WriteByte(flexBCD[(viw>>i)&0x0F])
	}
	if long && j+1 < PhaseWords {
		for i := uint(0); i <= 16; i += 4 {
			sb.WriteByte(flexBCD[(words[j+1]>>i)&0x0F])
		}
	}
	return strings.TrimRight(sb.String(), " ")
}
