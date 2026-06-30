package rfscope

import (
	"context"
	"fmt"
	"sort"
)

func init() { Register(expertAnalyzer{}) }

// Expert-info thresholds.
const (
	intermittentDuty   = 0.05 // duty below this with several bursts ⇒ intermittent
	intermittentBursts = 3
	noiseLikeFlatness  = 0.80  // spectral flatness above this ⇒ noise-like/spread
	wideCarrierHz      = 30000 // wider than the widest common narrowband channel
	narrowCarrierHz    = 5000  // narrower than the tightest standard raster
)

// expertAnalyzer is the RF Expert-Info layer — the analog of Wireshark's Expert
// Information. It runs cheap rules over the populated Scene and flags anything
// notable for an operator: intermittent emitters, frequency hoppers, abnormally
// wide/narrow carriers, noise-like (high-flatness) or high-randomness/encrypted
// carriers, and digital carriers that match no known protocol.
type expertAnalyzer struct{}

func (expertAnalyzer) Name() string        { return "expert" }
func (expertAnalyzer) Synopsis() string    { return "RF expert-info / anomaly flags" }
func (expertAnalyzer) DependsOn() []string { return []string{"topology", "timeline", "entropy"} }

func (expertAnalyzer) Analyze(_ context.Context, sc *Scene, _ *Input) error {
	var out []Anomaly

	// Frequency hoppers (from topology).
	for _, e := range sc.Emitters {
		if len(e.HopSet) > 1 {
			out = append(out, Anomaly{
				Severity: "note", Kind: "hopper", EmitterID: e.ID, FreqHz: e.Fingerprint.CenterHz,
				Detail: fmt.Sprintf("emitter hops %d channels", len(e.HopSet)),
			})
		}
	}

	// Per-channel rules (from timeline + segmentation metrics).
	for _, c := range sc.Channels {
		if c.DutyCycle < intermittentDuty && c.BurstCount >= intermittentBursts {
			out = append(out, Anomaly{
				Severity: "note", Kind: "intermittent", FreqHz: c.FreqHz,
				Detail: fmt.Sprintf("%d bursts at %.1f%% duty", c.BurstCount, c.DutyCycle*100),
			})
		}
		switch {
		case c.OccupiedBwHz > wideCarrierHz:
			out = append(out, Anomaly{Severity: "note", Kind: "wide-carrier", FreqHz: c.FreqHz,
				Detail: fmt.Sprintf("occupied %d Hz, wider than a standard narrowband channel", c.OccupiedBwHz)})
		case c.OccupiedBwHz > 0 && c.OccupiedBwHz < narrowCarrierHz:
			out = append(out, Anomaly{Severity: "note", Kind: "narrow-carrier", FreqHz: c.FreqHz,
				Detail: fmt.Sprintf("occupied %d Hz, below the tightest standard raster", c.OccupiedBwHz)})
		}
		if c.MedianFlatness >= noiseLikeFlatness {
			out = append(out, Anomaly{Severity: "note", Kind: "noise-like", FreqHz: c.FreqHz,
				Detail: fmt.Sprintf("spectral flatness %.2f (spread/encrypted-PHY or noise)", c.MedianFlatness)})
		}
	}

	// Encryption / structure findings (from the entropy triage).
	for _, r := range entropyResults(sc) {
		sev, kind := "note", "unknown-protocol"
		switch r.Class {
		case "strong-encrypted":
			sev, kind = "alert", "encrypted"
		case "repeating-xor", "lfsr-or-keyless-scrambler", "periodic-scrambler", "substitution-or-shift":
			sev, kind = "warn", "obfuscated"
		}
		out = append(out, Anomaly{
			Severity: sev, Kind: kind, EmitterID: r.EmitterID, FreqHz: r.FreqHz,
			Detail: fmt.Sprintf("%s (entropy %.2f) — %s", r.Class, r.EntropyBits, r.Recommended),
		})
	}

	// Deterministic ordering: severity (alert→warn→note), then frequency, kind.
	sort.SliceStable(out, func(i, j int) bool {
		if sevRank(out[i].Severity) != sevRank(out[j].Severity) {
			return sevRank(out[i].Severity) > sevRank(out[j].Severity)
		}
		if out[i].FreqHz != out[j].FreqHz {
			return out[i].FreqHz < out[j].FreqHz
		}
		return out[i].Kind < out[j].Kind
	})

	// Append so a caller that pre-seeded anomalies (e.g. the bridge) is preserved.
	sc.Anomalies = append(sc.Anomalies, out...)
	return nil
}

func sevRank(s string) int {
	switch s {
	case "alert":
		return 3
	case "warn":
		return 2
	default:
		return 1
	}
}
