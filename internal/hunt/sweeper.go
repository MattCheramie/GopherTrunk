package hunt

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/cmplx"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// Band is an inclusive frequency range to sweep, in Hz.
type Band struct {
	LowHz  uint32
	HighHz uint32
}

// SweepOptions configure a Sweeper.
type SweepOptions struct {
	Source IQSource
	Bands  []Band
	// FFTSize is the bins per step (power of two). 0 ⇒ 4096.
	FFTSize int
	// SweepDwell is how long to accumulate per step before detecting peaks.
	// Captures one FFT frame per ~10 ms of dwell, averaged. 0 ⇒ one frame.
	SweepDwell time.Duration
	// GuardFrac reserves this fraction of each step's bandwidth at the edges
	// (where the SDR rolloff lives) when advancing the center. 0 ⇒ 0.1.
	GuardFrac float64
	PeakOpts  PeakOptions
	Log       *slog.Logger
}

// Candidate is a carrier the sweep found, worth identifying.
type Candidate struct {
	FreqHz uint32  `json:"freq_hz"`
	SNRDb  float32 `json:"snr_db"`
}

// Sweeper walks operator-given bands on an IQSource and returns the candidate
// carriers it finds. It is the discovery front-end: its candidates feed the
// identify→decode→accumulate pipeline (see LiveHunter).
type Sweeper struct {
	opts      SweepOptions
	fftSize   int
	guardFrac float64
	log       *slog.Logger

	// FFT scratch reused across steps (single-goroutine use).
	plan   fft.Plan
	win    []float64
	winSum float64
	bufIQ  []complex128
	bufOut []complex128
}

// NewSweeper validates options and builds a Sweeper.
func NewSweeper(opts SweepOptions) (*Sweeper, error) {
	if opts.Source == nil {
		return nil, fmt.Errorf("hunt: sweeper requires an IQSource")
	}
	if len(opts.Bands) == 0 {
		return nil, fmt.Errorf("hunt: sweeper requires at least one band")
	}
	if opts.Source.SampleRateHz() == 0 {
		return nil, fmt.Errorf("hunt: sweeper IQSource has zero sample rate")
	}
	fftSize := opts.FFTSize
	if fftSize == 0 {
		fftSize = 4096
	}
	if err := ensurePow2(fftSize); err != nil {
		return nil, err
	}
	guardFrac := opts.GuardFrac
	if guardFrac <= 0 || guardFrac >= 0.5 {
		guardFrac = 0.1
	}
	log := opts.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	win := window.Hann(fftSize)
	var winSum float64
	for _, w := range win {
		winSum += w * w
	}
	return &Sweeper{
		opts:      opts,
		fftSize:   fftSize,
		guardFrac: guardFrac,
		log:       log,
		plan:      fft.New(fftSize),
		win:       win,
		winSum:    winSum,
		bufIQ:     make([]complex128, fftSize),
		bufOut:    make([]complex128, fftSize),
	}, nil
}

// OnStep, when set, is called once per tuned step with the step center and the
// peaks found there — used by the live hunt to publish progress.
type OnStep func(centerHz uint32, peaks []Peak)

// Sweep walks every band and returns the de-duplicated candidate carriers,
// sorted by descending SNR. ctx cancellation stops the sweep and returns
// ctx.Err(). onStep may be nil.
func (s *Sweeper) Sweep(ctx context.Context, onStep OnStep) ([]Candidate, error) {
	rate := s.opts.Source.SampleRateHz()
	// Advance the center by the usable bandwidth (minus the guard) so adjacent
	// steps overlap slightly and a carrier near a step edge is still seen
	// somewhere away from the rolloff.
	step := uint32(float64(rate) * (1 - 2*s.guardFrac))
	if step == 0 {
		step = rate
	}
	framesPerStep := 1
	if s.opts.SweepDwell > 0 {
		// ~one frame per 10 ms of dwell, capped so a long dwell can't run away.
		framesPerStep = int(s.opts.SweepDwell / (10 * time.Millisecond))
		if framesPerStep < 1 {
			framesPerStep = 1
		}
		if framesPerStep > 64 {
			framesPerStep = 64
		}
	}

	// Accumulate candidates keyed to a quantized frequency bucket so the same
	// carrier seen in two overlapping steps collapses to one (strongest wins).
	bucket := s.opts.PeakOpts.MinSpacingHz
	if bucket == 0 {
		bucket = 6_250
	}
	best := map[uint32]Candidate{}

	for _, band := range s.opts.Bands {
		if band.HighHz < band.LowHz {
			return nil, fmt.Errorf("hunt: band %d:%d is inverted", band.LowHz, band.HighHz)
		}
		// Center the first step half a usable-bandwidth above the low edge so
		// the band's low edge is inside the first step (not at the rolloff).
		halfUsable := step / 2
		for center := band.LowHz + halfUsable; ; center += step {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if err := s.opts.Source.Tune(center); err != nil {
				return nil, fmt.Errorf("hunt: tune %d Hz: %w", center, err)
			}
			frame, err := s.captureFrame(ctx, center, rate, framesPerStep)
			if err != nil {
				return nil, err
			}
			peaks := DetectPeaks(frame, s.opts.PeakOpts)
			// Keep only peaks that fall within the requested band.
			inBand := peaks[:0]
			for _, p := range peaks {
				if p.FreqHz >= band.LowHz && p.FreqHz <= band.HighHz {
					inBand = append(inBand, p)
				}
			}
			if onStep != nil {
				onStep(center, inBand)
			}
			for _, p := range inBand {
				key := p.FreqHz / bucket
				if cur, ok := best[key]; !ok || p.SNRDb > cur.SNRDb {
					best[key] = Candidate{FreqHz: p.FreqHz, SNRDb: p.SNRDb}
				}
			}
			// Stop once this step's coverage reached the band's high edge.
			if center+halfUsable >= band.HighHz {
				break
			}
		}
	}

	out := make([]Candidate, 0, len(best))
	for _, c := range best {
		out = append(out, c)
	}
	out = snapCandidatesToGrid(out, rate, s.fftSize)
	sort.Slice(out, func(i, j int) bool { return out[i].SNRDb > out[j].SNRDb })
	return out, nil
}

// snapCandidatesToGrid corrects the FFT-bin quantization offset in the swept
// candidates: it infers the channel raster the carriers sit on (auto-detected
// from their frequencies) and snaps each candidate to the nearest grid point
// within one FFT bin, so a carrier the bins reported a few hundred Hz off (e.g.
// 440.1247 MHz for a P25 trunk really on 440.125000) tunes to the true centre
// and actually locks. Snapping is bounded to one bin, so a carrier that
// genuinely sits off the raster is kept untouched. Candidates that collapse to
// the same snapped frequency merge, strongest SNR winning.
func snapCandidatesToGrid(cands []Candidate, rate uint32, fftSize int) []Candidate {
	if len(cands) == 0 || fftSize <= 0 {
		return cands
	}
	step, phase := inferGrid(cands)
	binHz := uint32(math.Round(float64(rate) / float64(fftSize)))
	merged := make(map[uint32]Candidate, len(cands))
	for _, c := range cands {
		c.FreqHz = snapHz(c.FreqHz, step, phase, binHz)
		if cur, ok := merged[c.FreqHz]; !ok || c.SNRDb > cur.SNRDb {
			merged[c.FreqHz] = c
		}
	}
	out := make([]Candidate, 0, len(merged))
	for _, c := range merged {
		out = append(out, c)
	}
	return out
}

// captureFrame captures framesPerStep FFT frames at the current center and
// returns their power-averaged spectrum (averaging lifts a weak carrier out of
// the noise). The returned Frame uses the spectrum.Frame bin convention.
func (s *Sweeper) captureFrame(ctx context.Context, centerHz, rate uint32, frames int) (spectrum.Frame, error) {
	n := s.fftSize
	acc := make([]float64, n) // accumulated linear power per (shifted) bin
	got := 0
	for f := 0; f < frames; f++ {
		iq, err := s.opts.Source.Capture(ctx, captureFrameSamples(n))
		if err != nil {
			return spectrum.Frame{}, err
		}
		if len(iq) < n {
			if got == 0 {
				return spectrum.Frame{}, fmt.Errorf("hunt: short capture (%d < %d) at %d Hz", len(iq), n, centerHz)
			}
			break
		}
		s.accumulatePower(acc, iq[:n])
		got++
	}
	bins := make([]float32, n)
	norm := 1.0 / (float64(n) * s.winSum)
	for i := 0; i < n; i++ {
		power := acc[i] / float64(got) * norm
		if power < 1e-30 {
			power = 1e-30
		}
		bins[i] = float32(10 * math.Log10(power))
	}
	return spectrum.Frame{
		Timestamp:  time.Now(),
		CenterHz:   centerHz,
		SampleRate: rate,
		Bins:       bins,
	}, nil
}

// accumulatePower windows + FFTs one chunk and adds its per-bin |X|² into acc,
// FFT-shifted so DC lands in the middle (matching spectrum.Frame).
func (s *Sweeper) accumulatePower(acc []float64, iq []complex64) {
	n := s.fftSize
	for i := 0; i < n; i++ {
		v := iq[i]
		w := s.win[i]
		s.bufIQ[i] = complex(float64(real(v))*w, float64(imag(v))*w)
	}
	out := s.plan.Forward(s.bufOut, s.bufIQ)
	half := n / 2
	for i := 0; i < n; i++ {
		dst := (i + half) % n
		mag := cmplx.Abs(out[i])
		acc[dst] += mag * mag
	}
}
