package trunking

import (
	"testing"
	"time"
)

// noFireAfterFunc replaces the engine's hold timer with one that never fires
// during a test (a 1 h timer), so the wakeup-page hold window is driven
// deterministically by the test rather than wall-clock. Stop still works, so the
// supersede path (dropHeldIndividual) exercises timer cancellation for real.
func noFireAfterFunc(d time.Duration, f func()) *time.Timer {
	return time.AfterFunc(time.Hour, f)
}

// TestEngineTETRAWakeupPageHeldUntilGroupGrant is the regression for the
// Energy-Economy wakeup-page bug (Discord report + attached bug report): the SwMI
// sends an individual-addressed page (individual=true, no source, dst = the
// calling party's radio SSI) ~160 ms before the authoritative group grant.
// Spawning the page immediately created a ghost call + a WAV directory named
// after the radio ID, torn down when the group grant superseded it. The page must
// instead be held so no ghost is ever created, and the group grant then starts
// the one true call under the GSSI.
func TestEngineTETRAWakeupPageHeldUntilGroupGrant(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	const (
		freq = uint32(467_912_500)
		ts   = uint8(2)
		issi = uint32(1005750) // paged radio ID (leaks as a phantom talkgroup today)
		gssi = uint32(1020545) // the real talkgroup
	)
	// The wakeup page. Before the fix this spawns a ghost call under the radio SSI.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: ts, Individual: true, TETRAUsageMarker: 8})
	if n := len(pool.Active()); n != 0 {
		t.Fatalf("active calls after wakeup page = %d, want 0 (the page must be held, not spawned)", n)
	}

	// The authoritative group grant lands within the hold window: it cancels the
	// held page and starts the one real call under the GSSI.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: gssi, SourceID: issi, FrequencyHz: freq, Timeslot: ts, TETRAUsageMarker: 23})
	act := pool.Active()
	if len(act) != 1 {
		t.Fatalf("active calls = %d, want 1 (only the authoritative group call)", len(act))
	}
	if got := act[0].Grant.GroupID; got != gssi {
		t.Errorf("call talkgroup = %d, want %d (no ghost under the radio SSI)", got, gssi)
	}
	if got := act[0].Grant.SourceID; got != issi {
		t.Errorf("call source = %d, want %d", got, issi)
	}
}

// TestEngineTETRAWakeupPageFlushesWithoutGroupGrant guards that a genuine
// unit-to-unit individual call — an individual page that no group grant ever
// supersedes — still records once the hold window expires, so the hold does not
// silently drop private calls.
func TestEngineTETRAWakeupPageFlushesWithoutGroupGrant(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	const (
		freq = uint32(467_912_500)
		ts   = uint8(1)
		issi = uint32(1005750) // the individually-called radio
	)
	page := Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: ts, Individual: true}
	e.HandleGrant(page)
	if n := len(pool.Active()); n != 0 {
		t.Fatalf("active calls after page = %d, want 0 (held)", n)
	}

	// The hold window expires with no group grant: the flush allocates the call.
	e.flushHeldGrant(channelKeyOf(page))
	act := pool.Active()
	if len(act) != 1 {
		t.Fatalf("active calls after flush = %d, want 1 (unit-to-unit call must record)", len(act))
	}
	if got := act[0].Grant.GroupID; got != issi {
		t.Errorf("flushed call destination = %d, want %d", got, issi)
	}
}

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

// TestEngineTETRANotificationSuperseded is the regression for the group
// notification vs channel grant bug (Discord report, log 01753d5b): a call bound
// from a provisional notification grant — labelled with the calling party's own
// SSI (the temporal race: the party is not yet known as a radio ID), no source,
// and the notification's usage marker (8) — must yield to the authoritative group
// grant for the same physical channel, which carries the real GSSI, the calling
// party as source, and the traffic slot's usage marker (23). Before the supersede
// the #915 channel backfill folded the group grant on as a source update, so the
// call stayed on the wrong talkgroup and followed marker 8, starving the vocoder.
func TestEngineTETRANotificationSuperseded(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()

	const (
		freq = uint32(467_912_500)
		ts   = uint8(2)
		issi = uint32(1005724) // calling party radio ID (also the notification address)
		gssi = uint32(1020543) // the real talkgroup
	)
	// Grant 1: the provisional notification — addressed to the individual's SSI
	// (surfaced as a phantom talkgroup because the ISSI is not yet known), no
	// source, the notification's usage marker.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: ts, TETRAUsageMarker: 8})
	// Grant 2: the authoritative group grant — real GSSI, the calling party as
	// source, and the traffic channel's downlink usage marker.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: gssi, SourceID: issi, FrequencyHz: freq, Timeslot: ts, TETRAUsageMarker: 23})

	act := pool.Active()
	if len(act) != 1 {
		t.Fatalf("active calls = %d, want 1", len(act))
	}
	if got := act[0].Grant.GroupID; got != gssi {
		t.Errorf("call talkgroup = %d, want %d (the authoritative GSSI must supersede the notification)", got, gssi)
	}
	if got := act[0].Grant.TETRAUsageMarker; got != 23 {
		t.Errorf("call usage marker = %d, want 23 (the traffic marker must supersede the notification marker)", got)
	}
	if got := act[0].Grant.SourceID; got != issi {
		t.Errorf("call source = %d, want %d", got, issi)
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
