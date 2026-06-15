package hunt

import "github.com/MattCheramie/GopherTrunk/internal/carriers"

// inferGrid + snapHz wrap the channel-grid helpers in internal/carriers so the
// sweeper (snapCandidatesToGrid) keeps calling them with hunt's Candidate type
// while the implementation lives in the neutral carriers package, shared with
// the siglab wideband survey.

func inferGrid(cands []Candidate) (stepHz, phaseHz uint32) {
	freqs := make([]uint32, len(cands))
	for i, c := range cands {
		freqs[i] = c.FreqHz
	}
	return carriers.InferGrid(freqs)
}

func snapHz(f, step, phase, maxCorrectionHz uint32) uint32 {
	return carriers.SnapHz(f, step, phase, maxCorrectionHz)
}
