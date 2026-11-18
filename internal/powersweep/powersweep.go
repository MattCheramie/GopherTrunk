// Package powersweep is an rtl_power-style wideband power logger (issue
// #1230): it steps a tunable IQ source across a frequency range, averages an
// FFT power spectrum at each step, and writes the result in rtl_power's CSV
// layout so existing tooling (heatmap.py, flatten.py, spreadsheet imports)
// reads it unchanged.
//
// The sweep is laid out in whole FFT bins. One "hop" is one tuning: it keeps
// the HopBins bins around the centre of the FFT (the rest — the 1−crop edges,
// where the tuner's anti-alias filter rolls off — are dropped), and
// consecutive hops abut exactly, so bin j of hop h sits at
// Low + (h·HopBins + j)·BinHz with no gap or overlap across the whole range.
package powersweep

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/cmplx"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// MaxFFTSize bounds the FFT a bin size can ask for. At 2.4 MS/s it allows
// ~9 Hz bins; finer than that is not a power survey.
const MaxFFTSize = 1 << 18

// MinFFTSize is the smallest FFT a plan uses, however wide the bins.
const MinFFTSize = 8

// Range is a sweep request: [LowHz, HighHz] in steps of at most BinHz.
type Range struct {
	LowHz, HighHz, BinHz float64
}

// ParseRange parses rtl_power's -f syntax, "lower:upper:bin_size", where each
// field takes an optional k / M / G suffix (e.g. "88M:108M:10k").
func ParseRange(s string) (Range, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 3 {
		return Range{}, fmt.Errorf("frequency range %q: want lower:upper:bin_size (e.g. 88M:108M:10k)", s)
	}
	var v [3]float64
	for i, p := range parts {
		f, err := ParseHz(p)
		if err != nil {
			return Range{}, fmt.Errorf("frequency range %q: %w", s, err)
		}
		v[i] = f
	}
	r := Range{LowHz: v[0], HighHz: v[1], BinHz: v[2]}
	if r.LowHz <= 0 || r.HighHz <= r.LowHz {
		return Range{}, fmt.Errorf("frequency range %q: need 0 < lower < upper", s)
	}
	if r.BinHz <= 0 {
		return Range{}, fmt.Errorf("frequency range %q: bin size must be > 0", s)
	}
	if r.HighHz > math.MaxUint32 {
		return Range{}, fmt.Errorf("frequency range %q: upper edge above %d Hz", s, uint32(math.MaxUint32))
	}
	return r, nil
}

// ParseHz parses a frequency with an optional k / M / G suffix (case-
// insensitive, as rtl_power's atofs).
func ParseHz(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty frequency")
	}
	mult := 1.0
	switch s[len(s)-1] {
	case 'k', 'K':
		mult = 1e3
	case 'm', 'M':
		mult = 1e6
	case 'g', 'G':
		mult = 1e9
	}
	if mult != 1 {
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("bad frequency %q", s)
	}
	return f * mult, nil
}

// Hop is one tuning of a sweep.
type Hop struct {
	// CenterHz is the frequency the source is tuned to.
	CenterHz uint32
	// LowHz is the frequency of the hop's first kept bin.
	LowHz float64
	// Bins is how many bins the hop contributes (HopBins, except possibly
	// fewer on the last hop).
	Bins int
}

// Plan is a laid-out sweep.
type Plan struct {
	SampleRateHz float64
	FFTSize      int
	BinHz        float64
	// HopBins is the number of bins every full hop keeps, centred in the
	// FFT: FFT indices [FirstBin, FirstBin+HopBins).
	HopBins  int
	FirstBin int
	Hops     []Hop
}

// NewPlan lays a Range out over a source running at sampleRateHz. crop is the
// fraction of each FFT discarded at the band edges (0 ≤ crop < 1), split
// evenly between them. The FFT size is the smallest power of two whose bins
// are no wider than r.BinHz.
func NewPlan(r Range, sampleRateHz, crop float64) (Plan, error) {
	if sampleRateHz <= 0 {
		return Plan{}, errors.New("sample rate must be > 0")
	}
	if crop < 0 || crop >= 1 {
		return Plan{}, fmt.Errorf("crop %.2f must be in [0, 1)", crop)
	}
	n := MinFFTSize
	for sampleRateHz/float64(n) > r.BinHz {
		n <<= 1
		if n > MaxFFTSize {
			return Plan{}, fmt.Errorf("bin size %.1f Hz needs an FFT over %d points at %.0f S/s; use a wider bin", r.BinHz, MaxFFTSize, sampleRateHz)
		}
	}
	binHz := sampleRateHz / float64(n)
	hopBins := int(float64(n) * (1 - crop))
	hopBins &^= 1 // even, so the hop centres on a bin boundary of the FFT
	if hopBins < 2 {
		return Plan{}, fmt.Errorf("crop %.2f leaves no usable bins in a %d-point FFT", crop, n)
	}
	count := int(math.Ceil((r.HighHz - r.LowHz) / binHz))
	if count < 1 {
		count = 1
	}
	p := Plan{
		SampleRateHz: sampleRateHz,
		FFTSize:      n,
		BinHz:        binHz,
		HopBins:      hopBins,
		FirstBin:     n/2 - hopBins/2,
	}
	for done := 0; done < count; done += hopBins {
		low := r.LowHz + float64(done)*binHz
		// Bin j of the hop sits at center + (FirstBin + j − n/2)·binHz =
		// center + (j − hopBins/2)·binHz, so bin 0 is at low.
		center := low + float64(hopBins/2)*binHz
		if center > math.MaxUint32 {
			return Plan{}, fmt.Errorf("hop centre %.0f Hz out of range", center)
		}
		p.Hops = append(p.Hops, Hop{
			CenterHz: uint32(math.Round(center)),
			LowHz:    low,
			Bins:     min(hopBins, count-done),
		})
	}
	return p, nil
}

// Source is the tunable IQ source a sweep drives. hunt.IQSource satisfies it.
type Source interface {
	// Tune retunes the source; IQ captured afterwards is centred there.
	Tune(centerHz uint32) error
	// Capture returns the next n samples at the current centre.
	Capture(ctx context.Context, n int) ([]complex64, error)
}

// Row is one hop's averaged power, in rtl_power's CSV fields.
type Row struct {
	Time    time.Time
	LowHz   float64
	HighHz  float64
	StepHz  float64
	Samples int
	DB      []float32
	// DC is what the hop's centre bins showed of the tuner's own DC line
	// (zero when the line was kept, or the hop's centre is not in DB).
	DC DCLine
}

// DCFinding is what a hop's centre bins showed once the part of the
// tuner's DC line that stood still had been removed.
type DCFinding int

const (
	// DCClean: nothing stands over the bins beside the centre.
	DCClean DCFinding = iota
	// DCLineLeft: a line with the shape of the tuner's own is left, on a
	// flat floor, and it does not turn. Either the tuner's offset moved
	// during the hop faster than the removal follows, or a narrowband
	// signal sits on the hop's centre (to within about a hundredth of a
	// bin for a steady carrier, a tenth for a modulated one, more on a
	// short hop). The spectrum cannot tell the two apart;
	// Integration.FillDC decides what to do with it.
	DCLineLeft
	// DCCarrier: what is left turns from frame to frame, as a carrier off
	// the hop's centre does and the tuner's line cannot. It is a signal.
	DCCarrier
	// DCSignal: what is left is wider than the tuner's line, or sits on a
	// slope (a station through the hop's centre). It is a signal.
	DCSignal
)

func (f DCFinding) String() string {
	switch f {
	case DCLineLeft:
		return "a line like the tuner's is left"
	case DCCarrier:
		return "a carrier off the centre (it turns)"
	case DCSignal:
		return "a signal (wider than the tuner's line, or on a slope)"
	}
	return "clean"
}

// DCLine is the tuner's DC line as one hop saw it. See accumulator.
type DCLine struct {
	// Measured is false when the hop could not look (the line was kept, the
	// hop had one frame, or the FFT is too small to have bins beside the
	// centre).
	Measured bool
	// StillDB is the centre-bin power that stood still within blocks of
	// frames and was removed, in dBFS (−Inf when there was none): the line
	// while the tuner's offset holds.
	StillDB float64
	// LeftDB is how far the centre bin then stood over the bins two either
	// side of it, in dB.
	LeftDB float64
	// Shape is the excess of the bins one either side of the centre over
	// that of the centre bin: 0.25 for an offset through the Hann window.
	// Zero when nothing was left.
	Shape float64
	// Coherence and PhaseRad are the lag-one correlation of the centre
	// bin's frames, each less its block's mean: how alike consecutive
	// frames of what moved are (0 = unrelated, 1 = equal), and how far it
	// turns from one frame to the next. An offset that jumps now and then
	// is alike between jumps and has phase 0; a carrier f Hz off the
	// centre turns 2π·f·FFTSize/rate.
	Coherence float64
	PhaseRad  float64
	Finding   DCFinding
	// Filled is set when the three centre bins were filled in from the
	// bins two either side (Finding is DCLineLeft and FillDC was asked).
	Filled bool
}

// captureBlock bounds how many samples one Capture call asks for, so a long
// integration streams through a fixed buffer instead of being held in memory.
const captureBlock = 1 << 16

// Integration is how long each hop of a sweep integrates.
type Integration struct {
	// Samples is the number of IQ samples a hop integrates (rounded up to
	// whole FFTs, at least one).
	Samples int
	// Budget, when > 0, caps the wall-clock time a hop spends integrating
	// (from just after its retune). A hop stops at whichever of Samples or
	// Budget comes first, having integrated at least one FFT. Without it a
	// CPU that cannot FFT every sample in real time stretched each hop — and
	// so each sweep — past the requested interval (#1230: a 30 s sweep took
	// about a minute on a 32-bit phone). The row's Samples reports what was
	// actually integrated.
	Budget time.Duration
	// Now is the clock Budget is measured on; nil means time.Now.
	Now func() time.Time
	// KeepDC leaves the tuner's own DC line in each hop's centre bins
	// (see accumulator). The default removes it.
	KeepDC bool
	// FillDC fills in the three centre bins of a hop from the bins beside
	// them when a line like the tuner's is left there after the removal
	// (Row.DC.Finding == DCLineLeft): the tuner's offset moved during the
	// hop. It also hides a narrowband signal that sits exactly on a hop's
	// centre, which looks the same; without it such a line is only
	// reported.
	FillDC bool
}

// Sweep runs one pass of p over src, integrating each hop as in says. stamp
// is the time written on every row of the pass, as rtl_power does.
func Sweep(ctx context.Context, src Source, p Plan, in Integration, stamp time.Time) ([]Row, error) {
	acc := newAccumulator(p.FFTSize)
	acc.keepDC = in.KeepDC
	acc.fill = in.FillDC
	frames := (in.Samples + p.FFTSize - 1) / p.FFTSize
	if frames < 1 {
		frames = 1
	}
	now := in.Now
	if now == nil {
		now = time.Now
	}
	perBlock := max(1, captureBlock/p.FFTSize)
	rows := make([]Row, 0, len(p.Hops))
	for _, h := range p.Hops {
		if err := src.Tune(h.CenterHz); err != nil {
			return rows, fmt.Errorf("tune %d Hz: %w", h.CenterHz, err)
		}
		acc.reset()
		var deadline time.Time
		if in.Budget > 0 {
			deadline = now().Add(in.Budget)
		}
		for acc.frames < frames {
			want := min(perBlock, frames-acc.frames) * p.FFTSize
			iq, err := src.Capture(ctx, want)
			if err != nil {
				return rows, err
			}
			if len(iq) < p.FFTSize {
				return rows, io.ErrUnexpectedEOF
			}
			acc.add(iq)
			if !deadline.IsZero() && !now().Before(deadline) {
				break
			}
		}
		row := Row{
			Time:    stamp,
			LowHz:   h.LowHz,
			HighHz:  h.LowHz + float64(h.Bins)*p.BinHz,
			StepHz:  p.BinHz,
			Samples: acc.frames * p.FFTSize,
			DB:      acc.dB(p.FirstBin, h.Bins),
		}
		// The centre bins are FFT-shifted indices n/2−1..n/2+1; a short last
		// hop may end before them.
		if p.FirstBin+h.Bins > p.FFTSize/2+1 {
			row.DC = acc.line
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// accumulator averages FFT periodograms in linear power, with the same Hann
// window and normalisation as spectrum.PowerDB (a unit-amplitude tone reads
// ~0 dBFS in its bin).
//
// The transform is fft.Radix2 on complex64: on the #1230 reporter's 32-bit
// phones the gonum (complex128) backend could not keep up with a 2.4 MS/s
// dongle. The window step writes each sample straight into the FFT's
// bit-reversed input slot, so there is no separate permutation pass.
//
// # The tuner's DC line
//
// A zero-IF tuner leaks its own local oscillator into the output as a
// constant offset, which a spectrum shows as a line at exactly the tuned
// frequency: with the Hann window, the centre bin and, 6 dB down, the bin on
// each side. A sweep retunes for every hop, so the line appears once per
// hop, as a carrier that is not on the air. The #1230 reporter's PlutoSky
// (AD9361) sweep of the FM band had one every 3.6 MHz, 11 dB over the noise
// around it; rtl_power overwrites its DC bin with the bin beside it for the
// same reason.
//
// The line differs from everything received in one way: it is the same
// complex value in every FFT frame, where a signal off the air, or noise,
// turns from frame to frame. So for the three centre bins the accumulator
// keeps the frames' complex sum next to their power sum, in blocks of
// dcBlock frames, and takes the power of what varied: sum|X|² − |sum X|²/m,
// scaled by m/(m−1) so that noise still reads its true power. What goes
// with the line is anything else standing still for a block: an unmodulated
// carrier within about a sixteenth of a bin of the hop's centre. Blocks,
// not the whole hop, because the offset moves when the tuner's gain does.
//
// # When the line does not stand still
//
// Blocks follow an offset that moves now and then. They do not follow one
// that moves several times a block, and on the #1230 PlutoSky that is what
// happened in the busy hops: the v1.4.1 retest still had the line, in its
// exact Hann shape and at its old level, in a hop with two stations of
// nearly equal strength, while hops with one dominant station came out
// clean. (An AGC holds still under one constant-envelope carrier and hunts
// under the beating of several; each gain step moves the offset. That
// cause is inferred from the CSV, not seen on the board.) The frame is the
// finest time step a bin has, and a per-frame estimate of the offset would
// use up the centre bin itself, so what is left of the line cannot be
// subtracted. It can be recognised, by three things at once:
//
//   - it stands over the bins two either side of the centre, which are
//     level with the bins beyond them (a flat floor, not a station), bar
//     the few percent of itself a jumping offset spreads that far;
//   - it has the offset's shape: the bins one either side carry a quarter
//     of the centre bin's excess;
//   - it does not turn. A carrier f Hz off the hop's centre advances
//     2π·f·n/rate from one frame to the next; the tuner's offset, however
//     it jumps about, has no direction to turn in. The lag-one product of
//     the centre bin's frames (each less its block's mean, so the part
//     that stood still does not vote) shows which.
//
// That is reported in the row's DCLine, and is all that happens by default,
// because a narrowband signal that sits on the hop's centre looks the
// same: its frames are alike from one to the next, and with a receiver on
// frequency to a hundredth of a bin it does not turn either. A real case
// is in the repository: the samples/mmr captures are centred on a
// narrowband channel (a C4FM carrier with the spectrum of the site's P25
// control channel, 19 dB weaker), which is line-shaped at 2.3 kHz bins and
// 12 dB over the bins beside it. Filling in whatever is left would erase
// it. Integration.FillDC asks for it anyway: the three centre bins are
// then filled in from the bins two either side, as rtl_power does with its
// DC bin.
type accumulator struct {
	n      int
	plan   *fft.Radix2
	rev    []int32
	win    []float32
	norm   float64
	buf    []complex64
	sum    []float64 // FFT-shifted linear power
	frames int

	// keepDC switches the DC line removal off. fill fills in the centre
	// bins when a line like the tuner's is left after it.
	keepDC bool
	fill   bool
	// The centre bins (FFT bins n−1, 0, 1): the current block's complex
	// and power sums, and the finished blocks' line-free power and frames.
	dcSum    [3]complex128
	dcPow    [3]float64
	dcN      int
	dcOut    [3]float64
	dcFrames int

	// The centre bin's frames of the current block, and over the finished
	// blocks the lag-one products p = r[t]·conj(r[t−1]) of what moved in
	// them (r = a frame less its block's mean): Σp, Σ|p|² and their count.
	dcBin0 [dcBlock]complex128
	lagSum complex128
	lagSq  float64
	lagN   int

	// line is the last dB call's report on the hop's DC line.
	line DCLine
}

// dcBlock is how many FFT frames the DC line is taken to stand still for.
const dcBlock = 16

const (
	// dcLeftMin is how far the centre bin must stand over the bins beside
	// it, as a fraction of them, before a line counts as left (1 dB): under
	// that it does not read as a carrier, and a tuner's own response can
	// ripple that much at its centre. A hop of few frames needs more, see
	// dcLeftSigmas.
	dcLeftMin = 0.26
	// dcLeftSigmas scales that margin on short hops: a bin averaged over f
	// frames scatters about 1/√f of its level, and the difference of the
	// centre bin and the pair beside it about 1.25 times that.
	dcLeftSigmas = 5.0
	// dcFlatTol is how far the bins two either side of the centre may
	// differ from the four beyond them for the floor to count as flat
	// (0.5 dB; 3 standard deviations of that difference on a short hop).
	dcFlatTol    = 0.12
	dcFlatSigmas = 3.0
	// dcSkirtMax is how much of the line's own power the bins two either
	// side may carry on top of that. An offset that jumps inside a frame
	// spreads a little: measured on a jumping offset, 1 to 3.5 % of the
	// line lands there. A station through the hop's centre puts 20 % and
	// more of its peak there, which is what tells it from a line.
	dcSkirtMax = 0.06
	// dcShapeMin / dcShapeMax bound the offset's shape: the bins one either
	// side of the centre carry a quarter of its excess (Hann), give or take
	// what the averaging scatters at the smallest line that counts.
	dcShapeMin = 0.05
	dcShapeMax = 0.50
	// dcTurnMinRad is the smallest turn per frame taken as a carrier's: a
	// hundredth of a bin off the hop's centre, about. Below it a carrier
	// and the tuner's offset are not told apart.
	dcTurnMinRad = 0.05
	// dcTurnSigmas is how many standard deviations of the lag-one product's
	// scatter its turning must clear.
	dcTurnSigmas = 4.0
)

func newAccumulator(n int) *accumulator {
	hann := window.Hann(n)
	win := make([]float32, n)
	var ws float64
	for i, w := range hann {
		ws += w * w
		win[i] = float32(w)
	}
	plan, err := fft.NewRadix2(n)
	if err != nil {
		panic(err) // Plan sizes are powers of two in [MinFFTSize, MaxFFTSize]
	}
	return &accumulator{
		n:    n,
		plan: plan,
		rev:  plan.Rev(),
		win:  win,
		norm: 1 / (float64(n) * ws),
		buf:  make([]complex64, n),
		sum:  make([]float64, n),
	}
}

func (a *accumulator) reset() {
	clear(a.sum)
	a.frames = 0
	a.dcSum, a.dcPow, a.dcN = [3]complex128{}, [3]float64{}, 0
	a.dcOut, a.dcFrames = [3]float64{}, 0
	a.lagSum, a.lagSq, a.lagN = 0, 0, 0
	a.line = DCLine{}
}

// closeDCBlock folds the current block of the centre bins into dcOut,
// without the part that stood still. A block of one frame has nothing to
// compare with and is left out.
func (a *accumulator) closeDCBlock() {
	if m := float64(a.dcN); a.dcN >= 2 {
		for i := range a.dcOut {
			re, im := real(a.dcSum[i]), imag(a.dcSum[i])
			varied := a.dcPow[i] - (re*re+im*im)*a.norm/m
			if varied > 0 {
				a.dcOut[i] += varied * m / (m - 1)
			}
		}
		a.dcFrames += a.dcN
		// What moved in the centre bin, frame against the frame before.
		// Taking the block's mean out makes unrelated frames look slightly
		// opposed (each pair by −|r|²/(m−1) on average); the block's mean
		// |r|² puts that back, so noise sums to nothing.
		mean := a.dcSum[1] / complex(m, 0)
		prev := a.dcBin0[0] - mean
		pow := real(prev)*real(prev) + imag(prev)*imag(prev)
		for _, x := range a.dcBin0[1:a.dcN] {
			r := x - mean
			p := r * cmplx.Conj(prev)
			a.lagSum += p
			a.lagSq += real(p)*real(p) + imag(p)*imag(p)
			a.lagN++
			pow += real(r)*real(r) + imag(r)*imag(r)
			prev = r
		}
		a.lagSum += complex(pow/m, 0)
	}
	a.dcSum, a.dcPow, a.dcN = [3]complex128{}, [3]float64{}, 0
}

// add folds every whole FFT frame of iq into the average.
func (a *accumulator) add(iq []complex64) {
	half := a.n / 2
	buf, rev, win := a.buf, a.rev[:a.n], a.win[:a.n]
	// FFT-shift as two straight runs instead of a per-bin modulo: bins
	// [0, half) land at [half, n) and [half, n) at [0, half).
	lowSum, highSum := a.sum[half:], a.sum[:half]
	for off := 0; off+a.n <= len(iq); off += a.n {
		frame := iq[off : off+a.n]
		for i, w := range win {
			s := frame[i]
			buf[rev[i]] = complex(real(s)*w, imag(s)*w)
		}
		a.plan.TransformBitReversed(buf)
		for i, v := range buf[:half] {
			re, im := float64(real(v)), float64(imag(v))
			lowSum[i] += (re*re + im*im) * a.norm
		}
		for i, v := range buf[half:] {
			re, im := float64(real(v)), float64(imag(v))
			highSum[i] += (re*re + im*im) * a.norm
		}
		a.frames++
		if !a.keepDC {
			for i, k := range [3]int{a.n - 1, 0, 1} {
				v := complex128(buf[k])
				a.dcSum[i] += v
				a.dcPow[i] += (real(v)*real(v) + imag(v)*imag(v)) * a.norm
			}
			a.dcBin0[a.dcN] = complex128(buf[0])
			if a.dcN++; a.dcN == dcBlock {
				a.closeDCBlock()
			}
		}
	}
}

// centrePower returns the line-free mean power of the three centre bins
// (FFT-shifted indices n/2−1, n/2, n/2+1) and whether it could be measured.
// A hop of a single frame cannot tell the line from a signal; its centre
// bins are then filled in from the bins either side of them.
func (a *accumulator) centrePower() (p [3]float64, ok bool) {
	a.closeDCBlock()
	if a.dcFrames > 0 {
		for i := range p {
			p[i] = a.dcOut[i] / float64(a.dcFrames)
		}
		return p, true
	}
	if a.frames == 0 {
		return p, false
	}
	half := a.n / 2
	side := (a.sum[half-2] + a.sum[half+2]) / 2 / float64(a.frames)
	return [3]float64{side, side, side}, true
}

// settleCentre looks at what the block removal left in the three centre
// bins (centre, their line-free power): see "When the line does not stand
// still" on accumulator. It returns the powers to report (filled in when
// a line like the tuner's is left and fill is set) and the hop's DCLine.
func (a *accumulator) settleCentre(centre [3]float64) ([3]float64, DCLine) {
	half := a.n / 2
	if a.dcFrames == 0 || half < 4 {
		return centre, DCLine{}
	}
	frames := float64(a.frames)
	line := DCLine{Measured: true, StillDB: math.Inf(-1), LeftDB: math.Inf(-1)}
	if still := a.sum[half]/frames - centre[1]; still > 0 {
		line.StillDB = 10 * math.Log10(still)
	}
	if a.lagN > 0 && a.lagSq > 0 {
		// Σ|p|² ≥ |Σp|²/N, with equality when every product is the same.
		line.Coherence = math.Min(cmplx.Abs(a.lagSum)/math.Sqrt(a.lagSq*float64(a.lagN)), 1)
		line.PhaseRad = cmplx.Phase(a.lagSum)
	}
	beside := (a.sum[half-2] + a.sum[half+2]) / 2 / frames
	beyond := (a.sum[half-4] + a.sum[half-3] + a.sum[half+3] + a.sum[half+4]) / 4 / frames
	if beside <= 0 || beyond <= 0 {
		return centre, line
	}
	line.LeftDB = 10 * math.Log10(math.Max(centre[1], 1e-30)/beside)
	scatter := 1 / math.Sqrt(float64(a.dcFrames))
	left := centre[1] - beside
	if left <= beside*math.Max(dcLeftMin, dcLeftSigmas*scatter) {
		return centre, line
	}
	line.Shape = ((centre[0]+centre[2])/2 - beside) / left
	flat := math.Abs(beside-beyond) <= beyond*math.Max(dcFlatTol, dcFlatSigmas*scatter)+left*dcSkirtMax
	switch {
	case !flat || line.Shape < dcShapeMin || line.Shape > dcShapeMax:
		line.Finding = DCSignal
	case a.turns():
		line.Finding = DCCarrier
	default:
		line.Finding = DCLineLeft
		if a.fill {
			line.Filled = true
			centre = [3]float64{beside, beside, beside}
		}
	}
	return centre, line
}

// turns reports whether what moved in the centre bin turns from one frame
// to the next, as a carrier off the hop's centre does. The lag-one products
// scatter about their mean; their sum's component across the real axis has
// variance ½(Σ|p|² − |Σp|²/N). The turning counts when the sum stands
// clear of that scatter, either off the real axis by more than
// dcTurnMinRad or on its negative half (half a turn a frame).
func (a *accumulator) turns() bool {
	if a.lagN < 2 {
		return false
	}
	n := float64(a.lagN)
	re, im := real(a.lagSum), imag(a.lagSum)
	sigma := math.Sqrt(math.Max(a.lagSq-(re*re+im*im)/n, 0) / 2)
	if re < 0 && -re > dcTurnSigmas*sigma {
		return true
	}
	return math.Abs(im) > dcTurnSigmas*sigma && math.Abs(cmplx.Phase(a.lagSum)) > dcTurnMinRad
}

// dB returns the averaged power of FFT-shifted bins [first, first+n) in dBFS.
func (a *accumulator) dB(first, n int) []float32 {
	out := make([]float32, n)
	var centre [3]float64
	haveCentre := false
	a.line = DCLine{}
	if !a.keepDC {
		if centre, haveCentre = a.centrePower(); haveCentre {
			centre, a.line = a.settleCentre(centre)
		}
	}
	half := a.n / 2
	for j := range out {
		p := a.sum[first+j] / float64(max(a.frames, 1))
		if k := first + j - (half - 1); haveCentre && k >= 0 && k < 3 {
			p = centre[k]
		}
		if p < 1e-30 {
			p = 1e-30
		}
		out[j] = float32(10 * math.Log10(p))
	}
	return out
}

// WriteCSV writes rows in rtl_power's layout, one line per row:
//
//	date, time, Hz low, Hz high, Hz step, samples, dB, dB, ...
//
// Date and time are the row's local time ("2006-01-02", "15:04:05").
func WriteCSV(w io.Writer, rows []Row) error {
	var b strings.Builder
	for _, r := range rows {
		b.Reset()
		fmt.Fprintf(&b, "%s, %s, %d, %d, %.2f, %d",
			r.Time.Format("2006-01-02"), r.Time.Format("15:04:05"),
			int64(math.Round(r.LowHz)), int64(math.Round(r.HighHz)), r.StepHz, r.Samples)
		for _, v := range r.DB {
			fmt.Fprintf(&b, ", %.2f", v)
		}
		b.WriteByte('\n')
		if _, err := io.WriteString(w, b.String()); err != nil {
			return err
		}
	}
	return nil
}
