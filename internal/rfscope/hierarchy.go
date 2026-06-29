package rfscope

import (
	"context"
	"fmt"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

func init() { Register(hierarchyAnalyzer{}) }

// hierarchyAnalyzer builds the RF "protocol hierarchy" — the analog of
// Wireshark's Protocol Hierarchy Statistics, one layer down at the physical
// layer. It groups bursts by modulation class → occupied-bandwidth bucket →
// identified protocol, attaching counts, airtime, spectrum/time share, and an
// estimated symbol volume to each node.
type hierarchyAnalyzer struct{}

func (hierarchyAnalyzer) Name() string { return "hierarchy" }
func (hierarchyAnalyzer) Synopsis() string {
	return "RF protocol hierarchy (class → bandwidth → protocol)"
}
func (hierarchyAnalyzer) DependsOn() []string { return nil }

func (hierarchyAnalyzer) Analyze(ctx context.Context, sc *Scene, in *Input) error {
	if len(sc.Bursts) == 0 {
		return nil
	}

	// class → bandwidth bucket (Hz) → burst indices.
	byClass := map[survey.SignalClass]map[uint32][]int{}
	for i, b := range sc.Bursts {
		bucket := survey.SnapChannelBandwidth(b.OccupiedBwHz)
		if byClass[b.Class] == nil {
			byClass[b.Class] = map[uint32][]int{}
		}
		byClass[b.Class][bucket] = append(byClass[b.Class][bucket], i)
	}

	sc.Hierarchy.Total = aggregateStats(sc, allIndices(len(sc.Bursts)))

	classes := make([]survey.SignalClass, 0, len(byClass))
	for c := range byClass {
		classes = append(classes, c)
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i] < classes[j] })

	var nodes []HierNode
	for _, class := range classes {
		if err := ctx.Err(); err != nil {
			return err
		}
		buckets := byClass[class]

		var classIdx []int
		for _, idxs := range buckets {
			classIdx = append(classIdx, idxs...)
		}
		nodes = append(nodes, HierNode{
			Depth: 0,
			Label: string(class),
			Class: class,
			Stats: aggregateStats(sc, classIdx),
		})

		bwKeys := make([]uint32, 0, len(buckets))
		for bw := range buckets {
			bwKeys = append(bwKeys, bw)
		}
		sort.Slice(bwKeys, func(i, j int) bool { return bwKeys[i] < bwKeys[j] })

		for _, bw := range bwKeys {
			idxs := buckets[bw]
			nodes = append(nodes, HierNode{
				Depth: 1,
				Label: formatBandwidth(bw),
				Class: class,
				Stats: aggregateStats(sc, idxs),
			})

			// Depth 2: name the protocol of a representative digital burst.
			if !survey.IsDigital(class) {
				continue
			}
			proto := identifyRepresentative(sc, in, idxs)
			if proto == "" {
				continue
			}
			st := aggregateStats(sc, idxs)
			nodes = append(nodes, HierNode{
				Depth:    2,
				Label:    proto,
				Class:    class,
				Protocol: proto,
				Stats:    st,
			})
		}
	}
	sc.Hierarchy.Nodes = nodes
	return nil
}

// identifyRepresentative runs siglab.IdentifyIQ on the longest burst in the
// bucket and returns the winning protocol, or "" when identification is
// inconclusive or no IQ is available.
func identifyRepresentative(sc *Scene, in *Input, idxs []int) string {
	if in == nil || in.BurstIQ == nil || len(idxs) == 0 {
		return ""
	}
	best := idxs[0]
	for _, i := range idxs {
		if sc.Bursts[i].DurationSec() > sc.Bursts[best].DurationSec() {
			best = i
		}
	}
	iq, rate, err := in.BurstIQ(sc.Bursts[best])
	if err != nil || len(iq) == 0 || rate <= 0 {
		return ""
	}
	res, err := siglab.IdentifyIQ(iq, "rfscope-burst", siglab.IdentifyConfig{
		SampleRateHz: rate,
		Format:       siglab.FormatF32,
		Log:          in.Log,
	})
	if err != nil || res == nil || res.Inconclusive || res.Winner == "" {
		return ""
	}
	return res.Winner
}

// aggregateStats sums the figures for a set of bursts: counts, airtime, the
// share of spectrum (distinct channel bandwidth over the span) and time
// (airtime over the window), distinct emitters, and an estimated symbol volume
// (Σ duration·baud).
func aggregateStats(sc *Scene, idxs []int) HierStats {
	var st HierStats
	st.BurstCount = len(idxs)
	emitters := map[uint64]struct{}{}
	chanBw := map[uint32]uint32{} // freq → max occupied bw on that channel
	for _, i := range idxs {
		b := sc.Bursts[i]
		st.AirtimeSec += b.DurationSec()
		st.SymbolVolume += int64(b.DurationSec() * b.Features.BaudHz)
		if b.EmitterID != 0 {
			emitters[b.EmitterID] = struct{}{}
		}
		if b.OccupiedBwHz > chanBw[b.FreqHz] {
			chanBw[b.FreqHz] = b.OccupiedBwHz
		}
	}
	st.EmitterCount = len(emitters)

	var spectrumHz uint32
	for _, bw := range chanBw {
		spectrumHz += bw
	}
	if sc.SpanHz > 0 {
		st.SpectrumPct = 100 * float64(spectrumHz) / float64(sc.SpanHz)
	}
	if sc.WindowSec > 0 {
		st.TimePct = 100 * st.AirtimeSec / sc.WindowSec
	}
	return st
}

func allIndices(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// formatBandwidth renders a channel bandwidth as a compact label (e.g. "12.5 kHz").
func formatBandwidth(hz uint32) string {
	switch {
	case hz == 0:
		return "unknown-bw"
	case hz >= 1_000_000:
		return fmt.Sprintf("%.3g MHz", float64(hz)/1e6)
	default:
		return fmt.Sprintf("%.4g kHz", float64(hz)/1e3)
	}
}
