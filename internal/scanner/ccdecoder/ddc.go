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
//
// Exposed as DDCTargetRateHz so the gophertrunk replay subcommand
// (which constructs its own Downconverter to mirror the production
// pipeline; issue #402 Phase 2) can target the same channelized rate
// without duplicating the constant.
const ddcTargetRateHz = 48000.0

// DDCTargetRateHz is the exported alias of ddcTargetRateHz. The
// in-package callers keep using the lowercase name for source
// stability; new external callers (replay, integration tests) use
// the exported value.
const DDCTargetRateHz = ddcTargetRateHz

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

// Downconverter decimates a wideband SDR IQ stream to a narrowband
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
//
// Exported (issue #402 Phase 2) so the gophertrunk replay subcommand
// can construct an identical down-converter and exactly mirror the
// production receiver chain instead of feeding the receiver raw
// wideband IQ — the latter sizes the matched filter and AFC/AGC
// time constants for 2.4 MHz instead of 48 kHz, so a replayed
// capture decodes nothing like its live counterpart.
type Downconverter struct {
	nco       *dsp.NCO       // nil ⇒ no tuning shift (channel already at DC)
	mixBuf    []complex64    // scratch for the NCO mix, so raw is never mutated
	resampler *dsp.Resampler // nil ⇒ pass-through (no decimation)
	outRateHz float64
}

// NewDownconverter builds a down-converter that decimates inRateHz to
// ~targetHz. The exact achieved output rate is reported by
// OutRateHz (it equals targetHz for every SDR rate that reduces to
// a sane L/M, and equals inRateHz in pass-through mode).
func NewDownconverter(inRateHz, targetHz float64) *Downconverter {
	return NewDownconverterWithOffset(inRateHz, targetHz, 0)
}

// NewDownconverterWithOffset is NewDownconverter plus a tuning offset: the
// stream is first frequency-shifted so a channel sitting at +offsetHz lands
// at 0 Hz, then decimated to ~targetHz. This is the "deliberate tuning
// offset" the package doc describes — needed to replay a wideband capture
// whose control channel is not centred (the live pipeline gets this for
// free from the SDR tuner). A zero offset is identical to NewDownconverter
// (no NCO is built, so existing centred-capture behaviour is byte-exact).
func NewDownconverterWithOffset(inRateHz, targetHz, offsetHz float64) *Downconverter {
	d := &Downconverter{outRateHz: inRateHz}
	if offsetHz != 0 && inRateHz > 0 {
		d.nco = dsp.NewNCO(offsetHz, inRateHz)
	}
	in := int(math.Round(inRateHz))
	target := int(math.Round(targetHz))
	if in <= 0 || target <= 0 || in <= target {
		return d // pass-through decimation: the NCO mix (if any) still runs
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

// OutRateHz returns the achieved narrowband output rate (equals the
// requested target for standard SDR rates that reduce to a sane L/M;
// equals inRateHz in pass-through mode). Callers building a receiver
// against the downconverter's output should use this value for
// SampleRateHz so matched-filter sizing matches the actual stream.
func (d *Downconverter) OutRateHz() float64 { return d.outRateHz }

// Process decimates one raw IQ chunk to the narrowband rate. dst is
// reused if it has capacity; the returned slice holds the narrowband
// output (len ≈ len(raw)·outRateHz/inRateHz). In pass-through mode
// raw is returned unchanged. raw is never mutated.
func (d *Downconverter) Process(dst, raw []complex64) []complex64 {
	src := raw
	if d.nco != nil {
		// Tune the channel to DC first. Mix into our own scratch buffer
		// so the caller's raw slice is never mutated (the doc contract).
		d.mixBuf = d.nco.Mix(d.mixBuf, raw)
		src = d.mixBuf
	}
	if d.resampler == nil {
		return src
	}
	return d.resampler.Process(dst, src)
}

// Reset clears the decimation filter history. Called on every
// pipeline swap so a retune doesn't bleed the previous channel's
// filter state into the new one.
func (d *Downconverter) Reset() {
	if d.nco != nil {
		d.nco.Reset()
	}
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
