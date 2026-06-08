// Package flex decodes Motorola FLEX paging — the high-throughput
// successor to POCSAG used by most modern paging networks (hospital
// / EMS dispatch, commercial carriers). FLEX multiplexes many pagers
// onto one channel with time-framed, interleaved, FEC-protected data
// at 1600 / 3200 / 6400 bps (2- or 4-level FSK).
//
// This package owns the FLEX *logical* layer: given the BCH-decoded
// 21-bit codeword values of one phase of a frame, it walks the Block
// Information Word → address words → vector words → message words and
// produces typed Messages. The streaming bit-level decoder
// (sync acquisition, frame-info word, block de-interleaving, BCH) is
// in the Decoder type below; the DSP frontend is in
// internal/radio/pager/flex/receiver.
//
// Scope: this first slice targets the dominant 1600 bps / 2-level mode
// (single phase A) and alphanumeric / numeric / tone vectors. The
// 3200 / 6400 bps and 4-level multi-phase modes, long (2-word)
// addresses, and multi-frame message reassembly are follow-ups. The
// FLEX constants (sync marker, speed table, field layouts, idle
// values) are reproduced from the de-facto reference decoders; the
// de-interleave + BCH + field parse are validated end-to-end against a
// synthetic encoder, with exact on-wire bit order / generator left for
// real-capture calibration.
//
// References:
//   - Motorola FLEX protocol (ARIB STD-T44 / TIA-TSB-58 family).
//   - multimon-ng demod_flex.c — sync table, FIW / BIW / VIW layout.
package flex

import "strings"

// PageType is the FLEX vector message type (VIW bits 4..6).
type PageType uint8

const (
	PageSecure        PageType = 0
	PageShortInstruct PageType = 3
	PageTone          PageType = 2
	PageNumeric       PageType = 1
	PageNumberedNum   PageType = 6
	PageAlphanumeric  PageType = 5
	PageBinary        PageType = 4
	PageUnknown       PageType = 0xFF
)

// pageTypeFromVIW maps the 3-bit VIW type field to a PageType.
func pageTypeFromVIW(t uint32) PageType {
	switch t {
	case 0, 1:
		return PageNumeric
	case 2:
		return PageTone
	case 3:
		return PageShortInstruct
	case 4:
		return PageBinary
	case 5:
		return PageAlphanumeric
	case 6:
		return PageNumberedNum
	case 7:
		return PageAlphanumeric // hex/secure variants carry text too
	}
	return PageUnknown
}

// TypeName returns a stable lowercase label for a PageType — used as
// the pager_log encoding column value and the panel's display tag.
func (p PageType) TypeName() string {
	switch p {
	case PageNumeric, PageNumberedNum:
		return "numeric"
	case PageAlphanumeric:
		return "alpha"
	case PageTone:
		return "tone"
	case PageShortInstruct:
		return "short"
	case PageBinary:
		return "binary"
	}
	return "unknown"
}

// Message is one decoded FLEX page.
type Message struct {
	Capcode   uint32   // recovered pager address (capcode)
	Type      PageType // vector message type
	Text      string   // decoded message body (alphanumeric / numeric)
	Cycle     int      // FIW cycle number (0..15)
	Frame     int      // FIW frame number (0..127)
	Corrected int      // BCH bit errors corrected across this page's words
}

// idle codeword info values per the FLEX spec / multimon: an all-zero
// word and the all-ones 21-bit message field both mark "no address".
const (
	idleZero uint32 = 0x000000
	idleOnes uint32 = 0x1FFFFF
)

// DecodePhase walks one phase's BCH-decoded codeword info values
// (words[0..n-1], each a 21-bit value) and returns the pages it
// carries. corr is the per-word BCH error count (index-aligned;
// negative = uncorrectable). frame / cycle come from the FIW.
//
// The structure (multimon demod_flex.c): word 0 is the Block
// Information Word; addresses occupy words [aoffset, voffset); each
// address's vector word sits at voffset + (i-aoffset); the vector
// names the message-word span.
func DecodePhase(words []uint32, corr []int, frame, cycle int) []Message {
	if len(words) < 2 {
		return nil
	}
	biw := words[0]
	voffset := int((biw >> 10) & 0x3F)
	aoffset := int((biw>>8)&0x03) + 1
	if voffset <= aoffset || voffset >= len(words) {
		return nil
	}

	var out []Message
	for i := aoffset; i < voffset; i++ {
		aw := words[i]
		if aw == idleZero || aw == idleOnes {
			continue // idle / invalid address slot
		}
		vIdx := voffset + (i - aoffset)
		if vIdx >= len(words) {
			break
		}
		viw := words[vIdx]
		ptype := pageTypeFromVIW((viw >> 4) & 0x07)
		mw1 := int((viw >> 7) & 0x7F)
		mlen := int((viw >> 14) & 0x7F)

		msg := Message{
			Capcode: capcode(aw),
			Type:    ptype,
			Frame:   frame,
			Cycle:   cycle,
		}
		if corr != nil && i < len(corr) && corr[i] > 0 {
			msg.Corrected += corr[i]
		}
		if mw1 > 0 && mlen > 0 && mw1+mlen <= len(words) {
			mwords := words[mw1 : mw1+mlen]
			switch ptype {
			case PageAlphanumeric:
				msg.Text = decodeAlpha(mwords)
			case PageNumeric, PageNumberedNum:
				msg.Text = decodeNumeric(mwords)
			}
			if corr != nil {
				for k := mw1; k < mw1+mlen && k < len(corr); k++ {
					if corr[k] > 0 {
						msg.Corrected += corr[k]
					}
				}
			}
		}
		out = append(out, msg)
	}
	return out
}

// capcode recovers the pager address from a short (single-word)
// address codeword. Short FLEX addresses in [0x8001, 0x1E0000] map to
// capcode = word − 0x8000; values outside that range (long / special
// addresses) pass through unmapped for now.
func capcode(aw uint32) uint32 {
	if aw >= 0x8001 && aw <= 0x1E0000 {
		return aw - 0x8000
	}
	return aw
}

// decodeAlpha decodes FLEX alphanumeric message words: three 7-bit
// characters per 21-bit word, low group first. The very first
// character of the first word is a control / fragment header and is
// skipped; an ETX (0x03) ends the message.
func decodeAlpha(words []uint32) string {
	var sb strings.Builder
	for wi, w := range words {
		for c := 0; c < 3; c++ {
			ch := byte((w >> (uint(c) * 7)) & 0x7F)
			if wi == 0 && c == 0 {
				continue // first-word header character
			}
			if ch == 0x03 {
				return sb.String() // ETX — end of message
			}
			if ch >= 0x20 && ch < 0x7F {
				sb.WriteByte(ch)
			}
		}
	}
	return sb.String()
}

// numericMap is the FLEX 4-bit numeric symbol table.
var numericMap = []byte("0123456789 U -][")

// decodeNumeric decodes FLEX numeric message words: 4-bit symbols
// packed low-first across the 21-bit words (5 symbols/word + carry).
// A best-effort decode — the dominant operational traffic is
// alphanumeric; numeric is surfaced for completeness.
func decodeNumeric(words []uint32) string {
	var sb strings.Builder
	for _, w := range words {
		for s := 0; s < 5; s++ {
			sym := byte((w >> (uint(s) * 4)) & 0x0F)
			sb.WriteByte(numericMap[sym])
		}
	}
	return strings.TrimRight(sb.String(), " ")
}
