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
//   - Down-converts every raw IQ chunk to a narrowband channel
//     stream — rational polyphase decimation, ~48 kHz for the
//     4800-baud C4FM family, wider for TETRA — before the pipeline
//     sees it. The per-protocol receivers size their matched
//     filters for this channelized rate, not the raw SDR rate, so
//     this stage is what lets a live 2.048 MHz RTL-SDR stream
//     actually lock (issue #275).
//   - Pumps every down-converted IQ chunk through the active
//     pipeline's Process method. The pipeline's CC state machine
//     publishes events.KindCCLocked / events.KindGrant on the same
//     bus, which the supervisor + engine consume to drive the rest
//     of the daemon.
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
	"math"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// IQPowerObserver is the minimal Metrics surface the decoder uses to
// publish its window-averaged dBFS gauge. internal/metrics.Metrics
// satisfies this; nil disables the gauge entirely.
type IQPowerObserver interface {
	RecordIQPowerDbFS(system string, dbfs float64)
	ClearIQPowerDbFS(system string)
}

// iqPowerWindow is how long the pump aggregates samples before
// recomputing the mean dBFS and updating the gauge / debug log.
const iqPowerWindow = time.Second

// iqLowPowerThresholdDbFS is the level below which pump emits a
// throttled debug log warning that the IQ stream looks dead — gain
// at 0, antenna disconnected, or USB stuck. -55 dBFS sits well below
// the idle-noise floor for an R820T2 at moderate gain (~-45 dBFS)
// so legitimate weak signal doesn't trip it.
const iqLowPowerThresholdDbFS = -55.0

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
	// SampleRateHz is the raw SDR IQ stream rate (e.g. 2_048_000).
	// The decoder's digital down-converter decimates it to a
	// narrowband channel rate (~48 kHz for most protocols); the
	// per-protocol receiver factories are handed that decimated
	// rate, not this one, so they size their matched filters for
	// the channelized stream.
	SampleRateHz float64
	// Metrics is the optional IQ-power observer the pump updates
	// once per iqPowerWindow. Nil disables the gauge but leaves the
	// low-power debug log in place — operators without Prometheus
	// still get a hint when the dongle goes silent.
	Metrics IQPowerObserver
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

	// ddc decimates the raw SDR IQ stream to pipelineRateHz before
	// the active pipeline sees a chunk; ddcTarget records the
	// channel rate it was built for so it is only rebuilt when a
	// retune crosses to a protocol with a different rate (see
	// ddcTargetForProtocol). pipelineRateHz is what the per-protocol
	// factories receive as PipelineOptions.SampleRateHz. All three
	// are owned by handleProgress / pump under mu.
	ddc            *downconverter
	ddcTarget      float64
	pipelineRateHz float64

	// sub is the bus subscription the Decoder uses to learn about
	// KindHuntProgress retunes. Subscribed in New so the
	// subscription is alive before any other goroutine
	// (notably the cchunt supervisor) starts publishing on the
	// same bus — eliminates the race where the first
	// HuntProgress fires before Run subscribes.
	sub *events.Subscription

	mu       sync.Mutex
	active   ProtocolPipeline
	activeAt string // system name the active pipeline is bound to

	// ddcOut is the reusable down-converter output buffer — pump
	// hands it to downconverter.Process each chunk so the decimated
	// stream doesn't allocate per call.
	ddcOut []complex64

	// IQ-power tracking — see pump for the math. Owned by Run's
	// goroutine via pump; not protected by mu because no other
	// goroutine reads it.
	metrics      IQPowerObserver
	pwSumSq      float64
	pwSamples    int
	pwWindowAt   time.Time
	pwLowLogAt   time.Time
	pwLastSystem string
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
		sub:          opts.Bus.Subscribe(),
		metrics:      opts.Metrics,
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

	defer d.sub.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-d.sub.C:
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
		d.clearActiveLocked()
	}
	// (Re)build the down-converter for this protocol's channel rate
	// before the factory call — the factory sizes its matched
	// filter from the decimated rate.
	d.ensureDownconverterLocked(ddcTargetForProtocol(sys.Protocol))
	rate := d.pipelineRateHz
	d.mu.Unlock()

	p2, err := factory(PipelineOptions{
		Bus:          d.bus,
		Log:          d.log,
		SystemName:   sys.Name,
		FrequencyHz:  p.AttemptedFreqHz,
		SampleRateHz: rate,
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
	// Flush the down-converter so decimation-filter state from the
	// previous channel doesn't bleed into the freshly-tuned one.
	d.ddc.Reset()
	d.mu.Unlock()
}

func (d *Decoder) clearActive() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clearActiveLocked()
}

// clearActiveLocked closes and forgets the active pipeline. Caller
// holds d.mu. Also drops the IQ-power gauge series for the previously
// active system so stale dBFS doesn't linger.
func (d *Decoder) clearActiveLocked() {
	if d.active != nil {
		_ = d.active.Close()
		d.active = nil
	}
	if d.activeAt != "" && d.metrics != nil {
		d.metrics.ClearIQPowerDbFS(d.activeAt)
	}
	d.activeAt = ""
	d.pwSumSq = 0
	d.pwSamples = 0
	d.pwLastSystem = ""
}

// ensureDownconverterLocked (re)builds the down-converter when the
// target channel rate changes. Protocols with different symbol
// rates need different channelized rates — the 4800-baud C4FM
// family channelizes to ~48 kHz, TETRA's 18000-baud modulation to a
// wider channel — so a retune that crosses protocols rebuilds it;
// retunes within the same rate reuse the existing filter. Caller
// holds d.mu.
func (d *Decoder) ensureDownconverterLocked(targetHz float64) {
	if d.ddc != nil && d.ddcTarget == targetHz {
		return
	}
	d.ddc = newDownconverter(d.sampleRateHz, targetHz)
	d.ddcTarget = targetHz
	d.pipelineRateHz = d.ddc.outRateHz
	d.log.Info("ccdecoder: digital down-converter configured",
		"sdr_rate_hz", d.sampleRateHz,
		"pipeline_rate_hz", d.pipelineRateHz)
}

// pump down-converts a raw IQ chunk and forwards it to the active
// pipeline. Holding the lock for the whole Process call serialises
// against handleProgress's pipeline swap so we never call Process on
// a half-constructed pipeline.
//
// The chunk is decimated to the pipeline's narrowband rate by the
// down-converter before Process; the IQ-power window below still
// measures the *raw* chunk so the gauge reflects the SDR's actual
// input level, not the post-decimation level.
//
// While we have the lock, also fold the chunk into the IQ-power
// window. The window aggregates |IQ|^2 across chunks; once a second
// of samples has been seen, the mean is converted to dBFS and pushed
// to the Metrics observer + a throttled debug log fires if the level
// looks dead. See iqLowPowerThresholdDbFS.
func (d *Decoder) pump(iq []complex64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.observeIQPowerLocked(iq)
	if d.active == nil {
		return
	}
	d.ddcOut = d.ddc.Process(d.ddcOut, iq)
	d.active.Process(d.ddcOut)
}

func (d *Decoder) observeIQPowerLocked(iq []complex64) {
	if len(iq) == 0 || d.sampleRateHz <= 0 {
		return
	}
	// Reset window state if the system the gauge is labelled with
	// just changed — different system means different gauge label;
	// don't fold old samples into the new one's average.
	if d.activeAt != d.pwLastSystem {
		d.pwSumSq = 0
		d.pwSamples = 0
		d.pwLastSystem = d.activeAt
	}
	for _, c := range iq {
		r := float64(real(c))
		i := float64(imag(c))
		d.pwSumSq += r*r + i*i
	}
	d.pwSamples += len(iq)

	now := time.Now()
	if d.pwWindowAt.IsZero() {
		d.pwWindowAt = now
		return
	}
	if now.Sub(d.pwWindowAt) < iqPowerWindow {
		return
	}
	mean := d.pwSumSq / float64(d.pwSamples)
	// 10*log10(0) is -Inf; clamp with a small epsilon so the gauge
	// reads as "very low" instead of breaking JSON encoders.
	if mean < 1e-12 {
		mean = 1e-12
	}
	dbfs := 10 * math.Log10(mean)
	if d.activeAt != "" && d.metrics != nil {
		d.metrics.RecordIQPowerDbFS(d.activeAt, dbfs)
	}
	if dbfs < iqLowPowerThresholdDbFS && now.Sub(d.pwLowLogAt) >= 5*time.Second {
		d.log.Debug("ccdecoder: iq power very low — check antenna, gain, USB",
			"system", d.activeAt, "dbfs", dbfs)
		d.pwLowLogAt = now
	}
	d.pwSumSq = 0
	d.pwSamples = 0
	d.pwWindowAt = now
}

// Close releases the active pipeline. Safe to call from outside Run;
// Run also runs Close on the active pipeline as part of normal swap
// cleanup.
func (d *Decoder) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.active != nil {
		err := d.active.Close()
		d.clearActiveLocked()
		return err
	}
	d.clearActiveLocked()
	return nil
}
