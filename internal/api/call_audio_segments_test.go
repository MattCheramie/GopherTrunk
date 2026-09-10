package api

import (
	"context"
	"encoding/binary"
	"io"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/storage"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
	"github.com/MattCheramie/GopherTrunk/internal/voice"
)

// writeTestWAV writes n samples of a constant value at rate to path.
func writeTestWAV(t *testing.T, path string, rate uint32, n int, value int16) {
	t.Helper()
	w, err := voice.NewWavFile(path, rate)
	if err != nil {
		t.Fatal(err)
	}
	pcm := make([]int16, n)
	for i := range pcm {
		pcm[i] = value
	}
	if err := w.WriteSamples(pcm); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestCallAudioEndpointPlaysEverySegment pins the multi-over playback fix:
// a call recorded in per-transmission grouping has one file per over, and
// /calls/{id}/audio used to serve only the row's single path — the first
// over — so a 30 s call played 4 s. Now every segment is concatenated in order
// with the real inter-over silence restored.
func TestCallAudioEndpointPlaysEverySegment(t *testing.T) {
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

	const rate = 8000
	dir := t.TempDir()
	seg0 := filepath.Join(dir, "over1.wav")
	seg1 := filepath.Join(dir, "over2.wav")
	writeTestWAV(t, seg0, rate, 4*rate, 1000)  // talker A: 4 s
	writeTestWAV(t, seg1, rate, 3*rate, -1000) // talker B: 3 s

	callStart := time.Now().UTC().Truncate(time.Microsecond)
	cs := trunking.CallStart{
		Grant:        trunking.Grant{System: "250_013", Protocol: "tetra", GroupID: 1020545, FrequencyHz: 467_912_500},
		DeviceSerial: "cc:same-carrier:1",
		StartedAt:    callStart,
	}
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	// Over 1 spans the call start → +6 s; over 2 starts 0.5 s after it ends.
	bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: trunking.CallComplete{
		Grant: cs.Grant, DeviceSerial: cs.DeviceSerial,
		StartedAt: callStart, EndedAt: callStart.Add(6 * time.Second),
		CallStartedAt: callStart, Segment: 0, AudioPath: seg0,
	}})
	bus.Publish(events.Event{Kind: events.KindCallComplete, Payload: trunking.CallComplete{
		Grant: cs.Grant, DeviceSerial: cs.DeviceSerial,
		StartedAt: callStart.Add(6500 * time.Millisecond), EndedAt: callStart.Add(30 * time.Second),
		CallStartedAt: callStart, Segment: 1, AudioPath: seg1,
	}})

	var id int64
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows, _ := db.History(context.Background(), storage.HistoryFilter{Limit: 1})
		if len(rows) == 1 && rows[0].HasRecording {
			if segs, _ := db.RecordingSegmentsByID(context.Background(), rows[0].ID); len(segs) == 2 {
				id = rows[0].ID
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if id == 0 {
		t.Fatal("setup: the two segments never attached to the call row")
	}

	base, teardown := mkServer(t, ServerOptions{Bus: bus, History: HistoryFromStorage(db)})
	defer teardown()
	resp := mustGet(t, base+"/api/v1/calls/"+strconv.FormatInt(id, 10)+"/audio")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "audio/wav" {
		t.Errorf("content-type = %q, want audio/wav", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) < 44 || string(body[:4]) != "RIFF" || string(body[8:12]) != "WAVE" {
		t.Fatalf("body is not a WAV (%d bytes)", len(body))
	}
	if got := binary.LittleEndian.Uint32(body[24:]); got != rate {
		t.Errorf("sample rate = %d, want %d", got, rate)
	}
	dataLen := int(binary.LittleEndian.Uint32(body[40:]))
	samples := dataLen / 2
	// 4 s + 0.5 s restored gap + 3 s. The old handler served over 1 alone (4 s).
	want := 4*rate + rate/2 + 3*rate
	if samples != want {
		t.Fatalf("samples = %d (%.2f s), want %d (%.2f s): every over must play",
			samples, float64(samples)/rate, want, float64(want)/rate)
	}
	pcm := body[44:]
	at := func(sec float64) int16 {
		return int16(binary.LittleEndian.Uint16(pcm[2*int(sec*rate):]))
	}
	if v := at(2); v != 1000 {
		t.Errorf("t=2s sample = %d, want talker A (1000)", v)
	}
	if v := at(4.25); v != 0 {
		t.Errorf("t=4.25s sample = %d, want restored silence (0)", v)
	}
	if v := at(6); v != -1000 {
		t.Errorf("t=6s sample = %d, want talker B (-1000)", v)
	}
}

// TestSegmentGapIsBounded: the restored inter-over silence follows the
// segments' own timestamps, falls back to a small default when they are
// missing or overlap, and is capped so a long idle never plays as dead air.
func TestSegmentGapIsBounded(t *testing.T) {
	t0 := time.Date(2026, 9, 10, 0, 15, 18, 0, time.UTC)
	seg := func(start, end time.Duration) RecordingSegment {
		s := RecordingSegment{StartedAt: t0.Add(start)}
		if end >= 0 {
			s.EndedAt = t0.Add(end)
		}
		return s
	}
	cases := []struct {
		name       string
		prev, next RecordingSegment
		want       time.Duration
	}{
		{"real short gap", seg(0, 6*time.Second), seg(6500*time.Millisecond, 30*time.Second), 500 * time.Millisecond},
		{"long idle capped", seg(0, 6*time.Second), seg(66*time.Second, 90*time.Second), segmentGapMax},
		{"unset ended_at", seg(0, -1), seg(6*time.Second, 30*time.Second), segmentGapDefault},
		{"overlapping spans", seg(0, 6*time.Second), seg(5*time.Second, 30*time.Second), segmentGapDefault},
		{"back to back", seg(0, 6*time.Second), seg(6*time.Second, 30*time.Second), 0},
	}
	for _, c := range cases {
		if got := segmentGap(c.prev, c.next); got != c.want {
			t.Errorf("%s: gap = %v, want %v", c.name, got, c.want)
		}
	}
}
