package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/powersweep"
)

func TestParsePowerDuration(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"10":    10 * time.Second,
		"0.5":   500 * time.Millisecond,
		"250ms": 250 * time.Millisecond,
		"1m30s": 90 * time.Second,
	} {
		got, err := parsePowerDuration(in)
		if err != nil || got != want {
			t.Errorf("%q = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ten", "NaN"} {
		if _, err := parsePowerDuration(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// fakeRTLTCP is a minimal rtl_tcp server: it sends the 12-byte header, obeys
// set-frequency and set-sample-rate commands, and streams u8 IQ in real time
// holding one carrier at toneHz whenever it is inside the tuned band.
type fakeRTLTCP struct {
	ln     net.Listener
	toneHz float64

	mu     sync.Mutex
	freq   float64
	rate   float64
	tunes  []uint32
	closed chan struct{}
}

func newFakeRTLTCP(t *testing.T, toneHz float64) *fakeRTLTCP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeRTLTCP{ln: ln, toneHz: toneHz, rate: 2_048_000, closed: make(chan struct{})}
	go s.serve()
	t.Cleanup(func() { close(s.closed); ln.Close() })
	return s
}

func (s *fakeRTLTCP) serve() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	hdr := []byte{'R', 'T', 'L', '0', 0, 0, 0, 5, 0, 0, 0, 29} // R820T, 29 gains
	if _, err := conn.Write(hdr); err != nil {
		return
	}
	go func() { // commands: 1-byte op + big-endian uint32
		var pkt [5]byte
		for {
			if _, err := io.ReadFull(conn, pkt[:]); err != nil {
				return
			}
			v := binary.BigEndian.Uint32(pkt[1:])
			s.mu.Lock()
			switch pkt[0] {
			case 0x01:
				s.freq = float64(v)
				s.tunes = append(s.tunes, v)
			case 0x02:
				s.rate = float64(v)
			}
			s.mu.Unlock()
		}
	}()
	const tick = 5 * time.Millisecond
	var n int64
	buf := make([]byte, 0, 1<<16)
	for {
		select {
		case <-s.closed:
			return
		case <-time.After(tick):
		}
		s.mu.Lock()
		freq, rate := s.freq, s.rate
		s.mu.Unlock()
		off := s.toneHz - freq
		inBand := math.Abs(off) < rate/2
		buf = buf[:0]
		for i := 0; i < int(rate*tick.Seconds()); i++ {
			iv, qv := 0.0, 0.0
			if inBand {
				ph := 2 * math.Pi * off * float64(n) / rate
				iv, qv = 0.5*math.Cos(ph), 0.5*math.Sin(ph)
			}
			buf = append(buf, byte(math.Round(127.5+127.5*iv)), byte(math.Round(127.5+127.5*qv)))
			n++
		}
		if _, err := conn.Write(buf); err != nil {
			return
		}
	}
}

func (s *fakeRTLTCP) tuned() []uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]uint32(nil), s.tunes...)
}

// TestPowerSweepOverRTLTCP drives the #1230 path end to end: the real rtl_tcp
// driver against a server streaming one carrier, the CLI's settling IQ source,
// and the CSV writer. Each hop must be tuned on the server, and the CSV's
// loudest bin must sit at the carrier's frequency.
func TestPowerSweepOverRTLTCP(t *testing.T) {
	const (
		rate   = 240_000
		toneHz = 100_333_000.0
	)
	srv := newFakeRTLTCP(t, toneHz)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dev, name, err := openPowerDevice(srv.ln.Addr().String(), "", log)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	if !strings.HasPrefix(name, "rtl_tcp ") {
		t.Fatalf("device name %q", name)
	}
	if err := dev.SetSampleRate(rate); err != nil {
		t.Fatal(err)
	}
	r, _ := powersweep.ParseRange("100.0M:100.6M:2k")
	plan, err := powersweep.NewPlan(r, rate, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	src, err := newDeviceIQSource(ctx, dev, rate, log)
	if err != nil {
		t.Fatal(err)
	}
	src.settle = 60 * time.Millisecond

	var out bytes.Buffer
	w := bufio.NewWriter(&out)
	err = runPowerSweeps(ctx, src, plan, 20*plan.FFTSize, powerSchedule{every: time.Second, single: true},
		w, time.Now, sleepCtx)
	if err != nil {
		t.Fatal(err)
	}

	got := srv.tuned()
	if len(got) != len(plan.Hops) {
		t.Fatalf("server saw %d tunes %v, want one per hop (%d)", len(got), got, len(plan.Hops))
	}
	for i, h := range plan.Hops {
		if got[i] != h.CenterHz {
			t.Fatalf("tune %d = %d Hz, want %d", i, got[i], h.CenterHz)
		}
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != len(plan.Hops) {
		t.Fatalf("%d CSV lines, want %d:\n%s", len(lines), len(plan.Hops), out.String())
	}
	bestDB, bestHz := -999.0, 0.0
	for _, line := range lines {
		f := strings.Split(line, ", ")
		low, _ := strconv.ParseFloat(f[2], 64)
		step, _ := strconv.ParseFloat(f[4], 64)
		for j, v := range f[6:] {
			db, err := strconv.ParseFloat(v, 64)
			if err != nil {
				t.Fatalf("bad dB %q in %q", v, line)
			}
			if db > bestDB {
				bestDB, bestHz = db, low+float64(j)*step
			}
		}
	}
	if math.Abs(bestHz-toneHz) > plan.BinHz {
		t.Fatalf("loudest bin at %.0f Hz, want within %.0f Hz of %.0f", bestHz, plan.BinHz, toneHz)
	}
	// A 0.5-amplitude carrier is about −6 dBFS (−8..−9.5 dBFS through the Hann window).
	if bestDB < -11 || bestDB > -6 {
		t.Fatalf("loudest bin %.1f dBFS, want about −8", bestDB)
	}
}

// TestRunPowerSweepsHonoursExitTimer: sweeps repeat every interval and stop
// once the next one would start after the exit time (rtl_power's -e).
func TestRunPowerSweepsHonoursExitTimer(t *testing.T) {
	r, _ := powersweep.ParseRange("100M:100.1M:10k")
	plan, err := powersweep.NewPlan(r, 240_000, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	sleep := func(_ context.Context, d time.Duration) error { clock = clock.Add(d); return nil }
	src := &silentSource{}
	var out bytes.Buffer
	w := bufio.NewWriter(&out)
	err = runPowerSweeps(context.Background(), src, plan, plan.FFTSize,
		powerSchedule{every: 10 * time.Second, stopAfter: 35 * time.Second}, w, now, sleep)
	if err != nil {
		t.Fatal(err)
	}
	// Sweeps at 0, 10, 20 and 30 s; the one at 40 s is past the 35 s exit.
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if want := 4 * len(plan.Hops); len(lines) != want {
		t.Fatalf("%d lines, want %d", len(lines), want)
	}
	if !strings.HasPrefix(lines[len(lines)-1], "2026-10-03, 12:00:30, ") {
		t.Fatalf("last sweep stamped %q, want 12:00:30", lines[len(lines)-1][:22])
	}
}

type silentSource struct{}

func (silentSource) Tune(uint32) error { return nil }
func (silentSource) Capture(_ context.Context, n int) ([]complex64, error) {
	return make([]complex64, n), nil
}
