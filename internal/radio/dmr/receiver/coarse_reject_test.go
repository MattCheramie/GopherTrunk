package receiver

import (
	"math"
	"testing"
)

// TestCoarseAcquirerRejectedOffsetNeverReengages pins the decode-evidence
// hook behind the 15 Sep IPSC log: the wideband tap's acquirer froze at
// −20.1 kHz on every idle gap of a GPSDO-locked X310 (a neighbour that
// dominates the tap while the wanted repeater is silent), deafening the
// channel until the deaf-tap heal reset it — 21 times in 6 minutes. Once the
// consumer rejects that offset the stage must (1) revert the live correction,
// (2) never engage near it again, (3) keep that across Reset (the heal resets
// the receiver), and (4) still engage on a genuinely different offset.
func TestCoarseAcquirerRejectedOffsetNeverReengages(t *testing.T) {
	const chunk = 4096
	stream := balancedStream(3000)
	run := func(r *Receiver, offsetHz float64) {
		iq := makeC4FMIQWithOffset(stream, offsetHz)
		for i := 0; i < len(iq); i += chunk {
			end := i + chunk
			if end > len(iq) {
				end = len(iq)
			}
			r.Process(iq[i:end])
		}
	}
	r := New(Options{SampleRateHz: 48_000, DeviationHz: 1944.0, DibitSink: func([]uint8, int) {}})

	// The "neighbour": an offset carrier the stage engages on, as designed.
	run(r, -6000)
	got := r.CoarseCarrierOffsetHz()
	if math.Abs(got+6000) > 400 {
		t.Fatalf("stage did not engage on the −6 kHz scene: offset %.1f Hz", got)
	}
	// Decode evidence rules it out: the live correction reverts at once.
	if !r.RejectCoarseCarrierOffset(got) {
		t.Fatalf("RejectCoarseCarrierOffset(%.1f) reported no reverted correction", got)
	}
	if o := r.CoarseCarrierOffsetHz(); o != 0 {
		t.Fatalf("offset after reject = %.1f Hz, want 0 (identity)", o)
	}
	// The same scene again: the stage must stay dormant.
	run(r, -6000)
	if o := r.CoarseCarrierOffsetHz(); o != 0 {
		t.Fatalf("stage re-engaged on the rejected offset: %.1f Hz", o)
	}
	// A full receiver Reset (what the deaf-tap heal does) keeps the rejection.
	r.Reset()
	run(r, -6000)
	if o := r.CoarseCarrierOffsetHz(); o != 0 {
		t.Fatalf("rejection did not survive Reset: stage engaged at %.1f Hz", o)
	}
	// A different offset — a real tuner error on the wanted carrier — is
	// untouched by the rejection.
	run(r, 4000)
	if o := r.CoarseCarrierOffsetHz(); math.Abs(o-4000) > 400 {
		t.Fatalf("stage did not engage on a different (+4 kHz) offset after a rejection: %.1f Hz", o)
	}
	// Rejecting an offset the stage is not frozen at reverts nothing.
	if r.RejectCoarseCarrierOffset(-9000) {
		t.Errorf("rejecting an unrelated offset reported a reverted correction")
	}
	if o := r.CoarseCarrierOffsetHz(); math.Abs(o-4000) > 400 {
		t.Errorf("unrelated rejection disturbed the live +4 kHz correction: %.1f Hz", o)
	}
}
