// Package receiver wires the POCSAG protocol layer (sync detection,
// codeword parsing, page assembly) onto a live IQ stream coming off
// the iqtap broker. The pipeline is:
//
//	IQ chunks (Fs Hz, complex64)
//	  → FM demod (internal/dsp/demod/fm.FM)
//	  → real resampler (internal/dsp/resampler_real) to baud × 8 sps
//	  → bit-by-bit integrator + mean-tracking slicer
//	  → pocsag.Syncer.Push(bit)
//	  → events.KindPagerMessage on the bus
//
// The single-channel layout (one Receiver per configured paging
// frequency) trades flexibility for simplicity: the daemon tunes
// the SDR straight to the paging centre frequency and the receiver
// consumes the entire IQ stream. A multi-channel polyphase + DDC
// path that lets one wideband SDR cover several paging channels at
// once is a planned follow-up; the protocol + syncer layers it
// would feed are unchanged.
package receiver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/pocsag"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// Oversample is the number of resampled samples per POCSAG bit the
// integrator + slicer expects. 8 is a comfortable balance — high
// enough that a small timing offset (worst case ±0.5 input sample
// after the rational resampler quantises) doesn't bleed into the
// next bit, low enough to keep the CPU cost trivial.
const Oversample = 8

// Options configures a Receiver.
type Options struct {
	// InputRateHz is the IQ sample rate the broker is feeding at.
	// The receiver derives the FM-demod → resampler ratios from
	// this and BaudHz.
	InputRateHz uint32

	// BaudHz is the POCSAG signalling rate. Standard rates are
	// 512, 1200, and 2400. Defaults to 1200.
	BaudHz uint32

	// SourceName is stamped on emitted PagerMessage rows. The
	// daemon typically uses the SDR's serial.
	SourceName string

	// Bus is required — Pages publish onto KindPagerMessage.
	Bus *events.Bus

	// Log is optional; defaults to slog.Default.
	Log *slog.Logger
}

// Receiver runs a POCSAG decode pipeline against a stream of IQ
// chunks. One Receiver per (SDR, paging-frequency) pair; the daemon
// instantiates one for every entry in `paging.pocsag` and pumps the
// broker subscription into Process.
type Receiver struct {
	inputRate uint32
	baudHz    uint32
	source    string
	bus       *events.Bus
	log       *slog.Logger

	fm        *demod.FM
	rsmp      *dsp.RealResampler
	syncer    *pocsag.Syncer
	demodBuf  []float32
	rsmpBuf   []float32
	intAcc    float32
	intCount  int
	meanEMA   float32
	meanReady bool

	// pagesEmitted counts pages published to the bus. Surfaces as a
	// /metrics counter once wired.
	pagesEmitted atomic.Uint64
}

// New constructs a Receiver. Returns an error if opts is incomplete
// or InputRateHz isn't a sensible multiple of BaudHz × Oversample.
func New(opts Options) (*Receiver, error) {
	if opts.Bus == nil {
		return nil, errors.New("pocsag/receiver: Bus is required")
	}
	if opts.InputRateHz == 0 {
		return nil, errors.New("pocsag/receiver: InputRateHz is required")
	}
	baud := opts.BaudHz
	if baud == 0 {
		baud = 1200
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}

	// Rational resampler from InputRateHz down to baud × Oversample.
	// Compute the smallest reduction of (out / in) — the rational
	// resampler accepts L (interpolate) and M (decimate); we pick
	// L = baud × Oversample, M = InputRateHz, then reduce by GCD.
	out := uint32(baud) * Oversample
	g := gcd(out, opts.InputRateHz)
	L := int(out / g)
	M := int(opts.InputRateHz / g)
	if L == 0 || M == 0 {
		return nil, fmt.Errorf("pocsag/receiver: bad resample ratio L=%d M=%d", L, M)
	}

	r := &Receiver{
		inputRate: opts.InputRateHz,
		baudHz:    baud,
		source:    opts.SourceName,
		bus:       opts.Bus,
		log:       log,
		fm:        demod.NewFM(),
		// Tap count is modest; the resampler's polyphase low-pass
		// is fine at 16 taps/branch for our wideband-to-narrow
		// step. The β picks a Kaiser shape with reasonable
		// stopband. Tuned empirically against POCSAG fixtures
		// can come later.
		rsmp:   dsp.NewRealResampler(L, M, 16, 7.0),
		syncer: pocsag.NewSyncer(),
	}
	return r, nil
}

// Process pumps IQ chunks from in through the decode pipeline until
// ctx cancels or in closes. The function returns ctx.Err() on
// context cancellation, or nil when the input channel closes
// cleanly (typically when the iqtap subscription is torn down).
func (r *Receiver) Process(ctx context.Context, in <-chan []complex64) error {
	if in == nil {
		return errors.New("pocsag/receiver: input channel is nil")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				// Flush any in-progress page before exiting.
				if p := r.syncer.Flush(); p != nil {
					r.publishPage(*p)
				}
				return nil
			}
			r.processChunk(chunk)
		}
	}
}

// processChunk runs one IQ chunk through FM demod, resample, and
// bit slicer; pages emerge from the syncer mid-chunk and publish
// to the bus.
func (r *Receiver) processChunk(chunk []complex64) {
	r.demodBuf = r.fm.Process(r.demodBuf, chunk)
	r.rsmpBuf = r.rsmp.Process(r.rsmpBuf, r.demodBuf)
	for _, s := range r.rsmpBuf {
		r.feedSample(s)
	}
}

// feedSample integrates one resampled sample into the current bit
// integrator. After Oversample samples it slices to a bit and
// pushes through the syncer.
func (r *Receiver) feedSample(s float32) {
	r.intAcc += s
	r.intCount++
	if r.intCount < Oversample {
		return
	}
	avg := r.intAcc / float32(Oversample)
	r.intAcc = 0
	r.intCount = 0

	// Track a slow mean to handle DC bias on the FM demod (the
	// receiver may not be perfectly tuned). 1/64 EMA gives ~64
	// bits of memory ≈ 50 ms at 1200 baud — short enough to
	// settle quickly, long enough that legitimate sync codeword
	// transitions don't yank it around.
	if !r.meanReady {
		r.meanEMA = avg
		r.meanReady = true
	} else {
		r.meanEMA += (avg - r.meanEMA) * (1.0 / 64.0)
	}

	var bit byte
	if avg > r.meanEMA {
		bit = 1
	}
	for _, p := range r.syncer.Push(bit) {
		r.publishPage(p)
	}
}

// publishPage converts a pocsag.Page into the events bus shape and
// publishes it. Encoding labels match the storage column values
// for the pager_log table.
func (r *Receiver) publishPage(p pocsag.Page) {
	enc := "alpha"
	if p.Encoding == pocsag.EncodingNumeric {
		enc = "numeric"
	}
	msg := storage.PagerMessage{
		Protocol:  "pocsag",
		RIC:       p.RIC,
		Func:      uint8(p.Func),
		Encoding:  enc,
		Body:      p.Text,
		Corrected: p.Corrected,
	}
	r.bus.Publish(events.Event{Kind: events.KindPagerMessage, Payload: msg})
	r.pagesEmitted.Add(1)
}

// PagesEmitted reports the cumulative pages this receiver has
// published. Useful for /metrics and end-to-end tests.
func (r *Receiver) PagesEmitted() uint64 { return r.pagesEmitted.Load() }

// gcd computes the greatest common divisor via Euclid's algorithm.
// Used to reduce the resample L/M ratio.
func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
