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

// ddcUpMaxNumer / ddcUpMaxDenom bound the L/M ratio the up-sampling
// branch of ddcRatio will accept as an *exact* reduced fraction. A
// pre-channelized capture below the target rate (e.g. an SDR++ baseband
// WAV at 39062.5 Hz, or the 48 kHz reference recording) rarely reduces
// to small terms against a 144 kHz / 48 kHz target, so beyond these
// bounds ddcRatio snaps to the nearest achievable rate within
// ddcUpTolerance instead — the receiver sizes its matched filter and
// clock loop from OutRateHz, so a sub-percent rate error is harmless.
const (
	ddcUpMaxNumer  = 4096
	ddcUpMaxDenom  = 64
	ddcUpTolerance = 0.01
)

// ddcUpMinTaps is the minimum anti-imaging prototype length for the
// up-sampling path. The decimation sizing (∝ M) collapses to a handful
// of taps when M is small — as it is when interpolating — and a short
// prototype leaves the L-1 spectral images the zero-stuffing creates
// poorly rejected, which measurably distorts the demodulated π/4-DQPSK
// constellation (whole dibit classes drop out). A fixed ~192-tap floor
// restores a >60 dB image stopband across the L values sub-target
// captures produce; the MACs run at the output rate, so the cost is a
// flat ~ddcUpMinTaps/L per output sample.
const ddcUpMinTaps = 192

// Downconverter resamples an SDR IQ stream to a narrowband channel
// rate — decimating a wideband live stream, or interpolating a
// pre-channelized sub-target capture.
//
// Resampling is a rational polyphase resample (dsp.Resampler) whose
// L/M ratio is chosen so the output rate lands on (or very near) the
// requested target for every standard SDR rate. A live wideband SDR
// decimates DOWN; a pre-channelized capture below the target (an SDR++
// baseband WAV, the 48 kHz reference recording) is resampled UP so the
// receiver still gets its designed samples-per-symbol. Only when the
// stream already sits exactly at the target is the resampler skipped
// and the chunk forwarded unchanged (the rate==target no-op path).
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
	resampler *dsp.Resampler // nil ⇒ pass-through (rate already == target)
	outRateHz float64
}

// NewDownconverter builds a down-converter that resamples inRateHz to
// ~targetHz (decimating a wideband SDR, or interpolating a
// pre-channelized sub-target capture up to the receiver's rate). The
// exact achieved output rate is reported by OutRateHz (it equals
// targetHz for every rate that reduces to a sane L/M, and equals
// inRateHz on the rate==target pass-through).
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
		return d // pass-through: the NCO mix (if any) still runs
	}
	l, m := ddcRatio(target, in)
	tapsPerBranch := (ddcStopbandTaps*m + l - 1) / l
	if tapsPerBranch < 8 {
		tapsPerBranch = 8
	}
	if in < target {
		// Interpolation: size the prototype for image rejection, not by
		// the decimation ∝M formula (which goes tiny when up-sampling).
		if up := (ddcUpMinTaps + l - 1) / l; up > tapsPerBranch {
			tapsPerBranch = up
		}
	}
	d.resampler = dsp.NewResampler(l, m, tapsPerBranch, ddcKaiserBeta)
	d.outRateHz = inRateHz * float64(l) / float64(m)
	return d
}

// OutRateHz returns the achieved narrowband output rate (equals the
// requested target for rates that reduce to a sane L/M; equals inRateHz
// on the rate==target pass-through). Callers building a receiver
// against the downconverter's output should use this value for
// SampleRateHz so matched-filter sizing matches the actual stream.
func (d *Downconverter) OutRateHz() float64 { return d.outRateHz }

// Process resamples one raw IQ chunk to the narrowband rate. dst is
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

// ddcRatio reduces the resample rate to L/M terms for the polyphase
// resampler. It handles both directions:
//
//   - Decimation (in > target): reduce target/in to lowest terms. A
//     non-standard SDR rate can reduce to a pathologically large ratio;
//     since the output rate only needs to be *roughly* the target,
//     those fall back to a pure integer decimator (L=1).
//   - Interpolation (in < target): a pre-channelized capture below the
//     target (an SDR++ baseband WAV, the 48 kHz reference recording) is
//     resampled UP so the receiver gets its designed ~8 samples/symbol
//     instead of the fractional sps that broke decode before (a
//     39062.5 Hz WAV ran the TETRA receiver at +8.5 % baud). Use the
//     exact reduced ratio when its terms are cheap; otherwise snap to
//     the smallest-denominator rational whose achieved rate lands
//     within ddcUpTolerance of target.
//
// in == target never reaches here — the caller pass-through handles it.
func ddcRatio(target, in int) (l, m int) {
	if in > target {
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

	// Interpolation. Exact reduced ratio first.
	g := gcd(target, in)
	l, m = target/g, in/g
	if l <= ddcUpMaxNumer && m <= ddcUpMaxDenom {
		return l, m
	}
	// Search for the smallest denominator whose achieved output rate is
	// within tolerance; smallest M within tolerance wins (cheapest
	// resampler). Fall back to the closest ratio found if none clears
	// the tolerance.
	bestL, bestM, bestErr := l, m, math.Inf(1)
	for mm := 1; mm <= ddcUpMaxDenom; mm++ {
		ll := int(math.Round(float64(target) * float64(mm) / float64(in)))
		if ll <= mm {
			continue // must interpolate (L > M)
		}
		achieved := float64(in) * float64(ll) / float64(mm)
		err := math.Abs(achieved-float64(target)) / float64(target)
		if err < bestErr {
			bestErr, bestL, bestM = err, ll, mm
		}
		if err <= ddcUpTolerance {
			return ll, mm
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
