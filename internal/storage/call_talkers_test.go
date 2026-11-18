package storage

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// talkerCall logs one call on its own device serial: it starts with the first
// over's radio, each further over arrives as a source update (which is what
// moves call_log.source_id to the LAST talker), and every over finishes a
// recording file stamped with its own radio. overs lists the radio of each
// file in order; 0 is a file whose talker never resolved.
func talkerCall(t *testing.T, bus *events.Bus, serial string, tg uint32, start time.Time, overs ...uint32) {
	t.Helper()
	g := trunking.Grant{System: "R13", Protocol: "p25", GroupID: tg, SourceID: overs[0], FrequencyHz: 167_850_000}
	publish := func(kind events.Kind, p any) { bus.Publish(events.Event{Kind: kind, Payload: p}) }
	publish(events.KindCallStart, trunking.CallStart{Grant: g, DeviceSerial: serial, StartedAt: start})
	end := start
	for i, src := range overs {
		if i > 0 && src != 0 {
			publish(events.KindCallSourceUpdate, trunking.CallSourceUpdate{
				DeviceSerial: serial, System: g.System, Protocol: g.Protocol, GroupID: tg,
				SourceID: src, FrequencyHz: g.FrequencyHz, At: end,
			})
		}
		g.SourceID = src
		overStart := end
		end = overStart.Add(3 * time.Second)
		publish(events.KindCallComplete, trunking.CallComplete{
			Grant: g, DeviceSerial: serial, StartedAt: overStart, EndedAt: end,
			CallStartedAt: start, Segment: i,
			AudioPath: "/rec/" + serial + "-" + string(rune('a'+i)) + ".wav",
		})
	}
	// The call row's own source is the last radio that resolved.
	last := overs[0]
	for _, src := range overs {
		if src != 0 {
			last = src
		}
	}
	g.SourceID = last
	publish(events.KindCallEnd, trunking.CallEnd{Grant: g, DeviceSerial: serial, StartedAt: start,
		EndedAt: end, Reason: trunking.EndReasonNormal})
}

// newTalkerLog opens a database with a running call log on a fresh bus.
func newTalkerLog(t *testing.T) (*DB, *events.Bus) {
	t.Helper()
	db := openTestDB(t)
	bus := events.NewBus(64)
	t.Cleanup(bus.Close)
	cl, err := NewCallLog(db, bus, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cl.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go cl.Run(ctx)
	return db, bus
}

// waitRecordings blocks until the call log has attached n recording files.
func waitRecordings(t *testing.T, db *DB, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var got int
		if err := db.SQL().QueryRow(`SELECT COUNT(*) FROM call_recordings`).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got == n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("call_recordings has %d rows, want %d", got, n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A call row's source_id is the call's LAST talker, so a reply inside the
// hangtime hid the radio that opened the call (#1379). The row now lists
// every radio its recording files name, in the order each first spoke.
func TestHistoryListsEveryTalkerOfACall(t *testing.T) {
	db, bus := newTalkerLog(t)
	t0 := time.Now().UTC().Truncate(time.Microsecond)
	const a, b, c = uint32(3200460), uint32(315529), uint32(77001)
	talkerCall(t, bus, "tap-0", 100, t0, a, b)                        // A, answered by B
	talkerCall(t, bus, "tap-1", 101, t0.Add(time.Minute), a, b, a, c) // A B A C: each radio once
	talkerCall(t, bus, "tap-2", 102, t0.Add(2*time.Minute), a, a)     // one radio, two overs
	talkerCall(t, bus, "tap-3", 103, t0.Add(3*time.Minute), a, 0)     // second over never resolved
	talkerCall(t, bus, "tap-4", 104, t0.Add(4*time.Minute), 0, b)     // opened with no source, then B
	waitRecordings(t, db, 12)

	rows, err := db.History(context.Background(), HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	got := map[uint32]CallRow{}
	for _, r := range rows {
		got[r.GroupID] = r
	}
	for _, tc := range []struct {
		tg      uint32
		source  uint32
		talkers []uint32
	}{
		{100, b, []uint32{a, b}},
		{101, c, []uint32{a, b, c}},
		{102, a, nil},
		{103, a, nil},
		{104, b, nil},
	} {
		r, ok := got[tc.tg]
		if !ok {
			t.Fatalf("tg %d: no call row", tc.tg)
		}
		if r.SourceID != tc.source {
			t.Errorf("tg %d: source_id = %d, want %d (the last talker, unchanged)", tc.tg, r.SourceID, tc.source)
		}
		if !slices.Equal(r.Talkers, tc.talkers) {
			t.Errorf("tg %d: talkers = %v, want %v", tc.tg, r.Talkers, tc.talkers)
		}
	}
}

// A radio that keyed but left no recording of its own (its over was discarded
// as too short, say) is still on the call row; the list names it after the
// radios the files carry.
func TestHistoryTalkersKeepTheRowSourceNoFileNames(t *testing.T) {
	db, bus := newTalkerLog(t)
	t0 := time.Now().UTC().Truncate(time.Microsecond)
	const a, b = uint32(3200460), uint32(315529)
	g := trunking.Grant{System: "R13", Protocol: "p25", GroupID: 200, SourceID: a, FrequencyHz: 167_850_000}
	publish := func(kind events.Kind, p any) { bus.Publish(events.Event{Kind: kind, Payload: p}) }
	publish(events.KindCallStart, trunking.CallStart{Grant: g, DeviceSerial: "tap-0", StartedAt: t0})
	publish(events.KindCallComplete, trunking.CallComplete{Grant: g, DeviceSerial: "tap-0",
		StartedAt: t0, EndedAt: t0.Add(4 * time.Second), CallStartedAt: t0, AudioPath: "/rec/only.wav"})
	g.SourceID = b
	publish(events.KindCallEnd, trunking.CallEnd{Grant: g, DeviceSerial: "tap-0", StartedAt: t0,
		EndedAt: t0.Add(5 * time.Second), Reason: trunking.EndReasonNormal})
	waitRecordings(t, db, 1)

	var rows []CallRow
	deadline := time.Now().Add(3 * time.Second)
	for {
		var err error
		if rows, err = db.History(context.Background(), HistoryFilter{Limit: 1}); err != nil {
			t.Fatal(err)
		}
		if (len(rows) == 1 && rows[0].SourceID == b) || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(rows) != 1 || rows[0].SourceID != b {
		t.Fatalf("call row = %+v, want one row whose source is %d", rows, b)
	}
	if want := []uint32{a, b}; !slices.Equal(rows[0].Talkers, want) {
		t.Errorf("talkers = %v, want %v", rows[0].Talkers, want)
	}
}

// The per-radio view of the call log (?source_id=, /rids/{id}/history) matched
// only call_log.source_id, so it missed every call the radio opened and
// another radio answered. It now also finds a call through its files' talkers.
func TestHistorySourceFilterFindsEarlierTalker(t *testing.T) {
	db, bus := newTalkerLog(t)
	t0 := time.Now().UTC().Truncate(time.Microsecond)
	const a, b, c = uint32(3200460), uint32(315529), uint32(77001)
	talkerCall(t, bus, "tap-0", 100, t0, a, b)                 // A opens, B answers
	talkerCall(t, bus, "tap-1", 101, t0.Add(time.Minute), b)   // B alone
	talkerCall(t, bus, "tap-2", 102, t0.Add(2*time.Minute), c) // C alone
	waitRecordings(t, db, 4)

	tgs := func(src uint32) []uint32 {
		t.Helper()
		rows, err := db.History(context.Background(), HistoryFilter{SourceID: src, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		var out []uint32
		for _, r := range rows {
			out = append(out, r.GroupID)
		}
		slices.Sort(out)
		return out
	}
	if got, want := tgs(a), []uint32{100}; !slices.Equal(got, want) {
		t.Errorf("calls with radio %d: tgs %v, want %v (it opened the call another radio answered)", a, got, want)
	}
	if got, want := tgs(b), []uint32{100, 101}; !slices.Equal(got, want) {
		t.Errorf("calls with radio %d: tgs %v, want %v", b, got, want)
	}
	if got, want := tgs(c), []uint32{102}; !slices.Equal(got, want) {
		t.Errorf("calls with radio %d: tgs %v, want %v", c, got, want)
	}
	if got := tgs(999); len(got) != 0 {
		t.Errorf("calls with an unknown radio: tgs %v, want none", got)
	}
}

// The index behind the per-radio filter is on a column an older database does
// not have. Created with the schema it would fail the open of every such
// database, so it is created after the column is added; this opens one.
func TestCallRecordingsSourceIndexOnLegacyDatabase(t *testing.T) {
	path := t.TempDir() + "/legacy.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DROP TABLE call_recordings`,
		`CREATE TABLE call_recordings (
			id INTEGER PRIMARY KEY AUTOINCREMENT, call_id INTEGER NOT NULL,
			seq INTEGER NOT NULL, path TEXT NOT NULL, started_at INTEGER NOT NULL,
			ended_at INTEGER, UNIQUE(call_id, path))`,
	} {
		if _, err := db.SQL().Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()

	db, err = Open(path)
	if err != nil {
		t.Fatalf("reopen a database written before call_recordings.source_id: %v", err)
	}
	defer db.Close()
	var n int
	if err := db.SQL().QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_call_recordings_source'`,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("idx_call_recordings_source missing after the migration")
	}
}
