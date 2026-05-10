package phase1

import "github.com/MattCheramie/GopherTrunk/internal/radio/framing"

// LDUDibitCount is the dibit length of a complete LDU on the air:
// 1728 bits / 2 = 864 dibits = 864 C4FM symbols. Used by the
// assembler as the trailing-dibit count from FSW start.
const LDUDibitCount = LDUTotalBits / 2

// LDUSink consumes a complete 1728-bit on-air LDU bit stream
// (one bit per byte, 0/1, MSB-first per dibit). The caller
// passes the LDU into ExtractVoiceFrames / ExtractLCESBlocks /
// ExtractLSDBlocks to get recorder-ready IMBE frames + metadata.
type LDUSink func(ldu []byte)

// LDUAssembler is a stateful stream consumer that turns a C4FM
// dibit stream (post-demodulation, post-symbol-clock-recovery)
// into complete 1728-bit LDU buffers ready for ExtractVoiceFrames.
//
// The state machine is:
//
//   - awaiting FSW (pending == -1): each incoming dibit slides
//     a 24-dibit window forward; when the window matches
//     FrameSyncWord within `tolerance` dibit positions, the
//     assembler records the FSW-start position and transitions
//     to "collecting".
//   - collecting (pending >= 0): each incoming dibit appends to
//     the buffer; once 864 dibits have arrived since the FSW
//     start, the assembler concatenates them into a 1728-bit
//     byte slice and hands it to the sink. The buffer is
//     compacted (dropping the consumed LDU) and the assembler
//     returns to "awaiting FSW".
//
// The assembler does NOT care about IQ demodulation, status
// symbols, or anything below the dibit layer — it expects the
// caller to have already produced clean {0,1,2,3} dibits via a
// C4FM demodulator + symbol clock recovery. It also doesn't
// know about NID parsing or LDU1 vs LDU2: the sink can inspect
// the emitted bits if it cares.
//
// Not safe for concurrent Process() calls. Instantiate one per
// per-call demod chain.
type LDUAssembler struct {
	sink      LDUSink
	tolerance int
	buf       []uint8
	pending   int // -1 = awaiting FSW; ≥0 = buf-relative FSW start
}

// NewLDUAssembler returns an LDUAssembler that forwards completed
// LDUs to sink. tolerance is the maximum dibit-position mismatch
// allowed when matching the 24-dibit FrameSyncWord; tolerance<0
// uses the SyncDetector default of 4 (one FSW out of every ~64
// captured bursts is statistically expected to land within 4
// mismatches by random chance, so set this lower for cleaner
// signals to reduce false-positives).
func NewLDUAssembler(sink LDUSink, tolerance int) *LDUAssembler {
	if tolerance < 0 {
		tolerance = 4
	}
	return &LDUAssembler{
		sink:      sink,
		tolerance: tolerance,
		buf:       make([]uint8, 0, 2*LDUDibitCount),
		pending:   -1,
	}
}

// Process feeds a chunk of dibits (0..3) into the assembler. The
// sink callback may be invoked zero or more times during the call
// — once per complete LDU detected in this chunk plus any LDU
// straddling a previous Process call's input.
func (a *LDUAssembler) Process(dibits []uint8) {
	for _, d := range dibits {
		a.buf = append(a.buf, d)
		// Detect FSW only when not already collecting an LDU.
		// Detection happens against the trailing 24 dibits of the
		// buffer — the most recent window.
		if a.pending < 0 && len(a.buf) >= 24 {
			if a.fswMismatch(a.buf[len(a.buf)-24:]) <= a.tolerance {
				a.pending = len(a.buf) - 24
			}
		}
		// Emit an LDU once 864 dibits have been buffered since the
		// FSW start.
		if a.pending >= 0 && len(a.buf)-a.pending >= LDUDibitCount {
			lduStart := a.pending
			lduDibits := a.buf[lduStart : lduStart+LDUDibitCount]
			ldu := framing.DibitsToBits(lduDibits)
			a.sink(ldu)
			// Compact: drop everything up to and including this LDU.
			// The next LDU's FSW must appear in subsequent dibits.
			tailStart := lduStart + LDUDibitCount
			copy(a.buf, a.buf[tailStart:])
			a.buf = a.buf[:len(a.buf)-tailStart]
			a.pending = -1
		}
	}
}

// Reset clears the assembler's internal state. Callers invoke
// it on stream re-sync (frame-loss recovery, channel retune,
// CallEnd cleanup) so the next dibit stream starts cleanly
// without a stale FSW match holding over.
func (a *LDUAssembler) Reset() {
	a.buf = a.buf[:0]
	a.pending = -1
}

// Buffered returns the number of dibits currently held in the
// assembler's internal buffer. Useful in tests to verify the
// state machine compacts correctly after each emission.
func (a *LDUAssembler) Buffered() int { return len(a.buf) }

// fswMismatch returns the number of dibit positions where the
// supplied 24-dibit window differs from FrameSyncWord. A return
// value > a.tolerance means "no match"; ≤ a.tolerance means
// "match within tolerance". Wrong-length inputs return a value
// guaranteed to exceed any reasonable tolerance.
func (a *LDUAssembler) fswMismatch(window []uint8) int {
	if len(window) != 24 {
		return 24
	}
	var mm int
	for i := range window {
		if window[i] != FrameSyncWord[i] {
			mm++
		}
	}
	return mm
}
