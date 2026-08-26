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

// TestCallAudioEndpointServesRelativeRecordingPath pins the "recording is
// unavailable while the file is right there" UI bug: a daemon started with a
// relative -config path used to store a cwd-relative recordings dir, so every
// call row's recording_path was relative and handleCallAudio's absolute-path
// guard 404'd it even though os.Open would have succeeded. A relative stored
// path must be resolved against the daemon's working directory — the same
// directory the recorder wrote it under — and served.
func TestCallAudioEndpointServesRelativeRecordingPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	bus := events.NewBus(8)
	defer bus.Close()
	db, err := storage.Open(filepath.Join(dir, "calls.db"))
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

	// The recording exists on disk, reachable via the cwd-relative path the
	// old config resolution produced.
	relPath := filepath.Join("recordings", "Alpha", "700", "call.wav")
	if err := os.MkdirAll(filepath.Dir(relPath), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("RIFF....WAVEfmt fake-pcm-bytes")
	if err := os.WriteFile(relPath, want, 0o644); err != nil {
		t.Fatal(err)
	}

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	call := trunking.CallStart{
		Grant:        trunking.Grant{System: "Alpha", Protocol: "tetra", GroupID: 700, FrequencyHz: 467_912_500},
		DeviceSerial: "cc:same-carrier:1",
		StartedAt:    startedAt,
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: call})
	bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: trunking.CallComplete{
		Grant: call.Grant, DeviceSerial: call.DeviceSerial,
		StartedAt: startedAt, EndedAt: startedAt.Add(2 * time.Second), AudioPath: relPath,
	}})

	var id int64
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), storage.HistoryFilter{Limit: 10})
		if len(rows) == 1 && rows[0].HasRecording {
			id = rows[0].ID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if id == 0 {
		t.Fatal("setup: call row with HasRecording never appeared")
	}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, History: HistoryFromStorage(db)})
	defer teardown()

	resp := mustGet(t, base+"/api/v1/calls/"+strconv.FormatInt(id, 10)+"/audio")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("relative-path audio status = %d, want 200 (file exists at %s)",
			resp.StatusCode, filepath.Join(dir, relPath))
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != string(want) {
		t.Errorf("body = %q, want %q", got, want)
	}
}
