package rfscope

import (
	"context"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/stats"
)

func init() { Register(timingAnalyzer{}) }

// timingHistBins is the default bucket count for the burst-length and
// inter-arrival histograms.
const timingHistBins = 10

// minPeriodConfidence is the autocorrelation peak height (0..1) below which a
// detected period is treated as noise and discarded.
const minPeriodConfidence = 0.2

// timingAnalyzer annotates each Channel with burst-timing statistics: the
// burst-length and inter-arrival histograms, and the periodicity/TDMA-frame
// period recovered by autocorrelating the channel's occupancy signal
// (dsp/stats.Correlate + FindPeaks). It is the time-domain counterpart to the
// byte-domain period detection cryptolab applies to recovered payloads.
type timingAnalyzer struct{}

func (timingAnalyzer) Name() string        { return "timing" }
func (timingAnalyzer) Synopsis() string    { return "burst timing, inter-arrival, and TDMA-period stats" }
func (timingAnalyzer) DependsOn() []string { return []string{"timeline"} }

func (timingAnalyzer) Analyze(ctx context.Context, sc *Scene, in *Input) error {
	if len(sc.Channels) == 0 {
		return nil
	}

	byFreq := map[uint32][]int{}
	for i, b := range sc.Bursts {
		byFreq[b.FreqHz] = append(byFreq[b.FreqHz], i)
	}

	binDur := 0.0
	if sc.WindowSec > 0 {
		binDur = sc.WindowSec / float64(timelineBins)
	}

	for ci := range sc.Channels {
		if err := ctx.Err(); err != nil {
			return err
		}
		ch := &sc.Channels[ci]
		idxs := byFreq[ch.FreqHz]

		// Burst-length and inter-arrival histograms.
		starts := make([]float64, 0, len(idxs))
		lengths := make([]float64, 0, len(idxs))
		for _, i := range idxs {
			starts = append(starts, sc.Bursts[i].StartSec)
			lengths = append(lengths, sc.Bursts[i].DurationSec())
		}
		sort.Float64s(starts)
		ch.BurstLenHist = histOf(lengths)

		gaps := make([]float64, 0, len(starts))
		for i := 1; i < len(starts); i++ {
			gaps = append(gaps, starts[i]-starts[i-1])
		}
		ch.InterArrivalHist = histOf(gaps)

		// Periodicity from the occupancy autocorrelation.
		if binDur > 0 && len(ch.Timeline) >= 4 {
			occ := make([]float64, len(ch.Timeline))
			for i, b := range ch.Timeline {
				if b.Occupied {
					occ[i] = 1
				}
			}
			ch.PeriodSec, ch.PeriodConfidence = detectPeriod(occ, binDur)
		}
	}
	return nil
}

// detectPeriod autocorrelates a (binary) occupancy signal and returns the period
// of its strongest non-zero lag, with the normalized autocorrelation height as
// confidence. Returns (0,0) when no lag clears minPeriodConfidence.
func detectPeriod(occ []float64, binDur float64) (periodSec, confidence float64) {
	n := len(occ)
	if n < 4 {
		return 0, 0
	}
	mean := stats.Mean(occ)
	sig := make([]float64, n)
	for i, v := range occ {
		sig[i] = v - mean
	}
	corr, _ := stats.Correlate(sig, sig)
	// Positive lags start at index n-1 (lag 0, value ~1) through 2n-2 (lag n-1).
	pos := corr[n-1:]
	if len(pos) < 3 {
		return 0, 0
	}
	peaks := stats.FindPeaks(pos[1:], stats.PeakOpts{Height: minPeriodConfidence, MinDistance: 1})
	if len(peaks) == 0 {
		return 0, 0
	}
	lag := peaks[0] + 1 // +1: pos[1:] dropped lag 0
	if lag <= 0 || lag >= len(pos) {
		return 0, 0
	}
	return float64(lag) * binDur, pos[lag]
}

// histOf builds a Hist over xs with a bucket count scaled to the sample size.
func histOf(xs []float64) Hist {
	if len(xs) == 0 {
		return Hist{}
	}
	nbins := timingHistBins
	if len(xs) < nbins {
		nbins = len(xs)
	}
	counts, edges := stats.Histogram(xs, nbins)
	return Hist{Counts: counts, Edges: edges}
}
