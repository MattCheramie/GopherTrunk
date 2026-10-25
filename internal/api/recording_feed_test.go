package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRecordingFeedFollowsCompletionOrder pins issue #1288 §10's recoverable
// cursor: GET /api/v1/recordings lists finalized recordings in the order they
// FINISHED, each with its call row and a URL for the original file, so a
// consumer that resumes after the last id it processed misses nothing - here
// call A starts first but finishes after call B, which a cursor on call ids
// (assigned at call start) would skip.
func TestRecordingFeedFollowsCompletionOrder(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	db, err := storage.Open(filepath.Join(t.TempDir(), "calls.db"))
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

	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.wav")
	fileB := filepath.Join(dir, "b.wav")
	writeTestWAV(t, fileA, 8000, 8000, 1000)
	writeTestWAV(t, fileB, 8000, 4000, -1000)

	t0 := time.Now().UTC().Truncate(time.Microsecond)
	callA := trunking.CallStart{Grant: trunking.Grant{System: "scanner", Protocol: "fm-conv", GroupID: 0x80100000,
		GroupLabel: "Fire", FrequencyHz: 460_100_000}, DeviceSerial: "wb:WB1:conv:460100000", StartedAt: t0}
	callB := trunking.CallStart{Grant: trunking.Grant{System: "scanner", Protocol: "p25", GroupID: 0x80100001,
		GroupLabel: "County", FrequencyHz: 460_200_000}, DeviceSerial: "wb:WB1:conv:460200000", StartedAt: t0.Add(time.Second)}
	publish := func(kind events.Kind, p any) { bus.Publish(events.Event{Kind: kind, Payload: p}) }
	publish(events.KindCallStart, callA)
	publish(events.KindCallStart, callB)
	// B finishes first, A after it.
	publish(events.KindCallEnd, trunking.CallEnd{Grant: callB.Grant, DeviceSerial: callB.DeviceSerial,
		StartedAt: callB.StartedAt, EndedAt: t0.Add(2 * time.Second), Reason: trunking.EndReasonNormal})
	publish(events.KindCallComplete, trunking.CallComplete{Grant: callB.Grant, DeviceSerial: callB.DeviceSerial,
		StartedAt: callB.StartedAt, EndedAt: t0.Add(2 * time.Second), AudioPath: fileB})
	publish(events.KindCallEnd, trunking.CallEnd{Grant: callA.Grant, DeviceSerial: callA.DeviceSerial,
		StartedAt: callA.StartedAt, EndedAt: t0.Add(5 * time.Second), Reason: trunking.EndReasonNormal})
	publish(events.KindCallComplete, trunking.CallComplete{Grant: callA.Grant, DeviceSerial: callA.DeviceSerial,
		StartedAt: callA.StartedAt, EndedAt: t0.Add(5 * time.Second), AudioPath: fileA})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if recs, _ := db.RecordingFeed(context.Background(), 0, 10); len(recs) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, History: HistoryFromStorage(db)})
	defer teardown()
	type feed struct {
		Recordings []FinalizedRecording `json:"recordings"`
		NextAfter  int64                `json:"next_after"`
	}
	get := func(q string) feed {
		t.Helper()
		resp := mustGet(t, base+"/api/v1/recordings"+q)
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("GET %s: status %d", q, resp.StatusCode)
		}
		var f feed
		if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
			t.Fatal(err)
		}
		return f
	}

	all := get("")
	if len(all.Recordings) != 2 {
		t.Fatalf("feed has %d recordings, want 2", len(all.Recordings))
	}
	first, second := all.Recordings[0], all.Recordings[1]
	if first.Call == nil || second.Call == nil {
		t.Fatalf("feed entries without their call rows: %+v", all.Recordings)
	}
	if first.Call.DeviceSerial != callB.DeviceSerial || second.Call.DeviceSerial != callA.DeviceSerial {
		t.Errorf("feed order %s, %s; want B (finished first) then A", first.Call.DeviceSerial, second.Call.DeviceSerial)
	}
	if first.Call.Protocol != "p25" || first.Call.FrequencyHz != 460_200_000 || first.Call.EndReason == "" {
		t.Errorf("B's call row %+v lacks its metadata", *first.Call)
	}
	if first.Format != "wav" || first.AudioURL != "/api/v1/recordings/"+strconv.FormatInt(first.ID, 10)+"/audio" {
		t.Errorf("B's format %q / audio_url %q", first.Format, first.AudioURL)
	}
	if all.NextAfter != second.ID {
		t.Errorf("next_after = %d, want the last id %d", all.NextAfter, second.ID)
	}

	// Resuming after B finds A, and nothing after A.
	if rest := get("?after=" + strconv.FormatInt(first.ID, 10)); len(rest.Recordings) != 1 || rest.Recordings[0].ID != second.ID {
		t.Errorf("after B: %+v, want only A", rest.Recordings)
	}
	if none := get("?after=" + strconv.FormatInt(second.ID, 10)); len(none.Recordings) != 0 || none.NextAfter != second.ID {
		t.Errorf("after A: %d entries, next_after %d; want none and the same cursor", len(none.Recordings), none.NextAfter)
	}
	// A replay from the start returns the same ids, so a duplicate is recognisable.
	if again := get("?limit=1"); len(again.Recordings) != 1 || again.Recordings[0].ID != first.ID {
		t.Errorf("replay: %+v, want B's id %d again", again.Recordings, first.ID)
	}

	// The audio URL serves the original file byte for byte.
	resp := mustGet(t, base+first.AudioURL)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	want, _ := os.ReadFile(fileB)
	if resp.StatusCode != 200 || !bytes.Equal(body, want) {
		t.Errorf("audio: status %d, %d bytes; want 200 and B's %d original bytes", resp.StatusCode, len(body), len(want))
	}
	if resp := mustGet(t, base+"/api/v1/recordings/99999/audio"); resp.StatusCode != 404 {
		t.Errorf("unknown recording: status %d, want 404", resp.StatusCode)
	}
	if resp := mustGet(t, base+"/api/v1/recordings?after=x"); resp.StatusCode != 400 {
		t.Errorf("bad cursor: status %d, want 400", resp.StatusCode)
	}
}
