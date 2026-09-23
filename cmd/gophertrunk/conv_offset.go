package main

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// Conventional-scanner LO offset tuning (issue #1184).
//
// The conventional scanner used to tune the SDR's LO exactly onto the analog FM
// channel (zero-IF). Three front-end artefacts then land INSIDE the channel, at
// frequencies set by the residual carrier offset δ between the transmitter and
// the LO — which on a real rig is a few hundred Hz, i.e. in the audio band:
//
//   - the DC spur (LO leakage / ADC offset) beats with the carrier at δ;
//   - the I/Q-imbalance image of the carrier sits at −δ and beats at 2δ;
//   - and — the one that dominated the reporter's captures — per-axis ADC
//     clipping. An overloaded 8-bit ADC clips I and Q independently, squaring
//     the unit-circle constellation. The phase error that introduces is periodic
//     in the carrier phase with period π/2, so the discriminator emits a tone at
//     4·|δ| plus its harmonics. Measured on the reporter's 447.100 MHz captures:
//     every Kenwood file is pinned to the rail (|IQ| ∈ [1.0, 1.414] — a square),
//     and the demodulated audio carries a tone 18–25 dB above the voice floor at
//     exactly 4× each file's carrier offset (NX-5000: δ≈−420 Hz → 1676 Hz;
//     NX-300: δ≈−850 Hz → 3381 Hz). The unclipped Radtel file at −12 dBFS shows
//     no such line, which is why it "sounded better".
//
// The operator's bench test pinned the same thing from the other side: an
// unmodulated −60 dBm carrier exactly on-channel was noisy, the same carrier
// ±3.25 kHz off was clean — moving the carrier away from the LO moves every one
// of those products out of the audio band. SDR#/SDR++ (which tune the LO off the
// VFO) never see it.
//
// The fix mirrors that: tune the LO convScannerOffsetHz BELOW the channel and mix
// the stream back to baseband with an NCO before either the scanner (squelch +
// CTCSS/DCS detectors) or the FM voice chain sees it. The DC spur then sits at
// −offset, the image at −2·offset, and the clipping products at ±4k·offset
// (aliased mod the sample rate) — all far outside the channel filter, provided
// the offset is chosen so that none of them alias back onto the channel. Note
// that the obvious sample_rate/4 (the dc_avoid default) is the WORST choice for
// a clipped signal: 4·(fs/4) ≡ 0 (mod fs), so the 3rd-order clipping product
// lands exactly on the channel. pickClipSafeLOOffsetHz searches for an offset
// that keeps every low-order product clear.

const (
	// convOffsetMinHz is the smallest offset considered: it must put the DC
	// spur well into the stopband of the FM chain's front-end decimating FIR,
	// whose transition band at a ~2.4 MS/s input is ~100 kHz wide.
	convOffsetMinHz = 150_000
	// convOffsetMaxFrac bounds the offset to a fraction of the sample rate so
	// the channel stays clear of the tuner's own band-edge roll-off.
	convOffsetMaxFrac = 0.35
	// convOffsetStepHz is the search granularity.
	convOffsetStepHz = 1_000
	// convOffsetMaxOrder is the highest clipping harmonic index k checked
	// (products at 4k·offset, i.e. up to the (4k+1)th order).
	convOffsetMaxOrder = 6
	// convOffsetDevHz is the FM deviation used to size how far each product's
	// modulation spreads: the (4k±1)th-order product carries (4k±1)× the
	// deviation. 5 kHz covers wideband (25 kHz) analog FM.
	convOffsetDevHz = 5_000
	// convOffsetChanHalfHz is the half-width of the channel the products must
	// stay out of.
	convOffsetChanHalfHz = 12_500
)

// convScannerLOOffsetHz resolves the conventional scanner's LO offset from
// scanner.lo_offset_hz: < 0 disables offset tuning (legacy on-channel tuning),
// 0 selects a clip-safe offset automatically, > 0 pins it. It returns 0 when
// the sample rate leaves no room for an offset (the scanner then tunes
// on-channel, as before) or when an explicit value is out of range.
func convScannerLOOffsetHz(rateHz uint32, configured int) (offsetHz float64, reason string) {
	switch {
	case configured < 0:
		return 0, "disabled (scanner.lo_offset_hz < 0)"
	case configured > 0:
		if float64(configured) > convOffsetMaxFrac*float64(rateHz) {
			return 0, "scanner.lo_offset_hz exceeds 35% of the sample rate"
		}
		return float64(configured), "configured"
	}
	off := pickClipSafeLOOffsetHz(float64(rateHz))
	if off == 0 {
		return 0, "no room for an LO offset at this sample rate"
	}
	return off, "auto"
}

// pickClipSafeLOOffsetHz returns the LO offset (Hz) in
// [convOffsetMinHz, convOffsetMaxFrac·fs] whose front-end products stay
// furthest outside the channel, or 0 when that range is empty. Ties go to the
// smaller offset, so the result is deterministic for a given rate.
func pickClipSafeLOOffsetHz(fs float64) float64 {
	maxOff := convOffsetMaxFrac * fs
	if fs <= 0 || maxOff < convOffsetMinHz {
		return 0
	}
	best, bestScore := 0.0, math.Inf(-1)
	for off := float64(convOffsetMinHz); off <= maxOff; off += convOffsetStepHz {
		if s := loOffsetClearanceHz(off, fs); s > bestScore {
			best, bestScore = off, s
		}
	}
	return best
}

// loOffsetClearanceHz scores an LO offset: the minimum, over every front-end
// product, of how far (Hz) the product's modulated extent stays outside the
// channel once the stream is mixed back to baseband. Negative means at least
// one product overlaps the channel. Products, relative to the channel after
// mixing (all wrapped mod fs):
//   - DC spur at −off and I/Q image at −2·off (unmodulated / 1× deviation);
//   - per-axis clipping at ±4k·off, the (4k∓1)th-order term carrying
//     (4k+1)× the deviation (the wider of the pair, conservatively).
func loOffsetClearanceHz(off, fs float64) float64 {
	clear := func(f, spread float64) float64 {
		return math.Abs(wrapHz(f, fs)) - convOffsetChanHalfHz - spread
	}
	score := clear(off, 0)
	score = math.Min(score, clear(2*off, convOffsetDevHz))
	for k := 1; k <= convOffsetMaxOrder; k++ {
		spread := float64(4*k+1) * convOffsetDevHz
		score = math.Min(score, clear(float64(4*k)*off, spread))
	}
	return score
}

// wrapHz folds f into (−fs/2, fs/2].
func wrapHz(f, fs float64) float64 {
	f = math.Mod(f, fs)
	if f > fs/2 {
		f -= fs
	} else if f <= -fs/2 {
		f += fs
	}
	return f
}

// convTunerIQ is what the conventional scanner drives: the iqtap broker (or a
// bare device) — tune + stream.
type convTunerIQ interface {
	SetCenterFreq(hz uint32) error
	StreamIQ(ctx context.Context) (<-chan []complex64, error)
}

// convScannerFrontEnd wraps the scanner SDR's broker: SetCenterFreq places the
// LO offsetHz below the requested channel, and StreamIQ mixes the channel back
// to DC so the scanner's squelch + tone detectors (and, via mixToChannel, the FM
// voice chain) see an on-channel stream with every front-end product parked
// far outside it. It also watches the RAW (pre-mix) samples for ADC-rail
// pinning — the root condition behind the clipping products — and WARNs,
// rate-limited, so the operator is told to lower the gain; the mix hides the
// artefacts but a clipped front end still loses weak signals. offsetHz == 0
// keeps the legacy on-channel tuning with only the overload watch.
type convScannerFrontEnd struct {
	inner    convTunerIQ
	offsetHz float64
	rateHz   uint32
	serial   string
	log      *slog.Logger

	mu        sync.Mutex
	centerHz  uint32 // requested channel (not the LO)
	winN      int64
	winClip   int64
	winStart  time.Time
	lastWarn  time.Time
	now       func() time.Time
	warnEvery time.Duration
}

const (
	// convClipWarnFrac is the rail-pinned fraction (~1 s window) above which
	// the front end is reported overloaded — the ccdecoder / widebandt2
	// threshold (issues #402, #749).
	convClipWarnFrac = 0.002
	convClipWindow   = time.Second
	convClipWarnGap  = 5 * time.Minute
)

func newConvScannerFrontEnd(inner convTunerIQ, offsetHz float64, rateHz uint32, serial string, log *slog.Logger) *convScannerFrontEnd {
	if log == nil {
		log = slog.Default()
	}
	return &convScannerFrontEnd{
		inner:     inner,
		offsetHz:  offsetHz,
		rateHz:    rateHz,
		serial:    serial,
		log:       log,
		now:       time.Now,
		warnEvery: convClipWarnGap,
	}
}

// SetCenterFreq tunes the LO offsetHz below hz (the channel). Voice channels
// are hundreds of MHz and the offset is sub-MHz, so the uint32 subtraction is
// safe.
func (f *convScannerFrontEnd) SetCenterFreq(hz uint32) error {
	f.mu.Lock()
	f.centerHz = hz
	f.mu.Unlock()
	return f.inner.SetCenterFreq(hz - uint32(f.offsetHz))
}

// StreamIQ opens the inner stream and returns it mixed back to the channel.
func (f *convScannerFrontEnd) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	in, err := f.inner.StreamIQ(ctx)
	if err != nil {
		return nil, err
	}
	return mixToChannel(ctx, in, f.offsetHz, f.rateHz, f.observeRaw), nil
}

// observeRaw folds a raw (pre-mix) chunk into the overload window and WARNs
// when a window's rail-pinned fraction exceeds convClipWarnFrac.
func (f *convScannerFrontEnd) observeRaw(chunk []complex64) {
	clipped := siglab.CountClipped(chunk)
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	if f.winStart.IsZero() {
		f.winStart = now
	}
	f.winN += int64(len(chunk))
	f.winClip += clipped
	if now.Sub(f.winStart) < convClipWindow {
		return
	}
	frac := 0.0
	if f.winN > 0 {
		frac = float64(f.winClip) / float64(f.winN)
	}
	f.winN, f.winClip, f.winStart = 0, 0, now
	if frac <= convClipWarnFrac {
		return
	}
	if !f.lastWarn.IsZero() && now.Sub(f.lastWarn) < f.warnEvery {
		return
	}
	f.lastWarn = now
	f.log.Warn("conv: front end overloaded — IQ pinned to the ADC rail; a strong nearby transmitter is clipping the SDR, which distorts analog FM audio (a whistle/tone at 4x the carrier offset when tuned on-channel). Reduce gain or add attenuation (do NOT raise gain). issue #1184",
		"serial", f.serial, "channel_hz", f.centerHz, "clipped_fraction", frac,
		"lo_offset_hz", int(f.offsetHz))
}

// mixToChannel returns in mixed down by offsetHz (the channel sits at +offsetHz
// because the LO is offsetHz low). observe, when non-nil, sees each raw chunk
// first. offsetHz == 0 still copies nothing extra: chunks pass through as-is.
func mixToChannel(ctx context.Context, in <-chan []complex64, offsetHz float64, rateHz uint32, observe func([]complex64)) <-chan []complex64 {
	if offsetHz == 0 && observe == nil {
		return in
	}
	out := make(chan []complex64, 8)
	var nco *dsp.NCO
	if offsetHz != 0 {
		nco = dsp.NewNCO(offsetHz, float64(rateHz)) // shifts +offsetHz → DC
	}
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-in:
				if !ok {
					return
				}
				if observe != nil {
					observe(chunk)
				}
				if nco != nil {
					// Fresh buffer: the broker's fan-out may share the
					// chunk with other subscribers.
					chunk = nco.Mix(make([]complex64, len(chunk)), chunk)
				}
				select {
				case out <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
