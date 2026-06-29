// Package blind estimates the symbol/baud rate and a coarse modulation class of
// an unknown carrier directly from its channelized IQ, with no protocol
// decoder. It is the "what is this?" front-end for SigLab's identify/wideband
// fallback: when nothing decodes, the blind estimate feeds a reference-database
// ranking (internal/siglab/sigref).
//
// The symbol-rate estimate is the reliable part — two independent methods
// (autocorrelation of the symbol-transition energy and the spectral line that
// same nonlinearity regenerates) must agree for high confidence. The
// modulation-class guess is deliberately coarse and reports "unknown" rather
// than risk a confidently-wrong label, because a wrong class hurts ranking more
// than a missing one.
package blind

import (
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/stats"
)

// Modulation-class labels. These mirror the coarse families the reference DB
// keys on; "unknown" is emitted whenever the features are inconclusive.
const (
	ModUnknown = "unknown"
	ModFSK2    = "fsk2" // 2-level FSK / GFSK / GMSK family
	ModFSK4    = "fsk4" // 4-level FSK / C4FM family
	ModPSK     = "psk"  // constant-envelope phase modulation (incl. π/4-DQPSK)
	ModQAM     = "qam"  // amplitude-bearing (non-constant-envelope) modulation
)

// BlindEstimate is the result of a blind analysis of one carrier.
type BlindEstimate struct {
	// SymbolRateHz is the best symbol-rate estimate; 0 when indeterminate.
	SymbolRateHz float64 `json:"symbol_rate_hz"`
	// SymbolRateConf is 0..1: high when the two methods agree, low otherwise.
	SymbolRateConf float64 `json:"symbol_rate_conf"`
	// SymbolRateCands carries both method estimates when they disagree.
	SymbolRateCands []float64 `json:"symbol_rate_cands,omitempty"`
	// ModClass is one of the Mod* labels; Levels is the detected symbol-level
	// count (2 or 4) or 0 when unknown.
	ModClass string `json:"mod_class"`
	Levels   int    `json:"levels"`
	// OccupiedBWHz is the 99% occupied bandwidth of the carrier.
	OccupiedBWHz float64 `json:"occupied_bw_hz"`
	// Method names the estimators that ran.
	Method string `json:"method"`
}

const (
	maxSamples = 16384 // cap the work per carrier
	acMaxLen   = 4096  // O(n²) autocorrelation input cap
	minBaudHz  = 200.0 // search-band floor
	constEnvCV = 0.35  // |x| coefficient-of-variation below this ⇒ constant envelope
	//            (loose enough to keep a channelized FSK carrier — whose filtered
	//            transitions ripple the envelope — out of the amplitude/QAM class)
	agreeRelTol = 0.10 // method agreement threshold
)

// Estimate analyses a channelized carrier (complex baseband at rateHz) and
// returns its blind symbol-rate / modulation estimate. Returns a zero value for
// too-short input or a non-positive rate.
func Estimate(iq []complex64, rateHz float64) BlindEstimate {
	if len(iq) < 256 || rateHz <= 0 {
		return BlindEstimate{}
	}
	s := iq
	if len(s) > maxSamples {
		s = s[:maxSamples]
	}

	// Instantaneous frequency (rad/sample) and its symbol-transition energy:
	// the FM-discriminator output spikes at every symbol boundary, so its
	// first difference squared is a near-impulse train at the symbol rate.
	d := make([]float64, len(s)-1)
	for i := 1; i < len(s); i++ {
		d[i-1] = phaseDiff(s[i], s[i-1])
	}
	tr := make([]float64, len(d)-1)
	for i := 1; i < len(d); i++ {
		x := d[i] - d[i-1]
		tr[i-1] = x * x
	}

	maxBaud := rateHz / 4
	baudA, okA := autocorrBaud(tr, rateHz, maxBaud)
	baudB, okB := spectralLineBaud(tr, rateHz, maxBaud)

	out := BlindEstimate{Method: "autocorr+spectral", ModClass: ModUnknown}
	switch {
	case okA && okB:
		if math.Abs(baudA-baudB)/baudB < agreeRelTol {
			out.SymbolRateHz = (baudA + baudB) / 2
			out.SymbolRateConf = 0.9
		} else {
			// Disagreement: trust the autocorrelation's smallest-strong-lag — it
			// pins the fundamental symbol period, whereas a lone spectral line can
			// latch a sub-harmonic from symbol run-length structure.
			out.SymbolRateHz = baudA
			out.SymbolRateConf = 0.4
			out.SymbolRateCands = []float64{baudA, baudB}
		}
	case okA:
		out.SymbolRateHz, out.SymbolRateConf = baudA, 0.5
	case okB:
		out.SymbolRateHz, out.SymbolRateConf = baudB, 0.5
	}

	out.ModClass, out.Levels = classify(s, d)
	out.OccupiedBWHz = occupiedBW(s, rateHz)
	return out
}

// phaseDiff returns the angle of a·conj(b) — the per-sample instantaneous
// frequency in radians/sample.
func phaseDiff(a, b complex64) float64 {
	ar, ai := float64(real(a)), float64(imag(a))
	br, bi := float64(real(b)), float64(imag(b))
	// a·conj(b) = (ar+jai)(br-jbi)
	re := ar*br + ai*bi
	im := ai*br - ar*bi
	return math.Atan2(im, re)
}

// autocorrBaud finds the symbol period as the strongest autocorrelation lag of
// the transition-energy signal within the baud search band, and converts it to
// a rate. Returns ok=false when no lag in the band stands out.
func autocorrBaud(tr []float64, rateHz, maxBaud float64) (float64, bool) {
	m := tr
	if len(m) > acMaxLen {
		m = m[:acMaxLen]
	}
	if len(m) < 8 {
		return 0, false
	}
	m = demean(m)
	corr, _ := stats.Correlate(m, m)
	center := len(m) - 1 // lag 0
	lagLo := int(rateHz / maxBaud)
	if lagLo < 2 {
		lagLo = 2
	}
	lagHi := int(rateHz / minBaudHz)
	if lagHi >= len(m) {
		lagHi = len(m) - 1
	}
	if lagHi <= lagLo {
		return 0, false
	}
	// Peak autocorrelation in the band, then take the SMALLEST lag that is a
	// local maximum near that peak — the fundamental symbol period, not one of
	// its (equally strong, for an impulse train) harmonics at 2×/3× the lag.
	var maxVal float64
	for lag := lagLo; lag <= lagHi; lag++ {
		if v := corr[center+lag]; v > maxVal {
			maxVal = v
		}
	}
	if maxVal < 0.05 {
		return 0, false
	}
	thresh := 0.5 * maxVal
	for lag := lagLo + 1; lag < lagHi; lag++ {
		v := corr[center+lag]
		if v >= thresh && v > corr[center+lag-1] && v >= corr[center+lag+1] {
			return rateHz / float64(lag), true
		}
	}
	return 0, false
}

// spectralLineBaud finds the symbol rate as the dominant non-DC spectral line of
// the transition-energy signal. Returns ok=false when no line stands out.
func spectralLineBaud(tr []float64, rateHz, maxBaud float64) (float64, bool) {
	n := pow2Floor(len(tr))
	if n < 64 {
		return 0, false
	}
	x := demean(tr[:n])
	in := make([]complex128, n)
	for i, v := range x {
		in[i] = complex(v, 0)
	}
	out := fft.New(n).Forward(make([]complex128, n), in)
	binHz := rateHz / float64(n)
	loBin := int(minBaudHz / binHz)
	if loBin < 1 {
		loBin = 1
	}
	hiBin := int(maxBaud / binHz)
	if hiBin >= n/2 {
		hiBin = n/2 - 1
	}
	if hiBin <= loBin {
		return 0, false
	}
	mag := make([]float64, hiBin+2)
	var meanMag float64
	for k := loBin; k <= hiBin; k++ {
		mag[k] = math.Hypot(real(out[k]), imag(out[k]))
		meanMag += mag[k]
	}
	meanMag /= float64(hiBin - loBin + 1)
	if meanMag <= 0 {
		return 0, false
	}
	// Take the LOWEST-frequency line that clears the band floor — the
	// fundamental, not a harmonic (an ideal symbol-clock impulse train has
	// equally strong lines at every multiple of the baud).
	thresh := 3 * meanMag
	for k := loBin + 1; k < hiBin; k++ {
		if mag[k] >= thresh && mag[k] > mag[k-1] && mag[k] >= mag[k+1] {
			return float64(k) * binHz, true
		}
	}
	return 0, false
}

// classify returns a coarse modulation class + level count from amplitude and
// instantaneous-frequency statistics. Conservative: returns ModUnknown when the
// features do not clearly fit a family.
func classify(s []complex64, d []float64) (string, int) {
	// Amplitude coefficient of variation: low ⇒ constant envelope.
	amp := make([]float64, len(s))
	for i, c := range s {
		amp[i] = math.Hypot(float64(real(c)), float64(imag(c)))
	}
	mean := stats.Mean(amp)
	if mean <= 0 {
		return ModUnknown, 0
	}
	cv := math.Sqrt(stats.Variance(amp, true)) / mean
	if cv >= constEnvCV {
		return ModQAM, 0 // amplitude carries information
	}
	// Constant envelope: count instantaneous-frequency clusters (FSK rails).
	modes := freqModes(d)
	switch modes {
	case 4:
		return ModFSK4, 4
	case 2:
		return ModFSK2, 2
	default:
		// Constant-envelope but no clean FSK rail structure ⇒ likely PSK.
		return ModPSK, 0
	}
}

// freqModes counts the dominant clusters in the instantaneous-frequency
// histogram — the FSK rail count (2 or 4) for an FSK signal.
func freqModes(d []float64) int {
	counts, _ := stats.Histogram(d, 41)
	y := make([]float64, len(counts))
	for i, c := range counts {
		y[i] = float64(c)
	}
	// IncludeEnds so the outer FSK rails — which land in the first/last
	// histogram bins — are counted, not silently dropped.
	peaks := stats.FindPeaks(y, stats.PeakOpts{RelHeight: 0.25, MinDistance: 4, IncludeEnds: true})
	return len(peaks)
}

// occupiedBW returns the 99% occupied bandwidth of s, reusing the spectrum
// package's occupancy math on a Welch-averaged spectrum.
func occupiedBW(s []complex64, rateHz float64) float64 {
	n := pow2Floor(len(s))
	if n < 256 {
		return 0
	}
	if n > 4096 {
		n = 4096
	}
	frame := spectrum.AverageDB(s, 0, rateHz, n)
	occ := spectrum.ComputeOccupancy(frame.Bins, rateHz, rateHz, 0, 0)
	return occ.OccupiedBandwidthHz
}

func demean(x []float64) []float64 {
	m := stats.Mean(x)
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = v - m
	}
	return out
}

func pow2Floor(n int) int {
	if n < 1 {
		return 0
	}
	p := 1
	for p<<1 <= n {
		p <<= 1
	}
	return p
}
