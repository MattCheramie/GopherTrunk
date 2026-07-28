package trunking

import "testing"

// TestEngineTETRAFoldsGrantsByChannel: a TETRA physical channel + timeslot hosts
// one call, but the SwMI grants it under several SSIs — the calling party's
// individual ISSI (a D-CONNECT) and the group GSSI (a D-SETUP). Those grants
// carry different GroupIDs; without the channel fold, a later group grant whose
// source is already known spawns a duplicate call that collides with the first on
// the shared usage marker. Assert the three grants collapse to one active call.
func TestEngineTETRAFoldsGrantsByChannel(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()

	const (
		freq = uint32(467_912_500)
		ts   = uint8(2)
		gssi = uint32(1020545) // talkgroup
		issi = uint32(1005372) // radio ID
	)
	// D-CONNECT to the individual binds the call; group D-SETUP grants for the same
	// channel + timeslot follow under the GSSI. All one physical call.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: ts, Individual: true})
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: gssi, SourceID: issi, FrequencyHz: freq, Timeslot: ts})
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: gssi, SourceID: issi, FrequencyHz: freq, Timeslot: ts})

	if n := len(pool.Active()); n != 1 {
		t.Fatalf("active calls = %d, want 1 (TETRA grants on one channel+slot must not duplicate)", n)
	}
	if src := pool.Active()[0].Grant.SourceID; src != issi {
		t.Errorf("call source = %d, want %d (backfilled onto the folded call)", src, issi)
	}
}

// TestEngineTETRAConcurrentSlotsStayDistinct guards that the channel fold keys on
// timeslot: two TETRA calls on the same carrier but different timeslots are
// distinct calls and must not be folded.
func TestEngineTETRAConcurrentSlotsStayDistinct(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: 100, FrequencyHz: 467_912_500, Timeslot: 1})
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: 200, FrequencyHz: 467_912_500, Timeslot: 2})

	if n := len(pool.Active()); n != 2 {
		t.Fatalf("active calls = %d, want 2 (distinct timeslots are distinct calls)", n)
	}
}

// TestEngineP25ChannelFoldGated proves the channel fold is TETRA-only: two P25
// grants with different talkgroups on the same frequency are not folded by it
// (P25 compressed/patch grants legitimately reuse a channel under different
// talkgroups, #915), so they remain two calls — the pre-existing behaviour.
func TestEngineP25ChannelFoldGated(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 100, FrequencyHz: 851_000_000})
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 200, FrequencyHz: 851_000_000})

	if n := len(pool.Active()); n != 2 {
		t.Fatalf("active P25 calls = %d, want 2 (the TETRA channel fold must not affect P25)", n)
	}
}
