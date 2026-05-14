package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestStreamingWAVHeader_Shape(t *testing.T) {
	h := streamingWAVHeader(8000)
	if len(h) != 44 {
		t.Fatalf("header length=%d want 44", len(h))
	}
	if !bytes.Equal(h[0:4], []byte("RIFF")) {
		t.Errorf("magic=%q want RIFF", h[0:4])
	}
	if !bytes.Equal(h[8:12], []byte("WAVE")) {
		t.Errorf("type=%q want WAVE", h[8:12])
	}
	if !bytes.Equal(h[12:16], []byte("fmt ")) {
		t.Errorf("fmt id=%q", h[12:16])
	}
	if !bytes.Equal(h[36:40], []byte("data")) {
		t.Errorf("data id=%q", h[36:40])
	}
	if got := binary.LittleEndian.Uint16(h[20:22]); got != 1 {
		t.Errorf("audio format=%d want 1 (PCM)", got)
	}
	if got := binary.LittleEndian.Uint16(h[22:24]); got != 1 {
		t.Errorf("channels=%d want 1 (mono)", got)
	}
	if got := binary.LittleEndian.Uint32(h[24:28]); got != 8000 {
		t.Errorf("sample rate=%d want 8000", got)
	}
	if got := binary.LittleEndian.Uint16(h[34:36]); got != 16 {
		t.Errorf("bits per sample=%d want 16", got)
	}
}

func TestAudioStream_NotWired(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	base, teardown := mkServer(t, ServerOptions{Bus: bus})
	defer teardown()

	resp, err := http.Get(base + "/api/v1/audio/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status=%d want 503", resp.StatusCode)
	}
}

func TestAudioStream_EmitsWAVHeaderAndPCM(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	pub, err := NewAudioPublisher(bus, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = pub.Run(ctx) }()
	defer pub.Close()

	fa := newFakeAudio()
	base, teardown := mkServer(t, ServerOptions{
		Bus:            bus,
		Audio:          fa,
		AudioPublisher: pub,
	})
	defer teardown()

	reqCtx, reqCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer reqCancel()
	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, base+"/api/v1/audio/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "audio/wav" {
		t.Errorf("Content-Type=%q want audio/wav", ct)
	}

	// Read just the header — first 44 bytes should be the WAV envelope.
	header := make([]byte, 44)
	if _, err := io.ReadFull(resp.Body, header); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if !bytes.Equal(header[0:4], []byte("RIFF")) {
		t.Errorf("response magic=%q want RIFF", header[0:4])
	}
	if !bytes.Equal(header[8:12], []byte("WAVE")) {
		t.Errorf("response type=%q want WAVE", header[8:12])
	}

	// Push a CallStart so the publisher's grants map is populated
	// for the test device, then write some PCM and confirm it
	// reaches the stream.
	bus.Publish(events.Event{
		Kind:      events.KindCallStart,
		Timestamp: time.Now().UTC(),
		Payload: trunking.CallStart{
			DeviceSerial: "test-sdr",
			Grant: trunking.Grant{
				System:      "TEST",
				Protocol:    "p25",
				GroupID:     1234,
				FrequencyHz: 851012500,
			},
		},
	})
	// Give the publisher time to register the grant.
	time.Sleep(50 * time.Millisecond)

	samples := []int16{1, -1, 2, -2, 3, -3, 4, -4}
	for i := 0; i < 4; i++ {
		_ = pub.WritePCM("test-sdr", samples)
	}

	// Read what we can within a short window — the stream is
	// open-ended so we set a per-read deadline via the request
	// context above.
	got := make([]byte, 0, 256)
	buf := make([]byte, 256)
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
		}
		if err != nil {
			break
		}
		if len(got) >= len(samples)*2 {
			break
		}
	}
	if len(got) < 2 {
		t.Fatalf("did not receive any PCM bytes after header (got %d)", len(got))
	}
}

func TestParseAudioStreamFilter(t *testing.T) {
	q, _ := url.ParseQuery("device=AAA&device=BBB&talkgroup=100&talkgroup=200&talkgroup=notanumber")
	r := &http.Request{URL: &url.URL{RawQuery: q.Encode()}}
	f := parseAudioStreamFilter(r)
	if want := []string{"AAA", "BBB"}; !equalStrings(f.DeviceSerials, want) {
		t.Errorf("DeviceSerials=%v want %v", f.DeviceSerials, want)
	}
	if want := []uint32{100, 200}; !equalUint32s(f.TalkgroupIDs, want) {
		t.Errorf("TalkgroupIDs=%v want %v", f.TalkgroupIDs, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalUint32s(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Sanity-check: the routes() registration includes the stream URL so
// a future refactor that drops it surfaces in tests rather than at
// runtime.
func TestAudioStream_RouteRegistered(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	s := &Server{addr: "127.0.0.1:0", bus: bus, log: slog.Default()}
	mux := s.routes()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/audio/stream", nil)
	_, pattern := mux.Handler(req)
	if !strings.Contains(pattern, "/api/v1/audio/stream") {
		t.Errorf("audio stream route not registered (pattern=%q)", pattern)
	}
}
