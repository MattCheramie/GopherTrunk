package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

// Issue #598 end-to-end: the live HTTP stream must deliver PCM bytes
// even when no CallStart grant was observed (the WebUI player opens an
// unfiltered stream and the grant cache rides a lossy bus). Mirrors
// TestAudioStream_EmitsWAVHeaderAndPCM but deliberately omits the
// CallStart publish.
func TestAudioStream_EmitsPCMWithoutCallStart(t *testing.T) {
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
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Audio: fa, AudioPublisher: pub})
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

	header := make([]byte, 44)
	if _, err := io.ReadFull(resp.Body, header); err != nil {
		t.Fatalf("read header: %v", err)
	}

	// No CallStart — write PCM straight away. Give the open stream a
	// moment to register its subscription, then drive samples.
	time.Sleep(50 * time.Millisecond)
	samples := []int16{1, -1, 2, -2, 3, -3, 4, -4}
	for i := 0; i < 4; i++ {
		_ = pub.WritePCM("unmapped-sdr", samples)
	}

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
		t.Fatalf("did not receive PCM bytes without a CallStart (got %d)", len(got))
	}
}

func TestParseRange(t *testing.T) {
	cases := []struct {
		name      string
		header    string
		wantStart int64
		wantEnd   int64
		wantOK    bool
	}{
		{"probe", "bytes=0-1", 0, 1, true},
		{"open ended", "bytes=0-", 0, -1, true},
		{"past header", "bytes=44-", 44, -1, true},
		{"bounded mid", "bytes=10-20", 10, 20, true},
		{"empty", "", 0, 0, false},
		{"suffix", "bytes=-100", 0, 0, false},
		{"multi range", "bytes=0-1,2-3", 0, 0, false},
		{"bad unit", "items=0-1", 0, 0, false},
		{"garbage", "bytes=abc", 0, 0, false},
		{"end before start", "bytes=20-10", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end, ok := parseRange(tc.header)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if start != tc.wantStart || end != tc.wantEnd {
				t.Errorf("got (%d,%d) want (%d,%d)", start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

// Safari probes the size with a small bounded Range. The endpoint must
// answer 206 with Accept-Ranges + Content-Range and the exact header
// bytes, then close — without a live subscription.
func TestAudioStream_ProbeRange(t *testing.T) {
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
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Audio: fa, AudioPublisher: pub})
	defer teardown()

	req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/audio/stream", nil)
	req.Header.Set("Range", "bytes=0-1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("status=%d want 206", resp.StatusCode)
	}
	if ar := resp.Header.Get("Accept-Ranges"); ar != "bytes" {
		t.Errorf("Accept-Ranges=%q want bytes", ar)
	}
	wantCR := "bytes 0-1/" + strconv.FormatInt(wavTotalSize, 10)
	if cr := resp.Header.Get("Content-Range"); cr != wantCR {
		t.Errorf("Content-Range=%q want %q", cr, wantCR)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(body, []byte("RI")) {
		t.Errorf("body=%q want first 2 header bytes %q", body, "RI")
	}
}

// openAudioStream starts a server with a live publisher and opens the
// stream with the given Range header ("" sends none). The response body
// and the server are torn down with the test.
func openAudioStream(t *testing.T, rangeHeader string) (*http.Response, *AudioPublisher) {
	t.Helper()
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)
	pub, err := NewAudioPublisher(bus, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = pub.Run(ctx) }()
	t.Cleanup(func() { pub.Close() })

	base, teardown := mkServer(t, ServerOptions{Bus: bus, Audio: newFakeAudio(), AudioPublisher: pub})
	t.Cleanup(teardown)

	reqCtx, reqCancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(reqCancel)
	req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, base+"/api/v1/audio/stream", nil)
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp, pub
}

// Issue #1405: the stream is live and has no length, so an open-ended
// Range, which every media player sends (VLC, mpv, ffmpeg and a
// browser's <audio> all open with `bytes=0-`), must be ignored. It used
// to be answered 206 with `Content-Range: bytes 0-2147483690/2147483691`,
// which told the player the stream is a seekable 2 GiB file. The reply
// must be the one a request with no Range header gets: 200, no
// Content-Range, no Accept-Ranges, the whole WAV header, then live PCM.
func TestAudioStream_OpenEndedRangeIsIgnored(t *testing.T) {
	cases := []struct{ name, header string }{
		{"no range header", ""},
		{"from the start", "bytes=0-"},
		{"from the data chunk", "bytes=44-"},   // ffmpeg's second request
		{"from inside the data", "bytes=100-"}, // a seek on the fake file
		{"from the fake end", "bytes=2147483691-"},
	}
	samples := []int16{1, -1, 2, -2, 3, -3, 4, -4}
	wantPCM := make([]byte, 2*len(samples))
	for i, v := range samples {
		binary.LittleEndian.PutUint16(wantPCM[2*i:], uint16(v))
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, pub := openAudioStream(t, tc.header)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status=%d want 200 (Content-Range=%q)", resp.StatusCode, resp.Header.Get("Content-Range"))
			}
			if cr := resp.Header.Get("Content-Range"); cr != "" {
				t.Errorf("Content-Range=%q: a live stream has no length to state", cr)
			}
			if ar := resp.Header.Get("Accept-Ranges"); ar != "" {
				t.Errorf("Accept-Ranges=%q: a live stream must not advertise ranges", ar)
			}
			if resp.ContentLength >= 0 {
				t.Errorf("Content-Length=%d: a live stream must not state a length", resp.ContentLength)
			}
			header := make([]byte, wavHeaderSize)
			if _, err := io.ReadFull(resp.Body, header); err != nil {
				t.Fatalf("read header: %v", err)
			}
			if want := streamingWAVHeader(8000); !bytes.Equal(header, want) {
				t.Fatalf("body starts %x, want the whole WAV header %x", header, want)
			}
			// The handler subscribes before it sends the header, so PCM
			// written now reaches this response.
			if err := pub.WritePCM("test-sdr", samples); err != nil {
				t.Fatal(err)
			}
			got := make([]byte, len(wantPCM))
			if _, err := io.ReadFull(resp.Body, got); err != nil {
				t.Fatalf("read PCM after the header: %v", err)
			}
			if !bytes.Equal(got, wantPCM) {
				t.Errorf("PCM=%x want %x", got, wantPCM)
			}
		})
	}
}

// The other half of #1405's contract: a BOUNDED range is unchanged. The
// header probe is pinned by TestAudioStream_ProbeRange; this pins the
// explicit range a client sends once the probe has told it the total.
func TestAudioStream_BoundedRangePastHeaderStillPartial(t *testing.T) {
	last := strconv.FormatInt(wavTotalSize-1, 10)
	resp, _ := openAudioStream(t, "bytes=0-"+last)
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("status=%d want 206", resp.StatusCode)
	}
	if ar := resp.Header.Get("Accept-Ranges"); ar != "bytes" {
		t.Errorf("Accept-Ranges=%q want bytes", ar)
	}
	wantCR := "bytes 0-" + last + "/" + strconv.FormatInt(wavTotalSize, 10)
	if cr := resp.Header.Get("Content-Range"); cr != wantCR {
		t.Errorf("Content-Range=%q want %q", cr, wantCR)
	}
	header := make([]byte, wavHeaderSize)
	if _, err := io.ReadFull(resp.Body, header); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if want := streamingWAVHeader(8000); !bytes.Equal(header, want) {
		t.Errorf("body starts %x, want the WAV header", header)
	}
}

// TestAudioStreamToneServer is a manual instrument, skipped unless
// GT_AUDIO_STREAM_SERVE is set to a number of seconds. It serves the real
// stream handler with a 1 kHz tone at real-time cadence and prints the
// URL, so a media player can be pointed at the endpoint without a radio:
//
//	GT_AUDIO_STREAM_SERVE=120 go test ./internal/api -run TestAudioStreamToneServer -v
//	vlc <url> ; mpv <url> ; ffmpeg -i <url> -t 3 -f null -
//
// GT_AUDIO_STREAM_BURST="3,4" sends three seconds of tone and then
// nothing for four, as the daemon sends nothing between calls.
func TestAudioStreamToneServer(t *testing.T) {
	secs, _ := strconv.Atoi(os.Getenv("GT_AUDIO_STREAM_SERVE"))
	if secs <= 0 {
		t.Skip("set GT_AUDIO_STREAM_SERVE=<seconds> to serve a test tone on the live stream")
	}
	var onS, offS float64
	if b := os.Getenv("GT_AUDIO_STREAM_BURST"); b != "" {
		if _, err := fmt.Sscanf(b, "%f,%f", &onS, &offS); err != nil || onS <= 0 || offS < 0 {
			t.Fatalf("GT_AUDIO_STREAM_BURST=%q: want \"<on seconds>,<off seconds>\"", b)
		}
	}
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
	base, teardown := mkServer(t, ServerOptions{Bus: bus, Audio: newFakeAudio(), AudioPublisher: pub})
	defer teardown()
	fmt.Fprintf(os.Stderr, "live stream with a test tone for %d s: %s/api/v1/audio/stream\n", secs, base)

	const rate, frame = 8000, 160 // 20 ms frames
	buf := make([]int16, frame)
	n := 0
	started := time.Now()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	end := time.After(time.Duration(secs) * time.Second)
	for {
		select {
		case <-end:
			return
		case <-tick.C:
			if onS > 0 && math.Mod(time.Since(started).Seconds(), onS+offS) >= onS {
				continue
			}
			for i := range buf {
				buf[i] = int16(8000 * math.Sin(2*math.Pi*1000*float64(n)/rate))
				n++
			}
			_ = pub.WritePCM("tone-sdr", buf)
		}
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
