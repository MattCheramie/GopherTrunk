package siglab

import "github.com/MattCheramie/GopherTrunk/internal/dsp/stats"

// SyncVariant is one named candidate sync pattern (e.g. one of DMR's 9 ETSI
// sync words, or a protocol's single FSW), as a slice of symbols (dibits in
// 0..3, or bits in 0..1).
type SyncVariant struct {
	Name    string  `json:"name" yaml:"name"`
	Pattern []uint8 `json:"-" yaml:"-"`
}

// SyncStat is the best correlation of one (variant, rotation) against the
// recovered-symbol stream. rotation is the cyclic symbol offset applied to the
// stream before comparison: for 4-level streams rotation 2 is the
// discriminator-polarity flip and 1/3 the I/Q-swap rotations; for 2-level
// streams rotation 1 is bit inversion.
type SyncStat struct {
	Variant  string `json:"variant" yaml:"variant"`
	Rotation int    `json:"rotation" yaml:"rotation"`
	BestDist int    `json:"best_dist" yaml:"best_dist"`
	BestPos  int    `json:"best_pos" yaml:"best_pos"`
	Hits     int    `json:"hits" yaml:"hits"`
}

// SyncLandscape summarizes sync-word correlation across a recovered-symbol
// stream — the generalization of P25's FSW-correlation landscape. It answers
// "does the demod produce canonical symbols, which sync/polarity aligns the
// stream, and at what cadence" for any protocol.
type SyncLandscape struct {
	SyncLen        int        `json:"sync_len" yaml:"sync_len"`
	Tolerance      int        `json:"tolerance" yaml:"tolerance"`
	Stats          []SyncStat `json:"stats" yaml:"stats"`
	WinnerVariant  string     `json:"winner_variant" yaml:"winner_variant"`
	WinnerRotation int        `json:"winner_rotation" yaml:"winner_rotation"`
	WinnerHits     int        `json:"winner_hits" yaml:"winner_hits"`
	// ModalSpacing is the most common symbol delta between consecutive clean
	// hits under the winner — a real control channel emits sync at a fixed
	// frame interval, so a single dominant spacing means genuine frames while
	// scattered spacings mean false positives.
	ModalSpacing int `json:"modal_spacing" yaml:"modal_spacing"`
	// positions holds the clean-hit positions under the winning (variant,
	// rotation) for downstream FEC re-runs. Not serialized.
	positions []int
}

// computeSyncLandscape correlates each variant against stream under every
// cyclic rotation of the given symbol cardinality (2 or 4), within tolerance.
// It returns the per-(variant,rotation) stats, picks the winner by hit count,
// and records the winner's clean-hit positions + modal spacing.
func computeSyncLandscape(stream []uint8, variants []SyncVariant, cardinality, tolerance int) *SyncLandscape {
	if len(variants) == 0 || cardinality < 2 {
		return nil
	}
	// Variants may differ in length (e.g. TETRA's 11/15/19-dibit training
	// sequences). Each is correlated at its own length; SyncLen reports the
	// winning variant's length.
	mask := uint8(cardinality - 1)
	land := &SyncLandscape{SyncLen: len(variants[0].Pattern), Tolerance: tolerance}

	bestHits, bestDist := -1, 1<<30
	var winLen int
	for _, v := range variants {
		syncLen := len(v.Pattern)
		if syncLen == 0 || len(stream) < syncLen {
			continue
		}
		for rot := 0; rot < cardinality; rot++ {
			st := SyncStat{Variant: v.Name, Rotation: rot, BestDist: syncLen + 1}
			for pos := 0; pos+syncLen <= len(stream); pos++ {
				mismatch := 0
				for k := 0; k < syncLen; k++ {
					if (stream[pos+k]+uint8(rot))&mask != v.Pattern[k] {
						mismatch++
						if mismatch > tolerance && mismatch >= st.BestDist {
							break
						}
					}
				}
				if mismatch <= tolerance {
					st.Hits++
				}
				if mismatch < st.BestDist {
					st.BestDist = mismatch
					st.BestPos = pos
				}
			}
			land.Stats = append(land.Stats, st)
			// Winner = most hits; ties break on lower best distance.
			if st.Hits > bestHits || (st.Hits == bestHits && st.BestDist < bestDist) {
				bestHits, bestDist = st.Hits, st.BestDist
				land.WinnerVariant, land.WinnerRotation, land.WinnerHits = st.Variant, st.Rotation, st.Hits
				winLen = syncLen
			}
		}
	}

	// Recover the winner's clean-hit positions + modal inter-hit spacing.
	if land.WinnerHits > 0 {
		win := variants[0]
		for _, v := range variants {
			if v.Name == land.WinnerVariant {
				win = v
				break
			}
		}
		land.SyncLen = winLen
		rot := uint8(land.WinnerRotation)
		for pos := 0; pos+winLen <= len(stream); pos++ {
			mismatch := 0
			for k := 0; k < winLen; k++ {
				if (stream[pos+k]+rot)&mask != win.Pattern[k] {
					mismatch++
				}
			}
			if mismatch <= tolerance {
				land.positions = append(land.positions, pos)
			}
		}
		land.ModalSpacing = modalSpacing(land.positions)
	}
	return land
}

// modalSpacing returns the most-frequent delta between consecutive positions
// (0 when fewer than two positions). The mode itself is computed by
// stats.ModeInt, which breaks ties toward the smaller spacing.
func modalSpacing(positions []int) int {
	if len(positions) < 2 {
		return 0
	}
	deltas := make([]int, 0, len(positions)-1)
	for i := 1; i < len(positions); i++ {
		deltas = append(deltas, positions[i]-positions[i-1])
	}
	return stats.ModeInt(deltas)
}

// dibitVariants / bitVariants wrap fixed patterns as single-variant slices.
func oneVariant(name string, pattern []uint8) []SyncVariant {
	return []SyncVariant{{Name: name, Pattern: pattern}}
}
