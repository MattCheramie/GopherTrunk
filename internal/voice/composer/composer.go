// Package composer bridges the trunking engine's CallStart events to
// the per-call demod chain that turns IQ samples on a freshly-tuned
// Voice device into 16-bit PCM the recorder can write.
//
// One Composer subscribes to events.KindCallStart / events.KindCallEnd
// from the bus. On a CallStart it looks up the Voice device by serial,
// opens its IQ stream, and starts a goroutine that runs an FM
// passthrough chain (LPF → decimate → quadrature FM demod → coarse
// resample → int16 PCM → recorder.WritePCM). The chain also calls
// Engine.Touch on a one-second cadence so the engine's silent-call
// watchdog doesn't kill the call mid-conversation.
//
// Digital protocols (P25 / DMR / NXDN) need vocoders that haven't
// landed yet — IMBE for P25 Phase 1 is in progress and AMBE+2 stays
// behind a build tag. Until they ship, a digital grant produces no
// PCM and only the raw-frame sidecar (when WriteRaw is enabled, or
// always for EDACS ProVoice grants) is useful. The composer logs a
// warning per digital grant so the behaviour is visible in operations.
package composer

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/equalizer"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// EqualizerConfig opts an adaptive blind equalizer (CMA) into the
// post-decimation, pre-FM-demod stage of the per-call analog chain.
// The win is simulcast-distortion mitigation: when multiple
// transmitters cover the same frequency at slightly different arrival
// delays, CMA drives the complex baseband back toward a constant
// modulus and unblurs the FM signal. Defaults are conservative — eight
// taps and a small step size — chosen so the equalizer behaves close
// to a pass-through on a clean signal and converges within a few
// hundred samples on a degraded one.
//
// FM voice has constant envelope on air, so the LMS variant (which
// needs a known training symbol stream) isn't useful here; the LMS
// type is exported from internal/dsp/equalizer for protocol decoders
// (P25 C4FM with a known FSW preamble) that need a directed update.
type EqualizerConfig struct {
	Enabled  bool
	Taps     int     // default 8
	StepSize float32 // default 1e-4
}

// IQSource is the subset of sdr.Device the composer needs. Decoupling
// keeps the package free of a hard import on internal/sdr and makes
// testing trivial with an in-memory channel.
type IQSource interface {
	StreamIQ(ctx context.Context) (<-chan []complex64, error)
}

// Devices resolves a Voice-role IQ source by its serial. The daemon
// supplies a wrapper around sdr.Pool; tests use a map.
type Devices interface {
	FindBySerial(serial string) IQSource
}

// PCMSink is the subset of voice.Recorder we touch. Recorder.WritePCM
// matches this signature exactly.
type PCMSink interface {
	WritePCM(deviceSerial string, samples []int16) error
}

// EngineHooks exposes the Touch / EndCall calls the chain uses to
// keep the engine in sync with what the chain hears. Stubbing this
// interface lets tests assert Touch fires on a real cadence.
type EngineHooks interface {
	Touch(deviceSerial string)
	EndCall(deviceSerial string, reason trunking.EndReason) bool
}

// Options configure a Composer.
type Options struct {
	Bus     *events.Bus
	Devices Devices
	Sink    PCMSink     // typically *voice.Recorder
	Engine  EngineHooks // typically *trunking.Engine
	Log     *slog.Logger
	// IQSampleRate is the per-second sample rate the SDR pool delivers
	// (typically 2.4e6). Required.
	IQSampleRate uint32
	// PCMSampleRate is the rate the recorder expects (default 8000).
	PCMSampleRate uint32
	// VoiceBandwidthHz is the cutoff of the front-end LPF (default
	// 12_500 — wide enough for analog FM voice with some margin).
	VoiceBandwidthHz uint32
	// TouchInterval is how often the chain pings Engine.Touch while
	// audio is flowing (default 1 s).
	TouchInterval time.Duration
	// Equalizer optionally enables a CMA blind equalizer between the
	// front-end LPF and the FM demod. Off by default; flip Enabled
	// to true and tune Taps / StepSize per site.
	Equalizer EqualizerConfig
}

// Composer is the long-lived event-driven bridge.
type Composer struct {
	bus    *events.Bus
	dev    Devices
	sink   PCMSink
	engine EngineHooks
	log    *slog.Logger

	iqHz         uint32
	pcmHz        uint32
	bw           uint32
	touchEvery   time.Duration
	eqCfg        EqualizerConfig

	sub       *events.Subscription
	runDone   chan struct{}
	closeOnce sync.Once

	mu     sync.Mutex
	chains map[string]*chain
}

type chain struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// New validates options and constructs a Composer. The bus subscription
// is created at construction time so callers can publish events before
// Run starts without losing them.
func New(opts Options) (*Composer, error) {
	if opts.Bus == nil {
		return nil, errors.New("composer: events.Bus is required")
	}
	if opts.Devices == nil {
		return nil, errors.New("composer: Devices is required")
	}
	if opts.IQSampleRate == 0 {
		return nil, errors.New("composer: IQSampleRate is required")
	}
	if opts.PCMSampleRate == 0 {
		opts.PCMSampleRate = 8000
	}
	if opts.VoiceBandwidthHz == 0 {
		opts.VoiceBandwidthHz = 12_500
	}
	if opts.TouchInterval <= 0 {
		opts.TouchInterval = time.Second
	}
	if opts.Equalizer.Enabled {
		if opts.Equalizer.Taps <= 0 {
			opts.Equalizer.Taps = 8
		}
		if opts.Equalizer.StepSize <= 0 {
			opts.Equalizer.StepSize = 1e-4
		}
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	c := &Composer{
		bus:        opts.Bus,
		dev:        opts.Devices,
		sink:       opts.Sink,
		engine:     opts.Engine,
		log:        log,
		iqHz:       opts.IQSampleRate,
		pcmHz:      opts.PCMSampleRate,
		bw:         opts.VoiceBandwidthHz,
		touchEvery: opts.TouchInterval,
		eqCfg:      opts.Equalizer,
		chains:     make(map[string]*chain),
		runDone:    make(chan struct{}),
	}
	c.sub = opts.Bus.Subscribe()
	return c, nil
}

// Run drains CallStart / CallEnd events until ctx cancels, spawning /
// reaping per-call demod goroutines. Every active chain is cancelled
// on context cancel so Close drains cleanly.
func (c *Composer) Run(ctx context.Context) error {
	defer close(c.runDone)
	for {
		select {
		case <-ctx.Done():
			c.cancelAll()
			return ctx.Err()
		case ev, ok := <-c.sub.C:
			if !ok {
				c.cancelAll()
				return nil
			}
			switch ev.Kind {
			case events.KindCallStart:
				if cs, ok := ev.Payload.(trunking.CallStart); ok {
					c.handleStart(ctx, cs)
				}
			case events.KindCallEnd:
				if ce, ok := ev.Payload.(trunking.CallEnd); ok {
					c.handleEnd(ce)
				}
			}
		}
	}
}

// Close releases the bus subscription and waits for Run to drain. It
// also cancels every active chain. Idempotent.
func (c *Composer) Close() error {
	c.closeOnce.Do(func() {
		c.sub.Close()
		select {
		case <-c.runDone:
		case <-time.After(time.Second):
		}
		c.cancelAll()
	})
	return nil
}

// ActiveChains returns the device serials with running chains. Test
// helper; takes the lock so it's race-free.
func (c *Composer) ActiveChains() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.chains))
	for k := range c.chains {
		out = append(out, k)
	}
	return out
}

func (c *Composer) handleStart(parent context.Context, cs trunking.CallStart) {
	if cs.Grant.Protocol != "" && cs.Grant.Protocol != "fm" && cs.Grant.Protocol != "analog" {
		// Digital protocols need a vocoder we don't have yet. The
		// recorder still gets the CallStart event itself, so the .raw
		// sidecar (when configured) and the call_log row land. We
		// simply don't push PCM.
		c.log.Info("composer: digital protocol; skipping FM chain",
			"device", cs.DeviceSerial, "protocol", cs.Grant.Protocol,
			"group", cs.Grant.GroupID)
		return
	}
	src := c.dev.FindBySerial(cs.DeviceSerial)
	if src == nil {
		c.log.Warn("composer: no device for serial", "serial", cs.DeviceSerial)
		return
	}
	c.mu.Lock()
	if existing := c.chains[cs.DeviceSerial]; existing != nil {
		// Engine should have ended the prior call first; defensive.
		existing.cancel()
		<-existing.done
		delete(c.chains, cs.DeviceSerial)
	}
	c.mu.Unlock()

	chainCtx, cancel := context.WithCancel(parent)
	iqCh, err := src.StreamIQ(chainCtx)
	if err != nil {
		cancel()
		c.log.Warn("composer: StreamIQ failed", "serial", cs.DeviceSerial, "err", err)
		return
	}
	ch := &chain{cancel: cancel, done: make(chan struct{})}
	c.mu.Lock()
	c.chains[cs.DeviceSerial] = ch
	c.mu.Unlock()

	go c.runFMChain(chainCtx, cs.DeviceSerial, iqCh, ch.done)
}

func (c *Composer) handleEnd(ce trunking.CallEnd) {
	c.mu.Lock()
	ch := c.chains[ce.DeviceSerial]
	delete(c.chains, ce.DeviceSerial)
	c.mu.Unlock()
	if ch == nil {
		return
	}
	ch.cancel()
	<-ch.done
}

func (c *Composer) cancelAll() {
	c.mu.Lock()
	chains := c.chains
	c.chains = make(map[string]*chain)
	c.mu.Unlock()
	for _, ch := range chains {
		ch.cancel()
		<-ch.done
	}
}

// runFMChain consumes IQ for one call. The chain is intentionally
// straightforward: LPF the IQ to voice bandwidth, naive-decimate to
// roughly 48 kHz, quadrature-FM-demod, naive-decimate again to the
// recorder's PCM rate, and convert to int16. A higher-fidelity version
// (proper polyphase resamplers, de-emphasis, post-demod LPF) is a
// follow-up; this is honest passthrough quality good enough to verify
// the wiring end-to-end and to land the operator-visible plumbing.
func (c *Composer) runFMChain(ctx context.Context, serial string, iqCh <-chan []complex64, done chan<- struct{}) {
	defer close(done)

	// Front-end LPF: cutoff = bw / iqHz (normalized 0..0.5).
	cutoff := float64(c.bw) / float64(c.iqHz)
	if cutoff > 0.45 {
		cutoff = 0.45
	}
	taps := filter.LowpassKaiser(81, cutoff, 8.6)
	lpf := filter.NewFIR(taps)

	// Naive decimation factors. They aren't exact; resampling-quality
	// audio lands with the polyphase resampler in a follow-up.
	const intermediateHz = 48_000
	decim1 := int(c.iqHz) / intermediateHz
	if decim1 < 1 {
		decim1 = 1
	}
	decim2 := intermediateHz / int(c.pcmHz)
	if decim2 < 1 {
		decim2 = 1
	}

	fm := demod.NewFM()

	// Optional CMA blind equalizer for simulcast-distortion mitigation.
	// Sits between the front-end LPF (decimated) and the FM demod so it
	// operates at the intermediate rate (~48 kHz) rather than 2.4 MS/s.
	// R^2 = 1 because FM has unit-magnitude carrier on air.
	var eq *equalizer.CMA
	if c.eqCfg.Enabled {
		eq = equalizer.NewCMA(c.eqCfg.Taps, c.eqCfg.StepSize, 1.0)
	}

	touchTicker := time.NewTicker(c.touchEvery)
	defer touchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-touchTicker.C:
			if c.engine != nil {
				c.engine.Touch(serial)
			}
		case iq, ok := <-iqCh:
			if !ok {
				return
			}
			filtered := lpf.Process(nil, iq)
			decimated := decimateComplex(filtered, decim1)
			if eq != nil {
				equalized := make([]complex64, len(decimated))
				for i, x := range decimated {
					y, _ := eq.Process(x)
					equalized[i] = y
				}
				decimated = equalized
			}
			audio := fm.Process(nil, decimated)
			pcm := decimateAndConvert(audio, decim2)
			if c.sink != nil && len(pcm) > 0 {
				_ = c.sink.WritePCM(serial, pcm)
			}
		}
	}
}

func decimateComplex(in []complex64, factor int) []complex64 {
	if factor <= 1 {
		return in
	}
	out := make([]complex64, 0, len(in)/factor+1)
	for i := 0; i < len(in); i += factor {
		out = append(out, in[i])
	}
	return out
}

// decimateAndConvert decimates a real audio stream and converts to
// 16-bit signed PCM. The FM demodulator emits radians/sample in
// roughly [-π, +π]; we scale by ~10 000 to fill the int16 range for
// reasonable deviation, then clamp.
func decimateAndConvert(in []float32, factor int) []int16 {
	if factor < 1 {
		factor = 1
	}
	out := make([]int16, 0, len(in)/factor+1)
	for i := 0; i < len(in); i += factor {
		v := float64(in[i]) * 10_000
		if v > math.MaxInt16 {
			v = math.MaxInt16
		}
		if v < math.MinInt16 {
			v = math.MinInt16
		}
		out = append(out, int16(v))
	}
	return out
}
