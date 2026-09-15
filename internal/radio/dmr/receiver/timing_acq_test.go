package receiver

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// syncsIn counts the DM data-sync words in a receiver's dibit stream, with
// the feed-forward timing acquisition on or off, over iq fed in
// RTL-realistic chunks.
func syncsIn(iq []complex64, acqTiming bool) (total int, positions []int) {
	var got []uint8
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1944.0,
		ClockGain:    0.015,
		DibitSink:    func(d []uint8, _ int) { got = append(got, d...) },
	})
	r.acqTiming = acqTiming
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		e := i + chunk
		if e > len(iq) {
			e = len(iq)
		}
		r.Process(iq[i:e])
	}
	det := dmr.NewSyncDetector([]dmr.SyncPattern{dmr.DMData1}, 2)
	matches, _ := det.Process(nil, got, 0)
	for _, m := range matches {
		positions = append(positions, m.Index)
	}
	return len(matches), positions
}

// shiftIQ prepends k samples of the stream's own noise level so the
// transmission's burst grid lands k samples later on the receiver's sample
// clock — k of sps sub-symbol start phases.
func shiftIQ(iq []complex64, k int, sigma float64, seed int64) []complex64 {
	rng := rand.New(rand.NewSource(seed))
	out := make([]complex64, 0, len(iq)+k)
	for i := 0; i < k; i++ {
		out = append(out, complex(float32(rng.NormFloat64()*sigma), float32(rng.NormFloat64()*sigma)))
	}
	return append(out, iq...)
}

// TestReceiverAcquiresDirectModeAtEverySymbolPhase is the issue #836 "first
// seconds of every PTT" regression. A keyup lands on the receiver's sample
// clock at an arbitrary sub-symbol phase, and the gated Mueller-Müller loop
// pulls a bad start phase in at gain·error per symbol — from near half a
// symbol that took 1.3–2.8 s on the reporter's capture, losing the whole
// header train. With the feed-forward phase estimate at the onset every
// start phase must decode from the second burst. The old receiver
// (acqTiming off) fails at some phases: that is the failing-first half.
func TestReceiverAcquiresDirectModeAtEverySymbolPhase(t *testing.T) {
	const frames = 24
	bursts := dmHeaderBursts(8)
	base := directModeIQ(frames, bursts, true, 900, 0.03, 836)
	var worstOld, worstNew = frames, frames
	for k := 0; k < 10; k++ {
		iq := shiftIQ(base, k, 0.03, int64(k))
		nNew, _ := syncsIn(iq, true)
		nOld, _ := syncsIn(iq, false)
		t.Logf("start phase %d/10: syncs with acquisition=%d of %d, without=%d", k, nNew, frames, nOld)
		if nNew < worstNew {
			worstNew = nNew
		}
		if nOld < worstOld {
			worstOld = nOld
		}
		if nNew < frames-2 {
			t.Errorf("start phase %d/10: only %d of %d bursts decoded with feed-forward acquisition", k, nNew, frames)
		}
	}
	// Failing-first evidence: without the acquisition at least one start
	// phase loses most of the transmission to the loop's slow pull-in.
	if worstOld > frames/2 {
		t.Errorf("the ungated-timing receiver decoded ≥ %d of %d bursts at every phase — the fixture no longer exhibits the slow pull-in this test pins", worstOld, frames)
	}
}

// TestReceiverReacquiresTimingOnSecondTransmission: the phase the loop held
// through the gap after one PTT is unrelated to the next PTT's, so the
// second transmission must be re-acquired feed-forward too — the reporter's
// second PTT lost 1.5 s (its ten header copies) to the held phase.
func TestReceiverReacquiresTimingOnSecondTransmission(t *testing.T) {
	const frames = 24
	bursts := dmHeaderBursts(8)
	first := directModeIQ(frames, bursts, true, 900, 0.03, 11)
	second := directModeIQ(frames, bursts, true, 900, 0.03, 12)
	rng := rand.New(rand.NewSource(5))
	gap := make([]complex64, 48_000/2) // 0.5 s of noise between the PTTs
	for i := range gap {
		gap[i] = complex(float32(rng.NormFloat64()*0.03), float32(rng.NormFloat64()*0.03))
	}
	worstOld := frames
	for k := 0; k < 10; k++ {
		iq := append(append(append([]complex64(nil), first...), gap...), shiftIQ(second, k, 0.03, int64(100+k))...)
		boundary := (len(first) + len(gap)) / 10 // dibit index where the second PTT begins
		count := func(acq bool) int {
			_, pos := syncsIn(iq, acq)
			n := 0
			for _, p := range pos {
				if p >= boundary {
					n++
				}
			}
			return n
		}
		nNew, nOld := count(true), count(false)
		t.Logf("second PTT start phase %d/10: syncs with acquisition=%d of %d, without=%d", k, nNew, frames, nOld)
		if nNew < frames-2 {
			t.Errorf("second PTT at start phase %d/10: only %d of %d bursts decoded", k, nNew, frames)
		}
		if nOld < worstOld {
			worstOld = nOld
		}
	}
	if worstOld > frames/2 {
		t.Errorf("without acquisition the second PTT decoded ≥ %d of %d at every phase — fixture no longer exhibits the held-phase pull-in", worstOld, frames)
	}
}
