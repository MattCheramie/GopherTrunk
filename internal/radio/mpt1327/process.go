package mpt1327

import (
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// processState is the cross-call bit buffering + frame-alignment
// state the Process adapter holds. Lazily initialised.
type processState struct {
	buf       []byte
	aligned   bool
	off       int
	consecBad int
}

// codewordInfoBits is the count of MPT 1327 address-codeword
// information bits the existing 38-bit Codeword struct models.
// Used as the slice width under BCHOff (where the adapter reads
// pre-stripped 38-bit info directly).
const codewordInfoBits = 38

// codewordWireBits is the full on-wire MPT 1327 codeword length:
// 48 information bits + 15 BCH parity + 1 overall parity. Used as
// the slice width under BCHOn, after which BCHDecodeMPT1327
// recovers the 48-bit info field.
const codewordWireBits = 64

// cwscBits is the length of the MPT 1327 Codeword Synchronisation
// Code (CWSC) per the standard: a 16-bit bit pattern transmitted
// immediately before the first codeword of every message on the
// control channel. The pattern is `1100010011010111` (= 0xC4D7
// MSB-first); the adapter scans for it during alignment search
// so a fixture / capture whose payload doesn't happen to parse
// as a recognised opcode can still synchronise.
const cwscBits = 16

// cwscPattern[i] is bit i (MSB-first) of the 16-bit Codeword
// Synchronisation Code. Indexing bit-by-bit (rather than packing
// into a uint16) keeps the matcher's hot loop branchless against
// the byte-slice bit representation the receiver delivers.
var cwscPattern = [cwscBits]byte{
	1, 1, 0, 0, 0, 1, 0, 0,
	1, 1, 0, 1, 0, 1, 1, 1,
}

// maxConsecBad is how many consecutive recognised-codeword-failed
// frames the adapter tolerates while aligned before unlocking and
// re-searching. Long quiet periods + the occasional bit error keep
// the threshold modest.
const maxConsecBad = 8

// Process consumes a window of raw bits from the MPT 1327 receiver
// (the IQ → FFSK bit chain in internal/radio/mpt1327/receiver/) and
// drives the MPT 1327 state machine.
//
// Alignment is two-stage: the adapter first searches the buffered
// stream for the 16-bit Codeword Synchronisation Code (CWSC,
// `1100010011010111`) the standard mandates before every message,
// and locks immediately when a CWSC match is found. If no CWSC
// appears (synthesized fixtures often skip it), the adapter falls
// back to the legacy "first recognised codeword wins" alignment.
// Either path unlocks + restarts the search after maxConsecBad
// consecutive frames whose Type or Kind fail the recognised-
// codeword check.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime; the adapter doesn't use it directly today.
//
// Under BCHOn (the default), the alignment search picks a 64-bit
// window that passes the BCH(63, 38) check; under BCHOff the
// window is the pre-stripped 38-bit information field. CWSC
// detection is mode-independent.
func (c *ControlChannel) Process(bits []byte, baseIdx int) int {
	if c.proc == nil {
		c.proc = &processState{}
	}
	p := c.proc
	c.mu.Lock()
	mode := c.bchMode
	c.mu.Unlock()
	p.buf = append(p.buf, bits...)

	frameLen := codewordInfoBits
	if mode == BCHOn {
		frameLen = codewordWireBits
	}

	for {
		if !p.aligned {
			// Try CWSC match first — much more selective than the
			// parseable-codeword fallback. CWSC + a parseable
			// following codeword locks us into the message stream
			// at a known boundary; CWSC + a corrupted following
			// codeword still locks us but lets the consecBad
			// counter unlock on the next 8 bad frames.
			if cwscOff, ok := findCWSC(p.buf, p.off); ok && cwscOff+cwscBits+frameLen <= len(p.buf) {
				// Lock immediately at the start of the codeword
				// window that follows CWSC. The first codeword
				// after CWSC is always an Address codeword per
				// spec, so even if it doesn't parse as one of the
				// recognised Kinds, we trust the sync match more
				// than the parser and just consume forward.
				p.off = cwscOff + cwscBits
				if w, ok := c.parseCodeword(p.buf[p.off:p.off+frameLen], mode); ok {
					c.Ingest(w)
				}
				p.aligned = true
				p.off += frameLen
				p.consecBad = 0
				continue
			}
			// Fallback: search forward for the first parseable
			// recognised codeword. Under BCHOff the alignment
			// discriminator is "the 38-bit window parses as a
			// recognised Address codeword"; under BCHOn it's "the
			// 64-bit window passes the BCH check + the recovered
			// codeword parses as a recognised Address codeword".
			found := false
			for ; p.off+frameLen <= len(p.buf); p.off++ {
				w, ok := c.parseCodeword(p.buf[p.off:p.off+frameLen], mode)
				if !ok || !isRecognisedAddressCodeword(w) {
					continue
				}
				c.Ingest(w)
				p.aligned = true
				p.off += frameLen
				p.consecBad = 0
				found = true
				break
			}
			if !found {
				break
			}
			continue
		}
		// Aligned: pull next frame at fixed offset.
		if p.off+frameLen > len(p.buf) {
			break
		}
		w, ok := c.parseCodeword(p.buf[p.off:p.off+frameLen], mode)
		if ok {
			c.Ingest(w)
		}
		recognised := ok && isRecognisedAddressCodeword(w)
		if recognised {
			p.consecBad = 0
		} else {
			p.consecBad++
			if p.consecBad >= maxConsecBad {
				p.aligned = false
				p.consecBad = 0
				// Re-search starts at the position AFTER the
				// failed frame so we don't immediately re-lock to
				// the same bad alignment.
				p.off++
				continue
			}
		}
		p.off += frameLen
	}

	// Trim consumed bits from the front, keeping the unconsumed
	// tail so a frame straddling a chunk boundary still parses on
	// the next call.
	if p.off > 0 {
		drop := p.off
		if drop > len(p.buf) {
			drop = len(p.buf)
		}
		copy(p.buf, p.buf[drop:])
		p.buf = p.buf[:len(p.buf)-drop]
		p.off = 0
	}
	return baseIdx + len(bits)
}

// findCWSC scans buf[from:] for the 16-bit CWSC pattern and returns
// the absolute index of the first matching bit (i.e. the index in
// buf of the leading '1' of `1100010011010111`). Returns (0, false)
// when no exact match is present in the buffer.
//
// The scan is exact-match: MPT 1327 receivers tolerate a small
// number of bit errors in CWSC in practice, but a 0-error first
// pass keeps the alignment selectivity high. Bit-error tolerance
// is a follow-up that lands together with on-air capture
// calibration.
func findCWSC(buf []byte, from int) (int, bool) {
	if from < 0 {
		from = 0
	}
	end := len(buf) - cwscBits
	for i := from; i <= end; i++ {
		match := true
		for j := 0; j < cwscBits; j++ {
			if buf[i+j]&1 != cwscPattern[j] {
				match = false
				break
			}
		}
		if match {
			return i, true
		}
	}
	return 0, false
}

// parseCodeword turns a wire-bit window of length frameLen into a
// Codeword. Under BCHOff the window is 38 bits of pre-stripped
// information; under BCHOn it's 64 bits of on-wire codeword that
// gets BCH-checked + corrected before its 38-bit info field is
// extracted. Returns (codeword, false) when BCHOn rejects the
// window (uncorrectable codeword) so the alignment search keeps
// scanning.
func (c *ControlChannel) parseCodeword(window []byte, mode BCHMode) (Codeword, bool) {
	if mode != BCHOn {
		w, _ := CodewordFromBits(window)
		return w, true
	}
	if len(window) != codewordWireBits {
		return Codeword{}, false
	}
	// Pack 64 wire bits into a uint64 with bit i of uint64
	// = window[i]. This matches the layout BCHEncodeMPT1327 /
	// BCHDecodeMPT1327 expect (info at bits 0..47, BCH at
	// bits 48..62, parity at bit 63).
	var cw uint64
	for i := 0; i < codewordWireBits; i++ {
		if window[i]&1 != 0 {
			cw |= uint64(1) << uint(i)
		}
	}
	info48, errs := framing.BCHDecodeMPT1327(cw)
	if errs == -1 {
		return Codeword{}, false
	}
	// The 48-bit info is laid out per the framing primitive
	// convention with info48 bit i = wire bit i (LSB-first packing).
	// The Codeword struct's CodewordFromBits48 helper expects
	// MSB-first wire bits, so we expand info48 directly into a
	// 48-bit wire array first. This surfaces the spec's full
	// information set (Type + Prefix + Ident + Op + Function).
	wire48 := make([]byte, 48)
	for i := 0; i < 48; i++ {
		wire48[i] = byte((info48 >> uint(i)) & 1)
	}
	w, _ := CodewordFromBits48(wire48)
	return w, true
}

// isRecognisedAddressCodeword reports whether a parsed Codeword is
// an Address codeword (Type=0) whose Kind matches one of the
// trunking-relevant opcodes the state machine acts on. Used as
// the alignment discriminator since MPT 1327 has no fixed sync
// pattern.
func isRecognisedAddressCodeword(w Codeword) bool {
	if w.Type != TypeAddress {
		return false
	}
	switch w.Kind() {
	case KindAloha, KindAhoy, KindAhoyChan, KindGoToChan,
		KindAck, KindDisconnect, KindData, KindEmergency:
		return true
	}
	return false
}
