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

	"github.com/MattCheramie/GopherTrunk/internal/dsp/tuner"
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
	// At an exact-divisor input rate the fractional accessor agrees exactly.
	if got := v.SampleRateExactHz(); got != float64(NarrowbandRateHz) {
		t.Errorf("SampleRateExactHz = %v, want %v", got, float64(NarrowbandRateHz))
	}
}

// TestSampleRateExactHzMatchesDDCBank verifies the virtual tuner reports the
// DDC's *actual* per-tap output rate (fractional part preserved), not the
// rounded nominal 48 kHz, so the composer can clock the voice symbol-recovery
// loop at the true rate. 390625 Hz (a 6.25 MS/s ÷16 polyphase bin rate) is the
// issue #550 case: 48000/390625 reduces to L=384/M=3125, trips the resampler's
// L≤64 cap, and the bank lands a fraction of a percent off 48 kHz — the exact
// regime where clocking the receiver at a rounded 48000 would slip symbols and
// produce voice spikes/glitches.
func TestSampleRateExactHzMatchesDDCBank(t *testing.T) {
	const fractionalRate = 390_625
	broker := iqtap.New(newFakeDevice(), 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	v, err := New(Options{
		Serial: "tap-0", Broker: broker,
		WidebandCenterHz: 851_500_000, SDRSampleRateHz: fractionalRate,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Must equal exactly what a bank built with the StreamIQ args reports —
	// the rate the live stream is actually clocked at.
	want := tuner.NewDDCBank(float64(fractionalRate), float64(NarrowbandRateHz), GuardFrac).OutputRateHz()
	if got := v.SampleRateExactHz(); got != want {
		t.Errorf("SampleRateExactHz = %v, want %v (DDCBank.OutputRateHz)", got, want)
	}
	// And it must be the fractional rate, not the nominal target: a symbol
	// clock built from 48000 here would drift off the true symbol phase.
	if want == float64(NarrowbandRateHz) {
		t.Fatalf("test input rate %d did not produce a fractional output rate; pick one that trips the resampler caps", fractionalRate)
	}
	if got := v.SampleRateExactHz(); got == float64(NarrowbandRateHz) {
		t.Errorf("SampleRateExactHz = %v, want a fractional rate ≠ %d", got, NarrowbandRateHz)
	}
	// The rounded integer accessor still rounds to the nearest Hz.
	if got, want := v.SampleRateHz(), uint32(math.Round(want)); got != want {
		t.Errorf("SampleRateHz = %d, want %d (rounded actual rate)", got, want)
	}
}

// TestStreamIQAcceptsWidebandSpanGrant is the regression for the field report
// (heavy-overrun DMR archive, 26 Jun): a 71-channel wideband DMR plan on a
// 10 MS/s capture has voice grants out to ±1.9 MHz of centre. CanTune already
// advertises the full ±4.5 MHz IQ window, so the trunking engine binds those
// grants — but StreamIQ used to build its per-call DDC with the no-span
// NewDDCBank, whose shared decimator floors the reduced rate at 2.5 MS/s and so
// rejects any offset beyond ±1.1 MHz with ErrOffsetOutOfBand. The outer
// carriers then produced no audio at all ("composer: StreamIQ failed ... offset
// is outside the usable IQ band"). The span-aware bank (mirroring widebandt2's
// maxAbsOffset sizing) must accept the grant, and the reported exact rate must
// match the bank the live stream is actually clocked at.
func TestStreamIQAcceptsWidebandSpanGrant(t *testing.T) {
	const sdrRate = 10_000_000.0
	const widebandHz = 441_700_000
	const targetHz = 443_600_000 // +1.9 MHz, an outer carrier of the reported plan
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

	// CanTune / SetCenterFreq accept it (advertised ±4.5 MHz window).
	if !v.CanTune(targetHz) {
		t.Fatalf("CanTune(%d) = false, want true (offset %.0f Hz inside ±%.0f Hz)",
			targetHz, float64(offsetHz), sdrRate*(0.5-GuardFrac))
	}
	if err := v.SetCenterFreq(targetHz); err != nil {
		t.Fatalf("SetCenterFreq(%d) = %v, want nil", targetHz, err)
	}

	// The crux: StreamIQ must build a DDC that keeps this offset in band.
	// Before the fix it returned a wrapped ErrOffsetOutOfBand here.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := v.StreamIQ(ctx); err != nil {
		t.Fatalf("StreamIQ at offset %.0f Hz = %v, want nil (outer wideband grant must tune)", float64(offsetHz), err)
	}

	// The reported exact rate must equal the span-aware bank's output rate —
	// the rate the live stream is clocked at — so symbol recovery doesn't slip
	// (issue #550).
	want := tuner.NewDDCBankForSpan(sdrRate, float64(NarrowbandRateHz), GuardFrac, math.Abs(offsetHz)).OutputRateHz()
	if got := v.SampleRateExactHz(); got != want {
		t.Errorf("SampleRateExactHz = %v, want %v (span-aware DDCBank.OutputRateHz)", got, want)
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
