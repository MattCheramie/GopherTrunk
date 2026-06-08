// Package receiver wires the M17 link-layer decoder onto a live IQ
// stream from the iqtap broker. The pipeline is:
//
//	IQ chunks (Fs Hz, complex64)
//	  → FM demod (internal/dsp/demod/fm.FM)
//	  → real resampler to BaudHz × Oversample
//	  → C4FM matched filter (internal/dsp/demod.C4FM)
//	  → Mueller-Müller symbol-timing recovery
//	  → 4FSK slice → dibit → bit pair
//	  → m17.Decoder.Push(bit)
//	  → events.KindM17LinkSetup on the bus
//
// One Receiver per configured M17 frequency. This frontend recovers
// link-setup metadata (source / destination callsigns, mode) via the
// stream-frame LICH path; the Codec2 voice payload is a later
// milestone. The slicer deviation / symbol-timing constants are
// nominal — like the POCSAG frontend, on-air calibration against a
// real capture is a follow-up; the link-decode logic is covered by the
// internal/radio/m17 package tests.
package receiver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	dspsync "github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/m17"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
)

// M17 4FSK signalling parameters (M17 spec, physical layer).
const (
	BaudHz      = 4800 // symbols/s
	Oversample  = 10   // matched-filter samples per symbol
	AudioRateHz = BaudHz * Oversample
	rrcAlpha    = 0.5    // M17 RRC roll-off
	deviationHz = 2400.0 // peak (outer-symbol) deviation
	mmGain      = 0.05
)

// Options configures a Receiver.
type Options struct {
	InputRateHz uint32
	SourceName  string
	Bus         *events.Bus
	Log         *slog.Logger
}

// Receiver runs an M17 link-layer decode pipeline against a stream of
// IQ chunks.
type Receiver struct {
	source string
	bus    *events.Bus
	log    *slog.Logger

	fm      *demod.FM
	rsmp    *dsp.RealResampler
	c4fm    *demod.C4FM
	mm      *dspsync.MuellerMuller
	decoder *m17.Decoder

	demodBuf []float32
	rsmpBuf  []float32
	mfBuf    []float32
	symBuf   []float32

	samplesSeen atomic.Uint64
	lsfEmitted  atomic.Uint64
}

// New constructs a Receiver. Returns an error if opts.Bus is nil or
// InputRateHz is unset.
func New(opts Options) (*Receiver, error) {
	if opts.Bus == nil {
		return nil, errors.New("m17/receiver: Bus is required")
	}
	if opts.InputRateHz == 0 {
		return nil, errors.New("m17/receiver: InputRateHz is required")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	out := uint32(AudioRateHz)
	g := gcd(out, opts.InputRateHz)
	L := int(out / g)
	M := int(opts.InputRateHz / g)
	if L == 0 || M == 0 {
		return nil, fmt.Errorf("m17/receiver: bad resample ratio L=%d M=%d", L, M)
	}
	// Post-FM-demod soft values are phase steps in radians; the outer
	// symbol lands at 2π·deviation/Fs, which is the slicer's scale.
	dev := 2 * 3.141592653589793 * deviationHz / float64(AudioRateHz)
	return &Receiver{
		source:  opts.SourceName,
		bus:     opts.Bus,
		log:     log,
		fm:      demod.NewFM(),
		rsmp:    dsp.NewRealResampler(L, M, 16, 7.0),
		c4fm:    demod.NewC4FM(Oversample, 8, rrcAlpha, dev),
		mm:      dspsync.NewMuellerMuller(float64(Oversample), mmGain),
		decoder: m17.NewDecoder(),
	}, nil
}

// Process pumps IQ chunks through the decode pipeline until ctx
// cancels or in closes.
func (r *Receiver) Process(ctx context.Context, in <-chan []complex64) error {
	if in == nil {
		return errors.New("m17/receiver: input channel is nil")
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
	r.samplesSeen.Add(uint64(len(chunk)))
	r.demodBuf = r.fm.Process(r.demodBuf, chunk)
	r.rsmpBuf = r.rsmp.Process(r.rsmpBuf, r.demodBuf)
	r.mfBuf = r.c4fm.MatchedFilter(r.mfBuf, r.rsmpBuf)
	r.symBuf = r.mm.Process(r.symBuf, r.mfBuf)
	for _, s := range r.symBuf {
		r.feedSymbol(s)
	}
}

// feedSymbol slices one recovered symbol to a 4FSK level, maps it to
// the M17 dibit, and pushes the two bits (MSB first) into the decoder.
func (r *Receiver) feedSymbol(s float32) {
	dibit := symbolToDibit(r.c4fm.Slice(s))
	if lsf, ok := r.decoder.Push(byte(dibit >> 1)); ok {
		r.publish(lsf)
	}
	if lsf, ok := r.decoder.Push(byte(dibit & 1)); ok {
		r.publish(lsf)
	}
}

// symbolToDibit maps a C4FM slicer level to the M17 dibit value
// (M17 spec: +1→00, +3→01, -1→10, -3→11).
func symbolToDibit(level int) int {
	switch level {
	case 1:
		return 0
	case 3:
		return 1
	case -1:
		return 2
	default: // -3
		return 3
	}
}

func (r *Receiver) publish(l m17.LSF) {
	out := storage.M17LinkSetup{
		Src:   l.Src,
		Dst:   l.Dst,
		Mode:  l.ModeString(),
		CAN:   l.CAN,
		Meta:  l.MetaHex(),
		CRCOK: l.CRCOK,
		Body:  l.String(),
	}
	r.bus.Publish(events.Event{Kind: events.KindM17LinkSetup, Payload: out})
	r.lsfEmitted.Add(1)
}

// Stats reports cumulative counters.
type Stats struct {
	IQSamplesSeen uint64
	LSFEmitted    uint64
}

func (r *Receiver) Stats() Stats {
	return Stats{IQSamplesSeen: r.samplesSeen.Load(), LSFEmitted: r.lsfEmitted.Load()}
}

func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
