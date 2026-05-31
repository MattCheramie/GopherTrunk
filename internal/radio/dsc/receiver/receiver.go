// Package receiver frames a demodulated DSC bit stream: it slides a
// 10-bit window through the bits, uses the BCH(10,7) syndrome check to
// lock onto the phasing-sequence DX characters (value 125, repeating
// every 20 bits), then samples the DX cadence to recover the 7-bit
// data symbols, detects the end-of-sequence character, and hands the
// symbol run to internal/radio/dsc.Decode. One events.KindDSCMessage
// publishes per decoded sequence.
//
// The DSP layer above this package (internal/radio/dsc/ffsk: FM demod
// → FFSK tone discrimination at 1300/2100 Hz → symbol-timing recovery
// → slicer) hands the receiver its bit stream. From bits down through
// bus event is what this package owns.
//
// One Receiver instance per DSC channel. The Push(bit) method is the
// single entry point — call it once per wire bit. DSC is direct FSK
// (no NRZI line code, unlike AX.25 / AIS), so the slicer's bit feeds
// the framer directly.
//
// # DX / RX time diversity
//
// DSC transmits each information character twice: once in the DX
// position and again in the RX position, the two interleaved at the
// 10-bit character cadence so the DX characters land on a 20-bit
// grid. This first slice takes the simplest viable path the handoff
// describes — lock the DX grid via the repeating phasing character
// and read DX only, dropping the RX copies. Comparing DX against its
// RX twin to recover characters that fail their BCH check is a
// follow-up (it lifts decode yield on marginal signals); it is not
// required for clean captures.
//
// # Polarity
//
// An FM discriminator can present the tones with either sense
// depending on tuning, so the phasing hunt accepts both the DX
// character and its bitwise complement; when it locks on the
// complement the sampled characters are inverted to recover the true
// symbols. This mirrors the MDC1200 receiver's approach.
//
// # On-wire calibration
//
// The 10-bit characters are assembled most-significant-bit-first into
// the codeword integer layout internal/radio/dsc/bch.go defines
// (7 data bits then 3 check bits). That convention is validated
// end-to-end against the FFSK modulator in the frontend test; the
// exact wire bit order, tone-frequency sense, and DX/RX offset of
// real ITU-R M.493 traffic should be confirmed against a captured
// signal before relying on field decodes.
package receiver

import (
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dsc"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// charBits is the DSC character length: 7 data bits + 3 BCH check
// bits per ITU-R M.493-15 §3.4.
const charBits = 10

// dxStride is the DX-character spacing in bits. DX and RX characters
// interleave at the 10-bit cadence, so successive DX characters land
// 20 bits apart.
const dxStride = 2 * charBits

// charMask isolates the low charBits of the sliding window.
const charMask = (uint16(1) << charBits) - 1

// phasingDX is the DSC phasing-sequence DX character (ITU-R M.493-15
// §3.2.1). It repeats on the DX grid through the phasing run, giving
// the receiver a stable pattern to lock the character cadence and
// polarity onto.
const phasingDX = 125

// maxSeqSyms caps the symbols collected for one sequence. A runaway
// capture with no end-of-sequence character is abandoned and the
// receiver returns to hunting rather than growing without bound. The
// longest standard DSC sequence (geographic / position call) is well
// under this.
const maxSeqSyms = 40

// eosSyms are the DSC end-of-sequence characters: 117 (acknowledge
// required), 122 (acknowledge), 127 (end of sequence). Any of them on
// the DX grid closes the sequence.
var eosSyms = map[byte]bool{117: true, 122: true, 127: true}

type state uint8

const (
	stateHunt   state = iota // searching for the phasing DX cadence
	stateLocked              // DX grid + polarity established, reading symbols
)

// Receiver consumes a DSC bit stream and publishes decoded messages
// on the shared events bus. Zero-value Receivers are NOT usable —
// call New.
type Receiver struct {
	bus *events.Bus

	// dropBadFCS, when true, discards sequences in which any DX
	// character failed its BCH check (DX-only decode cannot recover
	// those). Defaults to false — the sequence still publishes so the
	// panel can flag a marginal decode.
	dropBadFCS bool

	st    state
	reg   uint16 // sliding charBits window, newest bit in the LSB
	nbits int    // bits pushed since construction (monotonic)
	full  bool   // window has seen at least charBits bits

	// Hunt state: the most recent phasing-DX sighting, used to
	// confirm the 20-bit DX cadence.
	lastDXBit int
	lastDXPol bool
	dxSeen    bool

	// Lock state.
	inverted bool // locked on the complemented character sense
	lockBit  int  // nbits at the locked DX boundary; DX grid is lockBit + k·dxStride
	started  bool // saw the first non-phasing DX character (format specifier)
	syms     []byte
	badInSeq int // DX characters that failed BCH in the current sequence

	// Counters surfaced for /metrics.
	seqIn   atomic.Uint64 // sequences whose EOS was reached
	seqBad  atomic.Uint64 // sequences with ≥1 BCH-failed DX character
	seqEmit atomic.Uint64 // events published to the bus
}

// Options configures a Receiver.
type Options struct {
	// Bus is required — messages publish onto KindDSCMessage.
	Bus *events.Bus

	// DropBadFCS discards sequences with a BCH-failed DX character
	// when true.
	DropBadFCS bool
}

// New constructs a Receiver. Panics if opts.Bus is nil — receivers
// without a bus have nowhere to publish.
func New(opts Options) *Receiver {
	if opts.Bus == nil {
		panic("dsc/receiver: events.Bus is required")
	}
	return &Receiver{bus: opts.Bus, dropBadFCS: opts.DropBadFCS}
}

// Push feeds one wire bit through the framer. Bits outside {0, 1} are
// clamped to 1.
func (r *Receiver) Push(bit byte) {
	if bit > 1 {
		bit = 1
	}
	r.reg = ((r.reg << 1) | uint16(bit)) & charMask
	r.nbits++
	if !r.full {
		if r.nbits >= charBits {
			r.full = true
		} else {
			return
		}
	}
	switch r.st {
	case stateHunt:
		r.hunt()
	case stateLocked:
		r.collect()
	}
}

// hunt looks for two phasing-DX characters one DX stride apart at the
// same polarity. That confirms the DX cadence and the tone sense, and
// establishes the DX sampling grid.
func (r *Receiver) hunt() {
	pol, ok := r.windowIsPhasingDX()
	if !ok {
		return
	}
	if r.dxSeen && r.lastDXPol == pol && r.nbits-r.lastDXBit == dxStride {
		// Cadence confirmed — lock onto this DX boundary.
		r.st = stateLocked
		r.inverted = pol
		r.lockBit = r.nbits
		r.started = false
		r.syms = r.syms[:0]
		r.badInSeq = 0
		r.dxSeen = false
		return
	}
	r.dxSeen = true
	r.lastDXBit = r.nbits
	r.lastDXPol = pol
}

// windowIsPhasingDX reports whether the current window decodes to the
// phasing-DX character under either polarity. The bool result is the
// polarity (true = inverted) when ok.
func (r *Receiver) windowIsPhasingDX() (pol bool, ok bool) {
	if data, good := dsc.BCHCheck(r.reg & charMask); good && byte(data) == phasingDX {
		return false, true
	}
	if data, good := dsc.BCHCheck((^r.reg) & charMask); good && byte(data) == phasingDX {
		return true, true
	}
	return false, false
}

// collect samples the DX grid: at every DX boundary it BCH-checks the
// window, skips leading phasing characters, then appends data symbols
// until the end-of-sequence character closes the run.
func (r *Receiver) collect() {
	if (r.nbits-r.lockBit)%dxStride != 0 {
		return
	}
	cw := r.reg & charMask
	if r.inverted {
		cw = (^r.reg) & charMask
	}
	data, ok := dsc.BCHCheck(cw)
	sym := byte(data)

	if !r.started {
		if sym == phasingDX {
			return // still in the phasing run
		}
		if !ok {
			// Noise on the DX grid before the format specifier —
			// abandon the lock and re-hunt.
			r.reset()
			return
		}
		r.started = true
		r.syms = r.syms[:0]
		r.badInSeq = 0
	}

	if !ok {
		r.badInSeq++
	}
	r.syms = append(r.syms, sym)

	if eosSyms[sym] {
		r.finish()
		return
	}
	if len(r.syms) >= maxSeqSyms {
		r.reset() // runaway sequence, no EOS — give up the lock
	}
}

// finish decodes the collected symbol run, publishes it, and returns
// the receiver to hunting.
func (r *Receiver) finish() {
	r.seqIn.Add(1)
	if r.badInSeq > 0 {
		r.seqBad.Add(1)
	}
	if r.dropBadFCS && r.badInSeq > 0 {
		r.reset()
		return
	}

	m := dsc.Decode(r.syms)
	out := storage.DSCMessage{
		ReceivedAt: time.Now(),
		Format:     dsc.FormatString(m.Format),
		Category:   dsc.CategoryString(m.Category),
		SelfMMSI:   m.SelfMMSI,
		TargetMMSI: m.TargetMMSI,
		Nature:     dsc.NatureString(m.Nature),
		TimeUTC:    m.TimeUTC,
		Body:       m.String(),
		RawHex:     m.RawSymbols,
	}
	if m.Position != nil {
		out.Latitude = m.Position.Latitude
		out.Longitude = m.Position.Longitude
		out.HasPosition = m.Position.HasPosition
	}
	r.bus.Publish(events.Event{Kind: events.KindDSCMessage, Payload: out})
	r.seqEmit.Add(1)
	r.reset()
}

// reset drops the current lock and returns to hunting.
func (r *Receiver) reset() {
	r.st = stateHunt
	r.started = false
	r.syms = r.syms[:0]
	r.badInSeq = 0
	r.dxSeen = false
}

// Stats reports cumulative counters for /metrics + debugging.
type Stats struct {
	SequencesIn   uint64 // sequences whose EOS was reached
	SequencesBad  uint64 // subset with ≥1 BCH-failed DX character
	SequencesEmit uint64 // events published to the bus
}

func (r *Receiver) Stats() Stats {
	return Stats{
		SequencesIn:   r.seqIn.Load(),
		SequencesBad:  r.seqBad.Load(),
		SequencesEmit: r.seqEmit.Load(),
	}
}
