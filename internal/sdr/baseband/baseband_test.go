package baseband

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// iqTone builds n IQ samples of a complex exponential.
func iqTone(n int) []complex64 {
	s := make([]complex64, n)
	for i := range s {
		ph := 2 * math.Pi * 0.05 * float64(i)
		s[i] = complex(float32(0.6*math.Cos(ph)), float32(0.6*math.Sin(ph)))
	}
	return s
}

func TestIQWriterRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cap.wav")
	samples := iqTone(4000)

	w, err := NewIQWriter(path, 2_400_000)
	if err != nil {
		t.Fatalf("NewIQWriter: %v", err)
	}
	if err := w.Write(samples); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	info, err := ReadIQWavInfo(path)
	if err != nil {
		t.Fatalf("ReadIQWavInfo: %v", err)
	}
	if info.SampleRate != 2_400_000 {
		t.Fatalf("sample rate = %d, want 2400000", info.SampleRate)
	}
	if info.Channels != 2 {
		t.Fatalf("channels = %d, want 2", info.Channels)
	}
	if info.Samples != len(samples) {
		t.Fatalf("samples = %d, want %d", info.Samples, len(samples))
	}
}

func TestIQWriterRejectsZeroRate(t *testing.T) {
	if _, err := NewIQWriter(filepath.Join(t.TempDir(), "x.wav"), 0); err == nil {
		t.Fatal("NewIQWriter with rate 0 should error")
	}
}

// fakeDevice is a minimal sdr.Device that emits a fixed chunk list.
type fakeDevice struct {
	info   sdr.Info
	rate   uint32
	chunks [][]complex64
}

func (f *fakeDevice) Info() sdr.Info             { return f.info }
func (f *fakeDevice) SetCenterFreq(uint32) error { return nil }
func (f *fakeDevice) SetGain(int) error          { return nil }
func (f *fakeDevice) SetPPM(int) error           { return nil }
func (f *fakeDevice) SetBiasTee(bool) error      { return nil }
func (f *fakeDevice) Close() error               { return nil }
func (f *fakeDevice) SetSampleRate(hz uint32) error {
	f.rate = hz
	return nil
}
func (f *fakeDevice) StreamIQ(ctx context.Context) (<-chan []complex64, error) {
	out := make(chan []complex64)
	go func() {
		defer close(out)
		for _, c := range f.chunks {
			select {
			case out <- c:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func TestRecordingDeviceTeesToWav(t *testing.T) {
	dir := t.TempDir()
	inner := &fakeDevice{
		info:   sdr.Info{Serial: "00000111"},
		chunks: [][]complex64{iqTone(1000), iqTone(1000), iqTone(500)},
	}
	rec := NewRecordingDevice(inner, dir, nil)
	if err := rec.SetSampleRate(2_400_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}

	ch, err := rec.StreamIQ(context.Background())
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	forwarded := 0
	for chunk := range ch {
		forwarded += len(chunk)
	}
	if forwarded != 2500 {
		t.Fatalf("forwarded %d samples, want 2500", forwarded)
	}

	// The recorder closes the WAV when the inner stream ends; give the
	// teardown goroutine a moment to flush.
	var info IQWavInfo
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		matches, _ := filepath.Glob(filepath.Join(dir, "00000111_*.wav"))
		if len(matches) == 1 {
			if i, err := ReadIQWavInfo(matches[0]); err == nil && i.Samples == 2500 {
				info = i
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if info.Samples != 2500 {
		t.Fatalf("recorded WAV has %d samples, want 2500", info.Samples)
	}
	if info.SampleRate != 2_400_000 {
		t.Fatalf("recorded WAV rate = %d, want 2400000", info.SampleRate)
	}
}

func TestFileDriverEnumerateAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replay.wav")
	w, _ := NewIQWriter(path, 2_400_000)
	w.Write(iqTone(20000))
	w.Close()

	drv := NewFileDriver([]ReplaySpec{{Path: path, Serial: "replay-test", Loop: false}})
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) != 1 || infos[0].Serial != "replay-test" {
		t.Fatalf("Enumerate = %+v", infos)
	}

	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	got := 0
	for chunk := range ch { // non-looping: channel closes at EOF
		got += len(chunk)
	}
	if got != 20000 {
		t.Fatalf("replayed %d samples, want 20000", got)
	}
}

func TestFileDriverLoops(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loop.wav")
	w, _ := NewIQWriter(path, 2_400_000)
	w.Write(iqTone(2000))
	w.Close()

	drv := NewFileDriver([]ReplaySpec{{Path: path, Loop: true}})
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	// A looping replay never closes the channel; collect more samples
	// than the file holds, proving it restarted.
	got := 0
	deadline := time.After(4 * time.Second)
	for got <= 2000 {
		select {
		case chunk, ok := <-ch:
			if !ok {
				t.Fatal("looping replay closed the channel")
			}
			got += len(chunk)
		case <-deadline:
			t.Fatalf("only %d samples before deadline, want > 2000", got)
		}
	}
	cancel()
}

func TestFileDriverOpenRejectsBadIndex(t *testing.T) {
	drv := NewFileDriver(nil)
	if _, err := drv.Open(0); err == nil {
		t.Fatal("Open on an empty driver should error")
	}
}
