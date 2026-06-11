package dmrlcn

import (
	"math"
	"math/cmplx"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// CarrierOnset reports that a carrier on the DMR channel grid transitioned
// from idle to active — i.e. a voice/data call keyed up on it. The
// correlator pairs the onset time with a recently granted LCN.
type CarrierOnset struct {
	FreqHz uint32
	At     time.Time
	PeakDb float32
}

// onsetConfig parameterizes the onset detector. CenterHz/SampleRateHz come
// from the wideband dongle; GridHz is the DMR channel spacing the detector
// scans (12.5 kHz). The defaults in newOnsetDetector cover the rest.
type onsetConfig struct {
	CenterHz     uint32
	SampleRateHz uint32
	GuardFrac    float64
	GridHz       uint32
	FFTSize      int
	FrameRate    float64  // target FFTs per second
	OnThreshDb   float64  // rise this far above the noise floor to key up
	OffThreshDb  float64  // fall below floor+this to key down (hysteresis)
	Debounce     int      // consecutive frames required to flip state
	NoiseAlpha   float64  // EWMA factor for the per-channel noise floor
	Exclude      []uint32 // frequencies never reported (e.g. the CC carrier)
	Now          func() time.Time
}

// gridChannel is one candidate carrier on the DMR channel grid plus its
// onset/offset state.
type gridChannel struct {
	freqHz   uint32
	bins     []int // FFT-shifted bin indices that fall inside this channel
	excluded bool

	floorDb    float64
	floorInit  bool
	active     bool
	aboveCount int
	belowCount int
}

// onsetDetector runs windowed FFTs over a wideband IQ stream, integrates
// power onto the DMR channel grid, and emits a CarrierOnset on each
// channel's idle→active edge. It is fed IQ chunks via Process and reports
// onsets through the emit callback. Not safe for concurrent Process calls.
type onsetDetector struct {
	cfg  onsetConfig
	emit func(CarrierOnset)

	plan   fft.Plan
	win    []float64
	winSum float64
	size   int
	half   int

	bufIQ   []complex128
	bufOut  []complex128
	powBuf  []float64 // FFT-shifted linear power per bin
	pending []complex64

	channels []*gridChannel

	frameSkip int // run one FFT per (frameSkip+1) full windows
	winCount  int
}

func newOnsetDetector(cfg onsetConfig, emit func(CarrierOnset)) *onsetDetector {
	if cfg.GridHz == 0 {
		cfg.GridHz = 12_500
	}
	if cfg.FFTSize <= 0 || cfg.FFTSize&(cfg.FFTSize-1) != 0 {
		cfg.FFTSize = 4096
	}
	if cfg.FrameRate <= 0 {
		cfg.FrameRate = 30
	}
	if cfg.GuardFrac <= 0 {
		cfg.GuardFrac = 0.05
	}
	if cfg.OnThreshDb <= 0 {
		cfg.OnThreshDb = 10
	}
	if cfg.OffThreshDb <= 0 {
		cfg.OffThreshDb = 6
	}
	if cfg.Debounce <= 0 {
		cfg.Debounce = 3
	}
	if cfg.NoiseAlpha <= 0 {
		cfg.NoiseAlpha = 0.02
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	size := cfg.FFTSize
	d := &onsetDetector{
		cfg:    cfg,
		emit:   emit,
		plan:   fft.New(size),
		win:    window.Hann(size),
		size:   size,
		half:   size / 2,
		bufIQ:  make([]complex128, size),
		bufOut: make([]complex128, size),
		powBuf: make([]float64, size),
	}
	for _, w := range d.win {
		d.winSum += w * w
	}

	// One FFT per (frameSkip+1) windows hits ~FrameRate FFTs/sec.
	winRate := float64(cfg.SampleRateHz) / float64(size)
	if skip := int(math.Round(winRate/cfg.FrameRate)) - 1; skip > 0 {
		d.frameSkip = skip
	}

	d.buildGrid()
	return d
}

// buildGrid enumerates the DMR-grid channels inside the usable band and
// precomputes the FFT-shifted bins that fall within each channel's
// occupied bandwidth.
func (d *onsetDetector) buildGrid() {
	center := float64(d.cfg.CenterHz)
	rate := float64(d.cfg.SampleRateHz)
	binW := rate / float64(d.size)
	usableHalf := rate * (0.5 - d.cfg.GuardFrac)
	// Occupied half-bandwidth per channel: a touch under half the grid so
	// adjacent channels don't bleed into each other.
	halfBW := 0.45 * float64(d.cfg.GridHz)

	excluded := make(map[uint32]struct{}, len(d.cfg.Exclude))
	for _, f := range d.cfg.Exclude {
		excluded[f] = struct{}{}
	}

	grid := int64(d.cfg.GridHz)
	lo := int64(math.Ceil((center - usableHalf) / float64(grid)))
	hi := int64(math.Floor((center + usableHalf) / float64(grid)))
	for k := lo; k <= hi; k++ {
		freq := k * grid
		// Channel center must sit comfortably inside the usable band.
		if math.Abs(float64(freq)-center) > usableHalf-halfBW {
			continue
		}
		gc := &gridChannel{freqHz: uint32(freq)}
		if _, ok := excluded[gc.freqHz]; ok {
			gc.excluded = true
		}
		// FFT-shifted bin index for a frequency f:
		//   k = half + round((f-center)/binW)
		loBin := d.half + int(math.Ceil((float64(freq)-halfBW-center)/binW))
		hiBin := d.half + int(math.Floor((float64(freq)+halfBW-center)/binW))
		for b := loBin; b <= hiBin; b++ {
			if b >= 0 && b < d.size {
				gc.bins = append(gc.bins, b)
			}
		}
		if len(gc.bins) > 0 {
			d.channels = append(d.channels, gc)
		}
	}
}

// channels reporting helper for tests.
func (d *onsetDetector) channelCount() int { return len(d.channels) }

// Process accumulates IQ and runs a rate-limited FFT, advancing each
// grid channel's onset state machine per frame.
func (d *onsetDetector) Process(chunk []complex64) {
	d.pending = append(d.pending, chunk...)
	for len(d.pending) >= d.size {
		frameWin := d.pending[:d.size]
		d.pending = d.pending[d.size:]
		d.winCount++
		if d.frameSkip > 0 && (d.winCount-1)%(d.frameSkip+1) != 0 {
			continue
		}
		d.frame(frameWin)
	}
	// Bound pending growth if a consumer stalls.
	if len(d.pending) > 4*d.size {
		d.pending = d.pending[len(d.pending)-d.size:]
	}
}

// frame runs one FFT and updates every channel's state.
func (d *onsetDetector) frame(samples []complex64) {
	for i := 0; i < d.size; i++ {
		s := samples[i]
		w := d.win[i]
		d.bufIQ[i] = complex(float64(real(s))*w, float64(imag(s))*w)
	}
	out := d.plan.Forward(d.bufOut, d.bufIQ)
	norm := 1.0 / (float64(d.size) * d.winSum)
	for i := 0; i < d.size; i++ {
		dst := (i + d.half) % d.size
		mag := cmplx.Abs(out[i])
		d.powBuf[dst] = mag * mag * norm
	}

	now := d.cfg.Now()
	for _, gc := range d.channels {
		var p float64
		for _, b := range gc.bins {
			p += d.powBuf[b]
		}
		if p < 1e-30 {
			p = 1e-30
		}
		db := 10 * math.Log10(p)
		d.updateChannel(gc, db, now)
	}
}

// updateChannel runs the per-channel onset/offset state machine with
// hysteresis and an adaptive noise floor.
func (d *onsetDetector) updateChannel(gc *gridChannel, db float64, now time.Time) {
	if !gc.floorInit {
		gc.floorDb = db
		gc.floorInit = true
		return
	}

	onLevel := gc.floorDb + d.cfg.OnThreshDb
	offLevel := gc.floorDb + d.cfg.OffThreshDb

	switch {
	case db >= onLevel:
		gc.aboveCount++
		gc.belowCount = 0
		if !gc.active && gc.aboveCount >= d.cfg.Debounce {
			gc.active = true
			if !gc.excluded {
				d.emit(CarrierOnset{FreqHz: gc.freqHz, At: now, PeakDb: float32(db)})
			}
		}
	case db <= offLevel:
		gc.belowCount++
		gc.aboveCount = 0
		if gc.active && gc.belowCount >= d.cfg.Debounce {
			gc.active = false
		}
	default:
		// Inside the hysteresis band: relax both edge counters.
		gc.aboveCount = 0
		gc.belowCount = 0
	}

	// Track the noise floor only while idle so an active carrier doesn't
	// drag the floor up and mask itself.
	if !gc.active {
		a := d.cfg.NoiseAlpha
		gc.floorDb = (1-a)*gc.floorDb + a*db
	}
}
