package siglab

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestComputeSyncLandscape plants a known dibit sync word at a fixed cadence
// and asserts the landscape recovers the winning variant/rotation, the hit
// count, and the modal inter-hit spacing.
func TestComputeSyncLandscape(t *testing.T) {
	sync := []uint8{0, 1, 2, 3, 3, 2, 1, 0}
	const (
		gap  = 40 // dibits of filler between syncs
		hits = 25
	)
	var stream []uint8
	for h := 0; h < hits; h++ {
		stream = append(stream, sync...)
		for i := 0; i < gap; i++ {
			stream = append(stream, uint8((h*7+i)&3))
		}
	}

	land := computeSyncLandscape(stream, oneVariant("fsw", sync), 4, 2)
	if land == nil {
		t.Fatal("nil landscape")
	}
	if land.WinnerRotation != 0 {
		t.Errorf("winner rotation = %d, want 0", land.WinnerRotation)
	}
	if land.WinnerHits < hits {
		t.Errorf("winner hits = %d, want >= %d", land.WinnerHits, hits)
	}
	if want := len(sync) + gap; land.ModalSpacing != want {
		t.Errorf("modal spacing = %d, want %d", land.ModalSpacing, want)
	}
}

// TestComputeSyncLandscapeDetectsPolarityFlip confirms a discriminator-polarity
// inverted dibit stream (sym -> sym+2) is recovered at rotation 2.
func TestComputeSyncLandscapeDetectsPolarityFlip(t *testing.T) {
	// Non-symmetric sync (not invariant under any rotation) + constant filler
	// (which only matches an all-equal sync), so the only hits come from the
	// planted, polarity-flipped sync at rotation 2.
	sync := []uint8{0, 3, 1, 2, 3, 0, 2, 1}
	var stream []uint8
	for h := 0; h < 20; h++ {
		for _, s := range sync {
			stream = append(stream, (s+2)&3) // discriminator-polarity flip
		}
		for i := 0; i < 30; i++ {
			stream = append(stream, 0)
		}
	}
	land := computeSyncLandscape(stream, oneVariant("fsw", sync), 4, 1)
	if land == nil || land.WinnerRotation != 2 {
		t.Fatalf("expected winner rotation 2 (polarity flip), got %+v", land)
	}
}

// TestBuildProtocolDetailHistogram checks the symbol histogram + spec lookup
// for a bit protocol and a dibit protocol.
func TestBuildProtocolDetailHistogram(t *testing.T) {
	// Bit protocol (cardinality 2).
	bits := []uint8{0, 1, 0, 1, 1, 1}
	d, ok := buildProtocolDetail(trunking.ProtocolEDACS, bits, true, trunking.System{}).(*ProtocolDetail)
	if !ok || d == nil {
		t.Fatal("EDACS detail not built")
	}
	if d.SymbolCardinality != 2 || len(d.SymbolHistogram) != 2 {
		t.Errorf("cardinality/histogram wrong: %+v", d)
	}
	if d.SymbolHistogram[0] != 2 || d.SymbolHistogram[1] != 4 {
		t.Errorf("histogram = %v, want [2 4]", d.SymbolHistogram)
	}

	// Unregistered protocol (P25 P1 uses the deep path, not a spec) → nil.
	if got := buildProtocolDetail(trunking.ProtocolP25, []uint8{0, 1, 2, 3}, false, trunking.System{}); got != nil {
		t.Errorf("expected nil for P25 (no spec), got %T", got)
	}
}

// TestDetailSpecsCoverAllNonP25 asserts every registered protocol except P25
// Phase 1 has a detail spec, so the deep dive is uniform.
func TestDetailSpecsCoverAllNonP25(t *testing.T) {
	for _, p := range []trunking.Protocol{
		trunking.ProtocolDMR, trunking.ProtocolDMRTier2, trunking.ProtocolP25Phase2,
		trunking.ProtocolNXDN, trunking.ProtocolDPMR, trunking.ProtocolYSF,
		trunking.ProtocolTETRA, trunking.ProtocolEDACS, trunking.ProtocolMotorola,
		trunking.ProtocolLTR, trunking.ProtocolMPT1327, trunking.ProtocolDStar,
	} {
		if !hasDetailSpec(p) {
			t.Errorf("no detail spec for %s", p)
		}
	}
}
