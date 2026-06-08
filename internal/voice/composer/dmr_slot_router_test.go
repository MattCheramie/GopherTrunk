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

func TestSlotRouterDifferentTalkgroupNeverBinds(t *testing.T) {
	r := newSlotRouter(100)
	// Only ever see the other slot's talkgroup → nothing is accepted and
	// no phase is bound (so a later LC-less frame still can't leak).
	for _, ph := range []uint8{0, 1, 0, 1} {
		if r.accept(sfWithLC(ph, 200)) {
			t.Fatalf("accepted foreign talkgroup on phase %d", ph)
		}
	}
	if r.accept(sfNoLC(0)) || r.accept(sfNoLC(1)) {
		t.Error("accepted an LC-less superframe though our talkgroup was never seen")
	}
}
