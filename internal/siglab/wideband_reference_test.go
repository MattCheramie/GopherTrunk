package siglab

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab/sigref"
)

// TestKeepAsReference covers the wideband survey's decision to keep and name a
// non-decoding carrier from the reference DB: a confident reference match is
// kept, a weak or absent one is dropped. This is the wiring that turns the
// per-carrier identify fallback (blind estimate → sigref.Rank, proven in the
// identify tests) into a surfaced "looks like TETRA…" carrier.
func TestKeepAsReference(t *testing.T) {
	strong := CarrierResult{ReferenceMatches: []sigref.Match{
		{Entry: sigref.Entry{DisplayName: "TETRA"}, Score: 0.72, Why: "25 kHz, π/4-DQPSK, 18000 sym/s"},
	}}
	weak := CarrierResult{ReferenceMatches: []sigref.Match{
		{Entry: sigref.Entry{DisplayName: "FLEX"}, Score: 0.20},
	}}
	none := CarrierResult{}

	if !keepAsReference(strong) {
		t.Error("strong reference match should be kept")
	}
	if keepAsReference(weak) {
		t.Error("below-floor reference match should be dropped")
	}
	if keepAsReference(none) {
		t.Error("carrier with no reference matches should be dropped")
	}
	// A carrier that already locked is never routed here (keepCarrier handles
	// it), but the reference predicate must not depend on lock state.
	locked := strong
	locked.Locked = true
	if !keepAsReference(locked) {
		t.Error("reference predicate must judge by score, not lock state")
	}
}
