// Package receiver wires the IQ → GMSK bit chain that feeds the
// D-STAR repeater-channel state machine.
//
//	IQ samples
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → Gaussian matched filter + 2-level slicer (internal/dsp/demod.GFSK)
//	  → Mueller-Müller symbol clock recovery (internal/dsp/sync.MuellerMuller)
//	  → dstar.BitSink
//
// D-STAR is GMSK at 4800 bps with BT = 0.5 per the JARL DV-mode
// specification — same 2-level shape as EDACS but at half the symbol
// rate and a slightly more rounded pulse. The downstream framing
// (32-bit Frame Sync, 41-byte PCH header) lives in the parent dstar
// package.
//
// The convolutional rate-1/2 inner code + scrambler + interleaver
// the JARL spec wraps around the PCH header are deliberately deferred
// in this receiver — the downstream Process adapter consumes
// pre-FEC bits, which matches the existing dstar package's stated
// "honest deferral" for FEC. Synthesized fixtures (and operators
// reading from a host that has already de-FEC'd) work end-to-end;
// real-air capture-from-RF will need the inner FEC layer later.
//
// The receiver is stateful and not safe for concurrent Process
// calls. Instantiate one per tuned frequency / per call chain.
package receiver

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dstar"
)

// D-STAR DV-mode on-air parameters per the JARL specification.
const (
	// SymbolRate is the channel symbol rate. Each symbol carries
	// one bit on the wire — total channel capacity 4800 bps.
	SymbolRate = 4800.0
	// BT is the Gaussian premod product matching standard D-STAR
	// transmitters.
	BT = 0.5
	// PulseSpanSymbols is the half-span of the Gaussian pulse on
	// each side of the symbol time. 4 symbols is the typical
	// receiver-side compromise; the Gaussian shape is sharp enough
	// that longer spans add little selectivity.
	PulseSpanSymbols = 4
)

// Options configures a Receiver.
type Options struct {
	// SampleRateHz is the IQ sample rate after any upstream
	// channelization. Required; must be ≥ 2 × SymbolRate (9600 Hz).
	SampleRateHz float64
	// BitSink receives the raw bit stream the receiver decodes
	// from IQ. Required.
	BitSink dstar.BitSink
	// PulseSpanSymbols overrides the Gaussian half-span. <= 0
	// uses the package default.
	PulseSpanSymbols int
	// BTProduct overrides the Gaussian BT product. <= 0 uses BT.
	BTProduct float64
	// ClockGain is the Mueller-Müller loop gain. <= 0 uses 0.05.
	ClockGain float64

	// EnableAFC subtracts the carrier offset's DC bias from the FM
	// discriminator. GMSK at ±1.2 kHz deviation decodes nothing once the
	// carrier is ~150 Hz off (measured on a synthetic stream), and a real
	// radio is routinely a few hundred hertz off — the #1261 reporter's
	// NXDN captures sat ~400 Hz off. Off by default so existing fixtures
	// stay byte-identical; the voice chain turns it on.
	EnableAFC bool
	// EnableChannelFilter inserts a ±ChannelCutoffHz low-pass ahead of the
	// discriminator. The voice chain's front end barely band-limits at an
	// SDR rate of a few MS/s, so without it the discriminator sees the
	// whole ±24 kHz intermediate stream (the NXDN / DMR lesson). Off by
	// default; the voice chain turns it on.
	EnableChannelFilter bool
	// ChannelCutoffOverrideHz replaces ChannelCutoffHz when > 0.
	ChannelCutoffOverrideHz float64
}

// ChannelCutoffHz is the one-sided cutoff of the optional channel filter
// (Options.EnableChannelFilter). D-STAR's GMSK occupies ±~3 kHz. Swept on a
// synthetic stream 400 Hz off frequency, every cutoff from 3 to 6 kHz cut the
// bit error rate under a same-power neighbour 10–12.5 kHz away from 20–26 % to
// ~1.5 %, and 4 kHz did best in band-wide noise while still decoding cleanly
// 600 Hz off frequency.
const ChannelCutoffHz = 4000.0

// channelFilterSpanSymbols sizes the channel FIR at 2·span·sps+1 taps
// (Kaiser β 8.6): 181 taps at 48 kHz, a ~1.5 kHz skirt — the DMR / NXDN
// receivers' filter.
const channelFilterSpanSymbols = 9

// Receiver is the composed IQ → bit pipeline.
type Receiver struct {
	chanFilt *filter.FIR
	filtered []complex64
	afc      *demod.CoarseAFC
	fm       *demod.FM
	gfsk     *demod.GFSK
	clock    *sync.MuellerMuller
	bitSink  dstar.BitSink
	bitBase  int

	disc    []float32
	matched []float32
	symbols []float32
	bits    []byte
}

// New constructs a Receiver. Panics if SampleRateHz or BitSink are
// unset, or the resulting samples-per-symbol is below 2.
func New(opts Options) *Receiver {
	if opts.SampleRateHz <= 0 {
		panic("receiver: SampleRateHz is required")
	}
	if opts.BitSink == nil {
		panic("receiver: BitSink is required")
	}
	sps := opts.SampleRateHz / SymbolRate
	if sps < 2 {
		panic("receiver: SampleRateHz must be >= 2*SymbolRate (9600 Hz)")
	}
	span := opts.PulseSpanSymbols
	if span <= 0 {
		span = PulseSpanSymbols
	}
	bt := opts.BTProduct
	if bt <= 0 {
		bt = BT
	}
	gain := opts.ClockGain
	if gain <= 0 {
		gain = 0.05
	}

	var chanFilt *filter.FIR
	if opts.EnableChannelFilter {
		fc := ChannelCutoffHz
		if opts.ChannelCutoffOverrideHz > 0 {
			fc = opts.ChannelCutoffOverrideHz
		}
		taps := 2*channelFilterSpanSymbols*int(sps+0.5) + 1
		chanFilt = filter.NewFIR(filter.LowpassKaiser(taps, fc/opts.SampleRateHz, 8.6))
	}
	var afc *demod.CoarseAFC
	if opts.EnableAFC {
		afc = demod.NewCoarseAFC(sps)
	}

	return &Receiver{
		chanFilt: chanFilt,
		afc:      afc,
		fm:       demod.NewFM(),
		gfsk:     demod.NewGFSK(int(sps+0.5), span, bt),
		clock:    sync.NewMuellerMuller(sps, gain),
		bitSink:  opts.BitSink,
	}
}

// Process pushes one chunk of complex64 IQ samples through the chain.
// Zero or more bit batches may be emitted to BitSink during the call.
func (r *Receiver) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	if r.chanFilt != nil {
		r.filtered = r.chanFilt.Process(r.filtered, iq)
		iq = r.filtered
	}
	r.disc = r.fm.Process(r.disc, iq)
	if r.afc != nil {
		r.afc.Process(r.disc)
	}
	r.matched = r.gfsk.MatchedFilter(r.matched, r.disc)
	r.symbols = r.clock.Process(r.symbols, r.matched)
	if len(r.symbols) == 0 {
		return
	}
	if cap(r.bits) < len(r.symbols) {
		r.bits = make([]byte, len(r.symbols))
	} else {
		r.bits = r.bits[:len(r.symbols)]
	}
	for i, s := range r.symbols {
		r.bits[i] = byte(r.gfsk.Slice(s))
	}
	r.bitSink(r.bits, r.bitBase)
	r.bitBase += len(r.bits)
}

// Reset returns the receiver to its initial state. Call on stream
// re-sync (control-channel hunt success, IQ underrun recovery) so
// the BitSink baseIdx restarts at 0 and the matched filter / clock
// recovery shed their stale history.
func (r *Receiver) Reset() {
	r.bitBase = 0
	r.gfsk.Reset()
	if r.afc != nil {
		r.afc.Reset()
	}
	if r.chanFilt != nil {
		r.chanFilt.Reset()
	}
}
