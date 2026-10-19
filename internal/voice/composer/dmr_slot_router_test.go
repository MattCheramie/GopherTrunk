package composer

import (
	"strings"
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

// sfWithAliasLC is a superframe whose embedded LC decoded (CRC-valid) but
// is a talker-alias header: call metadata, naming no destination.
func sfWithAliasLC(phase uint8) dmrvoice.VoiceSuperframe {
	return dmrvoice.VoiceSuperframe{
		Phase:          phase,
		HasLC:          true,
		LC:             dmr.FLC{FLCO: dmr.FLCOTalkerAlias},
		HasTalkerAlias: true,
	}
}

func sfNoLC(phase uint8) dmrvoice.VoiceSuperframe {
	return dmrvoice.VoiceSuperframe{Phase: phase}
}

// sfWithU2ULC is sfWithLC for a unit-to-unit (private) call: the embedded
// LC's destination is the called subscriber, which is the private grant's
// GroupID.
func sfWithU2ULC(phase uint8, dest uint32) dmrvoice.VoiceSuperframe {
	return dmrvoice.VoiceSuperframe{
		Phase: phase,
		HasLC: true,
		LC:    dmr.FLC{FLCO: dmr.FLCOUnitToUnitVoice, DstAddr: dest, SrcAddr: 9},
	}
}

// TestSlotRouterRoutesUnitToUnitLC confirms a private call's embedded LC
// binds the slot by its destination subscriber, exactly like a group LC
// binds by talkgroup — otherwise a private call on an interleaved carrier
// would only ever record via the phase fallback.
func TestSlotRouterRoutesUnitToUnitLC(t *testing.T) {
	r := newSlotRouter(0x00ABCD) // grant GroupID = called subscriber

	if !r.accept(sfWithU2ULC(1, 0x00ABCD)) {
		t.Fatal("rejected our own unit-to-unit LC")
	}
	// The other slot carries a group call for a different TG.
	if r.accept(sfWithLC(0, 200)) {
		t.Error("accepted the other slot's group LC")
	}
	// Bound to phase 1: an LC-less phase-1 superframe is ours.
	if !r.accept(sfNoLC(1)) {
		t.Error("rejected a bound-phase superframe missing its LC")
	}
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

// TestSlotRouterPhaseFallbackRecords: when no embedded LC decodes (a weak
// signal or a late entry), the router must stop dropping the whole call
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

// TestSlotRouterFollowModeBindsFirstCall pins follow mode (groupID 0), the
// router a conventional scan-list DMR channel gets (issue #1261). Its grant's
// GroupID is a synthetic channel ID no on-air LC names, so a talkgroup-bound
// router marked every slot foreign and recorded nothing. Follow mode binds the
// first slot that carries a call, follows it across talkgroups (a channel's
// next over may be another group), and keeps the other slot out.
func TestSlotRouterFollowModeBindsFirstCall(t *testing.T) {
	r := newSlotRouter(0)

	if !r.accept(sfWithLC(1, 11)) {
		t.Fatal("follow mode rejected the first call's LC")
	}
	if r.accept(sfWithLC(0, 22)) {
		t.Error("follow mode accepted the other slot's call")
	}
	if !r.accept(sfNoLC(1)) {
		t.Error("follow mode rejected an LC-less superframe on the bound slot")
	}
	if !r.accept(sfWithLC(1, 33)) {
		t.Error("follow mode dropped a new talkgroup on the bound slot")
	}
	if r.accept(sfNoLC(0)) {
		t.Error("follow mode accepted an LC-less superframe on the other slot")
	}
}

// TestSlotRouterFollowModeLCOverridesFallback: with no LC at first the phase
// fallback binds the active slot, and the first LC to decode re-binds to its
// own slot (it is authoritative, the fallback a guess).
func TestSlotRouterFollowModeLCOverridesFallback(t *testing.T) {
	r := newSlotRouter(0)
	for i := 0; i < unboundPhaseFallbackGrace; i++ {
		r.accept(sfNoLC(0))
	}
	if !r.accept(sfNoLC(0)) {
		t.Fatal("phase fallback did not bind after the grace window")
	}
	if !r.accept(sfWithLC(1, 44)) {
		t.Fatal("an LC did not override the fallback bind")
	}
	if r.accept(sfNoLC(0)) {
		t.Error("still accepting the fallback slot after an LC re-bound to the other")
	}
}

// TestSlotRouterFallbackAfterAliasLCSaysItDecoded: a networked amateur
// repeater's first superframes can carry talker-alias embedded LCs before
// the group voice LC (issue #753). Those decode, so the fallback still binds
// the active slot, but its log must not claim the embedded LC failed to
// decode; that sent the reporter to #644 while their aliases decoded fine.
func TestSlotRouterFallbackAfterAliasLCSaysItDecoded(t *testing.T) {
	r := newSlotRouter(100)
	if r.accept(sfWithAliasLC(0)) {
		t.Fatal("accepted before the grace window elapsed")
	}
	if !r.accept(sfWithAliasLC(0)) {
		t.Fatal("did not fall back to the active phase after alias-only superframes")
	}
	if !r.byFallback || r.metadataLCs != 2 {
		t.Fatalf("byFallback=%v metadataLCs=%d, want true / 2", r.byFallback, r.metadataLCs)
	}
	if msg := r.fallbackLog(); strings.Contains(msg, "no embedded LC decoded") || !strings.Contains(msg, "talker alias") {
		t.Errorf("fallback log %q: want the decoded-but-metadata-only reason", msg)
	}
	// The group LC that follows re-binds by talkgroup.
	if !r.accept(sfWithLC(0, 100)) || r.byFallback {
		t.Error("the call's own LC did not re-bind after the fallback")
	}

	// LC-less superframes keep the decode-failure reason.
	q := newSlotRouter(100)
	q.accept(sfNoLC(1))
	q.accept(sfNoLC(1))
	if msg := q.fallbackLog(); !strings.Contains(msg, "no embedded LC decoded") {
		t.Errorf("fallback log %q after LC-less superframes: want the no-LC reason", msg)
	}

	// Follow mode (a conventional scan-list channel) counts them too.
	f := newSlotRouter(0)
	f.accept(sfWithAliasLC(1))
	if !f.accept(sfWithAliasLC(1)) || f.metadataLCs != 2 {
		t.Errorf("follow mode: metadataLCs=%d, want 2 and a fallback bind", f.metadataLCs)
	}
}
