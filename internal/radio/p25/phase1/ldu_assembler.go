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

// DibitSink consumes the raw stream of dibits the receiver
// decodes from IQ, before LDU framing. baseIdx is the absolute
// dibit index of dibits[0] across the stream lifetime —
// monotonically non-decreasing across calls, and reset to 0 by
// Receiver.Reset so a retune produces a fresh baseline. Wire
// this into ControlChannel.Process (which keeps its own
// FSW / NID / TSBK pipeline) to drive the control-channel state
// machine in parallel with — or instead of — the LDU voice path.
type DibitSink func(dibits []uint8, baseIdx int)

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
	// frameLen is the on-air dibit length of the data unit at pending, read
	// from its NID once nidPrefixDibits have arrived; 0 = not read yet.
	frameLen int
}

// On-air dibit lengths of the non-LDU data units a voice channel carries
// (TIA-102.BAAA; OP25 p25_framer: HDU 792 bits, TDU 144, TDULC 432). Each is
// a whole number of the LDU's 72-bit status-symbol blocks.
const (
	hduDibitCount   = 792 / 2
	tduDibitCount   = 144 / 2
	tdulcDibitCount = 432 / 2
)

// nidPrefixDibits is how much of a data unit carries its frame sync + NID:
// 48 + 64 bits, plus the status symbol after bit 70 — 114 bits.
const nidPrefixDibits = (LDUFrameSyncBits + LDUNIDBits + 2) / 2

// dataUnitDibits reads the NID that follows a frame sync and returns the data
// unit's on-air length in dibits. An undecodable NID, or a DUID with no fixed
// length here (LDU, or anything unexpected on a voice channel), returns the
// LDU length — the assembler's behaviour before it read NIDs.
func dataUnitDibits(prefix []uint8) int {
	bits := framing.DibitsToBits(prefix)
	payload := make([]byte, 0, LDUFrameSyncBits+LDUNIDBits)
	payload = append(payload, bits[:LDUStatusInterval]...)
	payload = append(payload, bits[LDUStatusInterval+2:]...)
	nid, _, err := ParseNID(payload[lduNIDOffset : lduNIDOffset+LDUNIDBits])
	if err != nil {
		return LDUDibitCount
	}
	switch nid.DUID {
	case DUIDHeader:
		return hduDibitCount
	case DUIDTerminator:
		return tduDibitCount
	case DUIDTerminatorWithLC:
		return tdulcDibitCount
	}
	return LDUDibitCount
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
				a.frameLen = 0
			}
		}
		if a.pending < 0 {
			continue
		}
		have := len(a.buf) - a.pending
		// Read the NID as soon as it has arrived, so a data unit shorter
		// than an LDU doesn't swallow the frame sync of the one after it
		// (#1242: every HDU swallowed its over's first LDU1).
		if a.frameLen == 0 && have >= nidPrefixDibits {
			a.frameLen = dataUnitDibits(a.buf[a.pending : a.pending+nidPrefixDibits])
		}
		// An HDU carries no voice: drop it once complete and hunt for the
		// LDU1 that follows. Handing it to the sink would decode header
		// bits as IMBE frames.
		if a.frameLen == hduDibitCount && have >= hduDibitCount {
			a.consume(a.pending + hduDibitCount)
			continue
		}
		// Emit an LDU-sized window once 864 dibits have been buffered since
		// the FSW start. Terminators are emitted in the same window (the
		// sink's contract is a 1728-bit buffer and it reads only their
		// NID/LC), but only their own length is consumed, and the dibits
		// after them are hunted again for the next frame sync.
		if have >= LDUDibitCount {
			lduStart := a.pending
			ldu := framing.DibitsToBits(a.buf[lduStart : lduStart+LDUDibitCount])
			a.sink(ldu)
			n := a.frameLen
			if n == 0 {
				n = LDUDibitCount
			}
			if n == LDUDibitCount {
				a.consume(lduStart + LDUDibitCount)
				continue
			}
			rest := append([]uint8(nil), a.buf[lduStart+n:]...)
			a.buf = a.buf[:0]
			a.pending = -1
			a.frameLen = 0
			a.Process(rest)
		}
	}
}

// consume drops the first n buffered dibits and returns to hunting for a
// frame sync in the dibits that arrive next.
func (a *LDUAssembler) consume(n int) {
	copy(a.buf, a.buf[n:])
	a.buf = a.buf[:len(a.buf)-n]
	a.pending = -1
	a.frameLen = 0
}

// Reset clears the assembler's internal state. Callers invoke
// it on stream re-sync (frame-loss recovery, channel retune,
// CallEnd cleanup) so the next dibit stream starts cleanly
// without a stale FSW match holding over.
func (a *LDUAssembler) Reset() {
	a.buf = a.buf[:0]
	a.pending = -1
	a.frameLen = 0
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
