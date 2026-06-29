package rfscope

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/stats"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Trunking-deferral bounds: a burst must be at least this long to be worth a
// full decode, and at most this many emitters are decoded per scene.
const (
	minTrunkDecodeSec = 0.5
	maxTrunkDecode    = 8
)

func init() { Register(topologyAnalyzer{}) }

// hopperPowerBucketDb groups emitters into power bands when looking for a
// frequency hopper: candidate hop members must sit within one bucket.
const hopperPowerBucketDb = 6.0

// topologyAnalyzer builds the protocol-agnostic network topology — the
// generalization of hunt's trunking map. It clusters bursts into Emitters by RF
// fingerprint (so a frequency hopper's scattered bursts collapse into one
// logical source), then links temporally correlated emitters into Conversations
// (request/response, paired slots, hop sequences, co-active). This is the RF
// analog of Wireshark's Conversations/Endpoints, one layer down.
type topologyAnalyzer struct{}

func (topologyAnalyzer) Name() string        { return "topology" }
func (topologyAnalyzer) Synopsis() string    { return "emitter clustering + conversation graph" }
func (topologyAnalyzer) DependsOn() []string { return nil }

func (topologyAnalyzer) Analyze(ctx context.Context, sc *Scene, in *Input) error {
	if len(sc.Bursts) == 0 {
		return nil
	}
	clusterEmitters(sc)
	if err := ctx.Err(); err != nil {
		return err
	}
	detectConversations(sc)
	deferToTrunkingTopology(sc, in)
	return nil
}

// deferToTrunkingTopology is the authoritative-topology bridge: when an emitter's
// representative burst decodes as a trunking control channel, GopherTrunk's own
// hunt accumulator is the authority on its WACN/SysID/RFSS/Site/neighbors/band
// plan. We decode such bursts with siglab, fold them into hunt.DiscoveredSystems,
// and attach the result under AnalyzerOutputs["topology"] — so hunt's
// trunking-specific map becomes one specialization of the generic emitter graph.
func deferToTrunkingTopology(sc *Scene, in *Input) {
	if in == nil || in.BurstIQ == nil {
		return
	}
	systems := map[string]*hunt.DiscoveredSystem{}
	var order []string
	decoded := 0

	for _, e := range sc.Emitters {
		if decoded >= maxTrunkDecode {
			break
		}
		if e.Fingerprint.ClassFamily != "digital" || e.Fingerprint.MedianBurstLenSec < minTrunkDecodeSec {
			continue
		}
		bestID := longestBurstID(sc, e)
		if bestID == 0 {
			continue
		}
		iq, rate, err := in.BurstIQ(sc.Bursts[burstIndexByID(sc, bestID)])
		if err != nil || len(iq) == 0 || rate <= 0 {
			continue
		}
		ident, err := siglab.IdentifyIQ(iq, "rfscope-burst", siglab.IdentifyConfig{SampleRateHz: rate, Format: siglab.FormatF32, Log: in.Log})
		if err != nil || ident == nil || ident.Inconclusive || ident.Winner == "" {
			continue
		}
		proto, err := trunking.ParseProtocol(ident.Winner)
		if err != nil {
			continue
		}
		decoded++
		res, err := siglab.RunReader(bytes.NewReader(siglab.EncodeCapture(iq, siglab.FormatF32)), "rfscope-burst",
			siglab.Config{Protocol: proto, SampleRateHz: rate, Format: siglab.FormatF32})
		if err != nil || res == nil || !res.Locked || res.Topology == nil {
			continue
		}
		key := ident.Winner
		sys, ok := systems[key]
		if !ok {
			sys = &hunt.DiscoveredSystem{}
			systems[key] = sys
			order = append(order, key)
		}
		hunt.Accumulate(sys, hunt.Observation{
			Protocol:       ident.Winner,
			Confidence:     ident.Confidence,
			Result:         res,
			FallbackFreqHz: e.Fingerprint.CenterHz,
		})
	}

	if len(order) == 0 {
		return
	}
	out := make([]*hunt.DiscoveredSystem, 0, len(order))
	for _, k := range order {
		out = append(out, systems[k])
	}
	if sc.AnalyzerOutputs == nil {
		sc.AnalyzerOutputs = map[string]any{}
	}
	sc.AnalyzerOutputs["topology"] = out
}

// longestBurstID returns the ID of the emitter's longest burst.
func longestBurstID(sc *Scene, e Emitter) uint64 {
	var best uint64
	var bestDur float64
	for _, bid := range e.BurstIDs {
		b := sc.Bursts[burstIndexByID(sc, bid)]
		if b.DurationSec() >= bestDur {
			bestDur, best = b.DurationSec(), bid
		}
	}
	return best
}

// Conversation-detection tuning.
const (
	maxConvEmitters = 24  // cap pairwise comparison to the top-N talkers
	coactiveThresh  = 0.5 // lag-0 correlation ⇒ co-active
	rrThresh        = 0.4 // off-lag correlation peak ⇒ request/response
)

// detectConversations links temporally correlated emitters into Conversations —
// the RF analog of a conversation/endpoint graph. A frequency hopper is its own
// hop-sequence; a pair of emitters whose activity coincides is co-active, and a
// pair that alternates (a correlation peak at a non-zero lag) is request/response.
func detectConversations(sc *Scene) {
	window := sc.WindowSec
	if window <= 0 || len(sc.Emitters) == 0 {
		return
	}
	nbins := timelineBins
	binDur := window / float64(nbins)

	byID := map[uint64]int{}
	for i := range sc.Bursts {
		byID[sc.Bursts[i].ID] = i
	}

	// Per-emitter binary activity signal.
	act := make(map[uint64][]float64, len(sc.Emitters))
	for _, e := range sc.Emitters {
		sig := make([]float64, nbins)
		for _, bid := range e.BurstIDs {
			b := sc.Bursts[byID[bid]]
			lo := int(b.StartSec / binDur)
			hi := int(b.EndSec / binDur)
			for k := lo; k <= hi && k < nbins; k++ {
				if k >= 0 {
					sig[k] = 1
				}
			}
		}
		act[e.ID] = sig
	}

	var convs []Conversation
	var nextID uint64

	// Hop sequences (intra-emitter).
	for _, e := range sc.Emitters {
		if len(e.HopSet) > 1 {
			nextID++
			convs = append(convs, Conversation{
				ID: nextID, EmitterIDs: []uint64{e.ID}, Kind: "hop-sequence",
				Correlation: 1, Exchanges: len(e.BurstIDs),
			})
		}
	}

	// Pairwise among the top talkers.
	top := sc.Emitters
	if len(top) > maxConvEmitters {
		top = top[:maxConvEmitters]
	}
	for i := 0; i < len(top); i++ {
		for j := i + 1; j < len(top); j++ {
			ai, aj := act[top[i].ID], act[top[j].ID]
			corr, _ := stats.Correlate(ai, aj)
			if len(corr) == 0 {
				continue
			}
			lag0 := corr[nbins-1]
			bestLag, bestVal := offPeak(corr, nbins, nbins/4)

			kind, score := "", 0.0
			switch {
			case lag0 >= coactiveThresh:
				kind, score = "co-active", lag0
			case bestVal >= rrThresh && bestLag != 0 && lag0 < 0.3:
				kind, score = "request-response", bestVal
			}
			if kind == "" {
				continue
			}
			nextID++
			convs = append(convs, Conversation{
				ID: nextID, EmitterIDs: []uint64{top[i].ID, top[j].ID}, Kind: kind,
				Correlation: score, Exchanges: overlapCount(ai, aj),
			})
		}
	}
	sc.Conversations = convs
}

// offPeak returns the lag (≠0, within ±maxLag) of the largest cross-correlation
// value and that value. corr is the full Correlate output of two length-n
// signals (length 2n-1, lag 0 at index n-1).
func offPeak(corr []float64, n, maxLag int) (lag int, val float64) {
	val = -2
	for d := -maxLag; d <= maxLag; d++ {
		if d == 0 {
			continue
		}
		idx := n - 1 + d
		if idx < 0 || idx >= len(corr) {
			continue
		}
		if corr[idx] > val {
			val, lag = corr[idx], d
		}
	}
	return lag, val
}

// overlapCount returns how many bins both signals are active.
func overlapCount(a, b []float64) int {
	n := 0
	for i := range a {
		if a[i] > 0 && b[i] > 0 {
			n++
		}
	}
	return n
}

// clusterEmitters groups bursts into emitters and sets each burst's EmitterID.
// Stage A clusters by exact channel + class family (per-channel emitters, the
// common case). Stage B conservatively merges several single-channel emitters
// into one frequency-hopping emitter when they share a fingerprint and never
// transmit at the same time.
func clusterEmitters(sc *Scene) {
	// Stage A: (FreqHz, classFamily) → emitter.
	type key struct {
		freq   uint32
		family string
	}
	groups := map[key][]int{}
	var order []key
	for i, b := range sc.Bursts {
		k := key{b.FreqHz, classFamily(b.Class)}
		if _, seen := groups[k]; !seen {
			order = append(order, k)
		}
		groups[k] = append(groups[k], i)
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].freq != order[j].freq {
			return order[i].freq < order[j].freq
		}
		return order[i].family < order[j].family
	})

	var emitters []Emitter
	for _, k := range order {
		emitters = append(emitters, buildEmitter(sc, groups[k]))
	}

	emitters = mergeHoppers(sc, emitters)

	// Assign IDs (stable: by first-seen time, then frequency) and back-reference.
	sort.Slice(emitters, func(i, j int) bool {
		if emitters[i].FirstSec != emitters[j].FirstSec {
			return emitters[i].FirstSec < emitters[j].FirstSec
		}
		return emitters[i].Fingerprint.CenterHz < emitters[j].Fingerprint.CenterHz
	})
	for i := range emitters {
		emitters[i].ID = uint64(i + 1)
		for _, bid := range emitters[i].BurstIDs {
			sc.Bursts[burstIndexByID(sc, bid)].EmitterID = emitters[i].ID
		}
	}
	// Present top talkers first (most airtime), keeping the assigned IDs.
	sort.Slice(emitters, func(i, j int) bool {
		if emitters[i].TotalAirtimeSec != emitters[j].TotalAirtimeSec {
			return emitters[i].TotalAirtimeSec > emitters[j].TotalAirtimeSec
		}
		return emitters[i].ID < emitters[j].ID
	})
	sc.Emitters = emitters
}

// buildEmitter constructs an emitter from a set of burst indices.
func buildEmitter(sc *Scene, idxs []int) Emitter {
	var e Emitter
	e.FirstSec = sc.Bursts[idxs[0]].StartSec
	e.LastSec = sc.Bursts[idxs[0]].EndSec
	var powers, lens, flats []float64
	freqSet := map[uint32]struct{}{}
	for _, i := range idxs {
		b := sc.Bursts[i]
		e.BurstIDs = append(e.BurstIDs, b.ID)
		e.TotalAirtimeSec += b.DurationSec()
		if b.StartSec < e.FirstSec {
			e.FirstSec = b.StartSec
		}
		if b.EndSec > e.LastSec {
			e.LastSec = b.EndSec
		}
		powers = append(powers, float64(b.PeakDbFS))
		lens = append(lens, b.DurationSec())
		flats = append(flats, b.SpectralFlatness)
		freqSet[b.FreqHz] = struct{}{}
	}
	b0 := sc.Bursts[idxs[0]]
	e.Fingerprint = Fingerprint{
		CenterHz:          b0.FreqHz,
		OccupiedBwHz:      b0.OccupiedBwHz,
		ClassFamily:       classFamily(b0.Class),
		MedianPowerDbFS:   float32(median(powers)),
		MedianBurstLenSec: median(lens),
		SpectralFlatness:  median(flats),
	}
	freqs := make([]uint32, 0, len(freqSet))
	for f := range freqSet {
		freqs = append(freqs, f)
	}
	sort.Slice(freqs, func(i, j int) bool { return freqs[i] < freqs[j] })
	if len(freqs) > 1 {
		e.HopSet = freqs
	}
	e.Label = emitterLabel(e)
	return e
}

// mergeHoppers collapses several single-frequency emitters into one hopping
// emitter when they share a class family and power band, span ≥2 frequencies,
// and never transmit simultaneously (a single radio cannot key two channels at
// once). It is deliberately conservative to avoid merging distinct co-channel
// radios.
func mergeHoppers(sc *Scene, emitters []Emitter) []Emitter {
	type hkey struct {
		family string
		powB   int
	}
	buckets := map[hkey][]int{}
	for i, e := range emitters {
		if len(e.HopSet) > 1 {
			continue // already multi-freq
		}
		k := hkey{e.Fingerprint.ClassFamily, int(float64(e.Fingerprint.MedianPowerDbFS) / hopperPowerBucketDb)}
		buckets[k] = append(buckets[k], i)
	}

	merged := map[int]bool{}
	var out []Emitter
	// Build merged hoppers first.
	keys := make([]hkey, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].family != keys[j].family {
			return keys[i].family < keys[j].family
		}
		return keys[i].powB < keys[j].powB
	})
	for _, k := range keys {
		members := buckets[k]
		if len(members) < 2 {
			continue
		}
		// Distinct frequencies and pairwise time-disjoint bursts.
		freqs := map[uint32]struct{}{}
		for _, mi := range members {
			freqs[emitters[mi].Fingerprint.CenterHz] = struct{}{}
		}
		if len(freqs) < 2 {
			continue
		}
		if !timeDisjoint(sc, emitters, members) {
			continue
		}
		// Merge.
		var idxs []int
		for _, mi := range members {
			merged[mi] = true
			for _, bid := range emitters[mi].BurstIDs {
				idxs = append(idxs, burstIndexByID(sc, bid))
			}
		}
		he := buildEmitter(sc, idxs)
		out = append(out, he)
	}
	// Keep unmerged emitters.
	for i, e := range emitters {
		if !merged[i] {
			out = append(out, e)
		}
	}
	return out
}

// timeDisjoint reports whether the bursts of all listed emitters are pairwise
// non-overlapping in time (a precondition for them being one hopping radio).
func timeDisjoint(sc *Scene, emitters []Emitter, members []int) bool {
	type iv struct{ start, end float64 }
	var ivs []iv
	for _, mi := range members {
		for _, bid := range emitters[mi].BurstIDs {
			b := sc.Bursts[burstIndexByID(sc, bid)]
			ivs = append(ivs, iv{b.StartSec, b.EndSec})
		}
	}
	sort.Slice(ivs, func(i, j int) bool { return ivs[i].start < ivs[j].start })
	for i := 1; i < len(ivs); i++ {
		if ivs[i].start < ivs[i-1].end {
			return false
		}
	}
	return true
}

// burstIndexByID maps a burst ID to its slice index (linear; burst counts are
// modest, and this keeps EmitterID back-references simple).
func burstIndexByID(sc *Scene, id uint64) int {
	for i := range sc.Bursts {
		if sc.Bursts[i].ID == id {
			return i
		}
	}
	return 0
}

// classFamily folds the modulation classes into the coarse families emitters are
// clustered on, so a hopper whose hops are classified inconsistently (c4fm on
// one, fsk on the next) still collapses to one emitter.
func classFamily(c survey.SignalClass) string {
	switch c {
	case survey.ClassAM:
		return "am"
	case survey.ClassNBFM, survey.ClassWideFM:
		return "analog-fm"
	case survey.ClassFSK, survey.ClassC4FM, survey.ClassPSK, survey.ClassContinuousData,
		survey.ClassPaging, survey.ClassTrunkControl, survey.ClassTrunkVoice:
		return "digital"
	default:
		return "unknown"
	}
}

// emitterLabel renders a short human label for an emitter.
func emitterLabel(e Emitter) string {
	if len(e.HopSet) > 1 {
		return fmt.Sprintf("%s hopper (%d ch)", e.Fingerprint.ClassFamily, len(e.HopSet))
	}
	return fmt.Sprintf("%s@%.4fMHz", e.Fingerprint.ClassFamily, float64(e.Fingerprint.CenterHz)/1e6)
}
