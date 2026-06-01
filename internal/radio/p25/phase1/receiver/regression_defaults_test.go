package receiver

import "testing"

// TestDefaultC4FMOptionsDisableDDAAndAdaptiveSlicer pins the issue #402
// reverts so a future change can't silently revive them. The production
// pipeline (internal/scanner/ccdecoder/pipelines.go) builds the C4FM receiver
// with neither EnableDecisionDirectedAFC nor EnableAdaptiveC4FMSlicer set:
//
//   - The decision-directed AFC (#412) stably false-locked onto a wrong
//     carrier offset with no FSW/CC-lock feedback to catch it, dropping the
//     MMR Site 9 capture from ~60% to 0% lock. Reverted by #430.
//   - The adaptive C4FM slicer (#432/#439/#447) went through truncation-bias
//     runaway, over- then under-regularization, and finally a catastrophic
//     0-FSW collapse before being made opt-in. Reverted to the fixed slicer
//     default by #450.
//
// Both remain available as explicit opt-ins (replay -dda / -adaptive-slicer)
// for experimentation, but the daemon's default C4FM path must stay
// CoarseAFC-alone with the fixed-threshold slicer.
func TestDefaultC4FMOptionsDisableDDAAndAdaptiveSlicer(t *testing.T) {
	// Mirror the production options from pipelines.go: DeviationHz>0 so the
	// adaptive slicer COULD activate — proving these are off by default, not
	// merely gated out by a zero deviation.
	def := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		DemodMode:    DemodC4FM,
		DibitSink:    func([]uint8, int) {},
	})
	if def.dda != nil {
		t.Error("default C4FM receiver constructed a DDA; #412 was reverted (#430) — DDA must stay off by default")
	}
	if def.slicer != nil {
		t.Error("default C4FM receiver constructed the adaptive slicer; #432/#439/#447 were reverted (#450) — fixed slicer must stay the default")
	}
	if def.DDAActive() {
		t.Error("default C4FM receiver reports DDAActive() true before any processing")
	}

	// Guard against the inverse rot: the opt-in flags must still wire the
	// components up, so the test above is pinning the DEFAULT rather than a
	// dead code path that can never activate.
	on := New(Options{
		SampleRateHz:              48_000,
		DeviationHz:               1800.0,
		DemodMode:                 DemodC4FM,
		EnableDecisionDirectedAFC: true,
		EnableAdaptiveC4FMSlicer:  true,
		DibitSink:                 func([]uint8, int) {},
	})
	if on.dda == nil {
		t.Error("EnableDecisionDirectedAFC=true did not construct the DDA")
	}
	if on.slicer == nil {
		t.Error("EnableAdaptiveC4FMSlicer=true did not construct the adaptive slicer")
	}
}
