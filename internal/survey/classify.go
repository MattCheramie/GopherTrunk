// Package survey turns a detected carrier into a classified signal. Where the
// hunt package's sweeper finds where energy is (carriers above the noise
// floor), survey answers what each carrier is: a cheap, blind, per-carrier
// classifier that names the modulation family (analog NBFM/WFM, AM, digital
// FSK/C4FM/PSK, continuous data) plus an occupied-bandwidth estimate, and the
// conventional decoders (POCSAG/FLEX paging, analog-FM activity) that turn a
// classified carrier into a decode summary.
//
// survey deliberately depends only downward — on the dsp primitives and the
// existing radio decoders — and never on hunt, so the hunt package can import
// it to route swept candidates without an import cycle. The trunking
// identify→decode→map path stays in siglab/hunt; survey hands a candidate off
// to it (via hunt) only when the class warrants the (expensive) full identify.
package survey

import (
	"math"
	"math/cmplx"
	"sort"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/fft"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/window"
)

// SignalClass names the modulation family the classifier assigns to a carrier.
// It is intentionally coarse: the goal is to route a carrier to the right
// decoder, not to fully demodulate it. A digital class that turns out to carry
// a trunking control channel is refined to ClassTrunkControl/ClassTrunkVoice by
// the router after the authoritative siglab identify.
type SignalClass string

const (
	ClassUnknown        SignalClass = "unknown"       // no carrier / too weak to classify
	ClassAM             SignalClass = "am"            // amplitude modulation (non-constant envelope)
	ClassNBFM           SignalClass = "nbfm"          // narrowband analog FM (≤ ~16 kHz)
	ClassWideFM         SignalClass = "wfm"           // wideband analog FM (broadcast / ~150–200 kHz)
	ClassFSK            SignalClass = "fsk"           // 2-level frequency-shift keying
	ClassC4FM           SignalClass = "c4fm"          // 4-level FSK (P25 C4FM / YSF family)
	ClassPSK            SignalClass = "psk"           // phase-shift keying (π/4-DQPSK and friends)
	ClassContinuousData SignalClass = "data"          // continuous digital carrier, family unresolved
	ClassPaging         SignalClass = "paging"        // FSK at a paging baud (512/1200/1600/2400)
	ClassTrunkControl   SignalClass = "trunk-control" // assigned by the router after a CC lock
	ClassTrunkVoice     SignalClass = "trunk-voice"   // assigned by the router: decoded, no CC lock
)

// ClassFeatures are the raw DSP measurements behind a classification. They are
// carried on the result so a survey is debuggable and golden-testable (the
// class is a thresholded view of these numbers).
type ClassFeatures struct {
	// OccupiedBwHz is the contiguous bandwidth around DC standing above the
	// noise floor (the carrier was tuned to baseband before classification).
	OccupiedBwHz uint32 `json:"occupied_bw_hz"`
	// SNRDb is the peak-to-noise-floor ratio of the baseband spectrum.
	SNRDb float64 `json:"snr_db"`
	// EnvelopeCV is the coefficient of variation (std/mean) of |z[n]|. AM has
	// a high CV; the constant-envelope angle modulations sit near zero.
	EnvelopeCV float64 `json:"envelope_cv"`
	// IFStd is the standard deviation of the FM-discriminator output
	// (rad/sample), a proxy for peak deviation.
	IFStd float64 `json:"if_std"`
	// IFKurtosis is the excess kurtosis of the discriminator output. PSK is
	// impulsive (high kurtosis: spikes at phase transitions, ~0 between);
	// FSK/analog are flatter.
	IFKurtosis float64 `json:"if_kurtosis"`
	// BaudHz is the cyclostationary symbol-rate line found in the rectified
	// discriminator spectrum, or 0 when none stands out (analog voice).
	BaudHz float64 `json:"baud_hz"`
	// BaudProminence is how far the baud line stands above the local median
	// (ratio). Higher ⇒ a more confident digital call.
	BaudProminence float64 `json:"baud_prominence"`
	// IFModality is the number of levels detected in the discriminator
	// histogram (2 ⇒ FSK, 4 ⇒ C4FM, 0/1 ⇒ analog or PSK-at-zero).
	IFModality int `json:"if_modality"`
}

// Classification is the verdict for one carrier.
type Classification struct {
	Class        SignalClass   `json:"class"`
	Confidence   float64       `json:"confidence"` // 0..1
	OccupiedBwHz uint32        `json:"occupied_bw_hz"`
	Features     ClassFeatures `json:"features"`
}

// minClassifySamples is the shortest baseband buffer worth classifying — below
// this the spectral estimates are too noisy to trust.
const minClassifySamples = 4096

// ClassifyConfig holds the decision thresholds the classifier uses. The zero
// value is invalid; callers either use Classify (which applies
// DefaultClassifyConfig) or pass a config whose zero fields are filled in by
// withDefaults. Exposing these lets operators tune for a noisy SDR front-end or
// a non-standard channel plan without recompiling.
type ClassifyConfig struct {
	// SNRGateDb is the minimum peak-over-noise-floor (dB) for a carrier to be
	// classified at all; below it the carrier reads as unknown.
	SNRGateDb float64
	// OccupiedThreshDb is how far above the noise floor a bin must stand to
	// count toward the occupied bandwidth.
	OccupiedThreshDb float64
	// AMEnvelopeCV is the envelope coefficient-of-variation above which a
	// carrier reads as AM (non-constant envelope).
	AMEnvelopeCV float64
	// AMMaxIFStd is the FM-discriminator std (rad/sample) below which a
	// high-envelope-CV carrier reads as AM. Above it the carrier carries real
	// angle modulation (FM / digital FM) and is not AM even when noise or
	// channel clipping has lifted its envelope CV past AMEnvelopeCV.
	AMMaxIFStd float64
	// DigitalProminence is the minimum rectified-discriminator baud-line
	// prominence for a carrier to read as digital.
	DigitalProminence float64
	// NBFMMaxBwHz is the occupied-bandwidth boundary between narrowband and
	// wideband analog FM.
	NBFMMaxBwHz uint32
	// PSKKurtosis is the discriminator excess-kurtosis above which a digital
	// carrier with no level structure reads as phase modulation.
	PSKKurtosis float64
}

// DefaultClassifyConfig returns the tuned defaults (validated against the
// synthesized fixtures in classify_test.go).
func DefaultClassifyConfig() ClassifyConfig {
	return ClassifyConfig{
		SNRGateDb:         3,
		OccupiedThreshDb:  6,
		AMEnvelopeCV:      0.20,
		AMMaxIFStd:        0.25,
		DigitalProminence: 15,
		NBFMMaxBwHz:       25_000,
		PSKKurtosis:       2,
	}
}

// withDefaults fills any zero field from DefaultClassifyConfig so a partially
// specified config (e.g. only DigitalProminence set from a CLI flag) still has
// sensible values everywhere.
func (c ClassifyConfig) withDefaults() ClassifyConfig {
	d := DefaultClassifyConfig()
	if c.SNRGateDb == 0 {
		c.SNRGateDb = d.SNRGateDb
	}
	if c.OccupiedThreshDb == 0 {
		c.OccupiedThreshDb = d.OccupiedThreshDb
	}
	if c.AMEnvelopeCV == 0 {
		c.AMEnvelopeCV = d.AMEnvelopeCV
	}
	if c.AMMaxIFStd == 0 {
		c.AMMaxIFStd = d.AMMaxIFStd
	}
	if c.DigitalProminence == 0 {
		c.DigitalProminence = d.DigitalProminence
	}
	if c.NBFMMaxBwHz == 0 {
		c.NBFMMaxBwHz = d.NBFMMaxBwHz
	}
	if c.PSKKurtosis == 0 {
		c.PSKKurtosis = d.PSKKurtosis
	}
	return c
}

// pagingBauds are the symbol rates that route a 2-level FSK carrier to the
// paging decoders. POCSAG runs at 512/1200/2400, FLEX at 1600.
var pagingBauds = []float64{512, 1200, 1600, 2400}

// Classify measures the baseband carrier in iq (already tuned to DC) with the
// default config, measuring occupied bandwidth from iq itself.
func Classify(iq []complex64, rateHz float64) Classification {
	return ClassifyWith(iq, rateHz, 0, 0, DefaultClassifyConfig())
}

// ClassifyWith classifies a baseband carrier with an explicit config. When
// occBwHz > 0 it is used as the occupied bandwidth (and snrDb as the SNR)
// instead of measuring from iq — the survey router passes a measurement taken
// on a wider, un-decimated view so a wideband signal isn't mis-measured by the
// narrow channel iq is decimated to. occBwHz == 0 measures from iq.
func ClassifyWith(iq []complex64, rateHz float64, occBwHz uint32, snrDb float64, cfg ClassifyConfig) Classification {
	cfg = cfg.withDefaults()
	feat := extractFeatures(iq, rateHz, cfg)
	if occBwHz > 0 {
		feat.OccupiedBwHz, feat.SNRDb = occBwHz, snrDb
	}
	class, conf := decide(feat, cfg)
	return Classification{
		Class:        class,
		Confidence:   conf,
		OccupiedBwHz: feat.OccupiedBwHz,
		Features:     feat,
	}
}

// extractFeatures computes the full feature vector for a baseband capture.
func extractFeatures(iq []complex64, rateHz float64, cfg ClassifyConfig) ClassFeatures {
	var f ClassFeatures
	if len(iq) < minClassifySamples || rateHz <= 0 {
		return f
	}

	// 1. Occupied bandwidth + SNR from the windowed IQ power spectrum. The
	//    blind (isolated-carrier) path is already band-limited, so no cap.
	f.OccupiedBwHz, f.SNRDb = OccupiedBandwidth(iq, rateHz, 0, cfg)

	// 2. Envelope coefficient of variation (AM vs constant-envelope).
	f.EnvelopeCV = envelopeCV(iq)

	// 3. FM-discriminator features: deviation, impulsiveness, the
	//    cyclostationary baud line, and the level structure.
	d := demod.NewFM().Process(nil, iq)
	mean := meanF32(d)
	f.IFStd = stddevF32(d, mean)
	f.IFKurtosis = excessKurtosisF32(d, mean, f.IFStd)
	f.BaudHz, f.BaudProminence = baudLine(d, mean, rateHz)
	// Level count is only meaningful for a digital carrier; compute it by
	// folding the discriminator at the detected symbol rate (integrate-and-dump
	// over each symbol period) so the levels are clean, the way a slicer sees
	// them — at the raw sample level a pulse-shaped carrier smears its levels.
	if f.BaudHz > 0 {
		f.IFModality = symbolModality(d, mean, rateHz, f.BaudHz)
	}
	return f
}

// decide maps the feature vector to a class and a 0..1 confidence. The cascade
// mirrors siglab's weighted-evidence style: each branch reports how cleanly its
// gate was met so a marginal call reads as low confidence and a clear one high.
func decide(f ClassFeatures, cfg ClassifyConfig) (SignalClass, float64) {
	// No usable carrier.
	if f.OccupiedBwHz == 0 || f.SNRDb < cfg.SNRGateDb {
		return ClassUnknown, 0
	}

	// Digital first: a strong cyclostationary baud line in the rectified
	// discriminator is the key-independent marker of symbol transitions, and it
	// is checked before AM because a linearly-modulated digital carrier
	// (π/4-DQPSK and friends) has real envelope variation from pulse shaping
	// that would otherwise trip the AM gate. Analog voice (continuous,
	// speech-shaped) produces only a smooth roll-off whose largest bin barely
	// clears the median, so it won't reach the prominence gate. The classifier
	// is the coarse router here — it names the family (PSK vs FSK vs C4FM); the
	// authoritative fine call is made downstream by the paging decoders and the
	// siglab identify the router hands a digital carrier to.
	if f.BaudHz > 0 && f.BaudProminence >= cfg.DigitalProminence {
		conf := 0.5 + 0.5*clamp01((f.BaudProminence-cfg.DigitalProminence)/30)
		// An impulsive discriminator (spikes at phase transitions) marks phase
		// modulation — including π/4-DQPSK, whose phase-change values also show
		// four levels, so kurtosis is checked before the 4-level C4FM branch.
		if f.IFKurtosis > cfg.PSKKurtosis {
			return ClassPSK, conf
		}
		if f.IFModality >= 4 {
			return ClassC4FM, conf // clean 4-level fold (best-effort upgrade)
		}
		return ClassFSK, conf
	}

	// AM: a non-constant envelope with no symbol structure AND little angle
	// modulation. Real AM carries its message in the envelope (high CV) with a
	// near-static carrier frequency (IFStd≈0); a constant-envelope FM / digital-FM
	// carrier (DMR, or an FM repeater with CTCSS) can have its envelope CV lifted
	// past the gate by noise or channel clipping, but its discriminator still
	// swings with the deviation. Requiring low IFStd keeps those out of AM — they
	// fall to the FM split below — so a DMR carrier whose fragile baud line was
	// lost isn't mislabelled am (issue #648).
	if f.EnvelopeCV > cfg.AMEnvelopeCV && f.IFStd < cfg.AMMaxIFStd {
		conf := clamp01((f.EnvelopeCV - cfg.AMEnvelopeCV) / 0.3)
		return ClassAM, 0.5 + 0.5*conf
	}

	// Analog FM: constant envelope, no baud line. Split by occupied bandwidth.
	if f.OccupiedBwHz <= cfg.NBFMMaxBwHz {
		return ClassNBFM, 0.6
	}
	return ClassWideFM, 0.6
}

// OccupiedBandwidth returns the contiguous bandwidth around DC that stands
// above the noise floor, and the carrier SNR. The capture is baseband (carrier
// at DC), so the occupied span is the run of above-threshold bins bridging the
// centre bin. Returns (0, 0) when no carrier stands out. The survey router
// calls this on the full-rate (un-decimated) capture so a wideband signal's
// bandwidth isn't truncated by the narrow channel decimation.
//
// maxBwHz caps the measured width: the outward walk from DC stops at ±maxBwHz/2
// even if the gap tolerance would otherwise bridge into an adjacent carrier.
// This keeps a dense band (e.g. DMR carriers on a 12.5 kHz grid, whose edges sit
// inside the small gap tolerance) from chaining its neighbours into one giant
// span. maxBwHz == 0 leaves the walk unbounded (a genuinely isolated wideband
// signal, where nothing adjacent can bridge in).
func OccupiedBandwidth(iq []complex64, rateHz float64, maxBwHz uint32, cfg ClassifyConfig) (uint32, float64) {
	cfg = cfg.withDefaults()
	n := pow2Floor(len(iq))
	if n > 8192 {
		n = 8192
	}
	if n < 256 {
		return 0, 0
	}
	pwr := powerSpectrumDb(iq[:n]) // FFT-shifted: DC in the middle
	floor := percentile(pwr, 0.25)
	peak := maxF32(pwr)
	snr := float64(peak - floor)

	thresh := floor + float32(cfg.OccupiedThreshDb)
	binHz := rateHz / float64(n)
	dc := n / 2

	// Walk outward from DC to the carrier edges, tolerating a few sub-threshold
	// bins (a DC notch, or a single noise dip inside the signal) so the measured
	// width isn't truncated to a sliver. The gap is kept small — a handful of
	// bins — so it bridges a narrow notch without chaining across the wide noise
	// field beyond the carrier's real edge.
	maxGap := n / 1024
	if maxGap < 3 {
		maxGap = 3
	}
	// Cap the outward walk at ±maxBwHz/2 bins so a dense band's adjacent
	// carriers can't bridge across the gap tolerance into one span.
	loLimit, hiLimit := 0, n-1
	if maxBwHz > 0 {
		half := int(math.Round((float64(maxBwHz) / 2) / binHz))
		if half < 1 {
			half = 1
		}
		if dc-half > loLimit {
			loLimit = dc - half
		}
		if dc+half < hiLimit {
			hiLimit = dc + half
		}
	}
	lo, hi := dc, dc
	for i, gap := dc-1, 0; i >= loLimit; i-- {
		if pwr[i] >= thresh {
			lo, gap = i, 0
		} else if gap++; gap > maxGap {
			break
		}
	}
	for i, gap := dc+1, 0; i <= hiLimit; i++ {
		if pwr[i] >= thresh {
			hi, gap = i, 0
		} else if gap++; gap > maxGap {
			break
		}
	}
	// No carrier near DC at all (centre and its immediate neighbourhood are all
	// noise) ⇒ nothing occupies this channel.
	if lo == dc && hi == dc && pwr[dc] < thresh {
		return 0, snr
	}
	bw := float64(hi-lo+1) * binHz
	return uint32(math.Round(bw)), snr
}

// powerSpectrumDb windows iq, FFTs it, and returns per-bin power in dB,
// FFT-shifted so DC lands at index n/2 (matching the sweeper's convention).
func powerSpectrumDb(iq []complex64) []float32 {
	n := len(iq)
	win := window.Hann(n)
	in := make([]complex128, n)
	for i, s := range iq {
		w := win[i]
		in[i] = complex(float64(real(s))*w, float64(imag(s))*w)
	}
	out := fft.New(n).Forward(nil, in)
	pwr := make([]float32, n)
	half := n / 2
	for i := 0; i < n; i++ {
		dst := (i + half) % n
		mag := cmplx.Abs(out[i])
		p := mag * mag
		if p < 1e-30 {
			p = 1e-30
		}
		pwr[dst] = float32(10 * math.Log10(p))
	}
	return pwr
}

// baudLine searches the rectified-discriminator spectrum for a cyclostationary
// symbol-rate line. Symbol transitions in a digital carrier recur at the baud
// rate; rectifying the discriminator (|d - mean|) exposes that as a discrete
// spectral line, while analog voice produces only a smooth roll-off. Returns
// the line frequency in Hz and its prominence over the local median, or (0, 0).
func baudLine(d []float32, mean float64, rateHz float64) (float64, float64) {
	n := pow2Floor(len(d))
	if n > 16384 {
		n = 16384
	}
	if n < 1024 {
		return 0, 0
	}
	// Rectify around the mean and remove DC so the transform isn't dominated
	// by the (large) zero-frequency bin.
	rect := make([]float64, n)
	var rmean float64
	for i := 0; i < n; i++ {
		rect[i] = math.Abs(float64(d[i]) - mean)
		rmean += rect[i]
	}
	rmean /= float64(n)
	win := window.Hann(n)
	in := make([]complex128, n)
	for i := 0; i < n; i++ {
		in[i] = complex((rect[i]-rmean)*win[i], 0)
	}
	out := fft.New(n).Forward(nil, in)

	binHz := rateHz / float64(n)
	// Plausible baud band: 300 Hz .. rate/2.4 (above any audio roll-off, below
	// the fold region near Nyquist). The upper edge reaches ~20 kHz at a 48 kHz
	// channel so TETRA's 18 kbaud symbol line is searchable, not just the
	// sub-16 kHz rates a rate/3 cap allowed.
	loBin := int(300 / binHz)
	if loBin < 1 {
		loBin = 1
	}
	hiBin := n / 2
	if maxBin := int((rateHz / 2.4) / binHz); maxBin < hiBin {
		hiBin = maxBin
	}
	if hiBin <= loBin {
		return 0, 0
	}
	mags := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		mags[i] = cmplx.Abs(out[i])
	}
	// Peak bin in the baud band.
	peakBin, peakMag := loBin, 0.0
	for i := loBin; i < hiBin; i++ {
		if mags[i] > peakMag {
			peakMag, peakBin = mags[i], i
		}
	}
	if peakMag == 0 {
		return 0, 0
	}
	// Prominence = peak / median of the band. A digital carrier's symbol-rate
	// line stands far above the broadband floor; analog voice produces only a
	// smooth roll-off whose largest bin barely clears the median.
	med := medianOf(mags[loBin:hiBin])
	if med <= 0 {
		return 0, 0
	}
	return float64(peakBin) * binHz, peakMag / med
}

// symbolModality estimates the number of discrete levels a digital carrier
// uses by folding the discriminator at the symbol rate: it integrates each
// symbol period to one value (an integrate-and-dump, like the pager slicer),
// then histograms those symbol values and counts the well-separated peaks.
// 2 ⇒ binary FSK, 4 ⇒ C4FM, 1 ⇒ a carrier with no level structure (PSK/analog).
func symbolModality(d []float32, mean, rateHz, baudHz float64) int {
	sps := int(math.Round(rateHz / baudHz))
	if sps < 2 || len(d) < sps*64 {
		return 0
	}
	syms := make([]float32, 0, len(d)/sps)
	for i := 0; i+sps <= len(d); i += sps {
		var acc float64
		for j := 0; j < sps; j++ {
			acc += float64(d[i+j])
		}
		syms = append(syms, float32(acc/float64(sps)))
	}
	smean := meanF32(syms)
	sstd := stddevF32(syms, smean)
	if sstd <= 0 {
		return 0
	}

	const bins = 64
	const span = 3.0 // ±3σ
	hist := make([]float64, bins)
	for _, v := range syms {
		x := (float64(v) - smean) / sstd // normalised
		idx := int((x + span) / (2 * span) * float64(bins))
		if idx < 0 || idx >= bins {
			continue
		}
		hist[idx]++
	}
	// Heavy smoothing (three box passes) so the level clusters survive but
	// quantisation ripple does not spawn spurious peaks.
	sm := hist
	for pass := 0; pass < 3; pass++ {
		sm = boxSmooth3(sm)
	}
	peak := maxFloat(sm)
	if peak == 0 {
		return 0
	}

	// Collect candidate local maxima (strict max over ±2 bins) at least 12% of
	// the global peak.
	const minRel = 0.12
	thresh := minRel * peak
	type pk struct {
		bin int
		h   float64
	}
	var cands []pk
	for i := 2; i < bins-2; i++ {
		if sm[i] < thresh {
			continue
		}
		if sm[i] >= sm[i-1] && sm[i] >= sm[i-2] && sm[i] > sm[i+1] && sm[i] >= sm[i+2] {
			cands = append(cands, pk{i, sm[i]})
		}
	}
	// Count peaks separated by a genuine valley: two adjacent candidates merge
	// unless the minimum between them dips below 55% of the shorter peak. This
	// yields 2 for binary FSK, 4 for C4FM, and 1 for a unimodal analog signal.
	const valleyRatio = 0.55
	modes := 0
	var lastBin int
	lastH := 0.0
	for _, c := range cands {
		if modes == 0 {
			modes, lastBin, lastH = 1, c.bin, c.h
			continue
		}
		valley := sm[lastBin]
		for i := lastBin; i <= c.bin; i++ {
			if sm[i] < valley {
				valley = sm[i]
			}
		}
		if valley < valleyRatio*math.Min(lastH, c.h) {
			modes++
			lastBin, lastH = c.bin, c.h
		} else if c.h > lastH {
			lastBin, lastH = c.bin, c.h // merge into the taller peak
		}
	}
	return modes
}

// boxSmooth3 returns a 3-bin moving average of v (edge-clamped).
func boxSmooth3(v []float64) []float64 {
	out := make([]float64, len(v))
	for i := range v {
		s, c := v[i], 1.0
		if i > 0 {
			s += v[i-1]
			c++
		}
		if i < len(v)-1 {
			s += v[i+1]
			c++
		}
		out[i] = s / c
	}
	return out
}

// standardChannelWidths are the occupied bandwidths of real narrowband
// land-mobile channels. A measured width close to one of these is the channel;
// off-grid values like 11.6 kHz are measurement artifacts (an off-centre carrier
// skews the outward DC walk), not bandwidths any system actually uses.
var standardChannelWidths = []uint32{6_250, 12_500, 25_000}

// SnapChannelBandwidth normalises a measured occupied bandwidth for reporting:
// when bwHz lands within ±20% of a standard narrowband channel width it returns
// that width (so a carrier reads as the 12.5 kHz channel it is, not "11.6 kHz");
// otherwise it returns bwHz unchanged so a genuinely wideband signal (broadcast
// FM, a wideband data carrier) is reported as-measured. This only adjusts the
// displayed width — the raw measurement stays on ClassFeatures.OccupiedBwHz.
func SnapChannelBandwidth(bwHz uint32) uint32 {
	const tol = 0.20
	for _, w := range standardChannelWidths {
		lo := float64(w) * (1 - tol)
		hi := float64(w) * (1 + tol)
		if float64(bwHz) >= lo && float64(bwHz) <= hi {
			return w
		}
	}
	return bwHz
}

// IsPagingBaud reports whether baud is close to a POCSAG/FLEX signalling rate.
// The router uses it to decide whether a digital FSK carrier is worth a paging
// decode attempt before the (more expensive) trunking identify.
func IsPagingBaud(baud float64) bool {
	for _, b := range pagingBauds {
		if math.Abs(baud-b)/b < 0.06 { // within 6%
			return true
		}
	}
	return false
}

// IsDigital reports whether c is one of the digital modulation families (the
// classes the router forwards to the paging decoders or the trunking identify).
func IsDigital(c SignalClass) bool {
	switch c {
	case ClassFSK, ClassC4FM, ClassPSK, ClassContinuousData, ClassPaging:
		return true
	default:
		return false
	}
}

// --- small numeric helpers (kept local so survey has no hunt dependency) ---

func envelopeCV(iq []complex64) float64 {
	if len(iq) == 0 {
		return 0
	}
	var sum, sumSq float64
	for _, s := range iq {
		m := math.Hypot(float64(real(s)), float64(imag(s)))
		sum += m
		sumSq += m * m
	}
	n := float64(len(iq))
	mean := sum / n
	if mean <= 0 {
		return 0
	}
	varr := sumSq/n - mean*mean
	if varr < 0 {
		varr = 0
	}
	return math.Sqrt(varr) / mean
}

func meanF32(d []float32) float64 {
	if len(d) == 0 {
		return 0
	}
	var s float64
	for _, v := range d {
		s += float64(v)
	}
	return s / float64(len(d))
}

func stddevF32(d []float32, mean float64) float64 {
	if len(d) == 0 {
		return 0
	}
	var s float64
	for _, v := range d {
		x := float64(v) - mean
		s += x * x
	}
	return math.Sqrt(s / float64(len(d)))
}

func excessKurtosisF32(d []float32, mean, std float64) float64 {
	if len(d) == 0 || std <= 0 {
		return 0
	}
	var s float64
	for _, v := range d {
		x := (float64(v) - mean) / std
		s += x * x * x * x
	}
	return s/float64(len(d)) - 3
}

// percentile returns the p-quantile (0..1) of vals.
func percentile(vals []float32, p float64) float32 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]float32(nil), vals...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(p * float64(len(cp)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func medianOf(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]float64(nil), vals...)
	sort.Float64s(cp)
	return cp[len(cp)/2]
}

func maxF32(vals []float32) float32 {
	m := float32(math.Inf(-1))
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

func maxFloat(vals []float64) float64 {
	m := math.Inf(-1)
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

// pow2Floor returns the largest power of two ≤ n (0 for n < 1).
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

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
