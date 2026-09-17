package receiver

import "math"

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

	ring    []float32 // discriminator delay line (lag samples)
	ringPos int

	// holdOpen is the number of leading samples (after construction or a
	// reset) the gate passes as present WITHOUT folding them into its
	// statistics: the receiver's channel filter starts from an empty
	// history, and the partial sums of its warm-up are a switch-on
	// transient whose instantaneous frequency is not the carrier's — read
	// as variance, it closed the gate for ~150 samples at the head of every
	// stream, muting the loops' first samples and starting the gated path
	// from a different state than the ungated one. Held samples still run
	// through the delay line, so the gate stays a pure lag over the head.
	holdOpen int
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
	return newCarrierGateForNoise(sps, fullBandNoiseDiscVariance)
}

// fullBandNoiseDiscVariance is the discriminator variance of white noise
// occupying the whole sample band: successive phase differences are uniform
// on ±π, so the variance is π²/3 ≈ 3.29 rad². The hysteresis constants above
// were measured against exactly that population (2.3–3.9 rad² in the
// direct-mode capture's gaps), so they are expressed as fractions of it.
var fullBandNoiseDiscVariance = math.Pi * math.Pi / 3

// newCarrierGateForNoise builds a gate whose hysteresis is scaled to the
// discriminator variance the receiver's front end produces on NOISE ALONE
// (noiseVariance, rad²). The gate's whole decision rests on the gap between
// a burst's variance (the modulation's own, ~0.04 rad², independent of the
// noise bandwidth) and the noise floor's — and the floor's is set by the
// bandwidth ahead of the discriminator: the full ±Fs/2 band gives π²/3, a
// channel filter a fraction of that (band-limited noise swings the
// instantaneous frequency only across its own band). The absolute rad²
// constants therefore hold only for the unfiltered front end they were
// measured on; behind a 12.5 kHz channel filter idle-gap noise fell below
// the OPEN threshold, the gate declared a carrier present on noise and the
// coarse acquirer engaged on that noise's mean (measured: −1 kHz, losing the
// next transmission). Scaling keeps the same fractions of the floor. The
// full-band value reproduces the historical constants exactly.
func newCarrierGateForNoise(sps, noiseVariance float64) *carrierGate {
	lag := int(carrierGateSymbols*sps + 0.5)
	scale := noiseVariance / fullBandNoiseDiscVariance
	return &carrierGate{
		rate:      1.0 / (carrierGateSymbols * sps),
		openBelow: carrierGateOpenBelow * scale,
		closeAt:   carrierGateCloseAt * scale,
		ring:      make([]float32, lag),
	}
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
		case g.holdOpen > 0:
			g.holdOpen--
			g.open = true
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
		dst[i] = g.open
		if g.open {
			disc[i] = delayed
		} else {
			disc[i] = 0
		}
	}
	return dst
}

// Reset clears the statistics and the delay line so a re-synced stream
// re-seeds.
// HoldOpen makes the next n samples pass as present without touching the
// gate's statistics (see holdOpen).
func (g *carrierGate) HoldOpen(n int) { g.holdOpen = n }

func (g *carrierGate) Reset() {
	g.mean = 0
	g.sq = 0
	g.seeded = false
	g.open = false
	g.holdOpen = 0
	for i := range g.ring {
		g.ring[i] = 0
	}
	g.ringPos = 0
}
