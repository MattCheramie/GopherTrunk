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
// Implementation: IQ samples → decimate to ~48 kHz + channel filter →
// quadrature FM discriminator (toneFrontEnd) → single-pole IIR low-pass
// at ~500 Hz to roll off the audio band → Goertzel detector at exactly
// the target tone frequency, compared against reverse bins at the
// adjacent EIA tones → magnitude threshold. The whole chain processes
// one IQ chunk at a time and runs only when a channel has tone
// gating configured, so the cost for un-gated channels is zero.
//
// DCS (Digital-Coded Squelch — also called DPL) is the digital
// cousin of CTCSS: a 23-bit Golay-coded codeword transmitted as a
// 134.4 baud sub-audible NRZ stream. Its detector is in dcs.go.

// CTCSSDetector matches a single CTCSS tone against a stream of IQ
// chunks. Construct via NewCTCSSDetector; feed IQ via Process. The
// detector keeps phase + Goertzel state across calls so block
// boundaries don't matter to the caller.
//
// Not safe for concurrent use — the conv scanner owns one detector
// per channel and processes each chunk serially.
type CTCSSDetector struct {
	// fe decimates the SDR-rate IQ to ~48 kHz, channel-filters it and
	// FM-discriminates (issue #1184); its output is radians per sample
	// at toneRefRateHz whatever the input rate.
	fe *toneFrontEnd

	// Single-pole IIR low-pass on the discriminator output. Cutoff
	// is set by NewCTCSSDetector to ~500 Hz so the audio band
	// rolls off before the Goertzel samples it. State is the
	// running output value.
	lpfAlpha float64
	lpfState float64

	// Goertzel detector at exactly the target tone frequency (no bin
	// rounding — see toneout.NewGoertzelExact).
	goertzel *toneout.Goertzel

	// reverseBins are Goertzel detectors at off-target frequencies: the
	// target ±ctcssReverseOffsetHz plus the adjacent tones of the EIA
	// table. A match requires target_mag > rejectRatio * max(reverse),
	// so a transmission on the next tone up or down the table — which
	// lands dead-on its own reverse bin — can never open the gate.
	reverseBins []*toneout.Goertzel

	// rejectRatio is how much the target bin must dominate the
	// largest reverse bin before declaring a match. Tunable per
	// detector.
	rejectRatio float64

	// Magnitude threshold above which the tone is considered present,
	// derived from ctcssMinDeviationHz. Tunable per detector via
	// SetMagnitudeThreshold.
	magThreshold float64

	// Detection state — present is true while the current matched
	// block keeps reporting magnitude above threshold. Sticky
	// across blocks so callers can poll between feeds.
	present bool

	// targetHz is preserved for inspection / tests.
	targetHz float64
}

// CTCSSConfig holds the sample rate of the input IQ stream + the
// CTCSS frequency to look for.
type CTCSSConfig struct {
	// SampleHz is the IQ sample rate (typically 2.4e6 for RTL-SDR).
	SampleHz float64
	// TargetHz is the CTCSS frequency to detect. Standard values
	// range from 67.0 to 254.1 Hz (EIACTCSSTones).
	TargetHz float64
	// AudioCutoffHz sets the single-pole IIR low-pass cutoff. The
	// LPF rolls off the audio band so it doesn't leak into the
	// sub-audible bins. Defaults to 500 Hz when zero — comfortably
	// above the highest CTCSS frequency and below the lowest voice
	// formant.
	AudioCutoffHz float64
	// BlockSize is the Goertzel block size in INPUT IQ samples.
	// Larger blocks → finer frequency resolution at the cost of
	// slower detection. Defaults to ctcssBlockSeconds of input.
	BlockSize int
}

// ctcssRefRateHz is the rate the detector's magnitude threshold is
// calibrated at (issue #1184); see toneRefRateHz.
const ctcssRefRateHz = toneRefRateHz

// ctcssBlockSeconds is the default Goertzel block: 250 ms, a 4 Hz
// resolution. Adjacent EIA tones sit 2.3–3.0 Hz apart; at the former
// 200 ms (5 Hz) block an on-tone signal leaked ~57% of its power into a
// neighbour 2.3 Hz away, leaving the rejection ratio almost no margin.
// At 4 Hz the leak is ~29%, and a detection still lands well inside the
// ~300 ms decode time radios quote.
const ctcssBlockSeconds = 0.25

// ctcssMinDeviationHz is the weakest tone the gate accepts, as peak FM
// deviation. Radios put ~15% of their peak deviation into CTCSS: ~350 Hz
// on a 2.5 kHz narrowband channel, ~750 Hz on a 5 kHz wideband one. The
// old fixed threshold (5e-4) needed ~540 Hz, so no narrowband radio ever
// opened the gate (issue #1184). 100 Hz keeps a >10 dB margin under an
// NFM tone; a carrier WITHOUT a tone still stays shut, because it puts
// nothing coherent in the target bin and the reverse-bin ratio holds.
const ctcssMinDeviationHz = 100

// ctcssReverseOffsetHz places the two fixed reverse bins either side of
// the target, catching non-EIA tones and broadband sub-audible energy.
const ctcssReverseOffsetHz = 5.0

// ctcssMinNeighbourSpacingHz: EIA neighbours closer than this are not
// used as reverse bins. Only 150.0/151.4 Hz (1.4 Hz apart) fall inside
// it; a 250 ms block cannot separate them (each leaks ~66% into the
// other), so a gate on either opens on both, as on most radios.
const ctcssMinNeighbourSpacingHz = 2.0

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
	fe := newToneFrontEnd(cfg.SampleHz)
	rate := fe.rate
	block := int(math.Round(rate * ctcssBlockSeconds))
	if cfg.BlockSize > 0 {
		block = cfg.BlockSize / fe.m // BlockSize is in INPUT samples
	}
	if block < 1 {
		block = 1
	}

	var reverseHz []float64
	for _, off := range []float64{-ctcssReverseOffsetHz, ctcssReverseOffsetHz} {
		reverseHz = append(reverseHz, cfg.TargetHz+off)
	}
	reverseHz = append(reverseHz, eiaNeighbourTones(cfg.TargetHz)...)
	reverseBins := make([]*toneout.Goertzel, 0, len(reverseHz))
	for _, hz := range reverseHz {
		if hz <= 0 {
			continue
		}
		reverseBins = append(reverseBins, toneout.NewGoertzelExact(hz, rate, block))
	}

	return &CTCSSDetector{
		fe:           fe,
		lpfAlpha:     onePoleAlpha(cfg.AudioCutoffHz, rate),
		goertzel:     toneout.NewGoertzelExact(cfg.TargetHz, rate, block),
		reverseBins:  reverseBins,
		rejectRatio:  1.5,
		magThreshold: ctcssToneMagnitude(ctcssMinDeviationHz, cfg.TargetHz, cfg.AudioCutoffHz),
		targetHz:     cfg.TargetHz,
	}
}

// ctcssToneMagnitude is the Goertzel magnitude a steady tone at toneHz
// with devHz of peak FM deviation produces at the detector's output: the
// discriminator reads 2π·dev/rate radians per sample at toneRefRateHz,
// the int16 scaling divides by π, and the single-pole low-pass
// attenuates it by 1/sqrt(1+(f/fc)²). The Goertzel reports squared
// amplitude.
func ctcssToneMagnitude(devHz, toneHz, cutoffHz float64) float64 {
	a := 2 * devHz / toneRefRateHz
	g := 1 / (1 + (toneHz/cutoffHz)*(toneHz/cutoffHz))
	return a * a * g
}

// eiaNeighbourTones returns the EIA tones immediately below and above
// targetHz (the tone itself excluded), skipping any closer than
// ctcssMinNeighbourSpacingHz.
func eiaNeighbourTones(targetHz float64) []float64 {
	var below, above float64
	for _, hz := range EIACTCSSTones {
		d := hz - targetHz
		if math.Abs(d) < ctcssMinNeighbourSpacingHz {
			continue
		}
		if d < 0 {
			below = hz
		} else if above == 0 {
			above = hz
		}
	}
	var out []float64
	if below > 0 && targetHz-below < 10 {
		out = append(out, below)
	}
	if above > 0 && above-targetHz < 10 {
		out = append(out, above)
	}
	return out
}

// SetRejectRatio tunes the reverse-bin rejection ratio: target bin
// magnitude must exceed rejectRatio × max(reverse_bins) to count as
// a match. Setting ratio ≤ 1 disables the check effectively (any
// target-bin magnitude above the leak floor will pass). Default
// is 1.5.
func (d *CTCSSDetector) SetRejectRatio(r float64) {
	if r < 0 {
		r = 0
	}
	d.rejectRatio = r
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
	d.fe.reset()
	d.lpfState = 0
	d.goertzel.Reset()
	for _, rb := range d.reverseBins {
		rb.Reset()
	}
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
	for _, demod := range d.fe.process(iq) {
		// Single-pole low-pass to reject the audio band that would
		// leak into the sub-audible Goertzel bins.
		d.lpfState = d.lpfState + d.lpfAlpha*(demod-d.lpfState)

		// Goertzel wants int16-scaled samples. Scale the [-π, π]
		// discriminator output into the int16 range; the Goertzel
		// normalises by sample-count so the absolute scale only
		// affects the magThreshold which is calibrated for this
		// scaling (ctcssToneMagnitude).
		const scale = 32768.0 / math.Pi
		sample := int16(d.lpfState * scale)

		// Feed every Goertzel — they all share the same block
		// size, so the ready signals fire on the same sample.
		targetMag, ready := d.goertzel.Process(sample)
		var maxReverseMag float64
		for _, rb := range d.reverseBins {
			if rmag, _ := rb.Process(sample); rmag > maxReverseMag {
				maxReverseMag = rmag
			}
		}
		if !ready {
			continue
		}
		// Match when target exceeds the magnitude floor AND
		// dominates the reverse bins by rejectRatio. The second
		// check guards against adjacent-code spectral leak.
		if targetMag < d.magThreshold {
			d.present = false
			continue
		}
		if d.rejectRatio > 0 && targetMag < d.rejectRatio*maxReverseMag {
			d.present = false
			continue
		}
		d.present = true
	}
	return d.present
}

// EIACTCSSTones is the standard CTCSS tone table: the 38 original EIA
// tones plus the 12 later additions (150.0, 159.8, 165.5, 171.3, 177.3,
// 183.5, 189.9, 196.6, 199.5, 206.5, 229.1, 254.1) that current
// commercial radios program. Sorted ascending.
var EIACTCSSTones = []float64{
	67.0, 69.3, 71.9, 74.4, 77.0, 79.7, 82.5, 85.4, 88.5, 91.5,
	94.8, 97.4, 100.0, 103.5, 107.2, 110.9, 114.8, 118.8, 123.0, 127.3,
	131.8, 136.5, 141.3, 146.2, 150.0, 151.4, 156.7, 159.8, 162.2, 165.5,
	167.9, 171.3, 173.8, 177.3, 179.9, 183.5, 186.2, 189.9, 192.8, 196.6,
	199.5, 203.5, 206.5, 210.7, 218.1, 225.7, 229.1, 233.6, 241.8, 250.3,
	254.1,
}
