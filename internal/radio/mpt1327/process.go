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

// maxConsecBad is how many consecutive recognised-codeword-failed
// frames the adapter tolerates while aligned before unlocking and
// re-searching. Long quiet periods + the occasional bit error keep
// the threshold modest.
const maxConsecBad = 8

// Process consumes a window of raw bits from the MPT 1327 receiver
// (the IQ → FFSK bit chain in internal/radio/mpt1327/receiver/) and
// drives the MPT 1327 state machine.
//
// MPT 1327 has no fixed inter-codeword sync pattern — codewords
// flow back-to-back at 1200 bps. The adapter searches the buffered
// stream for the first 38-bit window that parses as a recognised
// Address codeword (Aloha / AhoyChan / GoToChan), commits to that
// alignment, and follows it forward — unlocking + restarting the
// search after maxConsecBad consecutive frames whose Type or Kind
// fail the recognised-codeword check.
//
// baseIdx is the absolute bit index of bits[0] across the stream
// lifetime; the adapter doesn't use it directly today.
//
// The 64-bit on-air codeword + BCH(63,38) FEC isn't reversed
// here — the adapter reads 38 info bits straight from the wire.
// Real on-air signals require the BCH layer (a documented
// follow-up); until it ships the adapter sync-aligns on noise-
// free test fixtures but typically fails to lock on captured
// MPT 1327 traffic.
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
			// Search forward for a recognised codeword. Under
			// BCHOff the alignment discriminator is "the 38-bit
			// window parses as a recognised Address codeword";
			// under BCHOn it's "the 64-bit window passes the BCH
			// check + the recovered codeword parses as a
			// recognised Address codeword".
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
	// Extract the 38-bit info field expected by CodewordFromBits.
	// The 48-bit info is laid out as wire bits 0..47:
	//   wire 0..20  = Type (1) + Prefix (7) + Ident (13) — 21 bits
	//   wire 21..30 = Op (10) — dropped (not modelled by Codeword)
	//   wire 31..47 = Function (17)
	wire38 := make([]byte, 38)
	for i := 0; i < 21; i++ {
		wire38[i] = byte((info48 >> uint(i)) & 1)
	}
	for i := 0; i < 17; i++ {
		wire38[21+i] = byte((info48 >> uint(31+i)) & 1)
	}
	w, _ := CodewordFromBits(wire38)
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
