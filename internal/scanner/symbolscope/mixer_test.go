package symbolscope

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// argmax returns the index of the largest value in bins.
func argmax(bins []float32) int {
	best, bestI := float32(math.Inf(-1)), 0
	for i, v := range bins {
		if v > best {
			best, bestI = v, i
		}
	}
	return bestI
}

// TestMixerRawAndTunedPeaks feeds a pure off-centre complex tone and
// checks that the raw spectrum peaks at the tone's frequency while the
// tuned spectrum — re-mixed by a carrier-offset estimate equal to the
// tone — peaks at DC (centre bin). This is the core Mixer-plot behaviour:
// a locked loop pulls the carrier back to centre.
func TestMixerRawAndTunedPeaks(t *testing.T) {
	const (
		sampleHz = 48000.0
		fftSize  = 1024
		toneHz   = 5000.0
	)

	var frames []MixerFrame
	m, err := newMixerAccum(mixerOptions{
		fftSize:  fftSize,
		fps:      1000, // generous; we drive emit via nowNs below
		sampleHz: sampleHz,
		centerHz: 851_000_000,
		offsetHz: 0,
		nowNs:    func() int64 { return 0 },
		emit:     func(f MixerFrame) { frames = append(frames, f) },
	})
	if err != nil {
		t.Fatalf("newMixerAccum: %v", err)
	}

	// One full FFT window of a unit complex tone at +toneHz.
	tone := make([]complex64, fftSize)
	w := 2 * math.Pi * toneHz / sampleHz
	for n := 0; n < fftSize; n++ {
		c := cmplx.Rect(1, w*float64(n))
		tone[n] = complex64(c)
	}

	// Feed with a carrier-offset estimate equal to the tone, so the tuned
	// re-mix should centre it.
	m.feed(tone, toneHz)

	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(frames))
	}
	f := frames[0]
	if len(f.RawBins) != fftSize || len(f.TunedBins) != fftSize {
		t.Fatalf("bin lengths = (%d,%d), want %d", len(f.RawBins), len(f.TunedBins), fftSize)
	}
	if f.SampleRateHz != uint32(sampleHz) {
		t.Errorf("SampleRateHz = %d, want %d", f.SampleRateHz, uint32(sampleHz))
	}

	// Expected raw peak bin: FFT-shifted, centre (DC) at fftSize/2, the
	// tone sitting toneHz above it.
	center := fftSize / 2
	wantRaw := center + int(math.Round(toneHz*fftSize/sampleHz))
	rawPeak := argmax(f.RawBins)
	if abs(rawPeak-wantRaw) > 2 {
		t.Errorf("raw peak bin = %d, want ~%d", rawPeak, wantRaw)
	}

	// Tuned peak should land on the centre bin (within a bin or two).
	tunedPeak := argmax(f.TunedBins)
	if abs(tunedPeak-center) > 2 {
		t.Errorf("tuned peak bin = %d, want ~%d (centre)", tunedPeak, center)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TestMixerAccumValidation covers the constructor guards.
func TestMixerAccumValidation(t *testing.T) {
	now := func() int64 { return 0 }
	emit := func(MixerFrame) {}
	if _, err := newMixerAccum(mixerOptions{sampleHz: 48000, nowNs: now}); err == nil {
		t.Error("missing emit: want error")
	}
	if _, err := newMixerAccum(mixerOptions{emit: emit, nowNs: now}); err == nil {
		t.Error("zero sampleHz: want error")
	}
	if _, err := newMixerAccum(mixerOptions{emit: emit, sampleHz: 48000, fftSize: 1000, nowNs: now}); err == nil {
		t.Error("non-power-of-two FFT size: want error")
	}
	if _, err := newMixerAccum(mixerOptions{emit: emit, sampleHz: 48000}); err == nil {
		t.Error("missing nowNs: want error")
	}
	if _, err := newMixerAccum(mixerOptions{emit: emit, sampleHz: 48000, nowNs: now}); err != nil {
		t.Errorf("valid options: unexpected error %v", err)
	}
}

// TestEngineMixerEmits checks the Engine wires the mixer through Process
// end-to-end: a C4FM IQ stream produces MixerFrames with populated bins.
func TestEngineMixerEmits(t *testing.T) {
	const inRate = 960_000.0
	iq, _ := makeC4FMIQ(inRate, 4000)

	var mixFrames []MixerFrame
	eng, err := New(Options{
		Protocol:  trunking.ProtocolP25,
		DemodMode: "c4fm",
		InRateHz:  inRate,
		Emit:      func(Frame) {},
		MixerEmit: func(f MixerFrame) { mixFrames = append(mixFrames, f) },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer eng.Close()

	// Chunk the stream so Process runs repeatedly, like the broker drain.
	for off := 0; off < len(iq); off += 4096 {
		end := off + 4096
		if end > len(iq) {
			end = len(iq)
		}
		eng.Process(iq[off:end])
	}

	if len(mixFrames) == 0 {
		t.Fatal("no mixer frames emitted")
	}
	f := mixFrames[0]
	if len(f.RawBins) == 0 || len(f.TunedBins) == 0 {
		t.Errorf("empty bins: raw=%d tuned=%d", len(f.RawBins), len(f.TunedBins))
	}
	if f.SampleRateHz == 0 {
		t.Error("SampleRateHz = 0, want the DDC output rate")
	}
}
