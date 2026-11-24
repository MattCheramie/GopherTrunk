package phase2

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// A site update that names no network (an RFSS Status Broadcast before the
// Network Status Broadcast) is no seed; a later one that carries no NAC keeps
// the NAC learned before; and with no NAC anywhere there is no seed.
func TestSiteSeeds(t *testing.T) {
	const wacn, sysid, nac = 0xBEE00, 0x164, 0x161
	want := framing.PN44SeedFromIdentity(wacn, sysid, nac)
	var s SiteSeeds
	s.Note(trunking.SiteUpdate{System: "MMR", SystemID: sysid, RFSSID: 1, SiteID: 2})
	if got := s.Seed("MMR", nac); got != 0 {
		t.Fatalf("seed %#x from an update with no WACN", got)
	}
	s.Note(trunking.SiteUpdate{System: "MMR", WACN: wacn, SystemID: sysid})
	if got := s.Seed("MMR", 0); got != 0 {
		t.Fatalf("seed %#x with no NAC from the update or the grant", got)
	}
	if got := s.Seed("MMR", nac); got != want {
		t.Fatalf("seed %#x from the grant's NAC, want %#x", got, want)
	}
	s.Note(trunking.SiteUpdate{System: "MMR", WACN: wacn, SystemID: sysid, NAC: nac})
	s.Note(trunking.SiteUpdate{System: "MMR", WACN: wacn, SystemID: sysid})
	if got := s.Seed("MMR", 0); got != want {
		t.Fatalf("seed %#x after an update without a NAC, want %#x", got, want)
	}
	if got := s.Seed("other", nac); got != 0 {
		t.Fatalf("seed %#x for a system never heard", got)
	}
}

// SoftReliability is the distance of the less certain bit from its boundary.
func TestSoftReliability(t *testing.T) {
	got := SoftReliability([]complex64{complex(0.9, -0.2), complex(-0.1, 0.7), complex(-0.5, -0.6)}, nil)
	want := []float32{0.2, 0.1, 0.5}
	for i := range want {
		if d := got[i] - want[i]; d > 1e-6 || d < -1e-6 {
			t.Fatalf("reliabilities %v, want %v", got, want)
		}
	}
}
