package api

import (
	"context"
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

// TestCallAudioEndpointServesRecording drives the whole playback path: the
// recorder's KindCallComplete stamps a WAV onto the call row, and
// GET /api/v1/calls/{id}/audio streams that file back. A call without a
// recording, and an unknown id, both 404.
func TestCallAudioEndpointServesRecording(t *testing.T) {
	bus := events.NewBus(8)
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

	// A real WAV file on disk for the recorded call.
	wavPath := filepath.Join(t.TempDir(), "call.wav")
	want := []byte("RIFF....WAVEfmt fake-pcm-bytes")
	if err := os.WriteFile(wavPath, want, 0o644); err != nil {
		t.Fatal(err)
	}

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	recorded := trunking.CallStart{
		Grant:        trunking.Grant{System: "Alpha", Protocol: "p25", GroupID: 700, FrequencyHz: 851_000_000},
		DeviceSerial: "VOICE-1",
		StartedAt:    startedAt,
	}
	silent := trunking.CallStart{
		Grant:        trunking.Grant{System: "Alpha", Protocol: "p25", GroupID: 701, FrequencyHz: 851_000_000},
		DeviceSerial: "VOICE-2",
		StartedAt:    startedAt.Add(time.Second),
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: recorded})
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: silent})
	bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: trunking.CallComplete{
		Grant: recorded.Grant, DeviceSerial: recorded.DeviceSerial,
		StartedAt: startedAt, EndedAt: startedAt.Add(2 * time.Second), AudioPath: wavPath,
	}})

	// Wait until both rows exist and the recorded one is flagged.
	var recordedID, silentID int64
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), storage.HistoryFilter{Limit: 10})
		if len(rows) == 2 {
			for _, r := range rows {
				if r.GroupID == 700 && r.HasRecording {
					recordedID = r.ID
				}
				if r.GroupID == 701 {
					silentID = r.ID
				}
			}
		}
		if recordedID != 0 && silentID != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if recordedID == 0 || silentID == 0 {
		t.Fatalf("setup: recordedID=%d silentID=%d", recordedID, silentID)
	}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, History: HistoryFromStorage(db)})
	defer teardown()

	// The recorded call plays back with its bytes and an audio content type.
	resp := mustGet(t, base+"/api/v1/calls/"+strconv.FormatInt(recordedID, 10)+"/audio")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("recorded audio status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "audio/wav" {
		t.Errorf("content-type = %q, want audio/wav", ct)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != string(want) {
		t.Errorf("body = %q, want %q", got, want)
	}

	// A call with no recording 404s.
	resp2 := mustGet(t, base+"/api/v1/calls/"+strconv.FormatInt(silentID, 10)+"/audio")
	resp2.Body.Close()
	if resp2.StatusCode != 404 {
		t.Errorf("no-recording status = %d, want 404", resp2.StatusCode)
	}

	// An unknown id 404s.
	resp3 := mustGet(t, base+"/api/v1/calls/999999/audio")
	resp3.Body.Close()
	if resp3.StatusCode != 404 {
		t.Errorf("unknown-id status = %d, want 404", resp3.StatusCode)
	}
}
