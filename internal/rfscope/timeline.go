package rfscope

import (
	"context"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func init() { Register(timelineAnalyzer{}) }

// timelineBins is the default number of time buckets in a channel's activity
// timeline — the resolution of the I/O graph.
const timelineBins = 100

// timelineAnalyzer builds the per-channel activity view — the RF analog of
// Wireshark's I/O Graphs. It groups bursts by frequency into Channels and fills
// each with a time-bucketed occupancy/power series plus duty cycle, occupancy
// percentage, and burst rate. It is also the analyzer that materializes
// Scene.Channels, which the timing analyzer then annotates.
type timelineAnalyzer struct{}

func (timelineAnalyzer) Name() string        { return "timeline" }
func (timelineAnalyzer) Synopsis() string    { return "per-channel activity timeline / I/O graph" }
func (timelineAnalyzer) DependsOn() []string { return nil }

func (timelineAnalyzer) Analyze(ctx context.Context, sc *Scene, in *Input) error {
	if len(sc.Bursts) == 0 {
		return nil
	}
	window := sc.WindowSec
	if window <= 0 {
		return nil
	}

	byFreq := map[uint32][]int{}
	for i, b := range sc.Bursts {
		byFreq[b.FreqHz] = append(byFreq[b.FreqHz], i)
	}

	freqs := make([]uint32, 0, len(byFreq))
	for f := range byFreq {
		freqs = append(freqs, f)
	}
	sort.Slice(freqs, func(i, j int) bool { return freqs[i] < freqs[j] })

	nbins := timelineBins
	binDur := window / float64(nbins)
	channels := make([]Channel, 0, len(freqs))

	for _, f := range freqs {
		if err := ctx.Err(); err != nil {
			return err
		}
		idxs := byFreq[f]

		ch := Channel{FreqHz: f, BurstCount: len(idxs)}
		classCount := map[survey.SignalClass]int{}
		var powers, flats []float64
		for _, i := range idxs {
			b := sc.Bursts[i]
			ch.AirtimeSec += b.DurationSec()
			classCount[b.Class]++
			if b.OccupiedBwHz > ch.OccupiedBwHz {
				ch.OccupiedBwHz = b.OccupiedBwHz
			}
			powers = append(powers, float64(b.PeakDbFS))
			flats = append(flats, b.SpectralFlatness)
		}
		ch.DominantClass = mode(classCount)
		ch.MedianPowerDbFS = float32(median(powers))
		ch.MedianFlatness = median(flats)
		ch.DutyCycle = clamp01(ch.AirtimeSec / window)
		ch.BurstRatePerMin = float64(ch.BurstCount) / (window / 60)

		// Time-bucketed timeline.
		timeline := make([]Bin, nbins)
		occupiedBins := 0
		for k := 0; k < nbins; k++ {
			t0 := float64(k) * binDur
			t1 := t0 + binDur
			bin := Bin{TSec: t0 + binDur/2, PowerDbFS: sc.NoiseFloorDbFS}
			for _, i := range idxs {
				b := sc.Bursts[i]
				if b.StartSec < t1 && b.EndSec > t0 { // overlap
					bin.Occupied = true
					if b.PeakDbFS > bin.PowerDbFS {
						bin.PowerDbFS = b.PeakDbFS
					}
				}
				if b.StartSec >= t0 && b.StartSec < t1 { // started in this bin
					bin.BurstCount++
				}
			}
			if bin.Occupied {
				occupiedBins++
			}
			timeline[k] = bin
		}
		ch.Timeline = timeline
		ch.OccupancyPct = 100 * float64(occupiedBins) / float64(nbins)

		channels = append(channels, ch)
	}
	sc.Channels = channels
	return nil
}

// mode returns the most frequent class (ties broken by name for determinism).
func mode(counts map[survey.SignalClass]int) survey.SignalClass {
	best := survey.ClassUnknown
	bestN := -1
	keys := make([]survey.SignalClass, 0, len(counts))
	for c := range counts {
		keys = append(keys, c)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, c := range keys {
		if counts[c] > bestN {
			best, bestN = c, counts[c]
		}
	}
	return best
}

// median returns the median of xs (0 for empty). xs is copied before sorting.
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
