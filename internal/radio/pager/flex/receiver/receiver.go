// Package receiver wires the FLEX protocol layer onto a live IQ stream. The
// pipeline:
//
//	IQ chunks (Fs Hz, complex64)
//	  → FM discriminator (internal/dsp/demod.FM)
//	  → real resampler to 25.6 kHz (16 samples per 1600 symbols/s symbol)
//	  → Demod: per-frame sync search, level calibration, 2- / 4-level slicer
//	  → flex.Decoder (frame information word, phases, BCH, pages)
//	  → events.KindPagerMessage on the bus (Protocol="flex")
//
// One Receiver per configured FLEX paging frequency. Every FLEX mode is
// followed — 1600 bps (2-level), 3200 bps (2- or 4-level) and 6400 bps
// (4-level) — and the mode is read from each frame's sync, so a channel that
// mixes rates needs no configuration.
package receiver

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/pager/flex"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

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

	demod    *IQDemod
	lastMode flex.Mode // mode of the last frame logged

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
	dm, err := NewIQDemod(opts.InputRateHz)
	if err != nil {
		return nil, err
	}
	r := &Receiver{
		inputRate: opts.InputRateHz,
		source:    opts.SourceName,
		bus:       opts.Bus,
		log:       log,
		demod:     dm,
	}
	dm.OnFrame = r.noteFrame
	return r, nil
}

// noteFrame logs the channel's mode when it is first seen and whenever it
// changes (a network may switch rates frame by frame; that is logged at
// DEBUG after the first few so a mixed-rate channel does not flood the log).
func (r *Receiver) noteFrame(mode flex.Mode, inverted bool) {
	if mode == r.lastMode {
		return
	}
	first := r.lastMode == (flex.Mode{})
	r.lastMode = mode
	level := slog.LevelDebug
	if first {
		level = slog.LevelInfo
	}
	r.log.Log(context.Background(), level, "flex: frame sync",
		"source", r.source, "bps", mode.BitRate(), "symbol_rate", mode.Baud,
		"levels", mode.Levels, "inverted", inverted)
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
			for _, m := range r.demod.ProcessIQ(chunk) {
				r.publish(m)
			}
		}
	}
}

// publish converts a flex.Message into the events bus shape.
func (r *Receiver) publish(m flex.Message) {
	msg := storage.PagerMessage{
		Protocol:  "flex",
		Encoding:  m.Type.TypeName(),
		Body:      m.Text,
		Corrected: m.Corrected,
	}
	if m.Capcode <= math.MaxUint32 {
		msg.RIC = uint32(m.Capcode)
	} else {
		// The top sliver of the long-address range does not fit the log's
		// 32-bit address column; keep the page and say whose it is.
		msg.Body = "[capcode " + strconv.FormatUint(m.Capcode, 10) + "] " + msg.Body
	}
	r.bus.Publish(events.Event{Kind: events.KindPagerMessage, Payload: msg})
	r.pagesEmitted.Add(1)
}

// PagesEmitted reports the cumulative pages this receiver has
// published. Useful for /metrics and end-to-end tests.
func (r *Receiver) PagesEmitted() uint64 { return r.pagesEmitted.Load() }

// Stats reports the frame decoder's counters.
func (r *Receiver) Stats() flex.Stats { return r.demod.Decoder().Stats() }
