package receiver

import (
	"math"
	"math/rand"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

// carrierGate is a per-sample carrier-presence detector on the FM
// discriminator output — an FM noise-quieting squelch expressed as a
// statistic, with no absolute-power threshold anywhere.
//
// # Why the DMR receiver needs one
//
// A base-station DMR carrier (Tier II repeater, Tier III site) is continuous:
// both timeslots are always transmitted, so every earlier synthetic fixture
// modelled a back-to-back burst stream and the receiver's level and timing
// trackers were only ever exercised on one. A DIRECT-MODE transmission (Tier
// I / simplex, an MS talking to another MS on one frequency — the PMR446 /
// 446.500 "DMR simplex" case of issue #836) is not: the MS transmits one
// 27.5 ms burst per 60 ms TDMA frame and is off for the other 32.5 ms. In
// that gap the receiver sees only noise, and the FM discriminator of noise is
// uniformly distributed over ±π rad/sample — several times LARGER than the
// ±0.25 rad/sample the signal's own ±1944 Hz deviation produces at 48 kHz.
// Measured on the reporter's capture through the production DDC: the
// discriminator VARIANCE is 0.01–0.08 rad² inside every burst and 2.3–3.9 rad²
// in every gap and on band-limited noise alone.
//
// Left ungated, that gap noise poisons every slow tracker in the chain: the
// symbol AGC's 53 ms level EMA is inflated by each 32 ms gap so the next
// burst's outer symbols slice as inner ones (no sync word ever matches — sync
// words are all ±3); the post-clock AFC decays to zero and re-converges from
// scratch every burst; the coarse pre-clock acquirer's window is ~54 % noise so
// its offset estimate is halved and never clears its own deadband; and the
// Mueller-Müller error term random-walks the symbol phase between bursts.
// Reproduced failing-first: the production Tier I pipeline decoded ZERO sync
// words from a synthetic direct-mode stream at 30 dB SNR, while decoding the
// same bursts perfectly when laid back-to-back (receiver_burst_test.go).
//
// # What it measures
//
// The running mean-removed variance of the discriminator output over a short
// window (a few symbols). The mean removal makes it independent of the
// carrier offset — an uncorrected tuner ppm error is a constant discriminator
// bias, not variance — so a mistuned burst is still "present" and the coarse
// acquirer can measure it. Scale invariance is inherent: the discriminator
// output is a phase increment, blind to the IQ amplitude, so no gain staging
// can move the threshold (the repo's coherence-over-dBFS rule). The
// threshold sits between the two measured populations with ~5× margin to the
// noise side and ~12× to the signal side: a burst with a 10 dB in-channel SNR
// adds ~0.1 rad² of phase jitter and stays far inside it.
//
// # What it gates
//
// Two things. The trackers: the AGC level, the AFC bias, the acquirer's
// offset mean and the timing-loop error update all hold on absent samples.
// And the discriminator output itself is MUTED (zeroed) on absent samples —
// the noise-squelch every FM receiver applies — because the RRC matched
// filter's memory otherwise carries the ±π gap noise into the first span of
// every burst and corrupts its leading payload dibits (measured: half the
// direct-mode header bursts failed BPTC with the trackers gated but the
// discriminator unmuted). Every sample is still filtered, clocked and sliced,
// so the dibit stream stays contiguous and the framers downstream see the
// same timeline, delayed by the gate's detection lag (below).
//
// # Alignment
//
// The statistic is causal and needs ~one window to notice a burst edge, so
// the discriminator stream is delayed by that window while the decision is
// not: the decision made at sample n gates the sample from lag samples
// earlier, which puts the OPEN transition within a few samples of the true
// burst start. The close side uses a higher hysteresis threshold so it lands
// a little AFTER the true burst end (a couple of symbols of gap noise ride
// through — harmless, they only smear into more gap noise) rather than
// truncating the burst's last symbols. The delay shifts the dibit stream by
// lag/sps symbols; nothing downstream depends on absolute sample alignment.
// On a continuous carrier every sample is present, so the gated receiver's
// dibit stream is that of the ungated one shifted by exactly that lag.
type carrierGate struct {
	rate      float64 // EMA coefficient (1/window samples)
	openBelow float64 // variance (rad²) below which an absent carrier is declared present
	closeAt   float64 // variance (rad²) at or above which a present carrier is declared absent
	mean      float64 // running mean of the discriminator output
	sq        float64 // running mean of its square
	seeded    bool
	open      bool // current decision

	// hold keeps a closed gate reporting "present" for this many samples
	// after the variance crosses closeAt (calibrated gate only; 0 on the
	// wideband path). See newCarrierGateCalibrated: the delay line there
	// is sized to the OPEN latency, which is longer than the close latency,
	// so without the hold the last (open − close) samples of every burst
	// would be muted.
	hold      int
	closedFor int // samples since the gate last closed (saturates at hold)

	ring    []float32 // discriminator delay line (lag samples)
	ringPos int
}

// carrierGateSymbols is the gate's EMA window in symbol periods. Four symbols
// (40 samples at 10 sps) averages the per-sample noise of the variance
// estimate to ~0.1 rad² — well under the threshold — while lagging a burst
// edge by less than 1 ms: the first ~4 symbols of a burst are still flagged
// absent (the trackers hold the previous burst's values, which is the right
// answer) and the last ~4 symbols of gap noise are flagged present (a
// negligible 3 % of the AGC window).
const carrierGateSymbols = 4.0

// carrierGateOpenBelow / carrierGateCloseAt are the discriminator-variance
// (rad²) hysteresis thresholds. See the type doc for the measured populations
// they separate: ≤ 0.08 (signal, even ADC-clipped) vs ≥ 2.3 (gap /
// band-limited noise through the production DDC). The open threshold is
// reached ~1.1 windows after a burst starts (the EMA of the square decaying
// from ~3 toward ~0.05), which the delay line cancels; the higher close
// threshold is reached ~1.8 windows after it ends, so the gate closes ~0.7
// windows (~3 symbols) late rather than early.
const (
	carrierGateOpenBelow = 1.0
	carrierGateCloseAt   = 2.5
)

func newCarrierGate(sps float64) *carrierGate {
	return newCarrierGateCalibrated(sps, carrierGateNoiseVarianceWideband)
}

// carrierGateNoiseVarianceWideband is the discriminator variance of
// receiver noise that fills the whole passband — the population the fixed
// thresholds above were measured against (2.3–3.9 rad² on the reporter's
// capture through the production DDC; π²/3 ≈ 3.29 for a uniform ±π phase
// increment).
const carrierGateNoiseVarianceWideband = math.Pi * math.Pi / 3

// newCarrierGateCalibrated builds the gate for a stream whose noise-only
// discriminator variance is noiseVar rad². The thresholds are a FRACTION of
// the noise population, not absolute numbers: the channel-select filter
// (Options.EnableChannelFilter) narrows the receiver noise from the DDC's
// ±~22 kHz to ±6.25 kHz, and the discriminator of narrowband noise swings
// far less than ±π per sample — measured 0.49 rad² at 48 kHz against 3.29
// unfiltered — so the fixed 1.0 / 2.5 rad² thresholds read a filtered gap as
// a perfectly quiet carrier and the gate never closed (every tracker then
// drank the gap noise, exactly the #836 failure the gate exists to stop).
// The thresholds are set as fractions of noiseVar (carrierGateOpenFrac /
// carrierGateCloseFrac) rather than by scaling the wideband constants
// (which would be 0.30 / 0.76 of π²/3): filtered noise is CORRELATED over
// ~Fs/2fc ≈ 4 samples, so the gate's 4-symbol EMA of the variance averages
// ~10 independent samples instead of 40 and swings much wider around its
// mean. Measured at 48 kHz on the filtered fixture: gap EMA variance median
// 0.465, 1st percentile 0.25, minimum 0.14 over 2 s — a 0.30 fraction (0.15)
// is BELOW that minimum, so the gate flickered open on gap noise, seeded the
// timing loop on it, and the next burst train's headers were lost (the
// direct-mode pipeline fixture decoded one PTT of two). The filtered signal
// population sits at ≤ 0.01 rad² (27 dB) and 0.009 at 7 dB SNR — the filter
// removes most of the in-band noise too — so 0.15 of the noise variance
// (0.07 at 48 kHz) still clears the signal by ~7× while sitting 2× under the
// gap minimum.
func newCarrierGateCalibrated(sps, noiseVar float64) *carrierGate {
	lag := int(carrierGateSymbols*sps + 0.5)
	if noiseVar == carrierGateNoiseVarianceWideband {
		// The wideband path keeps its measured absolute constants exactly.
		return &carrierGate{
			rate:      1.0 / (carrierGateSymbols * sps),
			openBelow: carrierGateOpenBelow,
			closeAt:   carrierGateCloseAt,
			ring:      make([]float32, lag),
		}
	}
	// The delay line must match the decision latency of THESE thresholds,
	// not the wideband ones. At a burst start the EMA of the square decays
	// from the gap variance toward ~0 and crosses the open threshold after
	// ≈ −ln(openFrac) windows in theory (1.9) — measured ~2.3 windows on the
	// filtered direct-mode fixture, because the EMA sits above its median
	// when the burst lands and the channel filter's ring-in keeps the first
	// ~30 samples noisy. A one-window delay line therefore flagged the first
	// 4–8 dibits of every burst absent and muted them: enough clustered
	// errors at the burst start for BPTC to mis-correct the Voice LC Header
	// (the fixture's first PTT decoded nothing but RS-parity mismatches).
	// So the delay line is sized to the open latency, and the close side —
	// which the EMA reaches only ln(1/(1−closeFrac)) ≈ 1.4 windows after a
	// burst ends — gets a hold of (open − close) samples so the burst's last
	// samples are not muted either; the held samples map exactly onto the
	// burst tail, never into the gap.
	openWindows := carrierGateCalibratedOpenWindows
	closeWindows := math.Log(1 / (1 - carrierGateCloseFrac))
	lag = int(openWindows*carrierGateSymbols*sps + 0.5)
	hold := int((openWindows-closeWindows)*carrierGateSymbols*sps + 0.5)
	if hold < 0 {
		hold = 0
	}
	return &carrierGate{
		rate:      1.0 / (carrierGateSymbols * sps),
		openBelow: carrierGateOpenFrac * noiseVar,
		closeAt:   carrierGateCloseFrac * noiseVar,
		hold:      hold,
		closedFor: hold,
		ring:      make([]float32, lag),
	}
}

// carrierGateCalibratedOpenWindows is the measured open latency of the
// calibrated gate in EMA windows (see newCarrierGateCalibrated).
const carrierGateCalibratedOpenWindows = 2.3

// carrierGateOpenFrac / carrierGateCloseFrac are the calibrated gate's
// hysteresis thresholds as fractions of the noise-only discriminator
// variance (see newCarrierGateCalibrated for the measured populations).
const (
	carrierGateOpenFrac  = 0.15
	carrierGateCloseFrac = 0.75
)

// filteredNoiseDiscVariance measures the discriminator variance of white
// receiver noise after the channel-select FIR with the given taps: the
// calibration newCarrierGateCalibrated needs. Measured rather than derived —
// the instantaneous frequency of narrowband Gaussian noise is heavy-tailed
// and the per-sample wrap to ±π bounds it, so a closed form for the wrapped
// variance would be one more constant to get wrong. A fixed-seed Gaussian
// stream (deterministic, ~8k samples, a few hundred µs) through a fresh copy
// of the filter and a fresh discriminator, with the filter warm-up skipped.
func filteredNoiseDiscVariance(taps []float32) float64 {
	const n = 8192
	rng := rand.New(rand.NewSource(0x836))
	noise := make([]complex64, n+len(taps))
	for i := range noise {
		noise[i] = complex(float32(rng.NormFloat64()), float32(rng.NormFloat64()))
	}
	filtered := filter.NewFIR(taps).Process(nil, noise)
	disc := demod.NewFM().Process(nil, filtered)
	disc = disc[len(taps):]
	var mean, sq float64
	for _, x := range disc {
		mean += float64(x)
		sq += float64(x) * float64(x)
	}
	mean /= float64(len(disc))
	sq /= float64(len(disc))
	return sq - mean*mean
}

// Lag is the delay, in samples, the gate imposes on the discriminator stream.
func (g *carrierGate) Lag() int { return len(g.ring) }

// Process folds disc (the FM-discriminator output for one chunk, rad/sample)
// into the running statistics, then rewrites disc IN PLACE as the delayed,
// muted stream and returns one presence flag per (delayed) sample in dst. iq,
// when non-nil, is the discriminator's input for the same samples: a
// digitally dead sample (exactly zero — a squelched or zero-padded recording,
// or an SDR that stopped delivering) carries no carrier and closes the gate
// outright; its discriminator output is a constant 0 that the variance
// statistic alone would read as a perfectly quiet carrier.
func (g *carrierGate) Process(dst []bool, disc []float32, iq []complex64) []bool {
	if cap(dst) < len(disc) {
		dst = make([]bool, len(disc))
	} else {
		dst = dst[:len(disc)]
	}
	for i, x := range disc {
		v := float64(x)
		switch {
		case iq != nil && iq[i] == 0:
			g.open = false
		case !g.seeded:
			// Seed the mean on the first sample; the variance starts at zero,
			// so the stream opens present until the window fills. On a stream
			// that starts in noise that is one window of pollution — the same
			// as an ungated receiver has always absorbed.
			g.mean = v
			g.sq = v * v
			g.seeded = true
			g.open = true
		default:
			g.mean += g.rate * (v - g.mean)
			g.sq += g.rate * (v*v - g.sq)
			variance := g.sq - g.mean*g.mean
			if g.open {
				g.open = variance < g.closeAt
			} else {
				g.open = variance < g.openBelow
			}
		}
		// Delay line: emit the sample from lag samples ago under the decision
		// just made, muted when the carrier is absent.
		delayed := g.ring[g.ringPos]
		g.ring[g.ringPos] = x
		g.ringPos++
		if g.ringPos == len(g.ring) {
			g.ringPos = 0
		}
		present := g.open
		if g.open {
			g.closedFor = 0
		} else if g.closedFor < g.hold {
			g.closedFor++
			present = true
		}
		dst[i] = present
		if present {
			disc[i] = delayed
		} else {
			disc[i] = 0
		}
	}
	return dst
}

// Reset clears the statistics and the delay line so a re-synced stream
// re-seeds.
func (g *carrierGate) Reset() {
	g.mean = 0
	g.sq = 0
	g.seeded = false
	g.open = false
	g.closedFor = g.hold
	for i := range g.ring {
		g.ring[i] = 0
	}
	g.ringPos = 0
}
