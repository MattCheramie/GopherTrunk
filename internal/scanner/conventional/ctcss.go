package conventional

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/voice/toneout"
)

// CTCSS (Continuous Tone-Coded Squelch System) is the sub-audible
// (67.0 – 254.1 Hz) tone many analog FM repeaters mix into their
// transmissions so receivers only open audio when the right tone is
// present. Adding a per-channel tone gate to the conventional scanner
// is what turns "carrier-active" squelch into "right-system" squelch
// — without it the scanner stops on every nearby transmission on the
// same frequency, including marine, business, and adjacent-county
// traffic.
//
// Implementation: IQ samples → quadrature FM discriminator
// (inline atan2 over z[n]·conj(z[n-1])) → single-pole IIR low-pass
// at ~500 Hz to roll off the audio band that would otherwise alias
// into the sub-audible region → Goertzel detector at the target
// tone frequency → magnitude threshold. The whole chain processes
// one IQ chunk at a time and runs only when a channel has tone
// gating configured, so the cost for un-gated channels is zero.
//
// DCS (Digital-Coded Squelch — also called DPL) is the digital
// cousin of CTCSS: a 23-bit Golay-coded codeword transmitted as a
// 134.4 baud sub-audible NRZ stream. Decoding it requires a
// proper bit-level demodulator (clock recovery + Golay decoder)
// that is materially more work than the Goertzel pattern here;
// the conventional scanner's tone config validates the DCS mode
// so operators can configure it without churning the config
// surface later, but the detector itself is a tracked follow-up.

// CTCSSDetector matches a single CTCSS tone against a stream of IQ
// chunks. Construct via NewCTCSSDetector; feed IQ via Process. The
// detector keeps phase + Goertzel state across calls so block
// boundaries don't matter to the caller.
//
// Not safe for concurrent use — the conv scanner owns one detector
// per channel and processes each chunk serially.
type CTCSSDetector struct {
	// FM discriminator state (last IQ sample for the conjugate
	// multiply).
	last complex64

	// Single-pole IIR low-pass on the discriminator output. Cutoff
	// is set by NewCTCSSDetector to ~500 Hz so the audio band
	// rolls off before the Goertzel samples it. State is the
	// running output value.
	lpfAlpha float64
	lpfState float64

	// Goertzel detector at the target tone frequency. Reuses the
	// already-existing toneout primitive so the math is shared
	// with the paging-tone detector.
	goertzel *toneout.Goertzel

	// Magnitude threshold above which the tone is considered
	// present. Tunable per detector via SetMagnitudeThreshold; the
	// constructor picks a conservative default that works against
	// FM-demod normalised amplitudes for unit-amplitude tones.
	magThreshold float64

	// Detection state — present is true while the current matched
	// block keeps reporting magnitude above threshold. Sticky
	// across blocks so callers can poll between feeds.
	present bool

	// targetHz is preserved for inspection / tests.
	targetHz float64
}

// CTCSSConfig holds the sample rate of the input IQ stream + the
// CTCSS frequency to look for. Goertzel block size is derived from
// the sample rate so the frequency resolution is around 5 Hz at any
// reasonable SDR rate.
type CTCSSConfig struct {
	// SampleHz is the IQ sample rate (typically 2.4e6 for RTL-SDR).
	SampleHz float64
	// TargetHz is the CTCSS frequency to detect. Standard values
	// range from 67.0 to 254.1 Hz; the EIA list has 50 codes but
	// only 38 + 12 are widely used.
	TargetHz float64
	// AudioCutoffHz sets the single-pole IIR low-pass cutoff. The
	// LPF rolls off the audio band so it doesn't alias into the
	// sub-audible band when the Goertzel samples at its block
	// rate. Defaults to 500 Hz when zero — comfortably above the
	// highest CTCSS frequency and below the lowest voice formant.
	AudioCutoffHz float64
	// BlockSize is the Goertzel block size in IQ samples. Larger
	// blocks → finer frequency resolution at the cost of slower
	// detection. Defaults to SampleHz / 5 (≈ 5 Hz bin resolution
	// and ~200 ms detection latency, comfortably under typical
	// CTCSS reaction times on commercial radios).
	BlockSize int
}

// NewCTCSSDetector constructs a detector. TargetHz must be > 0 and
// inside the practical CTCSS range (50..300 Hz); SampleHz must be
// the IQ rate the detector will be fed.
func NewCTCSSDetector(cfg CTCSSConfig) *CTCSSDetector {
	if cfg.SampleHz <= 0 || cfg.TargetHz <= 0 {
		return nil
	}
	if cfg.AudioCutoffHz <= 0 {
		cfg.AudioCutoffHz = 500
	}
	if cfg.BlockSize <= 0 {
		cfg.BlockSize = int(cfg.SampleHz / 5)
	}
	// Single-pole IIR low-pass: alpha = dt / (RC + dt), where
	// RC = 1 / (2π·fc). Pre-warps the cutoff into the IIR's
	// per-sample step.
	dt := 1.0 / cfg.SampleHz
	rc := 1.0 / (2 * math.Pi * cfg.AudioCutoffHz)
	alpha := dt / (rc + dt)
	return &CTCSSDetector{
		last:         complex(1, 0),
		lpfAlpha:     alpha,
		goertzel:     toneout.NewGoertzel(cfg.TargetHz, cfg.SampleHz, cfg.BlockSize),
		// 5e-4 catches typical CTCSS injection (~500-1000 Hz peak
		// deviation on commercial repeaters) with ~3x headroom
		// over the noise floor measured on RTL-SDR captures.
		// Tunable per channel via SetMagnitudeThreshold.
		magThreshold: 5e-4,
		targetHz:     cfg.TargetHz,
	}
}

// SetMagnitudeThreshold tunes the detection threshold. Higher values
// reject low-level / spurious tones at the cost of slower lock onto
// a weak repeater. Defaults work for typical RTL-SDR captures.
func (d *CTCSSDetector) SetMagnitudeThreshold(t float64) {
	d.magThreshold = t
}

// TargetHz returns the configured CTCSS frequency. Useful for logs.
func (d *CTCSSDetector) TargetHz() float64 { return d.targetHz }

// Present reports the latest detection state. Stable between
// Process calls; flips inside Process when a Goertzel block
// completes.
func (d *CTCSSDetector) Present() bool { return d.present }

// Reset clears all internal state. Called by the scanner whenever
// it retunes so a tone match on a previous channel doesn't bleed
// into the new dwell.
func (d *CTCSSDetector) Reset() {
	d.last = complex(1, 0)
	d.lpfState = 0
	d.goertzel.Reset()
	d.present = false
}

// Process feeds an IQ chunk through the detector chain. Updates the
// internal Present() state when the Goertzel block boundary lands
// inside the chunk. Returns the most recent Present() value as a
// convenience for callers that gate on a single call.
func (d *CTCSSDetector) Process(iq []complex64) bool {
	if d == nil || len(iq) == 0 {
		return d != nil && d.present
	}
	for _, s := range iq {
		// FM discriminator: arg(z[n] · conj(z[n-1])).
		ar := real(s)*real(d.last) + imag(s)*imag(d.last)
		ai := imag(s)*real(d.last) - real(s)*imag(d.last)
		demod := math.Atan2(float64(ai), float64(ar))
		d.last = s

		// Single-pole low-pass to reject the audio band that would
		// alias into the sub-audible Goertzel bin.
		d.lpfState = d.lpfState + d.lpfAlpha*(demod-d.lpfState)

		// Goertzel wants int16-scaled samples. Scale the [-π, π]
		// discriminator output into the int16 range; the Goertzel
		// normalises by sample-count so the absolute scale only
		// affects the magThreshold which is calibrated for this
		// scaling.
		const scale = 32768.0 / math.Pi
		sample := int16(d.lpfState * scale)
		if mag, ready := d.goertzel.Process(sample); ready {
			d.present = mag >= d.magThreshold
		}
	}
	return d.present
}
