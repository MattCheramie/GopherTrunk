package ccdecoder

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// ddcTargetRateHz is the narrowband channel rate the down-converter
// decimates to for the 4800-baud C4FM family (P25 / DMR / NXDN /
// dPMR / YSF / D-STAR) and the other ≤9600-baud protocols.
//
// Those receivers size their matched filter + symbol-clock loop
// from PipelineOptions.SampleRateHz, expecting a channelized rate of
// roughly 48 kHz (≈10 samples per symbol at 4800 baud). Feeding the
// raw SDR rate (commonly 2.048 MHz) instead gives ≈427 samples per
// symbol and a matched filter spanning a ±1 MHz swath, so the Frame
// Sync Word never correlates and no protocol locks — see issue #275.
const ddcTargetRateHz = 48000.0

// tetraDDCTargetRateHz is the channel rate the down-converter uses
// for TETRA. TETRA's π/4-DQPSK runs at 18000 symbols/s — roughly
// four times the 4800-baud C4FM family — so a 48 kHz channel would
// leave under 3 samples per symbol. 144 kHz gives a comfortable
// 8 samples per symbol for the Gardner timing-recovery loop.
const tetraDDCTargetRateHz = 144000.0

// ddcTargetForProtocol picks the narrowband channel rate the down-
// converter decimates to for a protocol. The 4800-baud C4FM family
// (P25 / DMR / NXDN / dPMR / YSF / D-STAR) and the other ≤9600-baud
// protocols all channelize to ddcTargetRateHz; TETRA's 18000-baud
// π/4-DQPSK needs a wider channel, so it gets tetraDDCTargetRateHz.
func ddcTargetForProtocol(p trunking.Protocol) float64 {
	if p == trunking.ProtocolTETRA {
		return tetraDDCTargetRateHz
	}
	return ddcTargetRateHz
}

// ddcStopbandTaps sets the anti-alias prototype length as a multiple
// of the decimation factor M (total taps ≈ ddcStopbandTaps·M). The
// polyphase resampler runs its multiply-accumulates at the output
// rate, so a long prototype costs little; this length yields a
// >60 dB stopband for the M values typical SDR rates produce.
const ddcStopbandTaps = 12

// ddcKaiserBeta shapes the anti-alias prototype's Kaiser window —
// ~70 dB peak sidelobe attenuation.
const ddcKaiserBeta = 7.0

// downconverter decimates a wideband SDR IQ stream to a narrowband
// channel rate.
//
// Decimation is a rational polyphase resample (dsp.Resampler) whose
// L/M ratio is chosen so the output rate lands exactly on the
// requested target for every standard SDR rate. When the SDR already
// streams at (or below) the target the resampler is skipped and the
// chunk passes straight through — keeping the rate==target unit
// tests (and any future low-rate SDR) on a no-op path.
//
// It deliberately does NOT remove the front-end DC offset: a C4FM /
// FM control channel carries real signal energy at 0 Hz (the FM
// carrier component), so an IQ-domain DC blocker distorts the very
// signal being decoded — measured here as a >60% RMS error on a
// round-tripped C4FM stream. DC-spike handling, when a site needs
// it, belongs in the frequency domain (a deliberate tuning offset so
// the channel no longer sits at 0 Hz) or after the FM discriminator
// (coarse AFC on the real symbol stream); both are tracked as
// follow-ups to issue #275.
type downconverter struct {
	resampler *dsp.Resampler // nil ⇒ pass-through (no decimation)
	outRateHz float64
}

// newDownconverter builds a down-converter that decimates inRateHz to
// ~targetHz. The exact achieved output rate is reported by outRateHz
// (it equals targetHz for every SDR rate that reduces to a sane L/M,
// and equals inRateHz in pass-through mode).
func newDownconverter(inRateHz, targetHz float64) *downconverter {
	d := &downconverter{outRateHz: inRateHz}
	in := int(math.Round(inRateHz))
	target := int(math.Round(targetHz))
	if in <= 0 || target <= 0 || in <= target {
		return d // pass-through: nothing to decimate
	}
	l, m := ddcRatio(target, in)
	tapsPerBranch := (ddcStopbandTaps*m + l - 1) / l
	if tapsPerBranch < 8 {
		tapsPerBranch = 8
	}
	d.resampler = dsp.NewResampler(l, m, tapsPerBranch, ddcKaiserBeta)
	d.outRateHz = inRateHz * float64(l) / float64(m)
	return d
}

// Process decimates one raw IQ chunk to the narrowband rate. dst is
// reused if it has capacity; the returned slice holds the narrowband
// output (len ≈ len(raw)·outRateHz/inRateHz). In pass-through mode
// raw is returned unchanged. raw is never mutated.
func (d *downconverter) Process(dst, raw []complex64) []complex64 {
	if d.resampler == nil {
		return raw
	}
	return d.resampler.Process(dst, raw)
}

// Reset clears the decimation filter history. Called on every
// pipeline swap so a retune doesn't bleed the previous channel's
// filter state into the new one.
func (d *downconverter) Reset() {
	if d.resampler != nil {
		d.resampler.Reset()
	}
}

// ddcRatio reduces target/in to its lowest L/M terms. A non-standard
// SDR rate can reduce to a pathologically large ratio; since the
// output rate only needs to be *roughly* 48 kHz, those fall back to
// a pure integer decimator (L=1).
func ddcRatio(target, in int) (l, m int) {
	g := gcd(target, in)
	l, m = target/g, in/g
	if l > 64 || m > 8192 {
		l = 1
		m = int(math.Round(float64(in) / float64(target)))
		if m < 1 {
			m = 1
		}
	}
	return l, m
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		a = -a
	}
	return a
}
