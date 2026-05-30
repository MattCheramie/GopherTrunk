package wbvoice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iqtap"
)

// fakeDevice is the minimal sdr.Device we need to wire an iqtap.Broker
// around. The broker only calls Info / StreamIQ / Close on Forward
// paths; the rest are inert.
type fakeDevice struct {
	info   sdr.Info
	stream chan []complex64
	err    error
}

func newFakeDevice() *fakeDevice {
	return &fakeDevice{
		info:   sdr.Info{Driver: "fake", Serial: "FAKE"},
		stream: make(chan []complex64, 16),
	}
}

func (f *fakeDevice) Info() sdr.Info             { return f.info }
func (f *fakeDevice) SetCenterFreq(uint32) error { return nil }
func (f *fakeDevice) SetSampleRate(uint32) error { return nil }
func (f *fakeDevice) SetGain(int) error          { return nil }
func (f *fakeDevice) SetPPM(int) error           { return nil }
func (f *fakeDevice) SetBiasTee(bool) error      { return nil }
func (f *fakeDevice) SetAmp(bool) error          { return nil }
func (f *fakeDevice) Close() error               { return nil }
func (f *fakeDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.stream, nil
}

func TestNewValidatesInputs(t *testing.T) {
	broker := iqtap.New(newFakeDevice(), 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	cases := map[string]Options{
		"missing broker": {Serial: "x", WidebandCenterHz: 851_000_000, SDRSampleRateHz: 2_400_000},
		"missing center": {Serial: "x", Broker: broker, SDRSampleRateHz: 2_400_000},
		"missing rate":   {Serial: "x", Broker: broker, WidebandCenterHz: 851_000_000},
		"missing serial": {Broker: broker, WidebandCenterHz: 851_000_000, SDRSampleRateHz: 2_400_000},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(opts); err == nil {
				t.Errorf("New(%+v) = nil err, want one", opts)
			}
		})
	}
}

func TestSetCenterFreqAcceptsInBand(t *testing.T) {
	broker := iqtap.New(newFakeDevice(), 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	v, err := New(Options{
		Serial: "tap-0", Broker: broker,
		WidebandCenterHz: 851_500_000, SDRSampleRateHz: 2_400_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Edge of usable band = ± 2.4 MHz / 2 × 0.95 = ± 1,140,000 Hz.
	// Pick 851,500,000 + 1,000,000 = 852,500,000 (inside).
	if err := v.SetCenterFreq(852_500_000); err != nil {
		t.Errorf("in-band SetCenterFreq returned %v", err)
	}
}

func TestSetCenterFreqRejectsOutOfBand(t *testing.T) {
	broker := iqtap.New(newFakeDevice(), 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	v, err := New(Options{
		Serial: "tap-0", Broker: broker,
		WidebandCenterHz: 851_500_000, SDRSampleRateHz: 2_400_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 1.5 MHz away — outside ±1.14 MHz usable band.
	if err := v.SetCenterFreq(853_000_000); !errors.Is(err, ErrOutOfBand) {
		t.Errorf("out-of-band SetCenterFreq returned %v, want ErrOutOfBand", err)
	}
}

func TestStreamIQRequiresSetCenterFreqFirst(t *testing.T) {
	broker := iqtap.New(newFakeDevice(), 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	v, _ := New(Options{
		Serial: "tap-0", Broker: broker,
		WidebandCenterHz: 851_500_000, SDRSampleRateHz: 2_400_000,
	})
	if _, err := v.StreamIQ(context.Background()); err == nil {
		t.Errorf("StreamIQ before SetCenterFreq returned nil err, want one")
	}
}

func TestSampleRateHzReports48k(t *testing.T) {
	broker := iqtap.New(newFakeDevice(), 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	v, _ := New(Options{
		Serial: "tap-0", Broker: broker,
		WidebandCenterHz: 851_500_000, SDRSampleRateHz: 2_400_000,
	})
	if got := v.SampleRateHz(); got != NarrowbandRateHz {
		t.Errorf("SampleRateHz = %d, want %d", got, NarrowbandRateHz)
	}
}

// TestStreamIQShiftsCarrierToBaseband injects a complex sinusoid at the
// target offset through a fake broker; the virtual tuner should
// down-convert it to a near-DC tone after its DDC. Verifies that the
// NCO mix + decimation actually produces 48 kHz IQ centred on the
// SetCenterFreq target.
func TestStreamIQShiftsCarrierToBaseband(t *testing.T) {
	const sdrRate = 2_400_000.0
	const widebandHz = 851_500_000
	const targetHz = 852_000_000 // 500 kHz offset
	const offsetHz = targetHz - widebandHz

	fake := newFakeDevice()
	broker := iqtap.New(fake, 64, slog.New(slog.NewTextHandler(io.Discard, nil)))

	v, err := New(Options{
		Serial: "tap-0", Broker: broker,
		WidebandCenterHz: widebandHz, SDRSampleRateHz: sdrRate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := v.SetCenterFreq(targetHz); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Drive the broker's primary stream so the fanout to our
	// subscriber actually runs. We don't care about the primary
	// channel content — discard it.
	primary, err := broker.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for range primary {
		}
	}()

	iqCh, err := v.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Generate a continuous complex tone at +offsetHz against the
	// SDR rate, broken into chunks. Phase accumulates across chunks
	// to keep the tone coherent (a chunk-boundary phase jump would
	// broaden the spectrum and mask the DDC's accuracy).
	const chunkN = 4096
	const numChunks = 40 // 40 × 4096 = ~163 k input samples → ~3.2 k output samples
	step := 2 * math.Pi * float64(offsetHz) / sdrRate
	phase := 0.0
	var (
		mu        sync.Mutex
		all       []complex64
		collected = make(chan struct{})
	)
	const wantSamples = 2048
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case out, ok := <-iqCh:
				if !ok {
					return
				}
				mu.Lock()
				all = append(all, out...)
				done := len(all) >= wantSamples
				mu.Unlock()
				if done {
					select {
					case <-collected:
					default:
						close(collected)
					}
					return
				}
			}
		}
	}()

	for i := 0; i < numChunks; i++ {
		chunk := make([]complex64, chunkN)
		for j := range chunk {
			chunk[j] = complex64(complex(math.Cos(phase), math.Sin(phase)))
			phase += step
		}
		select {
		case fake.stream <- chunk:
		case <-time.After(2 * time.Second):
			t.Fatal("fake stream send blocked")
		}
	}

	select {
	case <-collected:
	case <-time.After(3 * time.Second):
		mu.Lock()
		got := len(all)
		mu.Unlock()
		t.Fatalf("collected only %d output samples within deadline, want %d", got, wantSamples)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(all) < 1024 {
		t.Fatalf("only %d output samples, want >= 1024", len(all))
	}
	// Skip the resampler warmup (first half) and analyse the
	// settled second half — exactly the pattern used by the
	// upstream tuner.DDCBank tests.
	settled := all[len(all)/2:]
	frac := powerNearDC(settled, NarrowbandRateHz, 500)
	if frac < 0.95 {
		t.Errorf("post-DDC power near DC = %.1f%%, want >= 95%% (carrier not shifted to baseband)", frac*100)
	}
}

// powerNearDC reports the fraction of total energy within ±halfWidthHz
// of DC. Mirrors the helper in internal/dsp/tuner/ddc_test.go so the
// virtual-tuner check uses the same spectral metric the DDC suite
// already relies on.
func powerNearDC(samples []complex64, sampleRateHz, halfWidthHz float64) float64 {
	N := len(samples)
	if N < 8 {
		return 0
	}
	binHz := sampleRateHz / float64(N)
	maxK := int(math.Ceil(halfWidthHz / binHz))
	totalPow := 0.0
	dcPow := 0.0
	for k := -N / 2; k < N/2; k++ {
		var sumR, sumI float64
		w := -2 * math.Pi * float64(k) / float64(N)
		for i, s := range samples {
			theta := w * float64(i)
			c := math.Cos(theta)
			si := math.Sin(theta)
			sumR += float64(real(s))*c - float64(imag(s))*si
			sumI += float64(real(s))*si + float64(imag(s))*c
		}
		p := sumR*sumR + sumI*sumI
		totalPow += p
		if k >= -maxK && k <= maxK {
			dcPow += p
		}
	}
	if totalPow == 0 {
		return 0
	}
	return dcPow / totalPow
}
