// Package ffsk wires the DSC DSP pipeline together: an IQ stream from
// the iqtap broker becomes 1200-baud FFSK audio, then a direct-FSK bit
// stream, then decoded DSC sequences on the events bus. The pipeline
// is:
//
//	IQ chunks (Fs Hz, complex64)
//	  → FM demod (internal/dsp/demod.FM)
//	  → real resampler (internal/dsp/resampler_real) to AudioRateHz
//	  → FFSK tone discriminator (internal/dsp/demod.FFSK,
//	    markHz=1300, spaceHz=2100 — ITU-R M.493 DSC tones)
//	  → Mueller-Müller symbol-timing recovery
//	    (internal/dsp/sync.MuellerMuller, 8 sps → 1 sample/symbol)
//	  → DC-tracking slicer (direct-FSK bit on the wire)
//	  → dsc/receiver.Receiver.Push(bit)
//	  → events.KindDSCMessage on the bus
//
// Layout mirrors internal/radio/mdc1200/afsk: one Receiver per (SDR,
// DSC-frequency) pair, the daemon pumps the broker subscription into
// Process. Like MDC1200 — and unlike APRS / AIS — DSC carries no NRZI
// line code, so the sliced bit feeds the framer directly; the tones
// are the DSC pair (1300/2100 Hz) rather than CCIR FFSK's 1200/1800.
//
// Channel 70 (156.525 MHz) is the VHF DSC calling channel; HF DSC
// rides 2187.5 / 8414.5 / 12577 / 16804.5 kHz among others. The
// receiver is frequency-agnostic — the daemon tunes the SDR.
package ffsk

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
	dscrx "github.com/MattCheramie/GopherTrunk/internal/radio/dsc/receiver"
)

// DSC FSK tone frequencies per ITU-R M.493-15 §3.1.
const (
	MarkHz  = 1300.0 // binary 1 ("B")
	SpaceHz = 2100.0 // binary 0 ("Y")
)

// BaudHz is the VHF DSC signalling rate (1200 baud; HF DSC uses 100
// baud, not handled by this frontend).
const BaudHz = 1200

// Oversample is the number of FFSK-discriminator samples per DSC bit
// fed into the Mueller-Müller timing-recovery loop. 8 matches the
// APRS / AIS / MDC1200 frontends — enough sub-sample resolution to
// track the symbol clock, cheap at 1200 baud.
const Oversample = 8

// AudioRateHz is the audio rate the FFSK discriminator runs at:
// baud × Oversample = 1200 × 8 = 9600.
const AudioRateHz = BaudHz * Oversample

// mmGain is the Mueller-Müller loop gain — the same conventional
// AFSK-paired starting point the other narrowband-FSK frontends use.
const mmGain = 0.05

// Options configures a Receiver.
type Options struct {
	// InputRateHz is the IQ sample rate the broker is feeding at.
	InputRateHz uint32

	// SourceName is stamped on log lines and surfaces in metrics.
	SourceName string

	// Bus is required — messages publish onto KindDSCMessage via the
	// dsc/receiver orchestrator.
	Bus *events.Bus

	// DropBadFCS, when true, drops sequences with a BCH-failed DX
	// character at the orchestrator. Defaults to false.
	DropBadFCS bool

	// Log is optional; defaults to slog.Default.
	Log *slog.Logger
}

// Receiver runs a DSC FFSK decode pipeline against a stream of IQ
// chunks. One Receiver per (SDR, DSC-frequency) pair.
type Receiver struct {
	inputRate uint32
	source    string
	log       *slog.Logger

	fm    *demod.FM
	rsmp  *dsp.RealResampler
	ffsk  *demod.FFSK
	mm    *dspsync.MuellerMuller
	inner *dscrx.Receiver

	// Per-call scratch buffers reused across Process iterations so the
	// hot path never allocates.
	demodBuf    []float32
	rsmpBuf     []float32
	ffskBuf     []float32
	symBuf      []float32
	meanEMA     float32
	meanReady   bool
	samplesSeen atomic.Uint64
	bitsEmitted atomic.Uint64
}

// New constructs a Receiver. Returns an error if opts.Bus is nil or
// InputRateHz is unset.
func New(opts Options) (*Receiver, error) {
	if opts.Bus == nil {
		return nil, errors.New("dsc/ffsk: Bus is required")
	}
	if opts.InputRateHz == 0 {
		return nil, errors.New("dsc/ffsk: InputRateHz is required")
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
		return nil, fmt.Errorf("dsc/ffsk: bad resample ratio L=%d M=%d", L, M)
	}

	return &Receiver{
		inputRate: opts.InputRateHz,
		source:    opts.SourceName,
		log:       log,
		fm:        demod.NewFM(),
		rsmp:      dsp.NewRealResampler(L, M, 16, 7.0),
		ffsk:      demod.NewFFSK(float64(AudioRateHz), MarkHz, SpaceHz),
		mm:        dspsync.NewMuellerMuller(float64(Oversample), mmGain),
		inner: dscrx.New(dscrx.Options{
			Bus:        opts.Bus,
			DropBadFCS: opts.DropBadFCS,
		}),
	}, nil
}

// Process pumps IQ chunks from in through the decode pipeline until
// ctx cancels or in closes.
func (r *Receiver) Process(ctx context.Context, in <-chan []complex64) error {
	if in == nil {
		return errors.New("dsc/ffsk: input channel is nil")
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

// processChunk runs one IQ chunk through FM → resample → FFSK
// discrimination → MM symbol-time recovery → slice → push.
func (r *Receiver) processChunk(chunk []complex64) {
	r.samplesSeen.Add(uint64(len(chunk)))
	r.demodBuf = r.fm.Process(r.demodBuf, chunk)
	r.rsmpBuf = r.rsmp.Process(r.rsmpBuf, r.demodBuf)
	r.ffskBuf = r.ffsk.Discriminate(r.ffskBuf, r.rsmpBuf)
	r.symBuf = r.mm.Process(r.symBuf, r.ffskBuf)
	for _, s := range r.symBuf {
		r.feedSymbol(s)
	}
}

// feedSymbol slices one recovered symbol to a direct-FSK bit (tracking
// DC bias with a slow EMA) and pushes it into the framer. DSC has no
// NRZI stage. Tone-sense ambiguity is resolved by the orchestrator's
// dual-polarity phasing hunt, so a fixed mark=1 slicer is sufficient
// here.
func (r *Receiver) feedSymbol(s float32) {
	if !r.meanReady {
		r.meanEMA = s
		r.meanReady = true
	} else {
		r.meanEMA += (s - r.meanEMA) * (1.0 / 64.0)
	}
	var bit byte
	if s > r.meanEMA {
		bit = 1
	}
	r.inner.Push(bit)
	r.bitsEmitted.Add(1)
}

// Inner returns the bit-stream orchestrator the frontend is driving.
// Exposed so the daemon can read Stats() for /metrics.
func (r *Receiver) Inner() *dscrx.Receiver { return r.inner }

// Stats reports cumulative DSP-frontend counters.
type Stats struct {
	IQSamplesSeen uint64 // raw IQ samples Process has consumed
	BitsEmitted   uint64 // bits handed to the orchestrator
}

func (r *Receiver) Stats() Stats {
	return Stats{
		IQSamplesSeen: r.samplesSeen.Load(),
		BitsEmitted:   r.bitsEmitted.Load(),
	}
}

// gcd computes the greatest common divisor via Euclid's algorithm.
func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
