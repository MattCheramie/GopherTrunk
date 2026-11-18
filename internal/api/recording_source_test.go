package api

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// twoRadioCall logs issue #1379's call through the real call log: radio A
// opens it, radio B answers inside the hangtime, and the recorder announces
// one file per over, each with the radio heard in it. The call row's own
// source ends as B, the last talker. It returns the server's base URL.
func twoRadioCall(t *testing.T, a, b uint32) string {
	t.Helper()
	bus := events.NewBus(32)
	t.Cleanup(bus.Close)
	db, err := storage.Open(filepath.Join(t.TempDir(), "calls.db"))
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

	const serial = "wb:test0001:tap-0"
	t0 := time.Now().UTC().Truncate(time.Microsecond)
	g := trunking.Grant{System: "R13", Protocol: "p25", GroupID: 30508, SourceID: a, FrequencyHz: 167_850_000}
	publish := func(kind events.Kind, p any) { bus.Publish(events.Event{Kind: kind, Payload: p}) }
	publish(events.KindCallStart, trunking.CallStart{Grant: g, DeviceSerial: serial, StartedAt: t0})
	publish(events.KindCallComplete, trunking.CallComplete{Grant: g, DeviceSerial: serial,
		StartedAt: t0, EndedAt: t0.Add(5 * time.Second), CallStartedAt: t0, Segment: 0, AudioPath: "/rec/over0.wav"})
	g.SourceID = b
	publish(events.KindCallSourceUpdate, trunking.CallSourceUpdate{DeviceSerial: serial, System: g.System,
		Protocol: g.Protocol, GroupID: g.GroupID, SourceID: b, FrequencyHz: g.FrequencyHz, At: t0.Add(5 * time.Second)})
	publish(events.KindCallComplete, trunking.CallComplete{Grant: g, DeviceSerial: serial,
		StartedAt: t0.Add(5 * time.Second), EndedAt: t0.Add(8 * time.Second), CallStartedAt: t0, Segment: 1,
		AudioPath: "/rec/over1.wav"})
	publish(events.KindCallEnd, trunking.CallEnd{Grant: g, DeviceSerial: serial, StartedAt: t0,
		EndedAt: t0.Add(8 * time.Second), Reason: trunking.EndReasonNormal})

	deadline := time.Now().Add(3 * time.Second)
	for {
		var files, ended int
		_ = db.SQL().QueryRow(`SELECT COUNT(*) FROM call_recordings`).Scan(&files)
		_ = db.SQL().QueryRow(`SELECT COUNT(*) FROM call_log WHERE ended_at IS NOT NULL AND source_id = ?`, b).Scan(&ended)
		if files == 2 && ended == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("call log has %d recording files and %d ended calls from radio %d; want 2 and 1", files, ended, b)
		}
		time.Sleep(10 * time.Millisecond)
	}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, History: HistoryFromStorage(db)})
	t.Cleanup(teardown)
	return base
}

// getGeneric decodes a GET's body into generic JSON, so a test reads exactly
// the keys a client would and does not depend on a Go field existing.
func getGeneric(t *testing.T, url string) map[string]any {
	t.Helper()
	var out map[string]any
	getJSON(t, url, &out)
	return out
}

// jsonUint reads a JSON number as a radio ID; ok is false when the key is
// absent or not a number.
func jsonUint(m map[string]any, key string) (uint32, bool) {
	f, ok := m[key].(float64)
	return uint32(f), ok
}

// Issue #1379: every file of a multi-radio call was attributed to the call's
// last radio, because the recording feed carried no talker of its own and its
// embedded call row holds only the last one. Each feed entry now names the
// radio heard in that file.
func TestRecordingFeedNamesEachOversTalker(t *testing.T) {
	const a, b = uint32(3200460), uint32(315529)
	base := twoRadioCall(t, a, b)

	recs, _ := getGeneric(t, base+"/api/v1/recordings")["recordings"].([]any)
	if len(recs) != 2 {
		t.Fatalf("feed has %d recordings, want 2", len(recs))
	}
	for i, want := range []uint32{a, b} {
		rec, _ := recs[i].(map[string]any)
		if seg, _ := rec["segment"].(float64); int(seg) != i {
			t.Fatalf("feed entry %d is segment %v", i, rec["segment"])
		}
		got, ok := jsonUint(rec, "source_id")
		if !ok || got != want {
			t.Errorf("segment %d: source_id = %v, want %d (the radio heard in that file)", i, rec["source_id"], want)
		}
		// The embedded call row is the whole call: its source stays the last
		// talker, as before.
		call, _ := rec["call"].(map[string]any)
		if got, _ := jsonUint(call, "source_id"); got != b {
			t.Errorf("segment %d: call.source_id = %v, want %d (the call's last talker)", i, call["source_id"], b)
		}
	}
}

// The call list showed only the last radio of such a call, and the per-radio
// views (?source_id=, /rids/{id}/history) did not find the call under the
// radio that opened it. The row now lists every radio, and both views find
// the call under either one.
func TestCallHistoryListsAndFindsEveryTalker(t *testing.T) {
	const a, b = uint32(3200460), uint32(315529)
	base := twoRadioCall(t, a, b)

	calls := func(path string) []any {
		t.Helper()
		rows, _ := getGeneric(t, base+path)["calls"].([]any)
		return rows
	}
	all := calls("/api/v1/calls/history")
	if len(all) != 1 {
		t.Fatalf("history has %d calls, want 1", len(all))
	}
	row, _ := all[0].(map[string]any)
	if got, _ := jsonUint(row, "source_id"); got != b {
		t.Errorf("source_id = %v, want %d (the last talker, unchanged)", row["source_id"], b)
	}
	talkers, _ := row["talkers"].([]any)
	if len(talkers) != 2 || talkers[0] != float64(a) || talkers[1] != float64(b) {
		t.Errorf("talkers = %v, want [%d %d] in the order they spoke", row["talkers"], a, b)
	}

	for _, rid := range []uint32{a, b} {
		id := strconv.FormatUint(uint64(rid), 10)
		for _, path := range []string{"/api/v1/calls/history?source_id=" + id, "/api/v1/rids/" + id + "/history"} {
			if got := calls(path); len(got) != 1 {
				t.Errorf("GET %s: %d calls, want the one call radio %d was heard in", path, len(got), rid)
			}
		}
	}
	if got := calls("/api/v1/calls/history?source_id=999"); len(got) != 0 {
		t.Errorf("history for a radio that never spoke: %d calls, want none", len(got))
	}
}
