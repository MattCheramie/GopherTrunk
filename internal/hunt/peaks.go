package hunt

import (
	"github.com/MattCheramie/GopherTrunk/internal/carriers"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
)

// Peak and PeakOptions are re-exported from internal/carriers so the live
// sweeper keeps its existing public surface while the peak detector itself
// lives in the neutral carriers package (shared with the siglab wideband
// survey). See internal/carriers for the implementation.
type Peak = carriers.Peak
type PeakOptions = carriers.PeakOptions

// DetectPeaks finds candidate carriers in a spectrum frame. Thin wrapper over
// carriers.DetectPeaks (kept for the sweeper's call sites).
func DetectPeaks(frame spectrum.Frame, opts PeakOptions) []Peak {
	return carriers.DetectPeaks(frame, opts)
}

// Occupancy and OccupancyOptions are re-exported from internal/carriers so the
// sweeper's wideband-occupancy pass keeps the hunt-package surface while the
// detector lives in the neutral carriers package.
type Occupancy = carriers.Occupancy
type OccupancyOptions = carriers.OccupancyOptions

// DetectOccupancy finds wideband occupancy spans in a spectrum frame. Thin
// wrapper over carriers.DetectOccupancy.
func DetectOccupancy(frame spectrum.Frame, opts OccupancyOptions) []Occupancy {
	return carriers.DetectOccupancy(frame, opts)
}

// absDiff is the |a-b| helper the sweeper + peak tests use. carriers has its
// own copy for DetectPeaks; this one stays so the hunt-package tests and
// sweeper logic don't depend on the carriers internal.
func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
