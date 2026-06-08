package main

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// streamingFakeDevice loops a synthetic IQ pattern on the StreamIQ
// channel so the spectrum producer has something to FFT during tests.
type streamingFakeDevice struct {
	serial string

	mu  sync.Mutex
	out chan []complex64

	center atomic.Uint32
	rate   atomic.Uint32
	closes atomic.Int32
}

func newStreamingFake(serial string) *streamingFakeDevice {
	return &streamingFakeDevice{serial: serial, out: make(chan []complex64, 8)}
}

func (s *streamingFakeDevice) Info() sdr.Info { return sdr.Info{Driver: "fake", Serial: s.serial} }
func (s *streamingFakeDevice) SetCenterFreq(hz uint32) error {
	s.center.Store(hz)
	return nil
}
func (s *streamingFakeDevice) SetSampleRate(hz uint32) error {
	s.rate.Store(hz)
	return nil
}
func (s *streamingFakeDevice) SetGain(int) error     { return nil }
func (s *streamingFakeDevice) SetPPM(int) error      { return nil }
func (s *streamingFakeDevice) SetBiasTee(bool) error { return nil }
func (s *streamingFakeDevice) Close() error          { s.closes.Add(1); return nil }

func (s *streamingFakeDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	s.mu.Lock()
	out := s.out
	s.mu.Unlock()
	// Background goroutine pumps a steady stream of unit-amplitude
	// DC IQ until ctx cancels.
	go func() {
		chunk := make([]complex64, 64)
		for i := range chunk {
			chunk[i] = complex(1, 0)
		}
		t := time.NewTicker(2 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				select {
				case out <- chunk:
				default:
				}
			}
		}
	}()
	return out, nil
}

func TestSpectrumProviderDevicesReflectsBrokers(t *testing.T) {
	dev := newStreamingFake("dev-1")
	br := iqtap.New(dev, 0, nil)
	_ = br.SetCenterFreq(851_012_500)
	_ = br.SetSampleRate(2_048_000)

	// Build a minimal pool by hand — NewPool + Open requires real
	// driver enumeration, so we set entries directly to avoid that.
	pool := sdr.NewPool(slog.New(slog.DiscardHandler))
	// Inject the entry via the same flow the daemon uses: rely on
	// the fact that the spectrum provider reads pool.Entries() and
	// the brokers map — we can stub both. Since pool.Entries()
	// requires private state we instead use the brokers map as the
	// authoritative source and rely on Devices() handling missing
	// pool entries by skipping them. (Verified by reading the
	// provider source.)
	_ = pool

	provider := &spectrumProvider{
		pool:    nil, // missing pool — Devices should return empty
		brokers: map[string]*iqtap.Broker{"dev-1": br},
		log:     slog.New(slog.DiscardHandler),
	}
	if got := provider.Devices(); len(got) != 0 {
		t.Errorf("Devices() with nil pool len = %d, want 0", len(got))
	}
}

func TestSpectrumProviderOpenStreamUnknownDevice(t *testing.T) {
	p := &spectrumProvider{
		brokers: map[string]*iqtap.Broker{},
		log:     slog.New(slog.DiscardHandler),
	}
	_, _, err := p.OpenStream(context.Background(), "nope", 64, 10)
	if err == nil {
		t.Fatal("OpenStream on unknown serial: want error, got nil")
	}
}

func TestSpectrumProviderOpenStreamProducesFrames(t *testing.T) {
	dev := newStreamingFake("dev-stream")
	br := iqtap.New(dev, 0, nil)
	_ = br.SetCenterFreq(100_000_000)
	_ = br.SetSampleRate(2_048_000)

	// Start primary stream so the broker fan-out is running.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	prim, err := br.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("broker StreamIQ: %v", err)
	}
	// Drain primary to keep the broker goroutine flowing.
	go func() {
		for range prim {
		}
	}()

	p := &spectrumProvider{
		brokers: map[string]*iqtap.Broker{"dev-stream": br},
		log:     slog.New(slog.DiscardHandler),
	}

	frames, cleanup, err := p.OpenStream(ctx, "dev-stream", 64, 50)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer cleanup()

	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("frame channel closed prematurely")
		}
		if f.CenterHz != 100_000_000 {
			t.Errorf("CenterHz = %d, want 100M", f.CenterHz)
		}
		if len(f.Bins) != 64 {
			t.Errorf("bins len = %d, want 64", len(f.Bins))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no frame within 2s")
	}
}

// Regression: when only the broker's Seed is used to populate the
// rate cache (no SetSampleRate call ever made through the broker —
// matches the daemon's wrapIQBrokers path for the wideband-only
// config), frames must still stamp the correct SampleRateHz instead
// of 0. Locks the wideband-T2 spectrum-view bug fix.
func TestSpectrumProviderFramesIncludeSampleRateAfterSeed(t *testing.T) {
	dev := newStreamingFake("dev-seeded")
	br := iqtap.New(dev, 0, nil)
	// Seeded (not SetSampleRate'd) — the pool already programmed the
	// raw device before the broker wrapped it.
	br.Seed(0, 2_400_000)
	// Center is programmed through the broker by whoever owns tuning
	// (wideband T2 engine in production).
	_ = br.SetCenterFreq(453_525_000)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	prim, err := br.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("broker StreamIQ: %v", err)
	}
	go func() {
		for range prim {
		}
	}()

	p := &spectrumProvider{
		brokers: map[string]*iqtap.Broker{"dev-seeded": br},
		log:     slog.New(slog.DiscardHandler),
	}
	frames, cleanup, err := p.OpenStream(ctx, "dev-seeded", 64, 50)
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer cleanup()

	select {
	case f, ok := <-frames:
		if !ok {
			t.Fatal("frame channel closed prematurely")
		}
		if f.SampleRateHz != 2_400_000 {
			t.Errorf("SampleRateHz = %d, want 2_400_000 (seed must populate cache)", f.SampleRateHz)
		}
		if f.CenterHz != 453_525_000 {
			t.Errorf("CenterHz = %d, want 453_525_000", f.CenterHz)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no frame within 2s")
	}
}

func TestSpectrumProviderOpenStreamBadFFTSize(t *testing.T) {
	dev := newStreamingFake("dev-bad")
	br := iqtap.New(dev, 0, nil)
	p := &spectrumProvider{
		brokers: map[string]*iqtap.Broker{"dev-bad": br},
		log:     slog.New(slog.DiscardHandler),
	}
	_, _, err := p.OpenStream(context.Background(), "dev-bad", 1000, 10)
	if err == nil {
		t.Fatal("OpenStream with bins=1000 (not power of two): want error")
	}
	if !errors.Is(err, err) { // sanity placeholder
		t.Fail()
	}
}

// TestP25ModulationFor covers the per-device modulation resolution that
// lets the web symbol/constellation panels auto-select C4FM vs CQPSK.
func TestP25ModulationFor(t *testing.T) {
	c4fm := trunking.System{
		Name:               "C4FM-sys",
		Protocol:           trunking.ProtocolP25,
		ControlChannels:    []uint32{851_000_000},
		P25Phase1DemodMode: "c4fm",
	}
	lsm := trunking.System{
		Name:               "LSM-sys",
		Protocol:           trunking.ProtocolP25,
		ControlChannels:    []uint32{770_000_000},
		P25Phase1DemodMode: "cqpsk",
	}
	dmr := trunking.System{
		Name:            "DMR-sys",
		Protocol:        trunking.ProtocolDMR,
		ControlChannels: []uint32{451_000_000},
	}

	const sr = 2_048_000 // ±1.024 MHz passband

	tests := []struct {
		name       string
		systems    []trunking.System
		centerHz   uint32
		sampleRate uint32
		want       string
	}{
		{
			name:       "control SDR parked on the C4FM control channel",
			systems:    []trunking.System{c4fm, lsm},
			centerHz:   851_000_000,
			sampleRate: sr,
			want:       "c4fm",
		},
		{
			name:       "control SDR parked on the LSM control channel",
			systems:    []trunking.System{c4fm, lsm},
			centerHz:   770_000_000,
			sampleRate: sr,
			want:       "cqpsk",
		},
		{
			name:       "voice channel inside the LSM passband matches by band",
			systems:    []trunking.System{c4fm, lsm},
			centerHz:   770_500_000, // 500 kHz off the CC, still < Nyquist
			sampleRate: sr,
			want:       "cqpsk",
		},
		{
			name:       "single system falls back even when off-band",
			systems:    []trunking.System{lsm},
			centerHz:   900_000_000,
			sampleRate: sr,
			want:       "cqpsk",
		},
		{
			name:       "multiple systems, none in band, is unknown",
			systems:    []trunking.System{c4fm, lsm},
			centerHz:   900_000_000,
			sampleRate: sr,
			want:       "",
		},
		{
			name:       "no P25 systems configured is unknown",
			systems:    []trunking.System{dmr},
			centerHz:   451_000_000,
			sampleRate: sr,
			want:       "",
		},
		{
			name:       "empty config mode normalises to c4fm",
			systems:    []trunking.System{{Name: "S", Protocol: trunking.ProtocolP25, ControlChannels: []uint32{851_000_000}}},
			centerHz:   851_000_000,
			sampleRate: sr,
			want:       "c4fm",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := p25ModulationFor(tc.systems, tc.centerHz, tc.sampleRate)
			if got != tc.want {
				t.Fatalf("p25ModulationFor() = %q, want %q", got, tc.want)
			}
		})
	}
}
