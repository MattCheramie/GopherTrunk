package siglab

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestScoreCandidateRequiresFramePeriodCadence: a short sync (Motorola's is
// 8 bits) collects chance hits in any 2-FSK data, about one per 256 bits. On
// the rfscope entropy fixture (random bits) 13 of them landed 10 bits apart,
// and because 10 ≥ the sync length the cadence gate credited them in full:
// Motorola scored 0.29 on random data and, once NXDN's own false evidence
// stopped narrowing the margin (#1261), identified it at 0.43, above the
// floor. A protocol that declares its frame period only earns cadence credit
// at a multiple of it.
func TestScoreCandidateRequiresFramePeriodCadence(t *testing.T) {
	score := func(modal int) float64 {
		r := &Result{Detail: &ProtocolDetail{Sync: &SyncLandscape{
			SyncLen: 8, WinnerHits: 13, ModalSpacing: modal, FramePeriod: 84,
		}}}
		return scoreCandidate(trunking.ProtocolMotorola, r).Score
	}
	if s := score(10); s > 0.1 {
		t.Errorf("chance hits 10 bits apart score %.2f, want ≤ 0.1", s)
	}
	if s := score(84); s < 0.25 {
		t.Errorf("hits at the 84-bit frame period score %.2f, want the full cadence credit", s)
	}
	if s := score(168); s < 0.25 {
		t.Errorf("hits two frames apart score %.2f, want the full cadence credit", s)
	}
	if detailSpecs[trunking.ProtocolMotorola].framePeriod != 84 {
		t.Error("the Motorola detail spec must declare its 84-bit frame period")
	}
}
