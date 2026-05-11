package mpt1327

// processState is the cross-call bit buffering + frame-alignment
// state the Process adapter holds. Lazily initialised.
type processState struct {
	buf       []byte
	aligned   bool
	off       int
	consecBad int
}

// codewordInfoBits is the count of MPT 1327 address-codeword
// information bits — 38. On-air codewords are 64 bits (38 info +
// 26 BCH(63,38) parity); the BCH FEC is consumed upstream and
// isn't implemented in this package yet (documented follow-up).
const codewordInfoBits = 38

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
	p.buf = append(p.buf, bits...)

	for {
		if !p.aligned {
			// Search forward for a recognised Address codeword.
			found := false
			for ; p.off+codewordInfoBits <= len(p.buf); p.off++ {
				w, _ := CodewordFromBits(p.buf[p.off : p.off+codewordInfoBits])
				if !isRecognisedAddressCodeword(w) {
					continue
				}
				c.Ingest(w)
				p.aligned = true
				p.off += codewordInfoBits
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
		if p.off+codewordInfoBits > len(p.buf) {
			break
		}
		w, _ := CodewordFromBits(p.buf[p.off : p.off+codewordInfoBits])
		c.Ingest(w)
		if isRecognisedAddressCodeword(w) {
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
		p.off += codewordInfoBits
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
