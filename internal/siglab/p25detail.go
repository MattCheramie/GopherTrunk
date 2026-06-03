package siglab

import p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"

// P25P1Detail is the P25-Phase-1-specific demod deep-dive, surfaced in
// Result.P25P1 when the protocol is P25 Phase 1 and CollectIQDiag is set. It
// reuses the dibit-domain analysis the historical iqdiag report pioneered
// (issue #275): a per-rotation Frame-Sync-Word correlation landscape that
// tells whether the demod produces canonical dibits at all and which
// rotation aligns the stream, plus NID decode attempts at the cleanest FSW
// hits. It operates purely on the recovered dibit stream (no soft samples),
// so it is available through the generic SymbolTap for every P25 P1 capture.
type P25P1Detail struct {
	DibitsBuffered  int             `json:"dibits_buffered" yaml:"dibits_buffered"`
	DibitHistogram  [4]int64        `json:"dibit_histogram" yaml:"dibit_histogram"`
	Rotations       [4]RotationStat `json:"rotations" yaml:"rotations"`
	WinningRotation int             `json:"winning_rotation" yaml:"winning_rotation"`
	WinningHits     int             `json:"winning_hits" yaml:"winning_hits"`
	// NIDDecodes are the NAC/DUID results from BCH-decoding the NID at the
	// cleanest (distance-0) FSW hits under the winning rotation. A mix of
	// successes and failures points at signal quality; zero successes
	// despite perfect FSW alignment points at framing.
	NIDDecodes []NIDDecode `json:"nid_decodes" yaml:"nid_decodes"`
}

// RotationStat summarizes the FSW Hamming-distance landscape for one of the
// four cyclic dibit rotations.
type RotationStat struct {
	BestDist int `json:"best_dist" yaml:"best_dist"`
	BestPos  int `json:"best_pos" yaml:"best_pos"`
	Hits     int `json:"hits" yaml:"hits"` // positions with distance ≤ tolerance
}

// NIDDecode is one NID BCH decode attempt at an FSW hit.
type NIDDecode struct {
	Pos  int    `json:"pos" yaml:"pos"`
	Errs int    `json:"errs" yaml:"errs"`
	NAC  uint16 `json:"nac" yaml:"nac"`
	DUID uint8  `json:"duid" yaml:"duid"`
	OK   bool   `json:"ok" yaml:"ok"`
}

const (
	p25FSWLen     = 24
	p25FSWTol     = 4
	maxNIDDecodes = 5
)

// buildP25Detail walks a recovered dibit stream and produces the P25 deep
// dive. Returns nil when there are too few dibits to analyze.
func buildP25Detail(dibits []uint8) *P25P1Detail {
	if len(dibits) < p25FSWLen {
		return nil
	}
	d := &P25P1Detail{DibitsBuffered: len(dibits)}
	for _, db := range dibits {
		d.DibitHistogram[db&3]++
	}

	for i := range d.Rotations {
		d.Rotations[i].BestDist = p25FSWLen + 1
	}
	for pos := 0; pos+p25FSWLen <= len(dibits); pos++ {
		for rot := uint8(0); rot < 4; rot++ {
			mismatch := 0
			for kk := 0; kk < p25FSWLen; kk++ {
				if ((dibits[pos+kk] + rot) & 3) != p25phase1.FrameSyncWord[kk] {
					mismatch++
				}
			}
			rs := &d.Rotations[rot]
			if mismatch <= p25FSWTol {
				rs.Hits++
			}
			if mismatch < rs.BestDist {
				rs.BestDist = mismatch
				rs.BestPos = pos
			}
		}
	}

	d.WinningRotation = 0
	for rot := 1; rot < 4; rot++ {
		if d.Rotations[rot].Hits > d.Rotations[d.WinningRotation].Hits {
			d.WinningRotation = rot
		}
	}
	d.WinningHits = d.Rotations[d.WinningRotation].Hits

	// Decode the NID at the cleanest (distance-0) hits under the winner.
	win := uint8(d.WinningRotation)
	for pos := 0; pos+p25FSWLen+32 <= len(dibits) && len(d.NIDDecodes) < maxNIDDecodes; pos++ {
		mismatch := 0
		for kk := 0; kk < p25FSWLen; kk++ {
			if ((dibits[pos+kk] + win) & 3) != p25phase1.FrameSyncWord[kk] {
				mismatch++
			}
		}
		if mismatch != 0 {
			continue
		}
		nid := make([]uint8, 32)
		for j := 0; j < 32; j++ {
			nid[j] = (dibits[pos+p25FSWLen+j] + win) & 3
		}
		n, errs, _, err := p25phase1.NIDFromDibitsWithErrors(nid)
		d.NIDDecodes = append(d.NIDDecodes, NIDDecode{
			Pos:  pos,
			Errs: errs,
			NAC:  n.NAC,
			DUID: uint8(n.DUID),
			OK:   err == nil,
		})
	}
	return d
}
