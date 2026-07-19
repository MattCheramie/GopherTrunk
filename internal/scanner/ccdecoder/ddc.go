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

// DDCTargetForProtocol is the exported wrapper over ddcTargetForProtocol.
// The offline siglab toolkit (and the replay subcommand) call it to size
// their own Downconverter to the SAME per-protocol channel rate the daemon
// uses — critical for TETRA, whose 18000-baud π/4-DQPSK needs the 144 kHz
// target rather than the 48 kHz C4FM-family default. Sizing replay's DDC
// to the C4FM target for a TETRA capture would leave under 3 samples per
// symbol and the Gardner loop would never lock.
func DDCTargetForProtocol(p trunking.Protocol) float64 {
	return ddcTargetForProtocol(p)
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
// Rate conversion is a rational polyphase resample (dsp.Resampler) whose
// L/M ratio is chosen so the output rate lands exactly on the
// requested target for every standard SDR rate. When the SDR already
// streams at the target the resampler is skipped and the chunk passes
// straight through — keeping the rate==target unit tests on a no-op path.
// A wider capture is decimated to the target; a narrowband capture recorded
// BELOW the target (e.g. a 50 kHz or 48 kHz TETRA slice, whose 144 kHz
// channel target the live path always delivers) is interpolated UP to it, so
// the receiver always runs at its designed samples-per-symbol rather than a
// rounded, decode-breaking one.
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

// NewDownconverter builds a down-converter that resamples inRateHz to
// ~targetHz (decimating a wider capture, interpolating a sub-target one).
// The exact achieved output rate is reported by OutRateHz (it equals
// targetHz for every SDR rate that reduces to a sane L/M, and equals
// inRateHz in the rate==target pass-through mode).
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
	if in <= 0 || target <= 0 || in == target {
		return d // pass-through: the NCO mix (if any) still runs, no resample
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
// equals inRateHz only when the capture is already at the target).
// Callers building a receiver
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

// ddcRatio reduces target/in to its lowest L/M terms so the resampler
// lands the output rate exactly on target. A non-standard SDR rate can
// reduce to a pathologically large ratio (L>64 branches or M>8192);
// rather than a crude integer decimator (L=1, M=round(in/target)) —
// which silently shifts the achieved rate, and therefore the recovered
// symbol rate, by up to a few percent — it falls back to the closest
// L/M under the caps. For a 3.019 MS/s capture the old fallback landed
// the 144 kHz TETRA channel at 143762 Hz (17970 baud, −0.165% — the
// "baud drift" signature); the bounded search lands it at 143998 Hz
// (17999.8 baud, −0.001%).
//
// This is the issue #550 fix already proven in internal/dsp/tuner
// (bestRatioUnderCaps); porting it here closes the "two separate DDC
// paths" divergence CLAUDE.md #764/#771 warns about. Standard SDR rates
// (2.4/2.5/10 MS/s, common USRP rates) reduce cleanly and never reach
// the fallback, so they are unaffected.
func ddcRatio(target, in int) (l, m int) {
	g := gcd(target, in)
	l, m = target/g, in/g
	if l > 64 || m > 8192 {
		l, m = bestRatioUnderCaps(float64(target) / float64(in))
	}
	return l, m
}

// bestRatioUnderCaps returns the L/M (L in 1..64, M in 1..8192) whose ratio is
// closest to r. O(64) and reached only on the cap-exceeded fallback, so it has
// no per-sample cost. Mirrors internal/dsp/tuner.bestRatioUnderCaps (issue
// #550) so both down-converter paths reduce a pathological rate the same way
// instead of the old integer decimator that shifted the achieved rate.
func bestRatioUnderCaps(r float64) (l, m int) {
	bestL, bestM := 1, 1
	bestErr := math.Inf(1)
	for li := 1; li <= 64; li++ {
		mi := int(math.Round(float64(li) / r))
		if mi < 1 {
			mi = 1
		} else if mi > 8192 {
			mi = 8192
		}
		if e := math.Abs(float64(li)/float64(mi) - r); e < bestErr {
			bestErr, bestL, bestM = e, li, mi
		}
	}
	return bestL, bestM
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
