// Package symbolscope recovers a live stream of demodulated symbols
// (the pre-slicer soft waveform plus the sliced dibit decisions) from a
// wideband IQ feed, for the web console's "Symbol" scope — GopherTrunk's
// take on OP25's Symbol plot.
//
// It deliberately reuses the production DSP rather than re-implementing
// demod: the same down-converter the live decoder channelizes with
// (ccdecoder.Downconverter) feeds the same protocol receiver, and the
// receiver's existing SoftSink / DibitSink taps surface the symbols.
// Run a separate Engine on an iqtap broker subscription (the
// diag_provider.go pattern) and production control-channel decode is
// never touched.
//
// Phase 1 supports P25 Phase 1: C4FM emits the FM-discriminator soft
// waveform aligned index-for-index with the sliced dibits; CQPSK emits
// the complex symbol-decision points (SymI/SymQ) — the true
// constellation — alongside the dibits, but no real soft track (its
// quality view is the constellation, not a 4-level eye). Other protocols
// — and a soft waveform for them — are a follow-up that adds a uniform
// soft tap to the remaining receivers; the Frame shape and this API are
// designed to absorb that without change.
package symbolscope

import (
	"errors"
	"fmt"
	"time"

	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	p25phase2rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25DeviationHz is the P25 Phase 1 nominal peak deviation (TIA-102),
// the slicer-scale the production pipeline calibrates against. Mirrors
// the siglab deep-path constant.
const p25DeviationHz = 1800.0

// dmrDeviationHz is the DMR nominal peak deviation at symbol ±3 (ETSI TS
// 102 361-1 §6.3), the slicer-scale the DMR receiver calibrates against —
// the same value the production DMR pipeline passes.
const dmrDeviationHz = 1944.0

// nxdnDeviationHz is the NXDN (9600-baud 4-FSK) nominal peak deviation at
// symbol ±3 per the Common Air Interface spec — the same value the
// production NXDN pipeline passes.
const nxdnDeviationHz = 1800.0

// defaultFrameSymbols batches recovered symbols into one Frame so the WS
// write cost stays reasonable — ~256 symbols is ~53 ms at 4800 baud,
// matching the diag stream's ~50 ms cadence.
const defaultFrameSymbols = 256

// maxEyeSamples bounds the oversampled eye window carried per frame. At
// 10 sps that is ~150 symbols of overlay — ample to render a stable eye —
// while keeping the WS payload modest (the oversampled stream is sps×
// the soft rate, so it is capped rather than sent in full).
const maxEyeSamples = 1536

// Frame is one batch of recovered symbols. Soft is the pre-slicer soft
// waveform (empty when the demod path exposes no soft tap, e.g. P25
// CQPSK); Dibits are the sliced decisions (values 0..3 when IsBits is
// false, 0..1 when true). When Soft is non-empty it is aligned
// index-for-index with Dibits. BaseIdx is the absolute symbol index the
// batch starts at, so a client can detect gaps.
//
// SymI/SymQ are the complex symbol-decision points — the true
// symbol-domain constellation sampled at each recovered symbol instant.
// They are populated only on the linear/CQPSK path (the post-Costas
// π/4-DQPSK points, which cluster at the four ±45°/±135° positions on a
// clean signal); the C4FM path leaves them empty because its symbol
// domain is the real 4-level soft already carried in Soft. When
// non-empty they are aligned index-for-index with Dibits.
type Frame struct {
	TimestampNs  int64     `json:"ts_ns"`
	SymbolRateHz float64   `json:"symbol_rate_hz"`
	CenterHz     uint32    `json:"center_hz"`
	OffsetHz     int32     `json:"offset_hz"`
	Soft         []float32 `json:"soft"`
	SymI         []float32 `json:"sym_i"`
	SymQ         []float32 `json:"sym_q"`
	// EyeSoft is a window of the oversampled, AGC-scaled matched-filter
	// output (EyeSPS samples per symbol) on the C4FM path — fold it over
	// the symbol period for the 4-level eye diagram. Empty on paths
	// without an eye tap (CQPSK). It is NOT aligned with Dibits (it runs
	// EyeSPS× faster) and is a bounded recent window, not the full
	// stream. EyeSPS is the integer samples-per-symbol to fold on.
	EyeSoft []float32 `json:"eye_soft"`
	EyeSPS  int       `json:"eye_sps"`
	Dibits  []uint8   `json:"dibits"`
	IsBits  bool      `json:"is_bits"`
	BaseIdx int       `json:"base_idx"`

	// Receiver-state metrics, read from the live receiver each frame —
	// the data behind the Tuning panel (OP25's Mixer / Tuner-FLL). The
	// getters are demod-specific: CarrierOffsetHz is the AFC estimate on
	// C4FM and the carrier-recovery estimate on CQPSK; ClockMu/ClockSPS
	// come from Mueller-Müller (C4FM) or Gardner (CQPSK); AGCLevel is the
	// C4FM symbol-AGC mean|x| or the CQPSK matched-filter AGC gain, with
	// AGCTarget set only on C4FM; CMAError is the CQPSK equalizer
	// convergence proxy (0 on C4FM).
	CarrierOffsetHz float64 `json:"carrier_offset_hz"`
	AGCLevel        float64 `json:"agc_level"`
	AGCTarget       float64 `json:"agc_target"`
	ClockMu         float64 `json:"clock_mu"`
	ClockSPS        float64 `json:"clock_sps"`
	CMAError        float64 `json:"cma_error"`
}

// Options configures an Engine.
type Options struct {
	// Protocol selects the receiver. Phase 1 supports trunking.ProtocolP25.
	Protocol trunking.Protocol
	// DemodMode is the P25 demod path: "c4fm" (default) or "cqpsk".
	// Ignored for other protocols.
	DemodMode string
	// InRateHz is the wideband input rate (the SDR's sample rate).
	InRateHz float64
	// OffsetHz tunes a channel sitting at this offset (relative to the
	// SDR centre) down to baseband before channelizing — pulls an
	// off-centre control/voice channel out from under the DC spike.
	OffsetHz int32
	// CenterHz stamps each frame with the SDR centre (informational).
	CenterHz uint32
	// SystemName / FrequencyHz are informational labels passed to the
	// receiver construction.
	SystemName  string
	FrequencyHz uint32
	// FrameSymbols is the per-frame symbol batch size. Zero picks
	// defaultFrameSymbols.
	FrameSymbols int
	// NowNs is an injectable clock for tests. Nil uses time.Now.
	NowNs func() int64
	// Emit receives each completed Frame. Required. The slices it
	// carries are freshly allocated per frame; the callee owns them.
	Emit func(Frame)

	// MixerEmit, when non-nil, enables the Mixer plot: the engine runs a
	// rate-limited FFT over the channelized baseband (raw) and over that
	// baseband re-mixed by the receiver's carrier-offset estimate (tuned),
	// delivering each pair as a MixerFrame. Nil disables the work entirely.
	MixerEmit func(MixerFrame)
	// MixerFFTSize is the FFT length for the Mixer plot (power of two).
	// Zero picks defaultMixerFFTSize. Ignored when MixerEmit is nil.
	MixerFFTSize int
	// MixerFPS caps the Mixer frame rate. Zero picks defaultMixerFPS.
	// Ignored when MixerEmit is nil.
	MixerFPS float64
}

// Engine channelizes a wideband IQ feed and runs a protocol receiver,
// surfacing the recovered symbols as Frames. Not safe for concurrent
// Process calls — drive it from a single goroutine (the broker drain
// loop), exactly like diag.Decimator.
type Engine struct {
	ddc *ccdecoder.Downconverter
	// Exactly one receiver is built per engine. rx is the P25 Phase 1 receiver;
	// tetraRx is the TETRA π/4-DQPSK receiver. Process/stampMetrics branch on
	// which is non-nil.
	rx           *p25phase1rx.Receiver
	tetraRx      *tetrarx.Receiver
	dmrRx        *dmrrx.Receiver
	nxdnRx       *nxdnrx.Receiver
	p25p2Rx      *p25phase2rx.Receiver
	symbolRateHz float64
	centerHz     uint32
	offsetHz     int32
	frameSymbols int
	nowNs        func() int64
	emit         func(Frame)

	// Channelized-IQ scratch, reused across Process calls.
	chanBuf []complex64

	// Per-frame accumulators. soft grows in lockstep with dibits on the
	// C4FM path; it stays empty on paths without a soft tap. pendSym
	// grows in lockstep with dibits on the CQPSK path (the complex
	// constellation points); it stays empty on the C4FM path.
	pendSoft   []float32
	pendSym    []complex64
	pendDibits []uint8
	// pendEye accumulates oversampled eye samples (C4FM); it runs faster
	// than the symbol stream and is a bounded recent window, attached to
	// each flushed frame and cleared. eyeSPS is the fold period.
	pendEye   []float32
	eyeSPS    int
	isCQPSK   bool // selects which receiver getters to read for metrics
	isBits    bool
	baseIdx   int // absolute symbol index of pendDibits[0]
	totalSyms int // symbols seen across the whole stream

	// mixer, when non-nil, accumulates channelized baseband into FFT
	// windows and emits raw/tuned spectra (the Mixer plot). Built only
	// when Options.MixerEmit is set.
	mixer *mixerAccum
}

// New constructs an Engine. Returns an error for an unsupported
// protocol, an absent Emit, or a non-positive input rate.
func New(opts Options) (*Engine, error) {
	if opts.Emit == nil {
		return nil, errors.New("symbolscope: Emit is required")
	}
	if opts.InRateHz <= 0 {
		return nil, errors.New("symbolscope: InRateHz must be > 0")
	}
	if opts.Protocol != trunking.ProtocolP25 && opts.Protocol != trunking.ProtocolTETRA &&
		opts.Protocol != trunking.ProtocolDMR && opts.Protocol != trunking.ProtocolNXDN &&
		opts.Protocol != trunking.ProtocolP25Phase2 {
		return nil, fmt.Errorf("symbolscope: protocol %s is not supported (P25 Phase 1/2, DMR, NXDN, and TETRA only)", opts.Protocol)
	}

	frameSymbols := opts.FrameSymbols
	if frameSymbols <= 0 {
		frameSymbols = defaultFrameSymbols
	}
	nowNs := opts.NowNs
	if nowNs == nil {
		nowNs = func() int64 { return time.Now().UnixNano() }
	}

	target := ccdecoder.DDCTargetForProtocol(opts.Protocol)
	ddc := ccdecoder.NewDownconverterWithOffset(opts.InRateHz, target, float64(opts.OffsetHz))

	e := &Engine{
		ddc:          ddc,
		centerHz:     opts.CenterHz,
		offsetHz:     opts.OffsetHz,
		frameSymbols: frameSymbols,
		nowNs:        nowNs,
		emit:         opts.Emit,
	}

	// Build the receiver directly (no control channel) so its symbol taps surface
	// the constellation. The DibitSink drives the frame accumulator; the
	// soft/symbol tap fires just before DibitSink on the same batch, so alignment
	// holds.
	switch opts.Protocol {
	case trunking.ProtocolTETRA:
		e.symbolRateHz = tetrarx.SymbolRate
		// The receiver's SoftSink emits the complex π/4-DQPSK differential
		// (s·conj(last)) per symbol — the constellation points (clustering at the
		// four ±45°/±135° positions on a clean signal). Route them into the same
		// complex-symbol accumulator the P25 CQPSK path uses. TETRA has no 4-level
		// soft eye, so no soft/eye tap. AFC + channel filter + equalizer mirror the
		// live decode so the scope shows what the decoder sees.
		e.tetraRx = tetrarx.New(tetrarx.Options{
			SampleRateHz:        ddc.OutRateHz(),
			DibitSink:           e.onDibits,
			SoftSink:            func(diffs []complex64, _ int) { e.onSymbols(diffs) },
			ClockMode:           tetrarx.ClockGardner,
			GardnerGain:         0.005,
			EnableAFC:           true,
			EnableChannelFilter: true,
			EnableEqualizer:     true,
		})
	case trunking.ProtocolDMR:
		// DMR is the same 4800-baud 4-level C4FM as P25 Phase 1, so it emits
		// the identical soft/eye/dibit shape — the soft track aligned with the
		// dibits and an oversampled eye. SymbolSink is wired for parity but
		// never fires (pure C4FM has no CQPSK constellation), so SymI/SymQ
		// stay empty and flush() renders the 4-level soft/eye view.
		e.symbolRateHz = dmrrx.SymbolRate
		e.dmrRx = dmrrx.New(dmrrx.Options{
			SampleRateHz: ddc.OutRateHz(),
			DeviationHz:  dmrDeviationHz,
			SoftSink:     e.onSoft,
			SymbolSink:   e.onSymbols,
			EyeSink:      e.onEye,
			DibitSink:    e.onDibits,
		})
	case trunking.ProtocolNXDN:
		// NXDN (9600-baud variant) is the same 4800-baud 4-level C4FM family
		// as DMR / P25 Phase 1, with the same tap shape: aligned soft track,
		// oversampled eye, parity-only SymbolSink that never fires.
		e.symbolRateHz = nxdnrx.SymbolRate
		e.nxdnRx = nxdnrx.New(nxdnrx.Options{
			SampleRateHz: ddc.OutRateHz(),
			DeviationHz:  nxdnDeviationHz,
			SoftSink:     e.onSoft,
			SymbolSink:   e.onSymbols,
			EyeSink:      e.onEye,
			DibitSink:    e.onDibits,
		})
	case trunking.ProtocolP25Phase2:
		// P25 Phase 2 H-DQPSK: the receiver's soft-decision path emits, per
		// dibit, the complex differential rotated onto the diagonals — the
		// true constellation, the same shape as TETRA's π/4-DQPSK view. With
		// SoftDecision set the hard DibitSink is never called (the receiver
		// contract), so the one callback feeds both accumulators: symbols
		// first, then dibits, so the flush pairs them on the same batch.
		// Gardner clock + the production gain mirror the live pipeline.
		// No 4-level soft/eye track (linear path).
		e.symbolRateHz = p25phase2rx.SymbolRate
		e.p25p2Rx = p25phase2rx.New(p25phase2rx.Options{
			SampleRateHz: ddc.OutRateHz(),
			ClockMode:    p25phase2rx.ClockGardner,
			GardnerGain:  0.03, // matches composer.p25p2VoiceGardnerGain
			SoftDecision: true,
			SoftSink: func(dibits []uint8, soft []complex64, baseIdx int) {
				e.onSymbols(soft)
				e.onDibits(dibits, baseIdx)
			},
		})
	default: // P25 Phase 1
		demodMode, ok := p25phase1rx.ParseDemodMode(opts.DemodMode)
		if !ok {
			return nil, fmt.Errorf("symbolscope: unknown demod mode %q (want c4fm or cqpsk)", opts.DemodMode)
		}
		e.symbolRateHz = p25phase1rx.SymbolRate
		e.isCQPSK = demodMode == p25phase1rx.DemodCQPSK
		e.rx = p25phase1rx.New(p25phase1rx.Options{
			SampleRateHz: ddc.OutRateHz(),
			DeviationHz:  p25DeviationHz,
			DemodMode:    demodMode,
			SoftSink:     e.onSoft,
			SymbolSink:   e.onSymbols,
			EyeSink:      e.onEye,
			DibitSink:    e.onDibits,
		})
	}

	if opts.MixerEmit != nil {
		mx, err := newMixerAccum(mixerOptions{
			fftSize:  opts.MixerFFTSize,
			fps:      opts.MixerFPS,
			sampleHz: ddc.OutRateHz(),
			centerHz: opts.CenterHz,
			offsetHz: opts.OffsetHz,
			nowNs:    nowNs,
			emit:     opts.MixerEmit,
		})
		if err != nil {
			return nil, err
		}
		e.mixer = mx
	}
	return e, nil
}

// Process channelizes one raw IQ chunk and runs the receiver over it.
// Completed frames are delivered via the Emit callback.
func (e *Engine) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	e.chanBuf = e.ddc.Process(e.chanBuf, iq)
	switch {
	case e.tetraRx != nil:
		e.tetraRx.Process(e.chanBuf)
	case e.dmrRx != nil:
		e.dmrRx.Process(e.chanBuf)
	case e.nxdnRx != nil:
		e.nxdnRx.Process(e.chanBuf)
	case e.p25p2Rx != nil:
		e.p25p2Rx.Process(e.chanBuf)
	default:
		e.rx.Process(e.chanBuf)
	}
	// Feed the Mixer accumulator after the receiver has updated its
	// carrier-offset estimate, so the "tuned" re-mix uses this chunk's
	// loop state. chanBuf is consumed synchronously (copied into the
	// FFT window), so reusing it as scratch next Process is safe.
	if e.mixer != nil {
		e.mixer.feed(e.chanBuf, e.carrierOffsetHz())
	}
}

// onSoft stashes the pre-slicer soft samples; onDibits, fired next on
// the same batch, pairs them with the sliced decisions.
func (e *Engine) onSoft(soft []float32) {
	e.pendSoft = append(e.pendSoft, soft...)
}

// onSymbols stashes the complex symbol-decision points (CQPSK path);
// onDibits pairs them with the sliced decisions, exactly as onSoft does
// for the C4FM soft track.
func (e *Engine) onSymbols(syms []complex64) {
	e.pendSym = append(e.pendSym, syms...)
}

// onEye accumulates the oversampled eye samples (C4FM), keeping only the
// most recent maxEyeSamples so the per-frame window stays bounded.
func (e *Engine) onEye(samples []float32, sps int) {
	e.eyeSPS = sps
	e.pendEye = append(e.pendEye, samples...)
	if len(e.pendEye) > maxEyeSamples {
		e.pendEye = e.pendEye[len(e.pendEye)-maxEyeSamples:]
	}
}

func (e *Engine) onDibits(dibits []uint8, _ int) {
	if e.pendDibits == nil {
		e.baseIdx = e.totalSyms
	}
	e.pendDibits = append(e.pendDibits, dibits...)
	e.totalSyms += len(dibits)
	for len(e.pendDibits) >= e.frameSymbols {
		e.flush(e.frameSymbols)
	}
}

// flush emits the first n accumulated symbols as a Frame and shifts the
// accumulators down. Soft is carried only when it stayed aligned with
// the dibit stream (C4FM); a length mismatch (a soft-less path) emits an
// empty soft track rather than a misaligned one.
func (e *Engine) flush(n int) {
	if n > len(e.pendDibits) {
		n = len(e.pendDibits)
	}
	if n == 0 {
		return
	}
	dibits := make([]uint8, n)
	copy(dibits, e.pendDibits[:n])

	var soft []float32
	if len(e.pendSoft) == len(e.pendDibits) {
		soft = make([]float32, n)
		copy(soft, e.pendSoft[:n])
		e.pendSoft = e.pendSoft[n:]
	} else {
		// Soft track absent or out of step — drop it for this batch and
		// resync so a one-off mismatch can't wedge the stream.
		e.pendSoft = e.pendSoft[:0]
	}

	// Complex symbol track (CQPSK), carried only when it stayed aligned
	// with the dibit stream — the same guard the soft track uses.
	var symI, symQ []float32
	if len(e.pendSym) == len(e.pendDibits) {
		symI = make([]float32, n)
		symQ = make([]float32, n)
		for i, s := range e.pendSym[:n] {
			symI[i] = real(s)
			symQ[i] = imag(s)
		}
		e.pendSym = e.pendSym[n:]
	} else {
		e.pendSym = e.pendSym[:0]
	}

	// Eye window (C4FM): hand the current bounded window to this frame and
	// clear it, so a later flush in the same drain doesn't duplicate it.
	var eye []float32
	if len(e.pendEye) > 0 {
		eye = make([]float32, len(e.pendEye))
		copy(eye, e.pendEye)
		e.pendEye = e.pendEye[:0]
	}

	frame := Frame{
		TimestampNs:  e.nowNs(),
		SymbolRateHz: e.symbolRateHz,
		CenterHz:     e.centerHz,
		OffsetHz:     e.offsetHz,
		Soft:         soft,
		SymI:         symI,
		SymQ:         symQ,
		EyeSoft:      eye,
		EyeSPS:       e.eyeSPS,
		Dibits:       dibits,
		IsBits:       e.isBits,
		BaseIdx:      e.baseIdx,
	}
	e.stampMetrics(&frame)
	e.pendDibits = e.pendDibits[n:]
	e.baseIdx += n
	e.emit(frame)
}

// stampMetrics reads the live receiver's state getters into the frame.
// The getters are demod-specific (Mueller-Müller vs Gardner, AFC vs
// carrier-recovery), so this routes by the configured demod mode.
func (e *Engine) stampMetrics(f *Frame) {
	if e.tetraRx != nil {
		// TETRA exposes its residual carrier offset; the other loop getters
		// (AGC/clock) aren't surfaced by the receiver, so they stay zero — the
		// constellation, offset, and dibits are the useful TETRA scope views.
		f.CarrierOffsetHz = e.tetraRx.CarrierOffsetHz()
		return
	}
	if e.dmrRx != nil {
		// DMR surfaces the scale-independent AGC/clock loop state. Its AFC
		// runs on the symbol stream through a unit-energy RRC (a DC gain the
		// P25 AFCOffsetHz formula doesn't share), so CarrierOffsetHz is left
		// at 0 rather than reported in the wrong scale — the constellation,
		// eye, and dibits are the useful DMR scope views.
		f.AGCLevel = e.dmrRx.AGCLevel()
		f.AGCTarget = e.dmrRx.AGCTarget()
		f.ClockMu = e.dmrRx.MMClockMu()
		f.ClockSPS = e.dmrRx.MMClockSPS()
		return
	}
	if e.nxdnRx != nil {
		// The scope's NXDN receiver runs no AFC (the opt-in nxdn_afc stage
		// is off by default, and even on it reads through a unit-energy RRC
		// — the same DC-gain scale mismatch as DMR), so CarrierOffsetHz
		// stays 0 rather than reported in the wrong scale — the AGC/clock
		// loop state plus the constellation, eye, and dibits are the useful
		// views, exactly the DMR posture.
		f.AGCLevel = e.nxdnRx.AGCLevel()
		f.AGCTarget = e.nxdnRx.AGCTarget()
		f.ClockMu = e.nxdnRx.MMClockMu()
		f.ClockSPS = e.nxdnRx.MMClockSPS()
		return
	}
	if e.p25p2Rx != nil {
		// Phase 2 exposes its carrier-recovery estimate (coarse seed +
		// Costas residual); the other loop getters aren't surfaced by the
		// receiver, so they stay zero — the TETRA posture: constellation,
		// offset, and dibits are the useful views.
		f.CarrierOffsetHz = e.p25p2Rx.CarrierOffsetHz()
		return
	}
	if e.rx == nil {
		return
	}
	if e.isCQPSK {
		f.CarrierOffsetHz = e.rx.CQPSKCarrierOffsetHz()
		f.AGCLevel = e.rx.CQPSKAGCGain()
		f.ClockMu = e.rx.GardnerMu()
		f.ClockSPS = e.rx.GardnerSPS()
		f.CMAError = e.rx.CMAError()
		return
	}
	f.CarrierOffsetHz = e.rx.AFCOffsetHz()
	f.AGCLevel = e.rx.AGCLevel()
	f.AGCTarget = e.rx.AGCTarget()
	f.ClockMu = e.rx.MMClockMu()
	f.ClockSPS = e.rx.MMClockSPS()
}

// carrierOffsetHz returns the receiver's current residual carrier-offset
// estimate (the AFC estimate on C4FM, the carrier-recovery estimate on
// CQPSK). The Mixer plot uses it to re-mix the raw baseband onto centre
// for the "tuned" view; it mirrors the routing stampMetrics uses.
func (e *Engine) carrierOffsetHz() float64 {
	if e.tetraRx != nil {
		return e.tetraRx.CarrierOffsetHz()
	}
	if e.p25p2Rx != nil {
		return e.p25p2Rx.CarrierOffsetHz()
	}
	if e.rx == nil {
		return 0
	}
	if e.isCQPSK {
		return e.rx.CQPSKCarrierOffsetHz()
	}
	return e.rx.AFCOffsetHz()
}

// Close releases the engine. Idempotent.
func (e *Engine) Close() error {
	e.pendSoft = nil
	e.pendSym = nil
	e.pendEye = nil
	e.pendDibits = nil
	return nil
}

// SymbolRateHz reports the recovered symbol rate (informational).
func (e *Engine) SymbolRateHz() float64 { return e.symbolRateHz }
