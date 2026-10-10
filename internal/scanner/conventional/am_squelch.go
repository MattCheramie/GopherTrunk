package conventional

import (
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// AM squelch (issue #1219).
//
// The FM channels' squelch is IQ power against squelch_dbfs. An AM channel
// cannot use a noise-quieting statistic (AM has no capture effect), and the
// rule for every new gate here is that it must not depend on absolute dBFS
// (a gain change would silently move it). So an AM channel's squelch is the
// carrier's CARRIER-TO-NOISE ratio, measured from the channel's own
// spectrum: an AM transmitter always radiates its carrier — a spectral line
// holding at least half the signal power — while receiver noise is flat.
//
// amCNMeter decimates the scanner's IQ to ~48 kHz, takes a Welch-averaged
// power spectrum (Hann, amCNFFTSize bins of ~188 Hz, amCNBlocks blocks),
// and reports the strongest bin within ±amCNCarrierHalfHz of the channel
// centre over the noise floor, taken as a low percentile of the bins within
// ±amCNFloorHalfHz. The percentile (not the mean) keeps the floor honest when
// the voice sidebands or an 8.33/25 kHz neighbour occupy part of the view.
// Both terms scale together with gain, so the ratio is level-independent;
// on noise alone it sits a few dB above 0 (the largest of ~48 averaged noise
// bins over the 20th percentile). The value is the C/N in one ~188 Hz bin.

const (
	// DefaultAMSquelchCNDb is the open threshold applied when an AM
	// channel leaves SquelchCNDb unset. Noise alone reads ~3–7 dB
	// (TestAMCNMeterNoiseStaysClosed pins the ceiling); 12 dB opens on a
	// carrier ~12 dB over the noise in a 188 Hz bin — a weak but
	// intelligible AM signal.
	DefaultAMSquelchCNDb = 12.0

	amCNRefRateHz = 48_000
	amCNFFTSize   = 256
	amCNBlocks    = 8
	// amCNCarrierHalfHz bounds where the carrier may sit relative to the
	// tuned frequency: air-band transmitters and the SDR's own ppm error
	// leave it within a few kHz, and ±4.5 kHz stays clear of an 8.33 kHz
	// neighbour's carrier.
	amCNCarrierHalfHz = 4_500
	// amCNFloorHalfHz is the span the noise floor is taken over — inside
	// the flat part of the decimator's passband.
	amCNFloorHalfHz = 12_000
	// amCNFloorPercentile picks the noise floor from the sorted bins.
	amCNFloorPercentile = 0.2
	// amMinDwell covers the meter's first full Welch average (8 × 256
	// samples at 48 kHz ≈ 43 ms) plus decimator settle and chunk margin.
	amMinDwell = 100 * time.Millisecond
)

// amCNMeter is one AM channel's carrier-to-noise meter.
type amCNMeter struct {
	pre     *dsp.Resampler // nil when the input is already near 48 kHz
	rate    float64
	win     []float64
	plan    fft.Plan
	pending []complex64 // decimated samples not yet in a block
	scratch []complex64
	in      []complex128
	out     []complex128
	blocks  [amCNBlocks][]float64 // per-block PSDs (ring)
	sum     []float64             // running sum over blocks
	next    int
	filled  int
	level   float64 // latest C/N (dB); -Inf until amCNBlocks blocks are in
	carrier []int   // bin indices within ±amCNCarrierHalfHz
	floor   []int   // bin indices within ±amCNFloorHalfHz
	sorted  []float64
}

func newAMCNMeter(sampleHz float64) *amCNMeter {
	m := int(sampleHz / amCNRefRateHz)
	if m < 1 {
		m = 1
	}
	a := &amCNMeter{
		rate:  sampleHz / float64(m),
		win:   window.Hann(amCNFFTSize),
		plan:  fft.New(amCNFFTSize),
		in:    make([]complex128, amCNFFTSize),
		out:   make([]complex128, amCNFFTSize),
		sum:   make([]float64, amCNFFTSize),
		level: math.Inf(-1),
	}
	if m > 1 {
		a.pre = dsp.NewResampler(1, m, m*8+1, 8.6)
	}
	for i := range a.blocks {
		a.blocks[i] = make([]float64, amCNFFTSize)
	}
	binHz := a.rate / amCNFFTSize
	floorHalf := math.Min(amCNFloorHalfHz, 0.4*a.rate)
	for k := 0; k < amCNFFTSize; k++ {
		f := float64(k) * binHz
		if k >= amCNFFTSize/2 {
			f = float64(k-amCNFFTSize) * binHz
		}
		if math.Abs(f) <= amCNCarrierHalfHz {
			a.carrier = append(a.carrier, k)
		}
		if math.Abs(f) <= floorHalf {
			a.floor = append(a.floor, k)
		}
	}
	a.sorted = make([]float64, len(a.floor))
	return a
}

// process folds iq into the meter and returns the latest C/N in dB
// (-Inf until the first full Welch average is in).
func (a *amCNMeter) process(iq []complex64) float64 {
	x := iq
	if a.pre != nil {
		a.scratch = a.pre.Process(a.scratch, iq)
		x = a.scratch
	}
	a.pending = append(a.pending, x...)
	for len(a.pending) >= amCNFFTSize {
		a.block(a.pending[:amCNFFTSize])
		a.pending = a.pending[amCNFFTSize:]
	}
	// Keep the backing array from growing without bound.
	if cap(a.pending) > 8*amCNFFTSize {
		a.pending = append([]complex64(nil), a.pending...)
	}
	return a.level
}

func (a *amCNMeter) block(x []complex64) {
	for i, s := range x {
		a.in[i] = complex(float64(real(s))*a.win[i], float64(imag(s))*a.win[i])
	}
	a.out = a.plan.Forward(a.out, a.in)
	psd := a.blocks[a.next]
	for k, c := range a.out {
		p := real(c)*real(c) + imag(c)*imag(c)
		a.sum[k] += p - psd[k]
		psd[k] = p
	}
	a.next = (a.next + 1) % amCNBlocks
	if a.filled < amCNBlocks {
		a.filled++
		if a.filled < amCNBlocks {
			return
		}
	}
	var peak float64
	for _, k := range a.carrier {
		peak = math.Max(peak, a.sum[k])
	}
	for i, k := range a.floor {
		a.sorted[i] = a.sum[k]
	}
	sort.Float64s(a.sorted)
	floor := a.sorted[int(amCNFloorPercentile*float64(len(a.sorted)))]
	switch {
	case peak <= 0:
		a.level = math.Inf(-1)
	case floor <= 0:
		a.level = math.Inf(1)
	default:
		a.level = 10 * math.Log10(peak/floor)
	}
}

func (a *amCNMeter) reset() {
	if a.pre != nil {
		a.pre.Reset()
	}
	a.pending = a.pending[:0]
	for i := range a.blocks {
		clear(a.blocks[i])
	}
	clear(a.sum)
	a.next, a.filled = 0, 0
	a.level = math.Inf(-1)
}

// ModeAM is the Channel.Mode of an AM channel (issue #1219).
const ModeAM = "am"

// ModeP25 is the Channel.Mode of a P25 Phase 1 conventional channel
// (issue #1239), e.g. the national P25 interoperability channels. It
// squelches on in-channel power like an FM channel and hands the call to
// the composer's P25 Phase 1 (IMBE) voice chain.
const ModeP25 = "p25"

// ValidMode reports whether mode is a Channel.Mode the scanner knows.
func ValidMode(mode string) bool {
	switch mode {
	case "fm", "nfm", ModeAM, ModeP25:
		return true
	}
	return false
}

// IsDigitalMode reports whether mode carries digital voice, which has no
// sub-audible tone and no analog data bursts to decode.
func IsDigitalMode(mode string) bool {
	return mode == ModeP25
}

// conventionalProtocol is the synthetic grant protocol for ch: the
// composer picks its voice chain from it.
func conventionalProtocol(ch Channel) string {
	switch ch.Mode {
	case ModeAM:
		return "am-conv"
	case ModeP25:
		return "p25"
	}
	return "fm-conv"
}

// buildAMMeter returns the carrier-to-noise meter for an AM channel, or nil
// for an FM channel. Without a sample rate the meter cannot be built; the
// channel then falls back to the IQ-power squelch, loudly.
func buildAMMeter(ch Channel, sampleHz float64, log *slog.Logger) *amCNMeter {
	if ch.Mode != ModeAM {
		return nil
	}
	if sampleHz <= 0 {
		if log != nil {
			log.Warn("conv: AM channel but the scanner sample rate is zero; using the squelch_dbfs power squelch instead of carrier-to-noise",
				"freq_hz", ch.FrequencyHz, "label", ch.Label)
		}
		return nil
	}
	return newAMCNMeter(sampleHz)
}

// amMeterFor returns channel idx's AM meter, or nil. Same ownership rule as
// detectorFor.
func (s *Scanner) amMeterFor(idx int) *amCNMeter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if idx < 0 || idx >= len(s.amMeters) {
		return nil
	}
	return s.amMeters[idx]
}

// squelchMeasure returns the per-chunk squelch statistic for channel idx and
// the level at which it opens: carrier-to-noise (dB) against SquelchCNDb on
// an AM channel, in-channel power (dBFS, channel_power.go) against
// SquelchDbFS otherwise — whole-span power only when the scanner has no
// sample rate to build the channel filter from. The dwell's hysteresis
// applies the same way to either.
func (s *Scanner) squelchMeasure(idx int, ch Channel) (level func([]complex64) float64, open float64) {
	if am := s.amMeterFor(idx); am != nil {
		return am.process, ch.SquelchCNDb
	}
	if pm := s.powerMeterFor(idx); pm != nil {
		return pm.process, ch.SquelchDbFS
	}
	return PowerDbFS, ch.SquelchDbFS
}
