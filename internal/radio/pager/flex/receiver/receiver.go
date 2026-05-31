// Package receiver wires the FLEX protocol layer (sync acquisition,
// frame-info word, block de-interleaving, BCH, page assembly) onto a
// live IQ stream from the iqtap broker. The pipeline mirrors the
// POCSAG receiver:
//
//	IQ chunks (Fs Hz, complex64)
//	  → FM demod (internal/dsp/demod/fm.FM)
//	  → real resampler (internal/dsp/resampler_real) to baud × 8 sps
//	  → bit-by-bit integrator + mean-tracking slicer
//	  → flex.Decoder.Push(bit)
//	  → events.KindPagerMessage on the bus (Protocol="flex")
//
// One Receiver per configured FLEX paging frequency. This frontend
// handles the 1600 bps / 2-level mode (the decoder's supported mode);
// the daemon tunes the SDR straight to the paging centre frequency and
// the receiver consumes the whole IQ stream.
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
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// BaudHz is the FLEX signalling rate this frontend handles (1600 bps,
// 2-level FSK — the decoder's supported mode).
const BaudHz = 1600

// Oversample is the resampled samples per FLEX bit fed to the
// integrator + slicer. 8 mirrors the POCSAG frontend.
const Oversample = 8

// Options configures a Receiver.
type Options struct {
	// InputRateHz is the IQ sample rate the broker is feeding at.
	InputRateHz uint32

	// SourceName is stamped on log lines / metrics.
	SourceName string

	// Bus is required — pages publish onto KindPagerMessage.
	Bus *events.Bus

	// Log is optional; defaults to slog.Default.
	Log *slog.Logger
}

// Receiver runs a FLEX decode pipeline against a stream of IQ chunks.
type Receiver struct {
	inputRate uint32
	source    string
	bus       *events.Bus
	log       *slog.Logger

	fm        *demod.FM
	rsmp      *dsp.RealResampler
	decoder   *flex.Decoder
	demodBuf  []float32
	rsmpBuf   []float32
	intAcc    float32
	intCount  int
	meanEMA   float32
	meanReady bool

	pagesEmitted atomic.Uint64
}

// New constructs a Receiver. Returns an error if opts.Bus is nil or
// InputRateHz is unset.
func New(opts Options) (*Receiver, error) {
	if opts.Bus == nil {
		return nil, errors.New("flex/receiver: Bus is required")
	}
	if opts.InputRateHz == 0 {
		return nil, errors.New("flex/receiver: InputRateHz is required")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	out := uint32(BaudHz) * Oversample
	g := gcd(out, opts.InputRateHz)
	L := int(out / g)
	M := int(opts.InputRateHz / g)
	if L == 0 || M == 0 {
		return nil, fmt.Errorf("flex/receiver: bad resample ratio L=%d M=%d", L, M)
	}
	return &Receiver{
		inputRate: opts.InputRateHz,
		source:    opts.SourceName,
		bus:       opts.Bus,
		log:       log,
		fm:        demod.NewFM(),
		rsmp:      dsp.NewRealResampler(L, M, 16, 7.0),
		decoder:   flex.NewDecoder(),
	}, nil
}

// Process pumps IQ chunks through the decode pipeline until ctx
// cancels or in closes.
func (r *Receiver) Process(ctx context.Context, in <-chan []complex64) error {
	if in == nil {
		return errors.New("flex/receiver: input channel is nil")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			r.processChunk(chunk)
		}
	}
}

func (r *Receiver) processChunk(chunk []complex64) {
	r.demodBuf = r.fm.Process(r.demodBuf, chunk)
	r.rsmpBuf = r.rsmp.Process(r.rsmpBuf, r.demodBuf)
	for _, s := range r.rsmpBuf {
		r.feedSample(s)
	}
}

// feedSample integrates one resampled sample; after Oversample samples
// it slices to a bit (tracking DC bias with a slow EMA, like POCSAG)
// and pushes it through the FLEX decoder.
func (r *Receiver) feedSample(s float32) {
	r.intAcc += s
	r.intCount++
	if r.intCount < Oversample {
		return
	}
	avg := r.intAcc / float32(Oversample)
	r.intAcc = 0
	r.intCount = 0

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
	for _, m := range r.decoder.Push(bit) {
		r.publish(m)
	}
}

// publish converts a flex.Message into the events bus shape.
func (r *Receiver) publish(m flex.Message) {
	msg := storage.PagerMessage{
		Protocol:  "flex",
		RIC:       m.Capcode,
		Encoding:  m.Type.TypeName(),
		Body:      m.Text,
		Corrected: m.Corrected,
	}
	r.bus.Publish(events.Event{Kind: events.KindPagerMessage, Payload: msg})
	r.pagesEmitted.Add(1)
}

// PagesEmitted reports the cumulative pages this receiver has
// published. Useful for /metrics and end-to-end tests.
func (r *Receiver) PagesEmitted() uint64 { return r.pagesEmitted.Load() }

// gcd computes the greatest common divisor via Euclid's algorithm.
func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
