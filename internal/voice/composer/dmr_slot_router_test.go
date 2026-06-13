package composer

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
)

func sfWithLC(phase uint8, group uint32) dmrvoice.VoiceSuperframe {
	return dmrvoice.VoiceSuperframe{
		Phase: phase,
		HasLC: true,
		LC:    dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: group, SrcAddr: 7},
	}
}

func sfNoLC(phase uint8) dmrvoice.VoiceSuperframe {
	return dmrvoice.VoiceSuperframe{Phase: phase}
}

func TestSlotRouterRoutesByEmbeddedLC(t *testing.T) {
	r := newSlotRouter(100)

	// Before any LC binds our phase, a phase-only superframe can't be
	// attributed and is dropped.
	if r.accept(sfNoLC(0)) {
		t.Error("accepted an unbound phase-only superframe")
	}
	// Our talkgroup's LC on phase 0 binds and is accepted.
	if !r.accept(sfWithLC(0, 100)) {
		t.Fatal("rejected our own talkgroup's LC")
	}
	// The other slot's LC (different TG, other phase) is dropped.
	if r.accept(sfWithLC(1, 200)) {
		t.Error("accepted the other talkgroup's LC")
	}
	// Once bound, a phase-0 superframe with no/garbled LC is ours.
	if !r.accept(sfNoLC(0)) {
		t.Error("rejected a bound-phase superframe missing its LC")
	}
	// A phase-1 superframe with no LC belongs to the other slot.
	if r.accept(sfNoLC(1)) {
		t.Error("accepted the other phase's LC-less superframe")
	}
}

func TestSlotRouterBothPhasesForeignNeverBinds(t *testing.T) {
	r := newSlotRouter(100)
	// Both phases positively decode as the other talkgroup → neither slot is
	// ours, so the phase fallback must never leak the foreign audio in.
	for _, ph := range []uint8{0, 1, 0, 1} {
		if r.accept(sfWithLC(ph, 200)) {
			t.Fatalf("accepted foreign talkgroup on phase %d", ph)
		}
	}
	for i := 0; i < 4; i++ {
		if r.accept(sfNoLC(0)) || r.accept(sfNoLC(1)) {
			t.Error("fell back onto a phase positively identified as another talkgroup")
		}
	}
}

// TestSlotRouterPhaseFallbackRecords: when no embedded LC ever decodes (the
// real-air capture-pending case), the router must stop dropping the whole call
// and fall back to the active slot's phase after the grace window so audio
// records (issue #644).
func TestSlotRouterPhaseFallbackRecords(t *testing.T) {
	r := newSlotRouter(100)
	// Within the grace window, an LC-less superframe is held (waiting for a
	// possible LC to bind the correct phase).
	for i := 0; i < unboundPhaseFallbackGrace-1; i++ {
		if r.accept(sfNoLC(1)) {
			t.Fatalf("accepted before the grace window elapsed (i=%d)", i)
		}
	}
	// At the grace threshold it falls back to this slot's phase and records.
	if !r.accept(sfNoLC(1)) {
		t.Fatal("did not fall back to the active phase after the grace window")
	}
	if !r.byFallback {
		t.Error("byFallback not set after a phase-fallback bind")
	}
	// Bound to phase 1 now: phase-1 superframes are ours, phase 0 is the other slot.
	if !r.accept(sfNoLC(1)) {
		t.Error("rejected a bound-phase superframe after fallback")
	}
	if r.accept(sfNoLC(0)) {
		t.Error("accepted the other phase after binding by fallback")
	}
}

// TestSlotRouterLCBeatsFallback: a matching-group LC inside the grace window
// binds its phase and the fallback never triggers; a later matching LC on a
// different phase re-binds and clears the fallback flag.
func TestSlotRouterLCBeatsFallback(t *testing.T) {
	r := newSlotRouter(100)
	// One LC-less superframe (still inside the grace window), then our LC.
	if r.accept(sfNoLC(0)) {
		t.Fatal("accepted before any bind")
	}
	if !r.accept(sfWithLC(0, 100)) {
		t.Fatal("rejected our talkgroup's LC")
	}
	if r.byFallback {
		t.Error("byFallback set even though an embedded LC bound the phase")
	}
	// A later matching LC on the other phase re-binds (corrects routing).
	if !r.accept(sfWithLC(1, 100)) {
		t.Fatal("rejected our talkgroup's LC on the other phase")
	}
	if !r.accept(sfNoLC(1)) || r.accept(sfNoLC(0)) {
		t.Error("did not re-bind to the phase named by the newer LC")
	}
}
