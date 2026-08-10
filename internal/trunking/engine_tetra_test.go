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

// TestEngineTETRADiscoveryRequiresCorroboration pins the residual RadioID→TGID
// leak fix: a TETRA unit-to-unit call's destination radio that never transmits is
// never revealed as an individual, so a single such grant (Individual=false,
// source-bearing, not a notification) would catalogue the callee ISSI as a phantom
// talkgroup that retraction never cleans up. TETRA discovery is therefore
// corroborated — a first sighting is held pending and only a second catalogues it
// (a real talkgroup recurs; a one-off callee does not). Non-TETRA protocols carry
// explicit individual-call opcodes and are NOT gated, so P25 discovery is
// unchanged. Fail-first: without corroboration the first TETRA grant catalogues
// the SSI and the "must not be catalogued yet" assertion goes red.
func TestEngineTETRADiscoveryRequiresCorroboration(t *testing.T) {
	e, _, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	// TETRA: a single source-bearing group grant to a fresh SSI is NOT catalogued.
	g := Grant{System: "X", Protocol: "tetra", GroupID: 1020600, SourceID: 1005001, FrequencyHz: 467_912_500}
	e.HandleGrant(g)
	if tg := e.talkgroups.Lookup(1020600); tg != nil {
		t.Fatalf("first TETRA sighting must not catalogue SSI 1020600 yet (corroboration), got %+v", tg)
	}
	// The second sighting corroborates and catalogues it.
	e.HandleGrant(g)
	if tg := e.talkgroups.Lookup(1020600); tg == nil || tg.Tag != discoveredTag {
		t.Fatalf("second TETRA sighting should catalogue SSI 1020600, got %+v", tg)
	}

	// Emergency bypasses corroboration (a first-sighting emergency is catalogued).
	em := Grant{System: "X", Protocol: "tetra", GroupID: 1020601, SourceID: 1005002, FrequencyHz: 467_912_500, Emergency: true}
	e.HandleGrant(em)
	if tg := e.talkgroups.Lookup(1020601); tg == nil {
		t.Errorf("emergency TETRA grant should catalogue on first sight, got nil")
	}

	// Non-TETRA (P25) discovery is unchanged: catalogued on the first sighting.
	e.HandleGrant(Grant{System: "X", Protocol: "p25", GroupID: 4321, SourceID: 7, FrequencyHz: 851_000_000})
	if tg := e.talkgroups.Lookup(4321); tg == nil {
		t.Errorf("P25 discovery must be unchanged (catalogued on first sight), got nil")
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
	// The authoritative source-bearing group grant discovers the real TG — but
	// TETRA discovery is now corroborated, so a second sighting is needed (a real
	// talkgroup recurs; a one-off unit-to-unit callee does not). See
	// TestEngineTETRADiscoveryRequiresCorroboration.
	authoritative := Grant{System: "X", Protocol: "tetra", GroupID: gssi, SourceID: issi, FrequencyHz: freq, Timeslot: 1}
	e.HandleGrant(authoritative)
	e.HandleGrant(authoritative)
	if tg := e.talkgroups.Lookup(gssi); tg == nil || tg.Tag != discoveredTag {
		t.Errorf("authoritative group grant should have discovered TG %d after corroboration, got %+v", gssi, tg)
	}
	// And the notification's SSI is still not a talkgroup afterwards.
	if tg := e.talkgroups.Lookup(issi); tg != nil {
		t.Errorf("radio SSI %d must not appear as a talkgroup, got %+v", issi, tg)
	}
}

// TestEngineTETRANotificationToKnownRadioRecordsIndividual: a held notification
// targeting a KNOWN radio SSI that no group grant ever supersedes is recorded as an
// INDIVIDUAL call to that SSI at flush — filed under recordings/<sys>/individual/
// <SSI>/, never under a phantom talkgroup named after the radio ID (the leak the
// hold exists to prevent). It is NOT dropped: dropping lost the voiced audio the
// over actually carried, which the operator wants preserved.
func TestEngineTETRANotificationToKnownRadioRecordsIndividual(t *testing.T) {
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

	// Flush with no group grant: recorded as an individual call to the SSI, never a TG.
	e.flushHeldGrant(channelKeyOf(page))
	act := pool.Active()
	if len(act) != 1 {
		t.Fatalf("active calls after flush = %d, want 1 (individual call must record)", len(act))
	}
	if !act[0].Grant.Individual {
		t.Errorf("flushed call Individual = false, want true (must not masquerade as a talkgroup)")
	}
	if got := act[0].Grant.GroupID; got != issi {
		t.Errorf("flushed call GroupID = %d, want %d (the addressed radio SSI)", got, issi)
	}
	if tg := e.talkgroups.Lookup(issi); tg != nil {
		t.Errorf("radio SSI %d catalogued as a talkgroup: %+v", issi, tg)
	}
}

// TestEngineTETRANotificationToUnknownSSIRecordsIndividual is the failing-first
// regression for the exact leak the operator reported (RID 1005497 / 1005557): a
// source-less notification whose dst was NEVER seen transmitting (so it is not yet
// a known radio) and that no group grant supersedes was recorded under the raw
// radio ID as a phantom talkgroup ("recordings/<sys>/1005497/…", Individual=false).
// A source-less TETRA notification is always addressed to the paged radio's own
// SSI, so even an as-yet-unknown dst is a radio, never a real talkgroup. It must now
// flush to an INDIVIDUAL call. (Old code: Individual=false → fails here.)
func TestEngineTETRANotificationToUnknownSSIRecordsIndividual(t *testing.T) {
	e, pool, bus, _ := mkEngine(t, 2)
	defer bus.Close()
	e.afterFunc = noFireAfterFunc

	const (
		freq = uint32(467_912_500)
		ts   = uint8(2)
		issi = uint32(1005497) // reported leak: a radio never seen transmitting
	)
	page := Grant{System: "X", Protocol: "tetra", GroupID: issi, FrequencyHz: freq, Timeslot: ts}
	e.HandleGrant(page)
	if n := len(pool.Active()); n != 0 {
		t.Fatalf("active calls after notification = %d, want 0 (held)", n)
	}

	e.flushHeldGrant(channelKeyOf(page))
	act := pool.Active()
	if len(act) != 1 {
		t.Fatalf("active calls after flush = %d, want 1 (individual call must record)", len(act))
	}
	if !act[0].Grant.Individual {
		t.Errorf("flushed call Individual = false, want true (RID must not become a phantom talkgroup)")
	}
	if got := act[0].Grant.GroupID; got != issi {
		t.Errorf("flushed call GroupID = %d, want %d", got, issi)
	}
	if tg := e.talkgroups.Lookup(issi); tg != nil {
		t.Errorf("RID %d catalogued as a phantom talkgroup: %+v", issi, tg)
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
