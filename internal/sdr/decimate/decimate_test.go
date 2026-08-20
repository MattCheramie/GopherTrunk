package decimate

import (
	"context"
	"math"
	"math/cmplx"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// fakeDevice is a minimal sdr.Device that streams a caller-supplied IQ buffer in
// fixed-size chunks and records the last rate it was programmed at. It optionally
// exposes ActualSampleRate so the wrapper's division can be exercised.
type fakeDevice struct {
	rate        uint32
	chunks      [][]complex64
	actual      uint32 // 0 = don't implement ActualSampleRate
	freqRangeLo uint32
	freqRangeHi uint32
}

func (f *fakeDevice) Info() sdr.Info                { return sdr.Info{Serial: "fake"} }
func (f *fakeDevice) SetCenterFreq(hz uint32) error { return nil }
func (f *fakeDevice) SetSampleRate(hz uint32) error { f.rate = hz; return nil }
func (f *fakeDevice) SetGain(tenthDB int) error     { return nil }
func (f *fakeDevice) SetPPM(ppm int) error          { return nil }
func (f *fakeDevice) SetBiasTee(enable bool) error  { return nil }
func (f *fakeDevice) Close() error                  { return nil }
func (f *fakeDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	ch := make(chan []complex64)
	go func() {
		defer close(ch)
		for _, c := range f.chunks {
			select {
			case ch <- c:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// actualDevice adds the optional ActualSampleRate extension to fakeDevice.
type actualDevice struct{ *fakeDevice }

func (a actualDevice) ActualSampleRate() (uint32, error) { return a.actual, nil }

// freqRangeDevice adds the optional FreqRanger extension.
type freqRangeDevice struct{ *fakeDevice }

func (f freqRangeDevice) FreqRange() (uint32, uint32) { return f.freqRangeLo, f.freqRangeHi }

func TestSetSampleRateProgramsNative(t *testing.T) {
	fd := &fakeDevice{}
	d := New(fd, 4)
	if err := d.SetSampleRate(250_000); err != nil {
		t.Fatal(err)
	}
	if fd.rate != 1_000_000 {
		t.Errorf("inner programmed at %d Hz, want 1_000_000 (250k * 4)", fd.rate)
	}
	if d.outRate != 250_000 {
		t.Errorf("wrapper output rate = %d, want 250_000", d.outRate)
	}
}

func TestActualSampleRateDivides(t *testing.T) {
	// Inner reports its real native rate; the wrapper must report native / M.
	fd := &fakeDevice{actual: 10_000_000}
	d := New(actualDevice{fd}, 4)
	got, err := d.ActualSampleRate()
	if err != nil {
		t.Fatal(err)
	}
	if got != 2_500_000 {
		t.Errorf("ActualSampleRate = %d, want 2_500_000 (10 MS/s / 4)", got)
	}

	// Inner without ActualSampleRate falls back to the programmed output rate.
	fd2 := &fakeDevice{}
	d2 := New(fd2, 5)
	_ = d2.SetSampleRate(200_000)
	if got, _ := d2.ActualSampleRate(); got != 200_000 {
		t.Errorf("fallback ActualSampleRate = %d, want 200_000", got)
	}
}

func TestFreqRangeForwarded(t *testing.T) {
	fd := &fakeDevice{freqRangeLo: 70_000_000, freqRangeHi: 2_000_000_000}
	d := New(freqRangeDevice{fd}, 2)
	lo, hi := d.FreqRange()
	if lo != 70_000_000 || hi != 2_000_000_000 {
		t.Errorf("FreqRange = (%d,%d), want (70e6, 2e9)", lo, hi)
	}
	// A device without a range reports (0,0).
	if lo, hi := New(&fakeDevice{}, 2).FreqRange(); lo != 0 || hi != 0 {
		t.Errorf("FreqRange without ranger = (%d,%d), want (0,0)", lo, hi)
	}
}

func TestNewRejectsFactorLE1(t *testing.T) {
	for _, m := range []int{0, 1, -3} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("New(_, %d) did not panic", m)
				}
			}()
			New(&fakeDevice{}, m)
		}()
	}
}

// makeTone returns n samples of a unit complex tone at normalized frequency f
// (cycles/sample), chopped into chunks of chunkLen.
func makeTone(n, chunkLen int, f float64) [][]complex64 {
	all := make([]complex64, n)
	for i := range all {
		all[i] = complex64(cmplx.Exp(complex(0, 2*math.Pi*f*float64(i))))
	}
	var chunks [][]complex64
	for i := 0; i < n; i += chunkLen {
		end := i + chunkLen
		if end > n {
			end = n
		}
		chunks = append(chunks, all[i:end])
	}
	return chunks
}

func drain(t *testing.T, d *Device) []complex64 {
	t.Helper()
	ch, err := d.StreamIQ(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var out []complex64
	for c := range ch {
		out = append(out, c...)
	}
	return out
}

func avgPower(s []complex64) float64 {
	if len(s) == 0 {
		return 0
	}
	var sum float64
	for _, v := range s {
		sum += float64(real(v)*real(v) + imag(v)*imag(v))
	}
	return sum / float64(len(s))
}

func TestStreamDecimatesRate(t *testing.T) {
	const (
		m        = 4
		n        = 40_000
		chunkLen = 997 // deliberately not a multiple of m, to exercise carry-over
	)
	fd := &fakeDevice{chunks: makeTone(n, chunkLen, 0.01)}
	out := drain(t, New(fd, m))
	want := n / m
	// The polyphase resampler emits floor(n/m) ± a couple of samples of edge
	// slack; allow a small tolerance rather than pinning an exact count.
	if diff := out; len(diff) < want-4 || len(diff) > want+4 {
		t.Errorf("decimated %d samples, want ~%d (n/m)", len(out), want)
	}
}

// TestAntiAliasRejectsOutOfBand is the substance of the stage: a tone inside the
// retained band (|f| < 0.5/M of the native normalized frequency) passes with
// roughly unit power, while a tone well outside it — which would alias into the
// band without a filter — is attenuated to a small fraction of that power.
func TestAntiAliasRejectsOutOfBand(t *testing.T) {
	const (
		m        = 4
		n        = 60_000
		chunkLen = 1024
		settle   = 500 // drop the filter's start-up transient before measuring
	)
	// In band: 0.02 « 0.5/M = 0.125.
	inBand := drain(t, New(&fakeDevice{chunks: makeTone(n, chunkLen, 0.02)}, m))
	// Out of band: 0.35 » 0.125 (and not near a passband image).
	outBand := drain(t, New(&fakeDevice{chunks: makeTone(n, chunkLen, 0.35)}, m))
	if len(inBand) < settle*2 || len(outBand) < settle*2 {
		t.Fatalf("too few output samples (in=%d out=%d)", len(inBand), len(outBand))
	}
	pIn := avgPower(inBand[settle:])
	pOut := avgPower(outBand[settle:])
	// The passband tone keeps ~unit power; allow generous slack for window droop.
	if pIn < 0.5 {
		t.Errorf("in-band power %.4f too low (expected ~1.0)", pIn)
	}
	// The stopband tone must be crushed well below the passband — a real
	// anti-alias filter, not a bare drop-every-Mth-sample decimator (which would
	// alias 0.35 straight into the band at full power).
	if pOut > pIn/1000 {
		t.Errorf("out-of-band power %.6g not rejected (in-band %.4f, ratio %.4g; want < 1/1000)",
			pOut, pIn, pOut/pIn)
	}
}
