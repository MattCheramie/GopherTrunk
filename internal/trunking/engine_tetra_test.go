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

// TestEngineTETRANotificationHeldUntilGroupGrant is the regression for the
// source-less notification bug (Discord reports + attached bug report): the SwMI
// sends a notification (no source, dst = the calling party's radio SSI) 50-400 ms
// before the authoritative group grant. Spawning it immediately created a ghost
// call + a WAV directory named after the radio ID, torn down when the group grant
// superseded it. Crucially the FIRST notification for a radio is published
// individual=FALSE (classifyParties only learns the SSI is a party later), so the
// hold must trigger on SourceID==0, NOT the Individual flag.
func TestEngineTETRANotificationHeldUntilGroupGrant(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	const (
		freq = uint32(467_912_500)
		ts   = uint8(2)
		issi = uint32(1005750) // paged radio ID (leaks as a phantom talkgroup today)
		gssi = uint32(1020545) // the real talkgroup
	)
	// The notification — individual=FALSE (the common first-sighting case), no
	// source. Before the fix this spawns a ghost call under the radio SSI.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: ts, TETRAUsageMarker: 8})
	if n := len(pool.Active()); n != 0 {
		t.Fatalf("active calls after notification = %d, want 0 (must be held, not spawned)", n)
	}

	// The authoritative group grant lands within the hold window: it cancels the
	// held notification and starts the one real call under the GSSI.
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

// TestEngineTETRANotificationDoesNotDiscoverTalkgroup is the regression for the
// residual RadioID→TGID leak seen in the reporter's captures (e.g. tg=1009311):
// a source-less TETRA notification grant addresses the paged radio's SSI, not a
// talkgroup, yet the discovery gate catalogued its GroupID as a "Discovered"
// talkgroup — a phantom that lingered in the /#/talkgroups list unless the SSI
// was later seen as a source. Discovery ran before the notification hold, so it
// leaked even for a notification that never spawned a call (held, folded as a
// repeat on an active channel, or dropped). A source-less notification must
// never self-discover a talkgroup; only a source-bearing group grant does.
func TestEngineTETRANotificationDoesNotDiscoverTalkgroup(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	const (
		freq = uint32(467_912_500)
		issi = uint32(1009311) // paged radio SSI — the leaked phantom in the capture
		gssi = uint32(1020539) // the real talkgroup
	)
	// A bare source-less notification for the radio SSI must NOT be catalogued.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: 1})
	if tg := e.talkgroups.Lookup(issi); tg != nil {
		t.Fatalf("source-less notification catalogued radio SSI %d as a talkgroup: %+v", issi, tg)
	}
	// The authoritative source-bearing group grant still discovers the real TG.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: gssi, SourceID: issi, FrequencyHz: freq, Timeslot: 1})
	if tg := e.talkgroups.Lookup(gssi); tg == nil || tg.Tag != discoveredTag {
		t.Errorf("authoritative group grant should have discovered TG %d, got %+v", gssi, tg)
	}
	// And the notification's SSI is still not a talkgroup afterwards.
	if tg := e.talkgroups.Lookup(issi); tg != nil {
		t.Errorf("radio SSI %d must not appear as a talkgroup, got %+v", issi, tg)
	}
}

// TestEngineTETRANotificationToKnownRadioDropped guards that a held notification
// targeting a KNOWN radio SSI that no group grant ever supersedes is DROPPED at
// flush, not recorded under the radio ID — recording under a radio ID is exactly
// the leak the hold exists to prevent.
func TestEngineTETRANotificationToKnownRadioDropped(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	const (
		freq = uint32(467_912_500)
		ts   = uint8(1)
		issi = uint32(1005750) // a radio the engine has already learned
	)
	e.noteRadio(issi, "X") // learned from an earlier call's source
	page := Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: ts}
	e.HandleGrant(page)
	if n := len(pool.Active()); n != 0 {
		t.Fatalf("active calls after notification = %d, want 0 (held)", n)
	}

	// Flush with no group grant: destination is a known radio → dropped, not recorded.
	e.flushHeldGrant(channelKeyOf(page))
	if n := len(pool.Active()); n != 0 {
		t.Fatalf("active calls after flush = %d, want 0 (notification to a known radio must not record)", n)
	}
}

// TestEngineTETRANotificationToTalkgroupFlushes guards the legitimate case: a
// source-less notification addressed to a real talkgroup (not a known radio) that
// no group grant supersedes still records under that talkgroup once the window
// expires — the hold must not silently drop a real (if unattributed) group call.
func TestEngineTETRANotificationToTalkgroupFlushes(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	const (
		freq = uint32(467_912_500)
		ts   = uint8(1)
		gssi = uint32(1020545) // a talkgroup — never learned as a radio
	)
	page := Grant{System: "X", Protocol: "tetra", GroupID: gssi, FrequencyHz: freq, Timeslot: ts}
	e.HandleGrant(page)
	if n := len(pool.Active()); n != 0 {
		t.Fatalf("active calls after notification = %d, want 0 (held)", n)
	}

	e.flushHeldGrant(channelKeyOf(page))
	act := pool.Active()
	if len(act) != 1 {
		t.Fatalf("active calls after flush = %d, want 1 (talkgroup call must record)", len(act))
	}
	if got := act[0].Grant.GroupID; got != gssi {
		t.Errorf("flushed call talkgroup = %d, want %d", got, gssi)
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

	// Authoritative group grants (source-bearing) on distinct timeslots: both spawn
	// immediately and must not fold into one.
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: 100, SourceID: 501, FrequencyHz: 467_912_500, Timeslot: 1})
	e.HandleGrant(Grant{System: "X", Protocol: "tetra", GroupID: 200, SourceID: 502, FrequencyHz: 467_912_500, Timeslot: 2})

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
