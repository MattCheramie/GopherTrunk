package receiver

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
)

// Feed-forward symbol-timing acquisition at a transmission's onset (issue
// #836).
//
// # Why the gate alone was not enough
//
// The carrier gate holds the Mueller-Müller timing loop across the 32.5 ms
// gaps of a direct-mode transmission, so all the bursts of ONE keyup share the
// symbol phase the loop converged to. But every keyup starts at a symbol phase
// of its own — a new transmission's burst grid has no relation to the phase
// held from the previous one, or to the loop's constructed phase on a cold
// start — and the loop then has to pull that phase error in by feedback at
// gain·error per symbol. From near a half-symbol offset (the loop's unstable
// equilibrium) that takes seconds. Measured on the #836 reporter's 446.500 MHz
// captures (2.4 MS/s, gain 200, two PTTs): the second PTT's first sync word
// came 1.5 s after its carrier, its whole Voice LC Header train (ten copies,
// 0.6 s) lost, and starting a cold receiver on the same PTT at ten sub-sample
// offsets decoded the first burst at seven of them and took 1.3–2.8 s at the
// other three. A PTT therefore had roughly a one-in-three chance of losing its
// header — and with it the grant, until late entry caught the embedded LC
// seconds later.
//
// # What this does
//
// When the carrier reappears after an absence longer than the inter-burst gap
// (timingAcqGapSymbols — i.e. a NEW transmission, not the next burst of the
// current one), the receiver skips the matched filter's transient and the
// burst ramp (timingAcqSkipSymbols), collects timingAcqSymbols symbol periods
// of contiguous matched-filter output from inside the first burst, estimates
// the symbol phase feed-forward (sync.EstimateSymbolPhase — an eye-opening
// search that needs no prior phase) and seeds the loop with it
// (MuellerMuller.SetPhase) between the two halves of the chunk that completed
// the window. The loop then refines from within a fraction of a sample instead
// of from wherever it happened to be. The first burst's own sync word (dibits
// 54–77 of 132) is inside the acquisition window and is still lost; DMR
// transmitters repeat their header burst, so the second copy is decoded.
//
// A window interrupted by an absent sample is discarded and re-armed for the
// next burst of the same transmission (the estimate needs contiguous samples).
// The same acquisition runs on a cold start and after the coarse carrier
// acquirer resets the loop on engage. Without the gate (NoCarrierGate, or the
// legacy DeviationHz<=0 path) there are no presence flags and nothing here
// runs, so those paths are byte-identical to before.
const (
	// timingAcqGapSymbols is the absence, in symbol periods, past which the
	// next presence is a new transmission. The direct-mode inter-burst gap
	// is 156 symbols (32.5 ms); 480 symbols is 100 ms.
	timingAcqGapSymbols = 480.0
	// timingAcqSkipSymbols is how far into the burst the window opens: past
	// the RRC matched filter's group delay (PulseSpanSymbols/2 = 4), the
	// gate's own detection lag (carrierGateSymbols = 4) and the ramp.
	timingAcqSkipSymbols = 10
	// timingAcqSymbols is the window length. 96 symbols keeps the kurtosis
	// estimate within a sample ≥ 95 % of the time (sync package tests) and,
	// with the skip, still fits inside one 132-symbol burst.
	timingAcqSymbols = 96
)

// armTimingAcq schedules an acquisition window on the next present samples.
func (r *Receiver) armTimingAcq() {
	if !r.acqTiming {
		return
	}
	r.acqPending = true
	r.acqSkip = int(timingAcqSkipSymbols * r.sps)
	r.acqBuf = r.acqBuf[:0]
}

// observeTimingAcq walks this chunk's presence flags (aligned with r.matched),
// arming a window at every new-transmission onset and filling the armed one.
// When a window completes inside this chunk it returns the chunk index k at
// which the clock must be seeded (samples [0,k) are timed as before, [k,…)
// after the seed) and the seed value mu: the loop's next symbol instant, in
// samples past chunk sample k−1, on the estimated phase's grid. At most one
// seed is applied per chunk.
func (r *Receiver) observeTimingAcq(present []bool) (k int, mu float64, seed bool) {
	if !r.acqTiming {
		return 0, 0, false
	}
	gap := int(timingAcqGapSymbols * r.sps)
	window := int(timingAcqSymbols * r.sps)
	for i, on := range present {
		if !on {
			r.absentRun++
			if r.acqPending && len(r.acqBuf) > 0 {
				// The burst ended (or the gate flickered) mid-window: the
				// estimate needs contiguous samples, so start over on the
				// next burst of this transmission.
				r.acqBuf = r.acqBuf[:0]
				r.acqSkip = int(timingAcqSkipSymbols * r.sps)
			}
			continue
		}
		if r.absentRun >= gap {
			r.armTimingAcq()
		}
		r.absentRun = 0
		if !r.acqPending || seed {
			continue
		}
		if r.acqSkip > 0 {
			r.acqSkip--
			continue
		}
		if len(r.acqBuf) == 0 {
			r.acqStart = r.sampleBase + i
		}
		r.acqBuf = append(r.acqBuf, r.matched[i])
		if len(r.acqBuf) < window {
			continue
		}
		tau, ok := sync.EstimateSymbolPhase(r.acqBuf, r.sps)
		r.acqPending = false
		r.acqBuf = r.acqBuf[:0]
		if !ok {
			continue
		}
		// The symbol instants sit at acqStart + tau + n·sps. The loop's next
		// instant must be the first of those past the last sample it will
		// have processed before the seed, chunk sample i.
		last := float64(r.sampleBase + i)
		inst := float64(r.acqStart) + tau
		if inst <= last {
			inst += math.Ceil((last-inst)/r.sps) * r.sps
			if inst <= last {
				inst += r.sps
			}
		}
		k, mu, seed = i+1, inst-last, true
		r.timingSeeds++
	}
	return k, mu, seed
}
