package powersweep

import (
	"context"
	"encoding/binary"
	"math"
	"math/cmplx"
	"math/rand"
	"os"
	"testing"
	"time"
)

// jumpSource is a tuner whose DC offset does not hold still: white noise
// plus an offset that jumps to a new, unrelated value every `every` samples
// on average, as an AGC stepping its gain moves it. Nothing of it stands
// still for a block of frames, so the block removal has nothing to take.
type jumpSource struct {
	noise, amp, every float64
	cur               complex128
	rng               *rand.Rand
}

func (s *jumpSource) Tune(uint32) error { s.cur = 0; return nil }

func (s *jumpSource) Capture(_ context.Context, n int) ([]complex64, error) {
	out := make([]complex64, n)
	for i := range out {
		if s.cur == 0 || s.rng.Float64() < 1/s.every {
			s.cur = complex(s.rng.NormFloat64(), s.rng.NormFloat64()) * complex(s.amp/math.Sqrt2, 0)
		}
		out[i] = complex64(s.cur + complex(s.rng.NormFloat64()*s.noise, s.rng.NormFloat64()*s.noise))
	}
	return out, nil
}

// dcAmp is the offset amplitude that reads overDB over the noise in the
// centre bin of an n-point Hann FFT (see TestSweepRemovesTunerDCLine).
func dcAmp(overDB, noise float64, n int) float64 {
	return math.Sqrt(math.Pow(10, overDB/10) * 2 * noise * noise * 1.5 / float64(n))
}

// TestSweepReportsAndFillsALineThatMoved is the #1230 retest. On v1.4.1 the
// reporter's PlutoSky sweep still had the tuner's DC line in the hop with
// two stations of nearly equal strength: 7.3 dB over the bins beside it, in
// the offset's Hann shape (the bins either side at 0.27 of its excess,
// where an offset gives 0.25), at the level the line had before the
// removal existed, while the hops with one dominant station came out
// clean. So the removal took nothing there: the offset did not stand still
// for a block of frames.
//
// The fixture is an offset that jumps every four frames on average, 10 dB
// over the noise. By default the sweep's numbers stay as the removal left
// them and the row says a line like the tuner's is left; FillDC fills it
// in. A steady offset is still removed outright, with nothing to report.
func TestSweepReportsAndFillsALineThatMoved(t *testing.T) {
	p := dcPlan(t)
	const noise = 0.003
	amp := dcAmp(10, noise, p.FFTSize)
	in := Integration{Samples: 2000 * p.FFTSize}
	jumping := func() Source {
		return &jumpSource{noise: noise, amp: amp, every: 4 * float64(p.FFTSize), rng: rand.New(rand.NewSource(21))}
	}

	rows, err := Sweep(context.Background(), jumping(), p, in, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for h, row := range rows {
		peak, _ := centreOverFloor(row)
		if peak < 6 {
			t.Fatalf("hop %d: the centre stands %.1f dB over the floor; the fixture should leave a line like the reported 7 dB one", h, peak)
		}
		d := row.DC
		if d.Finding != DCLineLeft || d.Filled {
			t.Errorf("hop %d: finding %q (filled %v), want a line like the tuner's left and not filled in", h, d.Finding, d.Filled)
		}
		if d.Shape < 0.18 || d.Shape > 0.36 {
			t.Errorf("hop %d: shape %.2f, want the offset's 0.25", h, d.Shape)
		}
		if d.Coherence < 0.3 || math.Abs(d.PhaseRad) > dcTurnMinRad {
			t.Errorf("hop %d: frames alike %.2f, turning %+.3f rad; a jumping offset is alike between jumps and does not turn",
				h, d.Coherence, d.PhaseRad)
		}
	}

	in.FillDC = true
	rows, err = Sweep(context.Background(), jumping(), p, in, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for h, row := range rows {
		if peak, centre := centreOverFloor(row); peak > 1 || centre < -1 {
			t.Errorf("hop %d with FillDC: the centre bins stand %.1f dB over the floor (centre %+.1f dB)", h, peak, centre)
		}
		if !row.DC.Filled {
			t.Errorf("hop %d with FillDC: the row does not say it was filled in (%q)", h, row.DC.Finding)
		}
	}

	// A steady offset: all of it is removed, nothing is left to report, and
	// FillDC changes no bin.
	steady := func() Source {
		return &dcSource{rate: p.SampleRateHz, noise: noise, dc: cmplx.Rect(amp, 0.7), rng: rand.New(rand.NewSource(22))}
	}
	in.FillDC = false
	plain, err := Sweep(context.Background(), steady(), p, in, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	in.FillDC = true
	filled, err := Sweep(context.Background(), steady(), p, in, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for h := range plain {
		if f := plain[h].DC.Finding; f != DCClean {
			t.Errorf("hop %d with a steady offset: finding %q, want clean", h, f)
		}
		// The offset reads 10 dB over the noise, which is 2·noise²/n a bin.
		want := 10 * math.Log10(10*2*noise*noise/float64(p.FFTSize))
		if got := plain[h].DC.StillDB; math.Abs(got-want) > 0.5 {
			t.Errorf("hop %d with a steady offset: %.1f dBFS reported as standing still, want the offset's %.1f", h, got, want)
		}
		for k := range plain[h].DB {
			if plain[h].DB[k] != filled[h].DB[k] {
				t.Fatalf("hop %d bin %d: FillDC changed a hop with nothing left (%.2f vs %.2f dBFS)", h, k, filled[h].DB[k], plain[h].DB[k])
			}
		}
	}
}

// TestSweepFillDCLeavesCarriersAndStationsAlone: FillDC fills in a line
// only when it has the offset's shape on a flat floor and does not turn. A
// carrier off the hop's centre turns from frame to frame, modulated or
// not, and a station through the centre is wider than a line; both must
// read as without the removal. What it cannot tell from the tuner's line
// is a narrowband signal exactly on the centre: that is the price of
// FillDC, and why it is not the default.
func TestSweepFillDCLeavesCarriersAndStationsAlone(t *testing.T) {
	// One hop of the #1230 layout (4.8 MS/s, 512-point FFT).
	p, err := NewPlan(Range{LowHz: 88e6, HighHz: 91.6e6, BinHz: 10e3}, 4_800_000, 0.25)
	if err != nil || len(p.Hops) != 1 || p.FFTSize != 512 {
		t.Fatalf("plan: %v (%d hops, %d-point FFT)", err, len(p.Hops), p.FFTSize)
	}
	centre := float64(p.Hops[0].CenterHz)
	for _, tc := range []struct {
		name    string
		offBins float64
		fmHz    float64
		want    DCFinding
		filled  bool
	}{
		{"unmodulated carrier a third of a bin off", 1.0 / 3, 0, DCCarrier, false},
		{"unmodulated carrier a tenth of a bin off", 0.1, 0, DCCarrier, false},
		{"modulated carrier a tenth of a bin off", 0.1, 3000, DCCarrier, false},
		{"signal two bins wide on the centre", 0, 12000, DCSignal, false},
		{"station through the centre", 0, 75000, DCClean, false},
		{"modulated carrier exactly on the centre", 0, 3000, DCLineLeft, true},
	} {
		run := func(in Integration) Row {
			src := &dcSource{
				rate: p.SampleRateHz, noise: 0.003, rng: rand.New(rand.NewSource(13)),
				carrierHz: centre + tc.offBins*p.BinHz, carrierAmp: 0.01, fmHz: tc.fmHz,
			}
			in.Samples = 3000 * p.FFTSize
			rows, err := Sweep(context.Background(), src, p, in, time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			return rows[0]
		}
		want, got := run(Integration{KeepDC: true}), run(Integration{FillDC: true})
		if got.DC.Finding != tc.want || got.DC.Filled != tc.filled {
			t.Errorf("%s: finding %q (filled %v), want %q (filled %v)", tc.name, got.DC.Finding, got.DC.Filled, tc.want, tc.filled)
		}
		c := len(want.DB) / 2
		for k := c - 1; k <= c+1; k++ {
			d := float64(got.DB[k] - want.DB[k])
			switch {
			case !tc.filled && math.Abs(d) > 0.5:
				t.Errorf("%s: bin %+d reads %.2f dBFS with FillDC and %.2f with the line kept", tc.name, k-c, got.DB[k], want.DB[k])
			case tc.filled && k == c && d > -10:
				t.Errorf("%s: the centre bin reads %.2f dBFS with FillDC and %.2f with the line kept; "+
					"FillDC is expected to hide a signal exactly on the centre", tc.name, got.DB[k], want.DB[k])
			}
		}
	}
}

// loadCfile reads a GNU Radio .cfile (interleaved little-endian float32 IQ),
// skipping the test when the capture is not in the checkout.
func loadCfile(t *testing.T, path string) []complex64 {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("capture not available: %v", err)
	}
	out := make([]complex64, len(b)/8)
	for i := range out {
		out[i] = complex(math.Float32frombits(binary.LittleEndian.Uint32(b[8*i:])),
			math.Float32frombits(binary.LittleEndian.Uint32(b[8*i+4:])))
	}
	return out
}

// realAirCentre runs one capture through an accumulator as one hop and
// returns the centre bin's level, the mean of the bins two either side,
// and the hop's DCLine.
func realAirCentre(iq []complex64, n int, set func(*accumulator)) (centre, beside float64, line DCLine) {
	a := newAccumulator(n)
	set(a)
	a.add(iq)
	db := a.dB(0, n)
	c := n / 2
	beside = 10 * math.Log10((math.Pow(10, float64(db[c-2])/10)+math.Pow(10, float64(db[c+2])/10))/2)
	return float64(db[c]), beside, a.line
}

// TestRealAirChannelOnTheCentreIsWhyFillIsNotTheDefault. The samples/mmr
// captures (RTL-SDR, 2.4 MS/s) are centred on a narrowband channel: a
// C4FM carrier with the spectrum of the site's P25 control channel 37.5 kHz
// above it, 19 dB weaker. At 2.3 kHz bins it stands 12 dB over the bins
// beside it, on a floor that happens to be level, in close to the offset's
// shape, and its frames do not turn: everything the tuner's line is when
// its offset moves. The detector says so, and cannot say more.
//
// By default the channel must read as it does with the line kept. FillDC
// erases it, which is why a sweep does not fill in on its own.
func TestRealAirChannelOnTheCentreIsWhyFillIsNotTheDefault(t *testing.T) {
	iq := loadCfile(t, "../../samples/mmr/mmr-s9-1.cfile")
	const n = 1024
	kept, _, _ := realAirCentre(iq, n, func(a *accumulator) { a.keepDC = true })
	got, beside, line := realAirCentre(iq, n, func(*accumulator) {})
	if got-beside < 10 {
		t.Fatalf("the centre bin stands %.1f dB over the bins beside it; the capture has a channel there, about 12 dB over them", got-beside)
	}
	if math.Abs(got-kept) > 1 {
		t.Errorf("the channel on the centre reads %.1f dBFS by default and %.1f with the DC line kept", got, kept)
	}
	if line.Finding != DCLineLeft {
		t.Errorf("finding %q: this channel has been indistinguishable from a line the tuner left (shape %.2f, alike %.2f, turn %+.3f rad); "+
			"if the detector now tells them apart, FillDC's default is worth another look", line.Finding, line.Shape, line.Coherence, line.PhaseRad)
	}
	filled, _, _ := realAirCentre(iq, n, func(a *accumulator) { a.fill = true })
	if got-filled < 10 {
		t.Errorf("with fill the channel reads %.1f dBFS against %.1f without: it was expected to be filled in (%.1f beside it)", filled, got, beside)
	}
}

// TestRealAirNoiseWithAMovingOffset puts the moving offset on a real
// receiver's noise instead of the test's white noise: a quiet stretch of
// the same capture is mixed to the centre, and an offset that jumps every
// four frames is added 10 dB over it. The removal leaves the line, the hop
// reports it, and fill takes it out.
func TestRealAirNoiseWithAMovingOffset(t *testing.T) {
	iq := loadCfile(t, "../../samples/mmr/mmr-s9-1.cfile")
	const (
		n    = 1024
		half = 20 // bins either side that must be quiet
	)
	// The quietest ±half bins of the usable band, away from the centre.
	a := newAccumulator(n)
	a.keepDC = true
	a.add(iq)
	db := a.dB(0, n)
	best, bestMax := -1, math.Inf(1)
	for c := n / 8; c < n-n/8; c++ {
		if c > n/2-3*half && c < n/2+3*half {
			continue
		}
		worst := math.Inf(-1)
		for _, v := range db[c-half : c+half+1] {
			worst = math.Max(worst, float64(v))
		}
		if worst < bestMax {
			best, bestMax = c, worst
		}
	}
	// Mix that stretch to the centre.
	step := -2 * math.Pi * float64(best-n/2) / float64(n)
	quiet := make([]complex64, len(iq))
	for i, v := range iq {
		quiet[i] = complex64(complex128(v) * cmplx.Rect(1, step*float64(i)))
	}
	level, beside, line := realAirCentre(quiet, n, func(*accumulator) {})
	if line.Finding != DCClean {
		t.Fatalf("the quiet stretch at the centre: finding %q (%.1f dBFS, %.1f beside it), want nothing left", line.Finding, level, beside)
	}

	// A DC of amplitude a reads a² in the centre bin (a unit tone is 0 dBFS).
	amp := math.Sqrt(10 * math.Pow(10, beside/10))
	rng := rand.New(rand.NewSource(5))
	var cur complex128
	for i := range quiet {
		if i == 0 || rng.Float64() < 1/(4*float64(n)) {
			cur = complex(rng.NormFloat64(), rng.NormFloat64()) * complex(amp/math.Sqrt2, 0)
		}
		quiet[i] += complex64(cur)
	}
	level, beside, line = realAirCentre(quiet, n, func(*accumulator) {})
	if level-beside < 6 || line.Finding != DCLineLeft {
		t.Errorf("the moving offset on real noise: centre %.1f dB over the bins beside it, finding %q; want the line left (6 dB or more) and reported",
			level-beside, line.Finding)
	}
	level, beside, line = realAirCentre(quiet, n, func(a *accumulator) { a.fill = true })
	if level-beside > 0.01 || !line.Filled {
		t.Errorf("with fill: centre %.2f dB over the bins beside it (filled %v), want it filled in", level-beside, line.Filled)
	}
}
