package trunking

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestEngineDoesNotReclassifyDMRGroupCallAsIndividual pins the DMR half of the
// classification fix. The engine's known-radio → individual reclassification
// rests on the TETRA-only invariant that talkgroup (GSSI) and radio (ISSI) IDs
// never overlap. On DMR they share one 24-bit space, so a fleet talkgroup
// number can equal some subscriber's radio ID. This asserts:
//
//   - a DMR call's source radio ID is NOT learned into knownRadios (the
//     mechanism is scoped to TETRA), and
//   - a DMR group call whose GroupID numerically equals a radio ID already
//     learned from a TETRA system is left classified as a GROUP call — not
//     flipped to individual and mis-filed under individual/<TG>.
//
// The equivalent TETRA case still reclassifies — see
// TestEngineReclassifiesKnownRadioGrantAsIndividual.
func TestEngineDoesNotReclassifyDMRGroupCallAsIndividual(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 2)
	defer bus.Close()

	const sharedID = 1009267

	// Learn sharedID as a subscriber radio via a TETRA group call (source).
	e.HandleGrant(Grant{System: "T", Protocol: "tetra", GroupID: 1020545, SourceID: sharedID,
		FrequencyHz: 467_912_500, Timeslot: 1})
	if !e.isKnownRadio(sharedID) {
		t.Fatalf("precondition: %d should be a known radio after a TETRA call source", sharedID)
	}

	sub := bus.Subscribe()
	defer sub.Close()
	collector := &testBusCollector{}
	done := make(chan struct{})
	go func() { collector.drain(sub, 1, 500*time.Millisecond); close(done) }()

	// A DMR *group* call whose talkgroup happens to equal that radio ID, with a
	// distinct source. It must NOT be reclassified individual.
	e.HandleGrant(Grant{System: "Fire2", Protocol: "dmr-tier2", GroupID: sharedID, SourceID: 13_000_013,
		FrequencyHz: 442_387_500, Timeslot: 1})
	<-done

	var found bool
	for _, ev := range collector.snapshot() {
		if ev.Kind != events.KindCallStart {
			continue
		}
		cs := ev.Payload.(CallStart)
		if cs.Grant.Protocol != "dmr-tier2" || cs.Grant.GroupID != sharedID {
			continue
		}
		found = true
		if cs.Grant.Individual {
			t.Errorf("DMR group call to TG %d was wrongly reclassified individual: %+v", sharedID, cs.Grant)
		}
	}
	if !found {
		t.Fatalf("no CallStart for the DMR group call; events=%+v", collector.snapshot())
	}

	// The DMR call's own source must not have polluted knownRadios.
	if e.isKnownRadio(13_000_013) {
		t.Errorf("DMR source 13000013 must not be learned as a known radio")
	}
}
