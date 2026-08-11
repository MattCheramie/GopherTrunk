package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

type fakeAffiliationProvider struct {
	units []trunking.UnitActivity
}

func (f *fakeAffiliationProvider) Affiliations() []trunking.UnitActivity { return f.units }

func TestListRIDsMergesConfiguredAndLive(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()

	db := trunking.NewRIDDB()
	db.Add(&trunking.RID{ID: 100, Alias: "CHIEF", Owner: "Cmdr. Riker", Watch: true})
	db.Add(&trunking.RID{ID: 200, Alias: "LOCKED", Lockout: true})

	live := &fakeAffiliationProvider{units: []trunking.UnitActivity{
		// 100 — configured + live observation
		{
			RadioID: 100, System: "Metro", Protocol: "p25-phase1",
			Talkgroup: 50, TalkerAlias: "CHIEF-ENG", CallCount: 3,
			LastSeen: time.Unix(1700, 0).UTC(),
		},
		// 300 — live only, no configured catalogue row
		{
			RadioID: 300, System: "Metro", Protocol: "p25-phase1",
			Talkgroup: 50, CallCount: 1,
			LastSeen: time.Unix(1800, 0).UTC(),
		},
	}}

	base, teardown := mkServer(t, ServerOptions{
		Bus:          bus,
		RIDs:         db,
		Affiliations: live,
	})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		RIDs []RIDDTO `json:"rids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.RIDs) != 3 {
		t.Fatalf("rids = %d, want 3 (100 configured+live, 200 configured-only, 300 live-only)", len(body.RIDs))
	}
	byID := map[uint32]RIDDTO{}
	for _, r := range body.RIDs {
		byID[r.ID] = r
	}
	// 100 — merged: configured alias and live observation both present.
	if r := byID[100]; r.Alias != "CHIEF" || !r.Configured || r.TalkerAlias != "CHIEF-ENG" ||
		r.LastTalkgroup != 50 || r.CallCount != 3 {
		t.Errorf("RID 100 = %+v", r)
	}
	// 200 — configured but never observed.
	if r := byID[200]; !r.Configured || r.CallCount != 0 || r.LastTalkgroup != 0 ||
		!r.Lockout || r.Alias != "LOCKED" {
		t.Errorf("RID 200 = %+v", r)
	}
	// 300 — live only, not in catalogue.
	if r := byID[300]; r.Configured || r.Alias != "" || r.CallCount != 1 || r.LastTalkgroup != 50 {
		t.Errorf("RID 300 = %+v", r)
	}
}

func TestListRIDsEmptyWhenNothingWired(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		RIDs []RIDDTO `json:"rids"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.RIDs) != 0 {
		t.Errorf("expected empty rids, got %d", len(body.RIDs))
	}
}

func TestGetRIDReturnsConfiguredOnlyRow(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	db := trunking.NewRIDDB()
	db.Add(&trunking.RID{ID: 42, Alias: "ANSWER", Watch: true})

	base, teardown := mkServer(t, ServerOptions{Bus: bus, RIDs: db})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids/42")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got RIDDTO
	json.NewDecoder(resp.Body).Decode(&got)
	if got.ID != 42 || got.Alias != "ANSWER" || !got.Configured {
		t.Errorf("got = %+v", got)
	}
}

func TestGetRIDReturnsLiveOnlyRow(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	live := &fakeAffiliationProvider{units: []trunking.UnitActivity{
		{RadioID: 7, System: "X", Talkgroup: 11, LastSeen: time.Unix(100, 0).UTC()},
	}}
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Affiliations: live})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids/7")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got RIDDTO
	json.NewDecoder(resp.Body).Decode(&got)
	if got.ID != 7 || got.Configured || got.LastTalkgroup != 11 {
		t.Errorf("got = %+v", got)
	}
}

func TestGetRIDNotFound(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids/999")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetRIDInvalidID(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids/notanumber")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRIDHistoryReturnsSourceFilteredRows(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	dbPath := filepath.Join(t.TempDir(), "calls.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cl, err := storage.NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cl.Run(ctx)

	now := time.Now().UTC().Truncate(time.Microsecond)
	for i, src := range []uint32{4242, 4242, 7777} {
		bus.Publish(events.Event{
			Kind: events.KindCallStart,
			Payload: trunking.CallStart{
				Grant: trunking.Grant{
					System: "Metro", Protocol: "p25",
					GroupID: 50, SourceID: src, FrequencyHz: 851_000_000,
				},
				DeviceSerial: "VOICE-1",
				// Distinct StartedAt per row — call_log is keyed by
				// (device_serial, started_at), so identical timestamps
				// collapse via INSERT OR REPLACE.
				StartedAt: now.Add(time.Duration(i) * time.Second),
			},
		})
	}
	// Wait for rows to flush.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), storage.HistoryFilter{Limit: 10})
		if len(rows) == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	base, teardown := mkServer(t, ServerOptions{
		Bus:     bus,
		History: HistoryFromStorage(db),
	})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids/4242/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Calls []CallRow `json:"calls"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Calls) != 2 {
		t.Fatalf("rid 4242 history = %d, want 2", len(body.Calls))
	}
	for _, c := range body.Calls {
		if c.SourceID != 4242 {
			t.Errorf("row source = %d, want 4242", c.SourceID)
		}
	}
}

// seedCallLog opens a fresh sqlite call log, publishes the given CallStart
// grants through the bus, and waits for them to flush. Returns the storage.DB so
// the test can wire it as the api History source. Each grant needs a distinct
// (device_serial, started_at) — the call_log is keyed on that pair.
func seedCallLog(t *testing.T, bus *events.Bus, grants []trunking.Grant, base time.Time) *storage.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "calls.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	cl, err := storage.NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cl.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go cl.Run(ctx)

	for i, g := range grants {
		bus.Publish(events.Event{
			Kind: events.KindCallStart,
			Payload: trunking.CallStart{
				Grant:        g,
				DeviceSerial: "VOICE-" + strconv.Itoa(i),
				StartedAt:    base.Add(time.Duration(i) * time.Minute),
			},
		})
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), storage.HistoryFilter{Limit: 100})
		if len(rows) == len(grants) {
			return db
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("call log did not flush %d rows in time", len(grants))
	return nil
}

// TestListRIDsBackedByCallLogSurvivesSweptTracker is the failing-first regression
// for the "RID list resets after each CC lock" report: the live affiliation
// tracker has a 30-minute idle TTL and no persistence, so during control-channel
// re-hunt gaps every radio ages out and the list collapsed to the static
// catalogue. With the list now backed by the persisted call log, radios seen over
// the daemon's lifetime stay listed even when the tracker has swept them all.
//
// The tracker is EMPTY here (modelling the post-sweep state); without the
// call-log source, /rids would return zero rows.
func TestListRIDsBackedByCallLogSurvivesSweptTracker(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()

	// Span the grants across >30 min so it is unambiguously past the tracker TTL.
	base := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	db := seedCallLog(t, bus, []trunking.Grant{
		// 4242 — a group call (tg 50) then a later INDIVIDUAL call (target 999).
		{System: "Metro", Protocol: "tetra", GroupID: 50, SourceID: 4242},
		{System: "Metro", Protocol: "tetra", GroupID: 999, SourceID: 4242, Individual: true},
		// 7777 — a single group call (tg 60).
		{System: "Metro", Protocol: "tetra", GroupID: 60, SourceID: 7777},
		// source 0 (unknown) must be excluded from the list entirely.
		{System: "Metro", Protocol: "tetra", GroupID: 70, SourceID: 0},
	}, base)

	// Empty live tracker: every radio has aged out, exactly as after a re-hunt gap.
	srv, teardown := mkServer(t, ServerOptions{
		Bus:          bus,
		Affiliations: &fakeAffiliationProvider{},
		History:      HistoryFromStorage(db),
	})
	defer teardown()

	resp, err := http.Get(srv + "/api/v1/rids")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		RIDs []RIDDTO `json:"rids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	byID := map[uint32]RIDDTO{}
	for _, r := range body.RIDs {
		byID[r.ID] = r
	}
	if len(body.RIDs) != 2 {
		t.Fatalf("rids = %d, want 2 (4242, 7777; source 0 excluded); got %+v", len(body.RIDs), body.RIDs)
	}
	// 4242 — durable all-time: 2 calls, last GROUP talkgroup 50 (the individual
	// call's target 999 must NOT surface as a talkgroup).
	if r := byID[4242]; r.CallCount != 2 || r.LastTalkgroup != 50 || r.Configured {
		t.Errorf("RID 4242 = %+v, want CallCount=2 LastTalkgroup=50 Configured=false", r)
	}
	if _, ok := byID[7777]; !ok {
		t.Errorf("RID 7777 missing from list")
	}
	if r := byID[7777]; r.CallCount != 1 || r.LastTalkgroup != 60 {
		t.Errorf("RID 7777 = %+v, want CallCount=1 LastTalkgroup=60", r)
	}
}

// TestListRIDsKeepsAllTimeCountOverLiveWindow pins the merge precedence: when a
// radio is in BOTH the durable call log and the live tracker, the freshest
// observation (last talkgroup / last-seen) comes from the tracker, but the
// all-time call count from the call log is kept — the live counter is only a
// post-sweep partial and must not shrink the displayed CALLS total.
func TestListRIDsKeepsAllTimeCountOverLiveWindow(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()

	base := time.Now().Add(-3 * time.Hour).UTC().Truncate(time.Second)
	db := seedCallLog(t, bus, []trunking.Grant{
		{System: "Metro", Protocol: "tetra", GroupID: 50, SourceID: 4242},
		{System: "Metro", Protocol: "tetra", GroupID: 50, SourceID: 4242},
		{System: "Metro", Protocol: "tetra", GroupID: 50, SourceID: 4242},
	}, base)

	// Live tracker holds 4242 with a smaller (windowed) count and a fresher TG.
	live := &fakeAffiliationProvider{units: []trunking.UnitActivity{
		{RadioID: 4242, System: "Metro", Protocol: "tetra", Talkgroup: 77,
			CallCount: 1, LastSeen: time.Now().UTC()},
	}}
	srv, teardown := mkServer(t, ServerOptions{Bus: bus, Affiliations: live, History: HistoryFromStorage(db)})
	defer teardown()

	resp, err := http.Get(srv + "/api/v1/rids")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		RIDs []RIDDTO `json:"rids"`
	}
	json.NewDecoder(resp.Body).Decode(&body)
	if len(body.RIDs) != 1 {
		t.Fatalf("rids = %d, want 1", len(body.RIDs))
	}
	r := body.RIDs[0]
	if r.ID != 4242 || r.CallCount != 3 || r.LastTalkgroup != 77 {
		t.Errorf("RID 4242 = %+v, want CallCount=3 (all-time) LastTalkgroup=77 (live)", r)
	}
}

// TestGetRIDBackedByCallLogWhenSwept covers handleGetRID's durable path: a radio
// swept from the live tracker is still found via the persisted call log
// (exercising the SourceID-filtered RIDSummaries query), instead of 404ing.
func TestGetRIDBackedByCallLogWhenSwept(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	base := time.Now().Add(-90 * time.Minute).UTC().Truncate(time.Second)
	db := seedCallLog(t, bus, []trunking.Grant{
		{System: "Metro", Protocol: "tetra", GroupID: 88, SourceID: 5150},
		{System: "Metro", Protocol: "tetra", GroupID: 88, SourceID: 5150},
	}, base)

	srv, teardown := mkServer(t, ServerOptions{
		Bus:          bus,
		Affiliations: &fakeAffiliationProvider{},
		History:      HistoryFromStorage(db),
	})
	defer teardown()

	resp, err := http.Get(srv + "/api/v1/rids/5150")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 (found via call log)", resp.StatusCode)
	}
	var got RIDDTO
	json.NewDecoder(resp.Body).Decode(&got)
	if got.ID != 5150 || got.CallCount != 2 || got.LastTalkgroup != 88 {
		t.Errorf("got = %+v, want ID=5150 CallCount=2 LastTalkgroup=88", got)
	}
}

// TestListRIDsFilteredBySystem pins the ?system= scope the RID roster UI uses:
// only radios with call-log or live activity on the named system are returned,
// the static-only catalogue rows (which carry no system) are excluded while a
// filter is active, and the unfiltered list still returns everything.
func TestListRIDsFilteredBySystem(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()

	base := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	db := seedCallLog(t, bus, []trunking.Grant{
		{System: "Metro", Protocol: "tetra", GroupID: 50, SourceID: 4242},
		{System: "Harbor", Protocol: "dmr-tier2", GroupID: 22, SourceID: 7777},
	}, base)

	// A static-only catalogue row (no system) and a live affiliation on Metro.
	riddb := trunking.NewRIDDB()
	riddb.Add(&trunking.RID{ID: 100, Alias: "CHIEF", Watch: true})
	live := &fakeAffiliationProvider{units: []trunking.UnitActivity{
		{RadioID: 300, System: "Metro", Protocol: "tetra", Talkgroup: 50,
			CallCount: 1, LastSeen: time.Now().UTC()},
		{RadioID: 400, System: "Harbor", Protocol: "dmr-tier2", Talkgroup: 22,
			CallCount: 1, LastSeen: time.Now().UTC()},
	}}

	srv, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: riddb, Affiliations: live, History: HistoryFromStorage(db),
	})
	defer teardown()

	ids := func(url string) map[uint32]bool {
		t.Helper()
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d for %s", resp.StatusCode, url)
		}
		var body struct {
			RIDs []RIDDTO `json:"rids"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		out := map[uint32]bool{}
		for _, r := range body.RIDs {
			out[r.ID] = true
		}
		return out
	}

	// Unfiltered: everything (static 100, live 300/400, call-log 4242/7777).
	all := ids(srv + "/api/v1/rids")
	for _, id := range []uint32{100, 300, 400, 4242, 7777} {
		if !all[id] {
			t.Errorf("unfiltered list missing RID %d: %v", id, all)
		}
	}

	// Metro: call-log 4242 + live 300; NOT Harbor (7777/400) and NOT static 100.
	metro := ids(srv + "/api/v1/rids?system=Metro")
	if !metro[4242] || !metro[300] {
		t.Errorf("Metro filter missing 4242/300: %v", metro)
	}
	for _, id := range []uint32{100, 400, 7777} {
		if metro[id] {
			t.Errorf("Metro filter wrongly included RID %d: %v", id, metro)
		}
	}

	// Harbor: call-log 7777 + live 400 only.
	harbor := ids(srv + "/api/v1/rids?system=Harbor")
	if !harbor[7777] || !harbor[400] {
		t.Errorf("Harbor filter missing 7777/400: %v", harbor)
	}
	for _, id := range []uint32{100, 300, 4242} {
		if harbor[id] {
			t.Errorf("Harbor filter wrongly included RID %d: %v", id, harbor)
		}
	}
}

func TestRIDHistory503WithoutHistoryWired(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/rids/1/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestUpdateRIDMutatesConfiguredRow(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	db := trunking.NewRIDDB()
	db.Add(&trunking.RID{ID: 100, Alias: "OLD", Watch: true})

	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		RIDs:           db,
		AllowMutations: true,
	})
	defer teardown()

	body, _ := json.Marshal(map[string]any{"alias": "NEW", "priority": 5, "watch": false})
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/rids/100", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got RIDDTO
	json.NewDecoder(resp.Body).Decode(&got)
	if got.Alias != "NEW" || got.Priority != 5 || got.Watch {
		t.Errorf("got = %+v", got)
	}
	// Confirm the in-memory DB reflects the change.
	if r := db.Lookup(100); r == nil || r.Alias != "NEW" || r.Priority != 5 || r.Watch {
		t.Errorf("db row = %+v", r)
	}
}

func TestUpdateRIDRejectsEmptyBody(t *testing.T) {
	bus := events.NewBus(4)
	defer bus.Close()
	db := trunking.NewRIDDB()
	db.Add(&trunking.RID{ID: 1, Watch: true})

	base, teardown := mkServer(t, ServerOptions{Bus: bus, RIDs: db, AllowMutations: true})
	defer teardown()

	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/rids/1", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUpdateRIDLiveOnlyReturns404(t *testing.T) {
	// A RID only seen over the air (no static catalogue row) cannot
	// be patched — operators must add it to the alias file first.
	bus := events.NewBus(4)
	defer bus.Close()
	live := &fakeAffiliationProvider{units: []trunking.UnitActivity{
		{RadioID: 5, Talkgroup: 1},
	}}
	base, teardown := mkServer(t, ServerOptions{
		Bus: bus, RIDs: trunking.NewRIDDB(), Affiliations: live, AllowMutations: true,
	})
	defer teardown()

	body, _ := json.Marshal(map[string]any{"alias": "NEW"})
	req, _ := http.NewRequest(http.MethodPatch, base+"/api/v1/rids/5", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
