package demod

import (
	"math"
	"testing"
)

func TestFMDemodLinearChirp(t *testing.T) {
	// Generate a complex exponential whose phase advances by a constant rate.
	const N = 4096
	const rate = 0.1 // radians per sample
	in := make([]complex64, N)
	phi := 0.0
	for i := 0; i < N; i++ {
		in[i] = complex(float32(math.Cos(phi)), float32(math.Sin(phi)))
		phi += rate
	}
	d := NewFM()
	out := d.Process(nil, in)
	// Skip first sample (depends on init). Rest should be ~rate.
	for i := 1; i < N; i++ {
		if math.Abs(float64(out[i])-rate) > 1e-3 {
			t.Fatalf("FM out[%d] = %f, want %f", i, out[i], rate)
		}
	}
}

func TestC4FMSlicer(t *testing.T) {
	// Deviation = 3.0 → outer-symbol threshold ±2.0.
	c := NewC4FM(8, 8, 0.2, 3.0)
	cases := []struct {
		in   float32
		want int
	}{
		{2.5, 3}, {1.0, 1}, {0.5, 1}, {0.01, 1},
		{-0.01, -1}, {-1.0, -1}, {-2.5, -3},
		{2.001, 3}, {1.999, 1}, // threshold corner
	}
	for _, tc := range cases {
		got := c.Slice(tc.in)
		if got != tc.want {
			t.Errorf("Slice(%f) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestGFSKSlicerThresholdAtZero(t *testing.T) {
	g := NewGFSK(10, 4, 0.3)
	cases := []struct {
		in   float32
		want int
	}{
		{1.0, 1}, {0.01, 1},
		{0.0, 0}, // tie-break: non-positive → 0
		{-0.01, 0}, {-1.0, 0},
	}
	for _, tc := range cases {
		if got := g.Slice(tc.in); got != tc.want {
			t.Errorf("Slice(%f) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestGFSKRecoversAlternatingNRZ: feed a square-wave stream that
// mimics the FM-discriminator output of alternating ±NRZ symbols
// (the same shape demod.FM would produce for a transmitter that
// shifted ±Δf every sps samples). The Gaussian matched filter
// rounds the edges; the symbol-centre slice must still recover the
// alternating 1 / 0 pattern.
func TestGFSKRecoversAlternatingNRZ(t *testing.T) {
	const sps = 10
	const span = 4
	const bt = 0.3
	const symbols = 64

	g := NewGFSK(sps, span, bt)

	in := make([]float32, symbols*sps)
	for s := 0; s < symbols; s++ {
		val := float32(1.0)
		if s%2 == 1 {
			val = -1.0
		}
		for k := 0; k < sps; k++ {
			in[s*sps+k] = val
		}
	}

	out := g.MatchedFilter(nil, in)

	// Sample at symbol centres. Skip the first `span` symbols
	// while the filter warms up and the last one to dodge edge
	// effects.
	mid := sps / 2
	for s := span; s < symbols-1; s++ {
		soft := out[s*sps+mid]
		want := 1
		if s%2 == 1 {
			want = 0
		}
		if got := g.Slice(soft); got != want {
			t.Errorf("symbol[%d] slice=%d, want %d (soft=%f)",
				s, got, want, soft)
		}
	}
}

// TestGFSKMatchedFilterStateAcrossChunks: splitting the input
// stream over multiple MatchedFilter calls must produce the same
// output as a single contiguous call. This is the invariant the
// receiver pipeline relies on when IQ arrives in arbitrary chunk
// sizes.
func TestGFSKMatchedFilterStateAcrossChunks(t *testing.T) {
	g1 := NewGFSK(10, 4, 0.3)
	g2 := NewGFSK(10, 4, 0.3)

	in := make([]float32, 1024)
	for i := range in {
		in[i] = float32(math.Sin(float64(i) * 0.1))
	}

	wantOut := g1.MatchedFilter(nil, in)

	var gotOut []float32
	for off := 0; off < len(in); off += 73 {
		end := off + 73
		if end > len(in) {
			end = len(in)
		}
		chunk := g2.MatchedFilter(nil, in[off:end])
		gotOut = append(gotOut, chunk...)
	}

	if len(gotOut) != len(wantOut) {
		t.Fatalf("chunked length %d, want %d", len(gotOut), len(wantOut))
	}
	for i := range wantOut {
		if math.Abs(float64(gotOut[i]-wantOut[i])) > 1e-6 {
			t.Errorf("chunked out[%d] = %f, want %f",
				i, gotOut[i], wantOut[i])
		}
	}
}

// TestGFSKResetClearsHistory: after Reset, a single impulse must
// produce the Gaussian impulse response as if from a fresh filter.
// TestGFSKResetClearsHistory: after Reset, a single impulse must
// produce the Gaussian impulse response as if from a fresh filter.
func TestGFSKResetClearsHistory(t *testing.T) {
	g := NewGFSK(10, 4, 0.3)

	noise := make([]float32, 256)
	for i := range noise {
		noise[i] = float32(math.Cos(float64(i) * 0.3))
	}
	g.MatchedFilter(nil, noise)
	g.Reset()

	impulse := make([]float32, 60)
	impulse[0] = 1.0
	out := g.MatchedFilter(nil, impulse)

	// After Reset, history is zero; the impulse propagates through
	// the filter and reaches its peak at sample mid = sps*span/2.
	const mid = 10 * 4 / 2
	peak := out[mid]
	for i, v := range out {
		if v > peak {
			t.Errorf("post-Reset: out[%d]=%f > out[%d]=%f", i, v, mid, peak)
		}
	}
}

// buildFFSKAudio synthesises a continuous-phase audio waveform whose
// each bit selects either markHz (bit=1) or spaceHz (bit=0) for `sps`
// samples. Continuous-phase keeps the discriminator clean.
func buildFFSKAudio(bits []int, markHz, spaceHz, sampleRate float64, sps int) []float32 {
	out := make([]float32, len(bits)*sps)
	phase := 0.0
	for b, bit := range bits {
		toneHz := spaceHz
		if bit == 1 {
			toneHz = markHz
		}
		dphi := 2 * math.Pi * toneHz / sampleRate
		for k := 0; k < sps; k++ {
			out[b*sps+k] = float32(math.Sin(phase))
			phase += dphi
		}
	}
	return out
}

func TestFFSKSlicerThresholdAtZero(t *testing.T) {
	f := NewFFSK(48_000, 1200, 1800)
	cases := []struct {
		in   float32
		want int
	}{
		{1.0, 1}, {0.01, 1},
		{0.0, 0}, // tie-break: non-positive → 0 (space)
		{-0.01, 0}, {-1.0, 0},
	}
	for _, tc := range cases {
		if got := f.Slice(tc.in); got != tc.want {
			t.Errorf("Slice(%f) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestFFSKRecoversCCIRBitPattern: synthesise an MPT 1327-style
// FFSK audio waveform (mark = 1200 Hz = 1, space = 1800 Hz = 0,
// 1200 baud, 48 kHz audio sample rate) for a known bit pattern,
// run it through Discriminate + symbol-centre slicing, and assert
// the recovered bits match. Sampling offset accounts for the LPF
// group delay (Delay() samples) so the bit centre lands on the
// right input symbol.
func TestFFSKRecoversCCIRBitPattern(t *testing.T) {
	const (
		sampleRate = 48_000.0
		markHz     = 1200.0
		spaceHz    = 1800.0
		bitRate    = 1200.0
	)
	sps := int(sampleRate / bitRate) // 40

	bits := []int{
		1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0, 0, 0, 1, 0,
		1, 1, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 0, 0, 1, 0,
		// Trailing tail so symbol-centre + group-delay offsets
		// land on real samples without overrunning the buffer.
		1, 1, 1, 1, 1, 1, 1, 1,
	}

	audio := buildFFSKAudio(bits, markHz, spaceHz, sampleRate, sps)

	f := NewFFSK(sampleRate, markHz, spaceHz)
	soft := f.Discriminate(nil, audio)

	// Skip the first few bits while the LPF + discriminator warm
	// up, and stop before the trailing tail.
	mid := sps / 2
	delay := f.Delay()
	var errors int
	for b := 4; b < 32; b++ {
		samp := soft[b*sps+mid+delay]
		got := f.Slice(samp)
		if got != bits[b] {
			errors++
			t.Logf("bit[%d] = %d, want %d (soft=%f)", b, got, bits[b], samp)
		}
	}
	if errors > 0 {
		t.Errorf("FFSK demod recovered %d bit(s) incorrectly out of %d",
			errors, 32-4)
	}
}

// TestFFSKWorksWhenMarkAboveSpace: some non-CCIR FFSK variants put
// mark on the higher frequency. Slice must still return 1 for the
// mark tone regardless of which side of the centre it sits on.
func TestFFSKWorksWhenMarkAboveSpace(t *testing.T) {
	const (
		sampleRate = 48_000.0
		markHz     = 2200.0
		spaceHz    = 1200.0
		bitRate    = 1200.0
	)
	sps := int(sampleRate / bitRate)

	bits := []int{
		1, 0, 1, 0, 1, 1, 0, 0, 1, 0, 0, 1, 1, 1, 0, 1,
		// Trailing tail to cover group delay.
		1, 1, 1, 1, 1, 1, 1, 1,
	}
	audio := buildFFSKAudio(bits, markHz, spaceHz, sampleRate, sps)

	f := NewFFSK(sampleRate, markHz, spaceHz)
	soft := f.Discriminate(nil, audio)

	mid := sps / 2
	delay := f.Delay()
	var errors int
	for b := 4; b < 16; b++ {
		samp := soft[b*sps+mid+delay]
		if got := f.Slice(samp); got != bits[b] {
			errors++
			t.Logf("bit[%d] = %d, want %d (soft=%f)", b, got, bits[b], samp)
		}
	}
	if errors > 0 {
		t.Errorf("FFSK (mark>space) recovered %d bit(s) incorrectly", errors)
	}
}

// TestFFSKDiscriminateStateAcrossChunks: splitting the input
// stream over multiple Discriminate calls must produce the same
// output as one contiguous call.
func TestFFSKDiscriminateStateAcrossChunks(t *testing.T) {
	const sampleRate = 48_000.0
	const markHz = 1200.0
	const spaceHz = 1800.0
	const sps = 40

	bits := []int{1, 0, 1, 1, 0, 0, 1, 0, 1, 1, 1, 0}
	audio := buildFFSKAudio(bits, markHz, spaceHz, sampleRate, sps)

	f1 := NewFFSK(sampleRate, markHz, spaceHz)
	wantOut := f1.Discriminate(nil, audio)

	f2 := NewFFSK(sampleRate, markHz, spaceHz)
	var gotOut []float32
	for off := 0; off < len(audio); off += 73 {
		end := off + 73
		if end > len(audio) {
			end = len(audio)
		}
		chunk := f2.Discriminate(nil, audio[off:end])
		gotOut = append(gotOut, chunk...)
	}

	if len(gotOut) != len(wantOut) {
		t.Fatalf("chunked length %d, want %d", len(gotOut), len(wantOut))
	}
	for i := range wantOut {
		if math.Abs(float64(gotOut[i]-wantOut[i])) > 1e-5 {
			t.Errorf("chunked out[%d] = %f, want %f",
				i, gotOut[i], wantOut[i])
		}
	}
}

func TestFFSKConstructorPanicsOnBadParams(t *testing.T) {
	cases := []struct {
		name             string
		sampleRate       float64
		markHz, spaceHz  float64
	}{
		{"zero sample rate", 0, 1200, 1800},
		{"zero mark", 48_000, 0, 1800},
		{"zero space", 48_000, 1200, 0},
		{"mark equals space", 48_000, 1500, 1500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic, got nil")
				}
			}()
			_ = NewFFSK(tc.sampleRate, tc.markHz, tc.spaceHz)
		})
	}
}

func TestDQPSKDibitsMatchPhaseSteps(t *testing.T) {
	d := NewDQPSK()
	// Build a signal whose phase advances by π/2 per symbol → dibit "01".
	const N = 64
	in := make([]complex64, N)
	phi := 0.0
	for i := 0; i < N; i++ {
		in[i] = complex(float32(math.Cos(phi)), float32(math.Sin(phi)))
		phi += math.Pi / 2
	}
	out := d.Decode(nil, in)
	for i := 1; i < N; i++ {
		if out[i] != 0b01 {
			t.Errorf("out[%d] = %02b, want 01", i, out[i])
		}
	}
}
