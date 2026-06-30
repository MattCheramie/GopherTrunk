package carriers

import (
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
)

// Occupancy is one contiguous span of spectrum standing above the noise
// floor — a signal far wider than a single land-mobile channel, such as an
// OFDM cellular/WiFi block. Where [DetectPeaks] finds narrow carriers,
// DetectOccupancy finds these plateaus.
//
// EdgeClippedLow/High flag a span whose energy runs right up to the frame's
// usable edge: the emitter is at least OccupiedBwHz wide, but its true extent
// continues past what a single tune can see. The live sweeper stitches such
// clipped spans across adjacent overlapping steps to recover the real width
// (see internal/hunt). The flags are how it knows two steps' spans are the
// same emitter seen at a seam.
type Occupancy struct {
	CenterHz     uint32  `json:"center_hz"`
	LowHz        uint32  `json:"low_hz"`
	HighHz       uint32  `json:"high_hz"`
	OccupiedBwHz uint32  `json:"occupied_bw_hz"`
	PowerDbFS    float32 `json:"power_dbfs"` // median power across the span
	SNRDb        float32 `json:"snr_db"`     // median power above the noise floor

	EdgeClippedLow  bool `json:"edge_clipped_low,omitempty"`
	EdgeClippedHigh bool `json:"edge_clipped_high,omitempty"`
}

// OccupancyOptions tune the wideband-occupancy detector.
type OccupancyOptions struct {
	// ThresholdDb is the minimum power above the estimated noise floor for a
	// bin to count as occupied. It is intentionally lower than the peak
	// detector's: a wide OFDM plateau is flatter and lower per-bin than a
	// narrow carrier, but contiguous. 0 ⇒ 6 dB.
	ThresholdDb float32
	// MinBwHz is the shortest contiguous run reported. The default sits well
	// above any land-mobile channel so a normal narrowband carrier never
	// registers as "wideband"; only genuinely wide emitters do. 0 ⇒ 200 kHz.
	MinBwHz uint32
	// GuardBins drops this many bins at each band edge (the SDR rolloff) from
	// detection; a run that reaches the first/last scanned bin is flagged
	// edge-clipped. 0 ⇒ a default proportional to the FFT size.
	GuardBins int
	// FloorDbFS overrides the per-frame noise-floor estimate. A signal wider
	// than the tune fills a whole step, so that step's own low-quartile floor
	// sits *inside* the signal and the per-frame estimate detects nothing. The
	// live sweeper passes a sweep-wide floor (learned from quiet steps) here so
	// a fully-occupied step is still recognised as occupied and stitches to its
	// neighbours. 0 ⇒ use the per-frame low-quartile estimate.
	FloorDbFS float32
}

// occupancyMinBwHz is the default floor: 200 kHz is ~16× the widest C4FM
// channel, so a narrowband carrier can never be mistaken for a wideband span.
const occupancyMinBwHz uint32 = 200_000

// DetectOccupancy finds contiguous spans of occupied spectrum in a frame —
// the wideband counterpart to [DetectPeaks]. It estimates the same robust
// noise floor (low-quartile), marks every non-guard bin standing ThresholdDb
// above it, coalesces adjacent occupied bins into runs, and returns each run
// at least MinBwHz wide as an [Occupancy] with absolute frequency bounds and
// edge-clip flags. Spans are returned sorted by ascending low edge so a
// caller can stitch neighbours.
func DetectOccupancy(frame spectrum.Frame, opts OccupancyOptions) []Occupancy {
	n := len(frame.Bins)
	if n < 8 || frame.SampleRate == 0 {
		return nil
	}
	threshold := opts.ThresholdDb
	if threshold <= 0 {
		threshold = 6
	}
	minBw := opts.MinBwHz
	if minBw == 0 {
		minBw = occupancyMinBwHz
	}
	guard := opts.GuardBins
	if guard <= 0 {
		guard = n / 64
		if guard < 1 {
			guard = 1
		}
	}

	floor := opts.FloorDbFS
	if floor == 0 {
		floor = spectrum.Percentile(frame.Bins, noiseFloorPercentile)
	}
	binHz := float64(frame.SampleRate) / float64(n)
	lastScan := n - guard - 1 // inclusive index of the last bin we examine

	var out []Occupancy
	runStart := -1
	flush := func(start, end int) {
		if start < 0 {
			return
		}
		// Bin centers; the span spans from the start bin's left edge to the
		// end bin's right edge, so it is (end-start+1) bins wide.
		lowHz := binToHz(frame, start, binHz)
		highHz := binToHz(frame, end, binHz)
		bwHz := uint32(float64(end-start+1) * binHz)
		if bwHz < minBw {
			return
		}
		run := frame.Bins[start : end+1]
		med := spectrum.Percentile(run, 0.5)
		out = append(out, Occupancy{
			CenterHz:        uint32((uint64(lowHz) + uint64(highHz)) / 2),
			LowHz:           lowHz,
			HighHz:          highHz,
			OccupiedBwHz:    bwHz,
			PowerDbFS:       med,
			SNRDb:           med - floor,
			EdgeClippedLow:  start == guard,
			EdgeClippedHigh: end == lastScan,
		})
	}

	for i := guard; i <= lastScan; i++ {
		occupied := frame.Bins[i]-floor >= threshold
		switch {
		case occupied && runStart < 0:
			runStart = i
		case !occupied && runStart >= 0:
			flush(runStart, i-1)
			runStart = -1
		}
	}
	if runStart >= 0 {
		flush(runStart, lastScan)
	}

	sort.Slice(out, func(a, b int) bool { return out[a].LowHz < out[b].LowHz })
	return out
}
