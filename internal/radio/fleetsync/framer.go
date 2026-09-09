package fleetsync

import "math/bits"

// Framer turns a demodulated FleetSync bit stream into decoded [Message]
// bursts. It hunts the 24-bit alternating preamble plus 16-bit sync word,
// captures the FrameBits payload that follows, hands it to [DecodeFrame],
// and invokes the OnMessage callback for each burst.
//
// This is the protocol-side framer only: the DSP layer above it (a
// 1200-baud FFSK demodulator + slicer, e.g. internal/dsp/demod.FFSK)
// feeds it one wire bit per Push call. Keeping the framer callback-based
// and free of the events bus / storage lets it be unit-tested in
// isolation; the bus/REST/web wiring is a separate, on-air-gated step
// (issue #437).
//
// Polarity: an FM discriminator can present the burst with either tone
// sense depending on tuning, so the sync hunt accepts both the sync word
// and its bitwise complement; when it locks on the complement the
// captured payload bits are inverted to recover the true data — the same
// approach internal/radio/mdc1200/receiver uses.
type Framer struct {
	onMsg func(Message)

	st       framerState
	reg      uint64 // sliding preamble+sync shift register (huntBits wide)
	inverted bool   // locked on the complemented sync word
	buf      [FrameBits]byte
	n        int // bits captured into buf

	burstsIn uint64 // sync words detected
	emitted  uint64 // messages passed to OnMessage
	badCRC   uint64 // frames whose block check failed
}

type framerState uint8

const (
	stateHunt    framerState = iota // searching for preamble + sync
	stateCapture                    // collecting the FrameBits payload
)

const (
	// preambleBits is the alternating lead-in checked ahead of the sync
	// word (three 0xAA bytes, per the reference).
	preambleBits = 24

	// huntBits is the width of the sync-hunt shift register.
	huntBits = preambleBits + SyncBits // 40

	// preambleMaxErrors tolerates a few sliced-bit errors in the
	// alternating preamble (matches the reference's tolerance).
	preambleMaxErrors = 3

	// syncMaxErrors tolerates at most one bit error in the 16-bit sync
	// word. The 16-bit block check downstream is the real gate, so this
	// stays tight to keep the false-lock rate low.
	syncMaxErrors = 1
)

const (
	huntMask     = (uint64(1) << huntBits) - 1
	preambleMask = (uint64(1) << preambleBits) - 1
	// preambleAlt is the 24-bit alternating pattern 0b1010...; its
	// complement covers the opposite phase.
	preambleAlt uint64 = 0xAAAAAA
)

// NewFramer constructs a Framer. onMsg is invoked once per decoded burst
// (including CRC-failed bursts, with Message.CRCOK == false, so a caller
// can choose to surface marginal signals); it must not be nil.
func NewFramer(onMsg func(Message)) *Framer {
	if onMsg == nil {
		panic("fleetsync: OnMessage callback is required")
	}
	return &Framer{onMsg: onMsg}
}

// Push feeds one sliced wire bit through the framer. Bits outside {0, 1}
// are masked to their low bit.
func (f *Framer) Push(bit byte) {
	bit &= 1
	switch f.st {
	case stateHunt:
		f.reg = ((f.reg << 1) | uint64(bit)) & huntMask
		if f.syncMatched() {
			f.st = stateCapture
			f.n = 0
			f.burstsIn++
		}
	case stateCapture:
		if f.inverted {
			bit ^= 1
		}
		f.buf[f.n] = bit
		f.n++
		if f.n == FrameBits {
			f.finish()
		}
	}
}

// syncMatched reports whether the shift register's low huntBits hold a
// valid alternating preamble followed by the sync word (or its
// complement). It sets f.inverted when locked on the complement.
func (f *Framer) syncMatched() bool {
	pre := (f.reg >> SyncBits) & preambleMask
	if !preambleOK(pre) {
		return false
	}
	syn := uint16(f.reg & 0xFFFF)
	if hamming16(syn, SyncWord) <= syncMaxErrors {
		f.inverted = false
		return true
	}
	if hamming16(syn, ^SyncWord) <= syncMaxErrors {
		f.inverted = true
		return true
	}
	return false
}

// preambleOK reports whether a 24-bit window is close enough to either
// phase of the alternating preamble.
func preambleOK(pre uint64) bool {
	dA := bits.OnesCount64(pre ^ preambleAlt)
	dB := bits.OnesCount64(pre ^ (^preambleAlt & preambleMask))
	return min(dA, dB) <= preambleMaxErrors
}

// finish decodes the captured payload, emits the burst, and returns the
// framer to the sync hunt with a cleared register so the just-decoded
// frame can't immediately re-trigger.
func (f *Framer) finish() {
	msg, ok := DecodeFrame(f.buf[:])
	if !ok {
		f.badCRC++
	}
	f.onMsg(msg)
	f.emitted++

	f.st = stateHunt
	f.reg = 0
	f.n = 0
	f.inverted = false
}

// hamming16 counts the differing bits between two 16-bit words.
func hamming16(a, b uint16) int { return bits.OnesCount16(a ^ b) }

// Stats reports cumulative framer counters for metrics / debugging.
type Stats struct {
	BurstsIn      uint64 // preamble+sync locks
	BurstsBadCRC  uint64 // decoded frames that failed the block check
	BurstsEmitted uint64 // messages passed to OnMessage
}

// Stats returns the current counters.
func (f *Framer) Stats() Stats {
	return Stats{
		BurstsIn:      f.burstsIn,
		BurstsBadCRC:  f.badCRC,
		BurstsEmitted: f.emitted,
	}
}
