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
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/autotune"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// IQPowerObserver is the minimal Metrics surface the decoder uses to
// publish its window-averaged dBFS gauge. internal/metrics.Metrics
// satisfies this; nil disables the gauge entirely.
//
// RecordIQDCRatioDb reports the per-window DC-bin power relative to
// total IQ power, in dB (so 0 means all power is in the DC bin, very
// negative means the DC content is negligible). It surfaces the
// RTL-SDR R820T2 DC-spike-on-channel failure mode behind issue #402:
// when the channel of interest sits on top of the tuner zero, the
// DC bin dominates the 48 kHz pipeline passband and the C4FM eye
// collapses. Healthy off-channel signals show ≤ -20 dB; a
// DC-dominated capture shows within ~5 dB of 0.
// RecordIQClipRatio reports the per-window fraction of IQ samples with
// an I or Q component pinned to the ADC rail (0 = none). Sustained
// non-zero values mean front-end overload/clipping — the average
// iq_power_dbfs gauge is RMS and hides peak clipping, so this is the
// authoritative clipping signal behind issue #402.
type IQPowerObserver interface {
	RecordIQPowerDbFS(system string, dbfs float64)
	ClearIQPowerDbFS(system string)
	RecordIQDCRatioDb(system string, ratioDb float64)
	ClearIQDCRatioDb(system string)
	RecordIQClipRatio(system string, ratio float64)
	ClearIQClipRatio(system string)
	// RecordDecodeOverrun counts one IQ chunk dropped at the decode queue
	// because decode could not keep up with real time (issue #402). It is
	// distinct from the SDR driver's iq_underruns_total: a non-zero value
	// here means the CPU/host can't sustain the decode, not that the USB
	// delivery channel overran.
	RecordDecodeOverrun()
}

// FECObserver is the minimal metrics surface for the opt-in per-protocol
// FEC correction-depth histograms (today: TETRA). internal/metrics.Metrics
// satisfies it; the daemon wires it only when metrics.detailed_fec is set,
// so nil is the normal case and disables the histogram path end-to-end at
// zero cost (the pipeline never computes a correction depth).
type FECObserver interface {
	RecordTetraViterbiCorrections(system, channel string, corrections int)
}

// ErrIQStreamClosed is returned by Run whenever the SDR's IQ stream is
// unexpectedly unavailable while the context is still live — either the
// channel closed mid-stream (the USB reaper died of an unrecoverable
// error: host controller hang, ENODEV, EPROTO storm under load) or
// StreamIQ failed to open at all on a Run retry (the underlying device
// has physically disconnected and the dead handle now rejects every
// control transfer with `usb: device disconnected`). Both shapes feed
// the same restart loop in the daemon; the underlying error is
// preserved via %w chaining so callers can still inspect the root
// cause. See issue #345.
var ErrIQStreamClosed = errors.New("ccdecoder: IQ stream closed unexpectedly")

// streamDeadCauser is the optional extension a driver (airspy, rtlsdr-purego,
// or the iqtap broker wrapping them) implements to report the concrete USB
// error that killed its last bulk-IN stream.
type streamDeadCauser interface{ StreamDeadCause() error }

// streamClosedErr returns ErrIQStreamClosed, wrapping the IQ source's concrete
// stream-death cause when it exposes one, so the daemon's "IQ stream died" log
// names the real USB error (usb.ErrStreamStalled / usb.ErrDeviceGone / ...)
// instead of a bare message. Still satisfies errors.Is(err, ErrIQStreamClosed),
// so the retry loop's classification is unchanged.
func streamClosedErr(iq IQSource) error {
	if c, ok := iq.(streamDeadCauser); ok {
		if cause := c.StreamDeadCause(); cause != nil {
			return fmt.Errorf("%w: %w", ErrIQStreamClosed, cause)
		}
	}
	return ErrIQStreamClosed
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

// iqDCRatioWarnDb is the dc-to-total power ratio (in dB) above which
// observeIQPowerLocked emits a throttled debug log noting that the
// IQ DC bin dominates the channel passband. -10 dB picks up the
// RTL-SDR R820T2 zero-IF DC-spike case (where the spike is within
// ~5 dB of total channel power) without firing on benign captures
// of a tone-rich broadcast channel. Issue #402.
const iqDCRatioWarnDb = -10.0

// iqClipThreshold is the normalized magnitude at or above which an I or
// Q component is treated as ADC rail-clipping. convertU8IQ maps u8 0 ->
// -1.0 and u8 255 -> +1.0 (rtlsdr/purego/stream.go convertU8IQ), so the
// rail samples u8 in {0,1,254,255} land at |x| >= ~0.992; 0.98 catches
// them without flagging healthy signal peaks. A front end driven into
// the rail distorts the C4FM eye and fails TSBK CRC continuously even
// while NID's stronger BCH still locks (issue #402).
const iqClipThreshold = 0.98

// iqClipWarnRatio is the per-window fraction of clipped samples above
// which observeIQPowerLocked emits a throttled WARN. A clean front end
// rarely rails on P25 C4FM, so a sustained fraction at this level means
// front-end overload — reduce gain or add attenuation (issue #402).
const iqClipWarnRatio = 0.002

// defaultCarrierOffsetWarnHz is the default magnitude of the TOTAL control-
// channel carrier offset (autotune correction + receiver residual) above which,
// while locked, the pump warns that GT may have locked onto an adjacent site's
// carrier rather than the configured frequency (issue #815). Overridable via
// Options.CarrierOffsetWarnHz.
//
// 4000 Hz sits in the gap between: a good front-end (Airspy/TCXO) after autotune
// leaves a sub-kHz residual (never trips); the P25 6.25/12.5 kHz adjacent-channel
// steps both clear it (an adjacent-site lock always trips); a reasonably-corrected
// dongle's 1-3 kHz crystal error stays under it.
const defaultCarrierOffsetWarnHz = 4000

// carrierOffsetWarnInterval throttles a sustained large-offset WARN, matching the
// iqClipLog / iqDCLog 30 s cadence so a stuck adjacent-channel lock logs steadily.
const carrierOffsetWarnInterval = 30 * time.Second

// decodeQueueDepth is the size of the internal queue between the
// lightweight IQ forwarder and the decode goroutine (issue #402). The
// SDR driver drops the instant its own small delivery channel backs up
// (~27 ms at 2.4 MS/s, ~68 ms at 960 kS/s), so any decode/retune/GC
// stall longer than that on a single combined goroutine corrupts the
// continuous symbol stream. The forwarder drains the driver channel into
// this larger queue so a transient stall backs up here instead of
// dropping RF; at 65 KB/chunk this is ~8 MB and buys ≈0.44 s of slack at
// 2.4 MS/s (≈1.1 s at 960 kS/s). Deeper would absorb longer stalls at the
// cost of higher decode latency under a sustained backlog; 128 balances
// the two. Sustained overload still drops here — attributably, via
// RecordDecodeOverrun — rather than silently in the driver.
const decodeQueueDepth = 128

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
	// FEC is the optional FEC correction-depth observer. Nil (the
	// default) disables the per-protocol FEC histograms; the daemon
	// supplies it only when metrics.detailed_fec is enabled.
	FEC FECObserver
	// IQCorrect enables blind I/Q-imbalance correction on the raw IQ
	// before decimation (issue #402). Off by default; opt-in per device
	// via config. Validate with `replay -iq-correct -diag` on a capture
	// before enabling in production.
	IQCorrect bool
	// Conjugate negates the imaginary part of every raw IQ sample before
	// channelization, undoing a spectrum-inverted / I-Q-swapped front end
	// (issue #264, the same correction as `replay -conjugate`). Off by
	// default; opt-in per device via config (iq_invert). A π/4-DQPSK
	// protocol such as TETRA cannot lock an inverted spectrum because the
	// inversion reverses every differential phase transition.
	Conjugate bool
	// LOOffsetHz is the live LO-offset (DC-spike-avoidance) tuning the daemon
	// applies on the control SDR (issue #402). The hardware LO is tuned
	// LOOffsetHz BELOW the logical control-channel frequency, so the wanted
	// channel sits at +LOOffsetHz in the delivered baseband; the
	// down-converter is built with this same offset so the channel is mixed
	// back to 0 Hz, off the front-end DC spur / 1/f noise / self-image. Zero
	// (the default) builds no NCO and is byte-exact with the pre-offset
	// behaviour — used when the path is disabled or at pass-through rates
	// where there is no room to offset.
	LOOffsetHz float64
	// Autotune, when non-nil, is the per-dongle carrier-error tracker for
	// the control SDR (sdr.autotune enabled). The decoder samples the
	// locked C4FM receiver's AFCOffsetHz once per second, folds it into the
	// running average, and shifts the digital down-converter by that
	// average at each (re)acquisition so the demod's AFC starts near lock.
	// Nil disables autotune at zero cost. See internal/autotune.
	Autotune *autotune.Manager
	// CarrierOffsetWarnHz is the magnitude of the total locked carrier offset
	// (autotune correction + receiver residual) above which the pump emits a
	// throttled WARN that GT may have locked onto an adjacent site's carrier
	// (issue #815). Zero selects defaultCarrierOffsetWarnHz. Raise it for a
	// high-drift dongle to avoid benign warnings.
	CarrierOffsetWarnHz int
}

// Decoder is the long-lived component that converts the control
// SDR's IQ stream into CC / grant events on the bus. Construct via
// New, run via Run.
type Decoder struct {
	bus          *events.Bus
	log          *slog.Logger
	iq           IQSource
	sampleRateHz float64
	// loOffsetHz is the DC-spike-avoidance LO offset (issue #402): the
	// channel sits at +loOffsetHz in the delivered baseband and the DDC is
	// built with this offset to mix it back to 0 Hz. Zero disables it.
	loOffsetHz float64
	// autotune, when non-nil, tracks the control dongle's carrier error
	// (Options.Autotune). autotuneApplied records the Hz correction folded
	// into the DDC offset at the last (re)build, so the next measurement
	// reports observed + applied as the source's true error, and the
	// published control_channel_carrier_offset_hz can add it back to the
	// receiver residual to report the TOTAL offset from the configured
	// frequency (issue #815). Atomic so the P25 pipeline's CarrierOffsetHz
	// closure can read it lock-free at site-update publish time. locked
	// mirrors the CC lock state from the bus so the pump only samples AFC on
	// a real lock; lastATSampleAt throttles sampling to roughly once per
	// second (owned by the pump goroutine).
	autotune        *autotune.Manager
	autotuneApplied atomic.Int64
	locked          atomic.Bool
	lastATSampleAt  time.Time
	// carrierOffsetWarnHz is the large-offset WARN threshold (Options.
	// CarrierOffsetWarnHz, defaulted in New). lastOffsetWarnAt throttles the
	// WARN to one line per carrierOffsetWarnInterval while the offset stays
	// over threshold; it resets to zero when the offset falls back under
	// threshold so a fresh excursion re-warns at once (owned by the pump
	// goroutine). Issue #815.
	carrierOffsetWarnHz int
	lastOffsetWarnAt    time.Time
	systems             map[string]trunking.System

	// ddc decimates the raw SDR IQ stream to pipelineRateHz before
	// the active pipeline sees a chunk; ddcTarget records the
	// channel rate it was built for so it is only rebuilt when a
	// retune crosses to a protocol with a different rate (see
	// ddcTargetForProtocol). pipelineRateHz is what the per-protocol
	// factories receive as PipelineOptions.SampleRateHz. All three
	// are owned by handleProgress / pump under mu.
	ddc       *Downconverter
	ddcTarget float64
	// ddcOffsetHz is the frequency offset the current DDC was built with
	// (loOffsetHz + the autotune correction). Part of the rebuild key so a
	// changed autotune average re-tunes the down-converter at the next
	// acquisition; without autotune it stays equal to loOffsetHz and the
	// build is byte-exact with before.
	ddcOffsetHz    float64
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
	activeAt string // system name the active pipeline is bound to (guarded by mu)
	// activeSystem mirrors activeAt as the gauge label the forwarder
	// goroutine uses, so observeIQPower can label its metrics without taking
	// mu (issue #402). Written under mu wherever activeAt is; read lock-free
	// by the forwarder. Never nil after New.
	activeSystem atomic.Pointer[string]
	// activeFreqHz is the frequency the active pipeline is tuned to. A
	// HuntProgress that repeats the same (system, frequency) — e.g. the
	// supervisor re-hunting a single-candidate system every dwell —
	// must NOT tear the pipeline down and flush the DDC + symbol clock,
	// or acquisition restarts from scratch every few seconds and never
	// converges on a live stream (issue #402: "no FSW hits in chunk"
	// cycling with "pipeline configured"). Zero until the first build.
	activeFreqHz uint32

	// ddcOut is the reusable down-converter output buffer — pump
	// hands it to Downconverter.Process each chunk so the decimated
	// stream doesn't allocate per call.
	ddcOut []complex64

	// IQ-power tracking — see observeIQPower for the math. Owned solely by
	// the forwarder goroutine, which observes every delivered chunk before
	// queueing it for decode (issue #402), so these need no lock. The
	// cross-goroutine IQHealth snapshot it derives is the separate,
	// healthMu-guarded block below.
	metrics      IQPowerObserver
	fec          FECObserver
	pwSumSq      float64
	pwSumI       float64 // running sum of I samples → DC-bin mean (issue #402)
	pwSumQ       float64 // running sum of Q samples → DC-bin mean (issue #402)
	pwClipped    int     // count of rail-clipped samples in the window (issue #402)
	pwSamples    int
	pwWindowAt   time.Time
	pwLowLogAt   time.Time
	pwDCLogAt    time.Time // throttle for the "DC bin dominant" debug line (issue #402)
	pwClipLogAt  time.Time // throttle for the "front-end clipping" warn line (issue #402)
	pwLastSystem string

	// iqPool recycles the decode-queue buffers (issue #402). The forwarder
	// copies each raw driver chunk into a pooled buffer before queueing it,
	// so a buffer sitting in the deep decode queue is owned by the decoder,
	// never the SDR driver's small reuse ring — without the copy a backlog
	// longer than that ring (issue #489) would let the driver overwrite a
	// chunk still awaiting decode. pump returns the buffer here when done.
	iqPool sync.Pool

	// iqCorrector, when non-nil (Options.IQCorrect), blindly removes the
	// front-end I/Q imbalance from the raw IQ in pump before decimation
	// (issue #402). Owned by pump.
	iqCorrector *rtlsdr.IQImbalanceCorrector

	// conjugate, when true (Options.Conjugate), negates Q on every raw IQ
	// chunk in pump before the I/Q-imbalance correction and decimation, to
	// undo a spectrum-inverted front end (issue #264). Owned by pump.
	conjugate bool

	// Persisted last-window IQ-health snapshot. Unlike the pw* window
	// accumulators above (reset every iqPowerWindow), these are kept so
	// IQHealth can answer the cchunt supervisor at hunt-failure time —
	// the supervisor attaches the result to the cchunt.failed event/log
	// so a field report explains itself. lastHealthSystem labels which
	// system the snapshot belongs to; totalSamples is cumulative since
	// that system became the active pipeline (reset on system change /
	// clearActive). Guarded by healthMu: written by the forwarder's
	// observeIQPower, read by IQHealth from the cchunt supervisor's
	// goroutine, reset by clearActive on the decode goroutine.
	healthMu         sync.Mutex
	lastHealthSystem string
	lastHealthAt     time.Time
	lastDbfs         float64
	lastClipRatio    float64
	lastDCRatioDb    float64
	totalSamples     int64
}

// getIQBuf returns a recycled []complex64 of length n for the decode
// queue, or allocates one when the pool is empty or holds a too-small
// buffer. putIQBuf returns it after pump is done. See the iqPool field.
func (d *Decoder) getIQBuf(n int) []complex64 {
	if v := d.iqPool.Get(); v != nil {
		s := v.(*[]complex64)
		if cap(*s) >= n {
			return (*s)[:n]
		}
	}
	return make([]complex64, n)
}

func (d *Decoder) putIQBuf(buf []complex64) {
	d.iqPool.Put(&buf)
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
		loOffsetHz:   opts.LOOffsetHz,
		systems:      make(map[string]trunking.System, len(opts.Systems)),
		sub:          opts.Bus.Subscribe(),
		metrics:      opts.Metrics,
		fec:          opts.FEC,
		autotune:     opts.Autotune,
	}
	empty := ""
	d.activeSystem.Store(&empty)
	if opts.IQCorrect {
		d.iqCorrector = rtlsdr.NewIQImbalanceCorrector()
	}
	d.conjugate = opts.Conjugate
	d.carrierOffsetWarnHz = opts.CarrierOffsetWarnHz
	if d.carrierOffsetWarnHz <= 0 {
		d.carrierOffsetWarnHz = defaultCarrierOffsetWarnHz
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
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		// On the daemon's retry path the previous Tuner handle is
		// often dead (USB device pulled and re-enumerated). Surface
		// the open failure as ErrIQStreamClosed so the daemon's
		// restart loop classifies it the same as a mid-stream reaper
		// death and either retries or escalates to a clean fatal.
		// Issue #345.
		return fmt.Errorf("%w: %w", ErrIQStreamClosed, err)
	}

	defer d.sub.Close()

	// Decouple real-time IQ ingestion from decode (issue #402). The SDR
	// driver drops a chunk the instant its small delivery channel backs up,
	// so running decode + retune inline on the goroutine that drains that
	// channel means any stall longer than a few tens of ms (a pipeline
	// rebuild, a GC pause, the OS scheduling away under host contention)
	// silently drops live IQ and splices the continuous symbol stream,
	// triggering the TSBK/NID CRC cascade. A dedicated lightweight forwarder
	// keeps the driver channel drained into a larger queue; decode and
	// retune run here, off the ingestion path, so a stall backs up in the
	// queue instead of dropping RF.
	decodeCh := make(chan []complex64, decodeQueueDepth)
	go d.forwardIQ(ctx, stream, decodeCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-d.sub.C:
			if !ok {
				return nil
			}
			switch ev.Kind {
			case events.KindCCLocked:
				// Gate autotune sampling on a real lock — the AFC estimate
				// only means anything once the FSW is correlating. Payload
				// shape is per-protocol, so we only read the lock edge.
				d.locked.Store(true)
				continue
			case events.KindCCLost:
				d.locked.Store(false)
				continue
			case events.KindHuntProgress:
			default:
				continue
			}
			p, ok := ev.Payload.(trunking.HuntProgress)
			if !ok {
				continue
			}
			d.handleProgress(p)
		case iq, ok := <-decodeCh:
			if !ok {
				// forwardIQ closed the queue: either the SDR stream ended
				// (surface as ErrIQStreamClosed for the daemon's #345 retry
				// loop) or ctx was cancelled (a clean shutdown, not a stream
				// death — return ctx.Err() so it isn't retried).
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return streamClosedErr(d.iq)
			}
			d.pump(iq)
			d.putIQBuf(iq) // decode done — recycle the queue buffer
		}
	}
}

// forwardIQ drains the SDR delivery channel into the decode queue (issue
// #402). For each delivered chunk it (1) folds the raw IQ into the
// power/clip/health window via observeIQPower, so the gauges reflect every
// chunk the SDR delivered even when decode is dropping, and (2) copies the
// chunk into a pooled, decoder-owned buffer before the non-blocking
// enqueue. The copy is essential: the SDR driver hands out slices from a
// small fixed reuse ring (issue #489) sized for its own delivery channel,
// so a chunk left sitting in the deep decodeCh would be overwritten by a
// later delivery once the ring wrapped — corrupting IQ still awaiting
// decode under exactly the backlog this queue exists to absorb. Copying
// hands the driver's ring slot straight back; only the pooled copy lives
// in the queue. Kept cheap (an O(n) observe + memcpy) so it always
// out-drains the driver channel. Under *sustained* overload the queue
// fills and chunks are dropped here, attributably (RecordDecodeOverrun +
// a throttled WARN), rather than silently in the driver. Closing out on
// return propagates stream EOF / ctx cancellation to Run.
func (d *Decoder) forwardIQ(ctx context.Context, in <-chan []complex64, out chan<- []complex64) {
	defer close(out)
	var (
		dropped   uint64
		lastLogAt time.Time
	)
	for {
		select {
		case <-ctx.Done():
			return
		case iq, ok := <-in:
			if !ok {
				return
			}
			d.observeIQPower(iq) // measures the raw driver chunk (every delivery)
			buf := d.getIQBuf(len(iq))
			copy(buf, iq) // decouple from the driver's reuse ring before queueing
			select {
			case out <- buf:
			default:
				// Decode is sustainedly behind real time: the queue is full.
				// Return the copy to the pool and drop the chunk (caps latency
				// at the queue depth). Surface it as a decode overrun, distinct
				// from the driver's iq_underruns_total, so "CPU can't keep up"
				// is attributable rather than looking like an RF fault.
				d.putIQBuf(buf)
				dropped++
				if d.metrics != nil {
					d.metrics.RecordDecodeOverrun()
				}
				if now := time.Now(); now.Sub(lastLogAt) >= time.Second {
					d.log.Warn("ccdecoder: decode can't keep up with real time; dropping IQ at the decode queue (raise CPU / lower sample rate / shed load) — issue #402",
						"dropped_since_last", dropped)
					dropped = 0
					lastLogAt = now
				}
			}
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
	// Idempotent retune: if the supervisor is re-attempting the same
	// (system, frequency) the active pipeline is already decoding, leave
	// it running. Rebuilding here would Close the pipeline and Reset the
	// DDC + symbol clock, discarding partial sync — and on a single-
	// candidate system the hunter re-attempts every dwell (~3s), so a
	// rebuild-on-every-attempt loop never lets a live stream acquire FSW
	// (issue #402). A genuine retune (different freq or protocol) still
	// rebuilds below.
	if d.active != nil && d.activeAt == sys.Name && d.activeFreqHz == p.AttemptedFreqHz {
		d.mu.Unlock()
		return
	}
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

	popts := PipelineOptions{
		Bus:          d.bus,
		Log:          d.log,
		SystemName:   sys.Name,
		FrequencyHz:  p.AttemptedFreqHz,
		SampleRateHz: rate,
		System:       sys,
		// Report the autotune correction already folded into the DDC so the
		// P25 pipeline's published control_channel_carrier_offset_hz is the
		// TOTAL offset from the configured frequency (applied + receiver
		// residual), matching the WARN in checkCarrierOffsetLocked. With
		// autotune off this is always 0, so the published value is unchanged
		// (issue #815).
		CarrierBiasHz: func() float64 { return float64(d.autotuneApplied.Load()) },
	}
	if d.fec != nil {
		// Bind the system name so the pipeline reports a per-burst FEC
		// correction depth without re-plumbing it. Nil observer ⇒ the
		// pipeline never computes one.
		sysName := sys.Name
		fec := d.fec
		popts.FECObserver = func(channel string, corrections int) {
			fec.RecordTetraViterbiCorrections(sysName, channel, corrections)
		}
	}
	p2, err := factory(popts)
	if err != nil {
		d.log.Warn("ccdecoder: pipeline factory failed",
			"system", p.System, "protocol", sys.Protocol, "err", err)
		return
	}

	d.mu.Lock()
	d.active = p2
	d.activeAt = sys.Name
	name := sys.Name
	d.activeSystem.Store(&name) // gauge label for the forwarder goroutine
	d.activeFreqHz = p.AttemptedFreqHz
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
		d.metrics.ClearIQDCRatioDb(d.activeAt)
		d.metrics.ClearIQClipRatio(d.activeAt)
	}
	d.activeAt = ""
	empty := ""
	d.activeSystem.Store(&empty)
	d.activeFreqHz = 0
	// The pw* window accumulators are owned by the forwarder's
	// observeIQPower, which re-zeros them when it next sees the system
	// label change — touching them here would race that goroutine.
	// Drop the persisted health snapshot (under healthMu, since the
	// forwarder writes it) so a torn-down system's IQ health can't answer
	// a later IQHealth query for a different one.
	d.healthMu.Lock()
	d.lastHealthSystem = ""
	d.lastHealthAt = time.Time{}
	d.totalSamples = 0
	d.healthMu.Unlock()
}

// ensureDownconverterLocked (re)builds the down-converter when the
// target channel rate changes. Protocols with different symbol
// rates need different channelized rates — the 4800-baud C4FM
// family channelizes to ~48 kHz, TETRA's 18000-baud modulation to a
// wider channel — so a retune that crosses protocols rebuilds it;
// retunes within the same rate reuse the existing filter. Caller
// holds d.mu.
func (d *Decoder) ensureDownconverterLocked(targetHz float64) {
	// The DDC mixes the channel sitting at +loOffsetHz in the delivered
	// baseband down to 0 Hz. A dongle crystal error δ leaves the wanted
	// carrier at loOffsetHz+δ instead, so the receiver's AFC has to drag it
	// in from δ away. Autotune estimates δ as a running average and folds it
	// into the mix offset here, so each fresh acquisition starts with the
	// carrier already near DC. The correction only changes between
	// acquisitions (the average is sampled while locked but applied at the
	// next build), so a stable lock never sees its DDC rebuilt out from
	// under it.
	offset := d.loOffsetHz
	applied := 0
	if d.autotune != nil {
		// Correction() stays 0 through warm-up, so a cold dongle doesn't get
		// its DDC shifted by an unsettled first measurement.
		applied = d.autotune.Correction()
		offset += float64(applied)
	}
	if d.ddc != nil && d.ddcTarget == targetHz && d.ddcOffsetHz == offset {
		return
	}
	d.ddc = NewDownconverterWithOffset(d.sampleRateHz, targetHz, offset)
	d.ddcTarget = targetHz
	d.ddcOffsetHz = offset
	d.autotuneApplied.Store(int64(applied))
	d.pipelineRateHz = d.ddc.outRateHz
	d.log.Info("ccdecoder: digital down-converter configured",
		"sdr_rate_hz", d.sampleRateHz,
		"pipeline_rate_hz", d.pipelineRateHz,
		"lo_offset_hz", d.loOffsetHz,
		"autotune_hz", applied)
}

// pump down-converts a raw IQ chunk and forwards it to the active
// pipeline. Holding the lock for the whole Process call serialises
// against handleProgress's pipeline swap so we never call Process on
// a half-constructed pipeline.
//
// IQ-power observation is NOT done here: the forwarder folds every
// delivered chunk into the window via observeIQPower before queueing it,
// so the gauges stay accurate even for chunks this decode path never sees
// because the queue was full (issue #402).
func (d *Decoder) pump(iq []complex64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conjugate {
		// Undo a spectrum-inverted / I-Q-swapped front end, in place,
		// before any other correction (issue #264).
		for i, s := range iq {
			iq[i] = complex(real(s), -imag(s))
		}
	}
	if d.iqCorrector != nil {
		d.iqCorrector.Process(iq) // blind I/Q-imbalance correction, in place (issue #402)
	}
	if d.active == nil {
		return
	}
	d.ddcOut = d.ddc.Process(d.ddcOut, iq)
	d.active.Process(d.ddcOut)
	d.sampleAutotuneLocked()
	d.checkCarrierOffsetLocked()
}

// afcReporter is the optional capability a pipeline exposes when its
// receiver tracks a carrier-frequency offset in Hz (today: P25 Phase 1
// C4FM). Autotune reads it to fold the residual offset into the running
// average. Pipelines without an AFC stage simply don't implement it.
type afcReporter interface {
	AFCOffsetHz() float64
}

// sampleAutotuneLocked folds the active receiver's residual carrier offset
// into the running average about once a second while the CC is locked, then
// logs the suggested correction. The new average takes effect at the next
// acquisition (ensureDownconverterLocked), not mid-lock, so a stable lock is
// never disturbed. Caller holds d.mu.
func (d *Decoder) sampleAutotuneLocked() {
	if d.autotune == nil || !d.locked.Load() {
		return
	}
	rep, ok := d.active.(afcReporter)
	if !ok {
		return
	}
	now := time.Now()
	if !d.lastATSampleAt.IsZero() && now.Sub(d.lastATSampleAt) < time.Second {
		return
	}
	d.lastATSampleAt = now

	observed := int(math.Round(rep.AFCOffsetHz()))
	applied := int(d.autotuneApplied.Load())
	d.autotune.AddErrorMeasurement(observed, applied, d.activeFreqHz)
	d.log.Info("ccdecoder: "+d.autotune.StatusString(d.activeFreqHz, 0),
		"system", d.activeAt, "freq_hz", d.activeFreqHz,
		"observed_hz", observed, "applied_hz", applied)
}

// checkCarrierOffsetLocked emits a throttled WARN when, while the CC is locked,
// the TOTAL carrier offset from the configured control frequency exceeds
// d.carrierOffsetWarnHz. Total = autotuneApplied + the receiver's residual
// AFCOffsetHz: with autotune off (the default) applied is 0 and the total is the
// raw residual; with autotune on, a large genuine offset is folded into applied,
// so summing the two still catches an adjacent-channel lock autotune would
// otherwise hide by chasing the wrong carrier (issue #815). Advisory only — it
// never retunes; it tells the operator the locked carrier sits far from the
// configured frequency, so the reported site identity may belong to a stronger
// neighbouring site bleeding through (or the front-end oscillator is mistuned).
// Caller holds d.mu.
func (d *Decoder) checkCarrierOffsetLocked() {
	if !d.locked.Load() {
		return
	}
	rep, ok := d.active.(afcReporter)
	if !ok {
		return
	}
	total := int(d.autotuneApplied.Load()) + int(math.Round(rep.AFCOffsetHz()))
	if total < 0 {
		total = -total
	}
	if total < d.carrierOffsetWarnHz {
		d.lastOffsetWarnAt = time.Time{} // re-arm so a fresh excursion warns at once
		return
	}
	now := time.Now()
	if !d.lastOffsetWarnAt.IsZero() && now.Sub(d.lastOffsetWarnAt) < carrierOffsetWarnInterval {
		return
	}
	d.lastOffsetWarnAt = now
	d.log.Warn("ccdecoder: control carrier offset far from configured frequency — "+
		"GT may be locked onto an adjacent site's stronger carrier (wrong site identity) "+
		"or the receiver oscillator is badly mistuned; verify the reported site against "+
		"the configured frequency (issue #815)",
		"offset_hz", total, "freq_hz", d.activeFreqHz, "system", d.activeAt)
}

// observeIQPower folds one raw IQ chunk into the per-second power / DC /
// clip window and, once a window elapses, publishes the gauges and the
// IQHealth snapshot and fires the throttled diagnostic logs. Runs only on
// the forwarder goroutine, so the pw* accumulators need no lock; the
// cross-goroutine IQHealth snapshot it derives (lastHealth*, totalSamples)
// is written under healthMu. The gauge label comes from activeSystem (an
// atomic) so this never takes d.mu.
func (d *Decoder) observeIQPower(iq []complex64) {
	if len(iq) == 0 || d.sampleRateHz <= 0 {
		return
	}
	system := *d.activeSystem.Load()
	// Reset window state if the system the gauge is labelled with
	// just changed — different system means different gauge label;
	// don't fold old samples into the new one's average.
	if system != d.pwLastSystem {
		d.pwSumSq = 0
		d.pwSumI = 0
		d.pwSumQ = 0
		d.pwClipped = 0
		d.pwSamples = 0
		d.pwLastSystem = system
		d.healthMu.Lock()
		d.totalSamples = 0
		d.healthMu.Unlock()
	}
	// Cumulative sample count for the active system (reset above on a
	// system change). Distinguishes "SDR delivered nothing" from "IQ
	// flowed but never locked" in the cchunt.failed diagnosis.
	d.healthMu.Lock()
	d.totalSamples += int64(len(iq))
	d.healthMu.Unlock()
	for _, c := range iq {
		r := float64(real(c))
		i := float64(imag(c))
		d.pwSumSq += r*r + i*i
		d.pwSumI += r
		d.pwSumQ += i
		// Count samples pinned to the ADC rail (issue #402): either
		// component at full scale means the 8-bit RTL ADC clipped, which
		// the RMS power gauge below averages away.
		if r >= iqClipThreshold || r <= -iqClipThreshold || i >= iqClipThreshold || i <= -iqClipThreshold {
			d.pwClipped++
		}
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
	n := float64(d.pwSamples)
	mean := d.pwSumSq / n
	// 10*log10(0) is -Inf; clamp with a small epsilon so the gauge
	// reads as "very low" instead of breaking JSON encoders.
	if mean < 1e-12 {
		mean = 1e-12
	}
	dbfs := 10 * math.Log10(mean)
	// DC-bin power: |mean(IQ)|² across the window. The DFT bin at 0 Hz
	// of a finite window is the sample mean; the per-sample power
	// contributed by a static carrier at DC is its squared magnitude.
	// Comparing it to the total mean power (in dB) tells the operator
	// what fraction of the channel passband is DC content — a smoking
	// gun for the R820T2 DC-spike-on-channel failure mode behind issue
	// #402. Healthy off-channel signals: ≤ -20 dB. A
	// DC-dominated capture sits within ~5 dB of 0.
	dcMeanI := d.pwSumI / n
	dcMeanQ := d.pwSumQ / n
	dcPower := dcMeanI*dcMeanI + dcMeanQ*dcMeanQ
	if dcPower < 1e-12 {
		dcPower = 1e-12
	}
	dcDbfs := 10 * math.Log10(dcPower)
	ratioDb := dcDbfs - dbfs
	// Fraction of samples that hit the ADC rail this window. RMS dbfs can
	// read merely "hot" (e.g. -5 dBFS) while a high-crest signal still
	// clips on peaks; this is the signal that distinguishes the two and
	// the smoking gun for the issue #402 overload failure mode.
	clipRatio := float64(d.pwClipped) / n
	// Persist this window under healthMu so IQHealth (cchunt supervisor
	// goroutine) can report it at hunt-failure time; the pw* accumulators
	// reset below every window.
	d.healthMu.Lock()
	d.lastHealthSystem = system
	d.lastHealthAt = now
	d.lastDbfs = dbfs
	d.lastClipRatio = clipRatio
	d.lastDCRatioDb = ratioDb
	d.healthMu.Unlock()
	if system != "" && d.metrics != nil {
		d.metrics.RecordIQPowerDbFS(system, dbfs)
		d.metrics.RecordIQDCRatioDb(system, ratioDb)
		d.metrics.RecordIQClipRatio(system, clipRatio)
	}
	if dbfs < iqLowPowerThresholdDbFS && now.Sub(d.pwLowLogAt) >= 5*time.Second {
		d.log.Debug("ccdecoder: iq power very low — check antenna, gain, USB",
			"system", system, "dbfs", dbfs, "dc_dbfs", dcDbfs, "dc_ratio_db", ratioDb)
		d.pwLowLogAt = now
	} else if clipRatio > iqClipWarnRatio && now.Sub(d.pwClipLogAt) >= 30*time.Second {
		// Front end is pinning the ADC rail — overload/clipping. This
		// shreds the C4FM eye and fails TSBK CRC continuously even while
		// the channel stays locked (issue #402). The fix is less RF, not
		// more: reduce gain or add attenuation/filtering. Warned (not
		// debug) because it silently defeats decode and the startup
		// low-gain hint points the wrong way for an overloaded front end.
		d.log.Warn("ccdecoder: iq clipping — front end overloaded; reduce gain or add attenuation (do NOT raise gain). issue #402",
			"system", system, "clip_ratio", clipRatio, "dbfs", dbfs)
		d.pwClipLogAt = now
	} else if ratioDb > iqDCRatioWarnDb && now.Sub(d.pwDCLogAt) >= 30*time.Second {
		// DC bin within iqDCRatioWarnDb of total — the channel of
		// interest sits on top of the tuner zero. Logged at debug so
		// the gauge is the primary signal; the throttled log line gives
		// operators without Prometheus a hint of the failure mode (issue
		// #402).
		d.log.Debug("ccdecoder: iq DC bin dominant — RTL-SDR DC spike on channel? see issue #402",
			"system", system, "dbfs", dbfs, "dc_dbfs", dcDbfs, "dc_ratio_db", ratioDb)
		d.pwDCLogAt = now
	}
	d.pwSumSq = 0
	d.pwSumI = 0
	d.pwSumQ = 0
	d.pwClipped = 0
	d.pwSamples = 0
	d.pwWindowAt = now
}

// IQHealth returns a best-effort snapshot of the control SDR's IQ
// health for the named system, for the cchunt supervisor to attach to
// a cchunt.failed event/log. ok is false when the decoder has no
// observations for that system — most often because the SDR delivered
// no IQ at all, or a different system is currently being decoded — and
// the supervisor then publishes its own "no IQ observed" diagnosis.
//
// Satisfies the cchunt.IQHealthProvider interface (structurally; the
// supervisor decouples via that interface so it doesn't import this
// package).
func (d *Decoder) IQHealth(system string) (trunking.HuntDiagnostics, bool) {
	// Pipeline state is guarded by mu (decode goroutine); the IQ-health
	// snapshot is guarded by healthMu (forwarder goroutine). Take mu then
	// healthMu — the same order clearActiveLocked uses — so the two never
	// deadlock. The forwarder only ever takes healthMu, never mu.
	d.mu.Lock()
	pipelineActive := d.active != nil && d.activeAt == system
	activeMatches := d.activeAt == system
	d.mu.Unlock()

	d.healthMu.Lock()
	haveWindow := d.lastHealthSystem == system && !d.lastHealthAt.IsZero()
	samples := d.totalSamples
	dbfs, clip, dc := d.lastDbfs, d.lastClipRatio, d.lastDCRatioDb
	d.healthMu.Unlock()

	haveSamples := activeMatches && samples > 0
	if !haveWindow && !haveSamples {
		return trunking.HuntDiagnostics{}, false
	}
	diag := trunking.HuntDiagnostics{
		IQObserved:     true,
		IQSamples:      samples,
		PipelineActive: pipelineActive,
	}
	if haveWindow {
		diag.IQPowerDbFS = dbfs
		diag.IQClipRatio = clip
		diag.IQDCRatioDb = dc
	}
	diag.Diagnosis = diagnoseIQ(diag, haveWindow)
	return diag, true
}

// diagnoseIQ turns an IQ-health snapshot into a one-line operator hint,
// reusing the same thresholds the live pump logs fire on so the two
// never disagree. haveWindow is false when only a partial (sub-window)
// sample count is available, so we can't yet judge signal level.
func diagnoseIQ(d trunking.HuntDiagnostics, haveWindow bool) string {
	if !d.PipelineActive {
		return "no live decode pipeline for this protocol — check the system's `protocol` setting"
	}
	if !haveWindow {
		return "IQ is flowing but the dwell was too short to gauge signal level — let it run, or raise scanner.cc_hunt.dwell_ms"
	}
	switch {
	case d.IQPowerDbFS <= iqLowPowerThresholdDbFS:
		return "IQ level very low — check the antenna is connected, raise gain, and confirm the dongle is streaming (`sdr list --probe`)"
	case d.IQClipRatio > iqClipWarnRatio:
		return "front-end overload (ADC clipping) — reduce gain or add attenuation; do NOT raise gain (issue #402)"
	case d.IQDCRatioDb > iqDCRatioWarnDb:
		return "an on-channel DC spike dominates the passband (RTL-SDR zero-IF) — offset the tuned frequency slightly or use a different dongle (issue #402)"
	default:
		return "signal present but no control-channel lock — verify the control-channel frequency is correct and current, set the right ppm, and confirm the system is active"
	}
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
