package rtltcp

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeServer mimics a librtlsdr rtl_tcp server: sends the 12-byte
// magic header on Accept, streams whatever IQ bytes the test feeds
// onto Feed, and records every 5-byte command packet the client
// sends so tests can assert SetCenterFreq / SetGain etc.
type fakeServer struct {
	t  *testing.T
	ln net.Listener

	mu   sync.Mutex
	cmds []cmdRecord
	feed chan []byte
	conn net.Conn

	tunerType uint32
	gainCount uint32
}

type cmdRecord struct {
	Op    uint8
	Param uint32
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	s := &fakeServer{
		t:         t,
		ln:        ln,
		feed:      make(chan []byte, 64),
		tunerType: 5, // R820T
		gainCount: 29,
	}
	go s.run()
	t.Cleanup(s.close)
	return s
}

func (s *fakeServer) Addr() string { return s.ln.Addr().String() }

func (s *fakeServer) Feed(b []byte) {
	s.feed <- b
}

func (s *fakeServer) Commands() []cmdRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]cmdRecord, len(s.cmds))
	copy(out, s.cmds)
	return out
}

func (s *fakeServer) close() {
	_ = s.ln.Close()
	s.mu.Lock()
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.mu.Unlock()
}

func (s *fakeServer) run() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	// Send the 12-byte header.
	var hdr [12]byte
	copy(hdr[:4], Magic[:])
	binary.BigEndian.PutUint32(hdr[4:8], s.tunerType)
	binary.BigEndian.PutUint32(hdr[8:12], s.gainCount)
	if _, err := conn.Write(hdr[:]); err != nil {
		return
	}

	// Reader goroutine: log every 5-byte command the client sends.
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 5)
		for {
			if _, err := io.ReadFull(conn, buf); err != nil {
				return
			}
			s.mu.Lock()
			s.cmds = append(s.cmds, cmdRecord{
				Op:    buf[0],
				Param: binary.BigEndian.Uint32(buf[1:]),
			})
			s.mu.Unlock()
		}
	}()

	// Writer loop: drain fed IQ onto the wire.
	for {
		select {
		case <-done:
			return
		case body, ok := <-s.feed:
			if !ok {
				_ = conn.Close()
				return
			}
			if _, err := conn.Write(body); err != nil {
				return
			}
		}
	}
}

func newDriver(t *testing.T, addr string) *Driver {
	t.Helper()
	return New([]Spec{{Addr: addr, Serial: "test-remote"}}, nil)
}

func TestEnumerateReportsConfiguredSpecs(t *testing.T) {
	d := New([]Spec{
		{Addr: "10.0.0.5:1234"},
		{Addr: "10.0.0.6:1234", Serial: "kitchen-pi"},
		{Addr: "", Serial: "skip-empty-addr"}, // skipped
	}, nil)
	infos, err := d.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("len(infos) = %d, want 2", len(infos))
	}
	if infos[1].Serial != "kitchen-pi" {
		t.Errorf("infos[1].Serial = %q, want kitchen-pi", infos[1].Serial)
	}
	if infos[0].Serial == "" {
		t.Error("infos[0].Serial is empty — generator should have filled it")
	}
	if infos[0].Driver != DriverName {
		t.Errorf("infos[0].Driver = %q, want %q", infos[0].Driver, DriverName)
	}
}

func TestOpenAndCloseAgainstFakeServer(t *testing.T) {
	srv := newFakeServer(t)

	d := newDriver(t, srv.Addr())
	dev, err := d.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if dev.Info().Serial != "test-remote" {
		t.Errorf("Info().Serial = %q", dev.Info().Serial)
	}
	if dev.Info().TunerName != "R820T" {
		t.Errorf("Info().TunerName = %q, want R820T", dev.Info().TunerName)
	}
	if err := dev.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestOpenRejectsNonMagicServer(t *testing.T) {
	// A listener that immediately writes 12 bogus bytes.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte("HTTP/1.1 200")) // 12 bytes
	}()

	d := newDriver(t, ln.Addr().String())
	_, err = d.Open(0)
	if err == nil {
		t.Fatal("Open against bogus server: want error, got nil")
	}
	if !errInString(err, "magic") {
		t.Errorf("Open err = %v, want a magic-mismatch error", err)
	}
}

func TestOpenDialTimeout(t *testing.T) {
	// 192.0.2.0/24 is reserved for documentation (RFC 5737) and
	// reliably non-routable, so the dial hangs until we cap it.
	d := New([]Spec{{
		Addr:           "192.0.2.1:1234",
		ConnectTimeout: 200 * time.Millisecond,
	}}, nil)
	start := time.Now()
	_, err := d.Open(0)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Open of unroutable address: want error, got nil")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Open took %v, expected ≤ 2s thanks to ConnectTimeout", elapsed)
	}
}

func TestCommandsAreEncodedCorrectly(t *testing.T) {
	srv := newFakeServer(t)
	d := newDriver(t, srv.Addr())
	dev, err := d.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	if err := dev.SetCenterFreq(851_012_500); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if err := dev.SetSampleRate(2_048_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if err := dev.SetGain(496); err != nil {
		t.Fatalf("SetGain(manual): %v", err)
	}
	if err := dev.SetGain(-1); err != nil {
		t.Fatalf("SetGain(auto): %v", err)
	}
	if err := dev.SetPPM(42); err != nil {
		t.Fatalf("SetPPM: %v", err)
	}
	if err := dev.SetBiasTee(true); err != nil {
		t.Fatalf("SetBiasTee: %v", err)
	}

	// Allow the reader goroutine to drain.
	waitFor(t, 500*time.Millisecond, func() bool {
		return len(srv.Commands()) >= 8
	})

	want := []cmdRecord{
		{cmdSetFreq, 851_012_500},
		{cmdSetSampleRate, 2_048_000},
		{cmdSetGainMode, 1}, // manual mode preamble
		{cmdSetGain, 496},   // gain value
		{cmdSetGainMode, 0}, // auto
		{cmdSetFreqCorrPPM, uint32(int32(42))},
		{cmdSetBiasTee, 1},
	}
	got := srv.Commands()
	if len(got) < len(want) {
		t.Fatalf("got %d commands, want at least %d (%+v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("cmd[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestStreamIQDeliversConvertedSamples(t *testing.T) {
	srv := newFakeServer(t)
	d := newDriver(t, srv.Addr())
	dev, err := d.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	// Feed one chunk's worth of zero-IQ bytes (127, 128 = ~DC, the
	// midpoint of the u8 range).
	const chunkSamples = 8192
	body := make([]byte, chunkSamples*2)
	for i := 0; i < chunkSamples; i++ {
		body[2*i] = 127
		body[2*i+1] = 128
	}
	srv.Feed(body)

	select {
	case samples, ok := <-ch:
		if !ok {
			t.Fatal("stream closed before sample arrived")
		}
		if len(samples) != chunkSamples {
			t.Errorf("samples len = %d, want %d", len(samples), chunkSamples)
		}
		// 127 → -0.5/127.5 ≈ -0.00392; 128 → 0.5/127.5 ≈ +0.00392.
		// Both well below 0.01 — the DC-midpoint sanity bound.
		got := samples[0]
		if absF(real(got)) > 0.01 || absF(imag(got)) > 0.01 {
			t.Errorf("DC-midpoint sample = %v, want |re|,|im| ≤ 0.01", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no samples received within 2s")
	}
}

func TestStreamIQEndsOnServerClose(t *testing.T) {
	srv := newFakeServer(t)
	d := newDriver(t, srv.Addr())
	dev, err := d.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	// Close the server end without sending any IQ; the stream
	// goroutine must drain and close out.
	srv.close()

	gotClose := false
	var received atomic.Int32
	deadline := time.After(2 * time.Second)
loop:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				gotClose = true
				break loop
			}
			received.Add(1)
		case <-deadline:
			break loop
		}
	}
	if !gotClose {
		t.Errorf("stream channel did not close after server hang-up (received %d chunks)", received.Load())
	}
}

// A quiet server is the normal state at shutdown, and the read loop only
// checked ctx BETWEEN reads — so a cancel arriving mid-read was not seen until
// the read deadline expired, putting a 30 s tail on every daemon stop that had
// an rtl_tcp source attached.
func TestStreamIQStopsPromptlyOnContextCancelWhileServerIsQuiet(t *testing.T) {
	srv := newFakeServer(t)
	d := newDriver(t, srv.Addr())
	dev, err := d.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	// Give the reader a moment to block in ReadFull on a server that will
	// never send anything.
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unexpected samples from a quiet server")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not close within 5 s of ctx cancel — the read is " +
			"blocked until its deadline expires")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("stream took %v to close after cancel, want < 1s", elapsed)
	}
}

func TestSetterAfterCloseFailsCleanly(t *testing.T) {
	srv := newFakeServer(t)
	d := newDriver(t, srv.Addr())
	dev, err := d.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := dev.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := dev.SetCenterFreq(100_000_000); err == nil {
		t.Error("SetCenterFreq after Close: want error, got nil")
	}
}

func TestSerialDefaultsAreUniqueAcrossSpecs(t *testing.T) {
	d := New([]Spec{
		{Addr: "10.0.0.1:1234"},
		{Addr: "10.0.0.2:1234"},
	}, nil)
	infos, _ := d.Enumerate()
	if infos[0].Serial == infos[1].Serial {
		t.Errorf("auto-serials collided: %q == %q", infos[0].Serial, infos[1].Serial)
	}
}

// waitFor polls cond up to d. Helper for fakeServer command reads
// arriving on a background goroutine.
func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func absF(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func errInString(err error, sub string) bool {
	if err == nil {
		return false
	}
	for {
		if containsCI(err.Error(), sub) {
			return true
		}
		next := errors.Unwrap(err)
		if next == nil {
			return false
		}
		err = next
	}
}

func containsCI(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			a := s[i+j]
			b := sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
