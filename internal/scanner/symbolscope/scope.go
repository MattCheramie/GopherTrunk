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
// the dibits only (the receiver has no soft tap on that path yet). Other
// protocols — and a soft waveform for them — are a follow-up that adds a
// uniform soft tap to the remaining receivers; the Frame shape and this
// API are designed to absorb that without change.
package symbolscope

import (
	"errors"
	"fmt"
	"time"

	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// p25DeviationHz is the P25 Phase 1 nominal peak deviation (TIA-102),
// the slicer-scale the production pipeline calibrates against. Mirrors
// the siglab deep-path constant.
const p25DeviationHz = 1800.0

// defaultFrameSymbols batches recovered symbols into one Frame so the WS
// write cost stays reasonable — ~256 symbols is ~53 ms at 4800 baud,
// matching the diag stream's ~50 ms cadence.
const defaultFrameSymbols = 256

// Frame is one batch of recovered symbols. Soft is the pre-slicer soft
// waveform (empty when the demod path exposes no soft tap, e.g. P25
// CQPSK); Dibits are the sliced decisions (values 0..3 when IsBits is
// false, 0..1 when true). When Soft is non-empty it is aligned
// index-for-index with Dibits. BaseIdx is the absolute symbol index the
// batch starts at, so a client can detect gaps.
type Frame struct {
	TimestampNs  int64     `json:"ts_ns"`
	SymbolRateHz float64   `json:"symbol_rate_hz"`
	CenterHz     uint32    `json:"center_hz"`
	OffsetHz     int32     `json:"offset_hz"`
	Soft         []float32 `json:"soft"`
	Dibits       []uint8   `json:"dibits"`
	IsBits       bool      `json:"is_bits"`
	BaseIdx      int       `json:"base_idx"`
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
}

// Engine channelizes a wideband IQ feed and runs a protocol receiver,
// surfacing the recovered symbols as Frames. Not safe for concurrent
// Process calls — drive it from a single goroutine (the broker drain
// loop), exactly like diag.Decimator.
type Engine struct {
	ddc          *ccdecoder.Downconverter
	rx           *p25phase1rx.Receiver
	symbolRateHz float64
	centerHz     uint32
	offsetHz     int32
	frameSymbols int
	nowNs        func() int64
	emit         func(Frame)

	// Channelized-IQ scratch, reused across Process calls.
	chanBuf []complex64

	// Per-frame accumulators. soft grows in lockstep with dibits on the
	// C4FM path; it stays empty on paths without a soft tap.
	pendSoft   []float32
	pendDibits []uint8
	isBits     bool
	baseIdx    int // absolute symbol index of pendDibits[0]
	totalSyms  int // symbols seen across the whole stream
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
	if opts.Protocol != trunking.ProtocolP25 {
		return nil, fmt.Errorf("symbolscope: protocol %s is not supported yet (P25 Phase 1 only)", opts.Protocol)
	}
	demodMode, ok := p25phase1rx.ParseDemodMode(opts.DemodMode)
	if !ok {
		return nil, fmt.Errorf("symbolscope: unknown demod mode %q (want c4fm or cqpsk)", opts.DemodMode)
	}

	frameSymbols := opts.FrameSymbols
	if frameSymbols <= 0 {
		frameSymbols = defaultFrameSymbols
	}
	nowNs := opts.NowNs
	if nowNs == nil {
		nowNs = func() int64 { return time.Now().UnixNano() }
	}

	target := ccdecoder.DDCTargetForProtocol(trunking.ProtocolP25)
	ddc := ccdecoder.NewDownconverterWithOffset(opts.InRateHz, target, float64(opts.OffsetHz))

	e := &Engine{
		ddc:          ddc,
		symbolRateHz: p25phase1rx.SymbolRate,
		centerHz:     opts.CenterHz,
		offsetHz:     opts.OffsetHz,
		frameSymbols: frameSymbols,
		nowNs:        nowNs,
		emit:         opts.Emit,
	}

	// Build the receiver directly (no control channel) so the SoftSink
	// surfaces the pre-slicer waveform. The DibitSink drives the frame
	// accumulator; soft↔dibit alignment holds because the receiver
	// fires SoftSink then DibitSink on the same symbol batch.
	e.rx = p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz: ddc.OutRateHz(),
		DeviationHz:  p25DeviationHz,
		DemodMode:    demodMode,
		SoftSink:     e.onSoft,
		DibitSink:    e.onDibits,
	})
	return e, nil
}

// Process channelizes one raw IQ chunk and runs the receiver over it.
// Completed frames are delivered via the Emit callback.
func (e *Engine) Process(iq []complex64) {
	if len(iq) == 0 {
		return
	}
	e.chanBuf = e.ddc.Process(e.chanBuf, iq)
	e.rx.Process(e.chanBuf)
}

// onSoft stashes the pre-slicer soft samples; onDibits, fired next on
// the same batch, pairs them with the sliced decisions.
func (e *Engine) onSoft(soft []float32) {
	e.pendSoft = append(e.pendSoft, soft...)
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

	frame := Frame{
		TimestampNs:  e.nowNs(),
		SymbolRateHz: e.symbolRateHz,
		CenterHz:     e.centerHz,
		OffsetHz:     e.offsetHz,
		Soft:         soft,
		Dibits:       dibits,
		IsBits:       e.isBits,
		BaseIdx:      e.baseIdx,
	}
	e.pendDibits = e.pendDibits[n:]
	e.baseIdx += n
	e.emit(frame)
}

// Close releases the engine. Idempotent.
func (e *Engine) Close() error {
	e.pendSoft = nil
	e.pendDibits = nil
	return nil
}

// SymbolRateHz reports the recovered symbol rate (informational).
func (e *Engine) SymbolRateHz() float64 { return e.symbolRateHz }
