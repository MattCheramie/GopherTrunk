package hunt

import "sort"

// wbStep is the per-step record the wideband-occupancy pass keeps while
// sweeping. Frames are not retained (a whole-device sweep is hundreds of
// steps); only this small summary is, so the stitch pass can run after the
// sweep-wide noise floor is known.
type wbStep struct {
	floorDb float32     // this step's per-frame low-quartile noise floor
	lowHz   uint32      // low edge of the step's scanned (non-guard) coverage
	highHz  uint32      // high edge of the step's scanned coverage
	spans   []Occupancy // occupancy spans found with this step's per-frame floor
}

// wbInterval is a frequency interval accumulated during stitching.
type wbInterval struct {
	lowHz, highHz uint32
	powerDb       float32 // strongest median power contributing to the interval
}

// defaultFullStepClipDb is how far a step's own noise floor must sit above the
// sweep-wide floor before the whole step is treated as occupied (the interior
// of a signal wider than the tune). 6 dB matches the occupancy threshold.
const defaultFullStepClipDb float32 = 6

// stitchWideband turns per-step occupancy observations into wideband
// Candidates. It learns the sweep-wide noise floor (the quietest step's
// floor), then:
//
//   - takes every per-frame occupancy span (these pin the true edges of a wide
//     signal at the steps where it enters/leaves the tune), and
//   - for any step whose own floor sits fullStepClipDb above the sweep-wide
//     floor — i.e. the whole step is inside a signal too wide to show a quiet
//     region — synthesizes a full-step span over its coverage.
//
// It then interval-merges everything (overlapping or within mergeTolHz) so a
// signal seen across N adjacent overlapping steps collapses into one span whose
// width is its true occupied bandwidth. Each merged span becomes a Candidate
// with IsWideband=true and BwHz set.
func stitchWideband(steps []wbStep, fullStepClipDb float32, mergeTolHz uint32) []Candidate {
	if len(steps) == 0 {
		return nil
	}
	if fullStepClipDb <= 0 {
		fullStepClipDb = defaultFullStepClipDb
	}

	globalFloor := steps[0].floorDb
	for _, s := range steps {
		if s.floorDb < globalFloor {
			globalFloor = s.floorDb
		}
	}

	var spans []wbInterval
	for _, s := range steps {
		for _, o := range s.spans {
			spans = append(spans, wbInterval{lowHz: o.LowHz, highHz: o.HighHz, powerDb: o.PowerDbFS})
		}
		// Fully-occupied interior step: its per-frame floor is itself elevated,
		// so DetectOccupancy (run with the per-frame floor) saw nothing. Fill in
		// the step's whole coverage so the interior stitches to the edge spans.
		if s.floorDb-globalFloor >= fullStepClipDb && s.highHz > s.lowHz {
			spans = append(spans, wbInterval{lowHz: s.lowHz, highHz: s.highHz, powerDb: s.floorDb})
		}
	}
	if len(spans) == 0 {
		return nil
	}

	sort.Slice(spans, func(a, b int) bool { return spans[a].lowHz < spans[b].lowHz })

	merged := []wbInterval{spans[0]}
	for _, s := range spans[1:] {
		cur := &merged[len(merged)-1]
		// Overlapping, or within the seam tolerance ⇒ same emitter.
		if s.lowHz <= cur.highHz+mergeTolHz {
			if s.highHz > cur.highHz {
				cur.highHz = s.highHz
			}
			if s.powerDb > cur.powerDb {
				cur.powerDb = s.powerDb
			}
			continue
		}
		merged = append(merged, s)
	}

	out := make([]Candidate, 0, len(merged))
	for _, m := range merged {
		out = append(out, Candidate{
			FreqHz:     uint32((uint64(m.lowHz) + uint64(m.highHz)) / 2),
			SNRDb:      m.powerDb - globalFloor,
			IsWideband: true,
			BwHz:       m.highHz - m.lowHz,
		})
	}
	return out
}
