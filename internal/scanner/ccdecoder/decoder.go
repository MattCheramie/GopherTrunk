// Package ccdecoder is the connector that closes the IQ → control-
// channel decoder gap listed in the README "Status & known gaps".
//
// What this package does:
//
//   - Owns the control SDR's IQ stream (one StreamIQ loop per
//     Decoder lifetime).
//   - Subscribes to events.KindHuntProgress so it learns which
//     system / frequency the CC Hunter supervisor is currently
//     attempting.
//   - On every HuntProgress transition, swaps the active per-
//     protocol pipeline (IQ → symbol-domain decoder → CC state
//     machine) via the package-local factory map keyed on
//     trunking.Protocol.
//   - Pumps every IQ chunk arriving on the StreamIQ channel
//     through the active pipeline's Process method. The pipeline's
//     CC state machine publishes events.KindCCLocked /
//     events.KindGrant on the same bus, which the supervisor +
//     engine consume to drive the rest of the daemon.
//
// What this package does NOT do:
//
//   - It doesn't retune the SDR — that's the CC Hunter
//     supervisor's job (`internal/scanner/cchunt`). The Decoder
//     follows the supervisor's lead via HuntProgress events.
//   - It doesn't open / close the SDR device — the daemon does
//     that and hands a Tuner + IQSource through Options.
//   - It doesn't decode every protocol from day one. Each
//     pipeline is gated on having a control-channel state
//     machine that accepts a raw dibit / bit stream. Protocols
//     whose CC state machine still consumes pre-parsed PDUs
//     (DMR / NXDN / dPMR / EDACS / MPT 1327 / LTR / Motorola /
//     P25 P2 / TETRA) need a Process(...) adapter on their
//     control package first — a documented follow-up per the
//     per-protocol receiver PRs that already shipped.
package ccdecoder

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Tuner is the subset of sdr.Device the decoder uses for retuning.
// Matches the same interface cchunt + conventional consume so the
// daemon can hand the same Device to all three.
type Tuner interface {
	SetCenterFreq(hz uint32) error
}

// IQSource is the subset of sdr.Device the decoder consumes for IQ
// samples. Matches conventional.IQSource so the same Device satisfies
// both interfaces.
type IQSource interface {
	StreamIQ(ctx context.Context) (<-chan []complex64, error)
}

// Options configure a Decoder.
type Options struct {
	Bus     *events.Bus
	Log     *slog.Logger
	Tuner   Tuner    // currently unused but kept for API symmetry with cchunt
	IQ      IQSource // control SDR providing the live IQ stream
	Systems []trunking.System
	// SampleRateHz is the IQ stream rate. Forwarded to the per-
	// protocol receiver factories so they can size their matched
	// filters correctly.
	SampleRateHz float64
}

// Decoder is the long-lived component that converts the control
// SDR's IQ stream into CC / grant events on the bus. Construct via
// New, run via Run.
type Decoder struct {
	bus          *events.Bus
	log          *slog.Logger
	iq           IQSource
	sampleRateHz float64
	systems      map[string]trunking.System

	mu       sync.Mutex
	active   ProtocolPipeline
	activeAt string // system name the active pipeline is bound to
}

// New constructs a Decoder. Returns an error when required Options
// are missing.
func New(opts Options) (*Decoder, error) {
	if opts.Bus == nil {
		return nil, errors.New("ccdecoder: events.Bus is required")
	}
	if opts.IQ == nil {
		return nil, errors.New("ccdecoder: IQSource is required")
	}
	if opts.SampleRateHz <= 0 {
		return nil, errors.New("ccdecoder: SampleRateHz must be > 0")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	d := &Decoder{
		bus:          opts.Bus,
		log:          log,
		iq:           opts.IQ,
		sampleRateHz: opts.SampleRateHz,
		systems:      make(map[string]trunking.System, len(opts.Systems)),
	}
	for _, s := range opts.Systems {
		d.systems[s.Name] = s
	}
	return d, nil
}

// Run blocks until ctx cancels. It opens one StreamIQ loop on the
// control SDR, subscribes to KindHuntProgress, swaps the active
// per-protocol pipeline whenever the supervisor reports a new
// (system, frequency) under attempt, and pumps every IQ chunk
// through the active pipeline.
//
// Returns ctx.Err() on shutdown; any StreamIQ error from the SDR
// surfaces as the return value.
func (d *Decoder) Run(ctx context.Context) error {
	stream, err := d.iq.StreamIQ(ctx)
	if err != nil {
		return err
	}

	sub := d.bus.Subscribe()
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-sub.C:
			if !ok {
				return nil
			}
			if ev.Kind != events.KindHuntProgress {
				continue
			}
			p, ok := ev.Payload.(trunking.HuntProgress)
			if !ok {
				continue
			}
			d.handleProgress(p)
		case iq, ok := <-stream:
			if !ok {
				return nil
			}
			d.pump(iq)
		}
	}
}

// handleProgress swaps the active pipeline to match the supervisor's
// freshly-attempted system + frequency. Unknown systems / protocols
// without a registered factory log + leave the active pipeline as-is
// (so a stale pipeline doesn't decode noise into spurious events).
func (d *Decoder) handleProgress(p trunking.HuntProgress) {
	sys, ok := d.systems[p.System]
	if !ok {
		d.log.Debug("ccdecoder: HuntProgress for unknown system",
			"system", p.System)
		return
	}
	factory, ok := factories[sys.Protocol]
	if !ok {
		d.log.Debug("ccdecoder: no pipeline factory for protocol",
			"system", p.System, "protocol", sys.Protocol)
		d.clearActive()
		return
	}
	d.mu.Lock()
	// Close the previous pipeline before constructing a new one so
	// resources don't accumulate on rapid retune storms.
	if d.active != nil {
		_ = d.active.Close()
		d.active = nil
		d.activeAt = ""
	}
	d.mu.Unlock()

	p2, err := factory(PipelineOptions{
		Bus:          d.bus,
		Log:          d.log,
		SystemName:   sys.Name,
		FrequencyHz:  p.AttemptedFreqHz,
		SampleRateHz: d.sampleRateHz,
		System:       sys,
	})
	if err != nil {
		d.log.Warn("ccdecoder: pipeline factory failed",
			"system", p.System, "protocol", sys.Protocol, "err", err)
		return
	}

	d.mu.Lock()
	d.active = p2
	d.activeAt = sys.Name
	d.mu.Unlock()
}

func (d *Decoder) clearActive() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.active != nil {
		_ = d.active.Close()
		d.active = nil
		d.activeAt = ""
	}
}

// pump forwards an IQ chunk to the active pipeline. Holding the lock
// for the whole Process call serialises against handleProgress's
// pipeline swap so we never call Process on a half-constructed
// pipeline.
func (d *Decoder) pump(iq []complex64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.active == nil {
		return
	}
	d.active.Process(iq)
}

// Close releases the active pipeline. Safe to call from outside Run;
// Run also runs Close on the active pipeline as part of normal swap
// cleanup.
func (d *Decoder) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.active != nil {
		err := d.active.Close()
		d.active = nil
		d.activeAt = ""
		return err
	}
	return nil
}
