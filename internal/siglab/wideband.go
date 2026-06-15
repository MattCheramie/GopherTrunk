package siglab

import (
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/carriers"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/tuner"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// Wideband-survey defaults. The channelization constants mirror
// internal/scanner/widebandt2 so the offline survey decimates each carrier the
// same way the live wide-band engine does.
const (
	wbDefaultFFTSize       = 4096
	wbDefaultChannelRateHz = 48_000.0
	wbDefaultGuardFrac     = 0.05
	wbDefaultPeakThreshDb  = 10.0
	wbDefaultMinSpacingHz  = 6_250

	// pickStrategy mirror (widebandt2): DDC ≤ 6 taps, polyphase above.
	wbStrategyAutoThreshold = 6
	wbChannelizerBins       = 16
	wbChannelizerTaps       = 16
	wbChannelizerKaiserBeta = 9.0
)

// WidebandConfig configures a wideband multi-carrier survey: the whole-capture
// sample rate + format, the channel rate each carrier is decimated to, and the
// per-carrier identify knobs.
type WidebandConfig struct {
	SampleRateHz       float64             // wideband input rate (required)
	CenterHz           uint32              // absolute capture center (0 ⇒ offsets only)
	Format             SampleFormat        //
	ChannelRateHz      float64             // per-carrier target; 0 ⇒ 48 kHz
	FFTSize            int                 // spectrum FFT; 0 ⇒ 4096
	GuardFrac          float64             // band-edge guard; 0 ⇒ 0.05
	PeakThresholdDb    float32             // 0 ⇒ 10
	MinSpacingHz       uint32              // 0 ⇒ 6.25 kHz
	TunerStrategy      string              // "" | "auto" | "ddc" | "polyphase"
	Candidates         []trunking.Protocol // per-carrier identify set; nil ⇒ all
	IdentifyMaxSamples int64               // per-carrier identify cap; 0 ⇒ whole carrier
	MaxCarriers        int                 // defensive cap on taps; 0 ⇒ no cap
	Conjugate          bool                //
	IQCorrect          bool                //
	// CollectSpectrum retains the Welch-averaged power spectrum on the result
	// (WidebandResult.Spectrum) so callers can plot the band with the detected
	// carriers overlaid. Off by default to keep the offline hunt path lean.
	CollectSpectrum bool         //
	Log             *slog.Logger //
}

// CarrierRole labels a surveyed carrier within a trunked system.
type CarrierRole string

const (
	RoleUnknown CarrierRole = ""
	RoleControl CarrierRole = "control"
	RoleVoice   CarrierRole = "voice"
)

// CarrierResult is one detected carrier surveyed end-to-end.
type CarrierResult struct {
	OffsetHz     float64       `json:"offset_hz" yaml:"offset_hz"`
	FreqHz       uint32        `json:"freq_hz" yaml:"freq_hz"`
	SNRDb        float32       `json:"snr_db" yaml:"snr_db"`
	Protocol     string        `json:"protocol" yaml:"protocol"`
	Confidence   float64       `json:"confidence" yaml:"confidence"`
	Inconclusive bool          `json:"inconclusive" yaml:"inconclusive"`
	Locked       bool          `json:"locked" yaml:"locked"`
	Role         CarrierRole   `json:"role" yaml:"role"`
	ColorCode    uint8         `json:"color_code,omitempty" yaml:"color_code,omitempty"`
	SystemID     uint32        `json:"system_id,omitempty" yaml:"system_id,omitempty"`
	Grants       []GrantRecord `json:"grants,omitempty" yaml:"grants,omitempty"`
	Score        float64       `json:"score" yaml:"score"`

	identify *IdentifyResult // retained for the hunt bridge
	result   *Result         // full Run of the winning protocol
}

// Result returns the winning protocol's full run Result (topology + grants),
// or nil. Used by the offline hunt bridge to fold each carrier into a system.
func (c *CarrierResult) Result() *Result { return c.result }

// Identify returns the carrier's ranked identification.
func (c *CarrierResult) Identify() *IdentifyResult { return c.identify }

// SystemRollup clusters carriers into one trunked-system verdict.
type SystemRollup struct {
	Protocol     string `json:"protocol" yaml:"protocol"`
	Tier3        bool   `json:"tier3" yaml:"tier3"`
	SystemID     uint32 `json:"system_id,omitempty" yaml:"system_id,omitempty"`
	ColorCode    uint8  `json:"color_code,omitempty" yaml:"color_code,omitempty"`
	ControlCount int    `json:"control_count" yaml:"control_count"`
	VoiceCount   int    `json:"voice_count" yaml:"voice_count"`
	LoHz         uint32 `json:"lo_hz" yaml:"lo_hz"`
	HiHz         uint32 `json:"hi_hz" yaml:"hi_hz"`
	CarrierIdx   []int  `json:"carrier_idx" yaml:"carrier_idx"`
	Verdict      string `json:"verdict" yaml:"verdict"`
}

// SurveySpectrum is the Welch-averaged power spectrum of the whole band, in
// the same FFT-shifted dBFS layout as spectrum.Frame: Bins[0] is the lowest
// frequency (CenterHz - SampleRateHz/2) and successive bins step by
// SampleRateHz/len(Bins). Populated only when WidebandConfig.CollectSpectrum is
// set; it lets a UI plot the band with the detected carriers marked.
type SurveySpectrum struct {
	CenterHz     uint32    `json:"center_hz" yaml:"center_hz"`
	SampleRateHz uint32    `json:"sample_rate_hz" yaml:"sample_rate_hz"`
	Bins         []float32 `json:"bins" yaml:"bins"`
}

// WidebandResult is the full survey outcome.
type WidebandResult struct {
	Source        string  `json:"source" yaml:"source"`
	SampleRateHz  float64 `json:"sample_rate_hz" yaml:"sample_rate_hz"`
	CenterHz      uint32  `json:"center_hz" yaml:"center_hz"`
	ChannelRateHz float64 `json:"channel_rate_hz" yaml:"channel_rate_hz"`
	Strategy      string  `json:"strategy" yaml:"strategy"`
	// Spectrum is the averaged band power spectrum for display; nil unless
	// WidebandConfig.CollectSpectrum was set.
	Spectrum *SurveySpectrum `json:"spectrum,omitempty" yaml:"spectrum,omitempty"`
	// DetectedCount is the number of spectral carrier peaks before the
	// per-carrier identify filter dropped the ones that didn't decode to a
	// real protocol (leakage / noise peaks).
	DetectedCount int             `json:"detected_count" yaml:"detected_count"`
	Carriers      []CarrierResult `json:"carriers" yaml:"carriers"`
	Systems       []SystemRollup  `json:"systems" yaml:"systems"`
}

// detectedCarrier is a snapped spectral peak before identification.
type detectedCarrier struct {
	offsetHz float64
	freqHz   uint32
	snr      float32
}

// SurveyWideband decodes the whole capture from r into memory and surveys it.
func SurveyWideband(r io.ReadSeeker, source string, cfg WidebandConfig) (*WidebandResult, error) {
	if cfg.SampleRateHz <= 0 {
		return nil, fmt.Errorf("siglab: wideband sample rate must be > 0")
	}
	decode, bytesPerSample := cfg.Format.Decoder()
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("siglab: read wideband capture: %w", err)
	}
	pairs := len(raw) / bytesPerSample
	if pairs == 0 {
		return nil, fmt.Errorf("siglab: wideband capture %s has no samples", source)
	}
	iq := make([]complex64, pairs)
	decode(raw[:pairs*bytesPerSample], iq)
	return SurveyWidebandIQ(iq, source, cfg)
}

// SurveyWidebandIQ surveys an already-decoded wideband buffer: averaged power
// spectrum → carrier peaks → grid-snap → per-carrier DDC → per-carrier identify
// → cluster into systems. It is the in-memory core the hunt bridge and tests
// share.
func SurveyWidebandIQ(iq []complex64, source string, cfg WidebandConfig) (*WidebandResult, error) {
	if cfg.SampleRateHz <= 0 {
		return nil, fmt.Errorf("siglab: wideband sample rate must be > 0")
	}
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	if cfg.Conjugate {
		for i, s := range iq {
			iq[i] = complex(real(s), -imag(s))
		}
	}
	if cfg.IQCorrect {
		rtlsdr.NewIQImbalanceCorrector().Process(iq)
	}

	chRate := cfg.ChannelRateHz
	if chRate <= 0 {
		chRate = wbDefaultChannelRateHz
	}
	guard := cfg.GuardFrac
	if guard <= 0 {
		guard = wbDefaultGuardFrac
	}
	fftSize := cfg.FFTSize
	if fftSize <= 0 {
		fftSize = wbDefaultFFTSize
	}

	out := &WidebandResult{
		Source:       baseName(source),
		SampleRateHz: cfg.SampleRateHz,
		CenterHz:     cfg.CenterHz,
	}

	// 1-3. Spectrum → peaks → grid-snap.
	frame, cands := detectCarriers(iq, cfg, fftSize, guard)
	if cfg.CollectSpectrum {
		out.Spectrum = &SurveySpectrum{
			CenterHz:     frame.CenterHz,
			SampleRateHz: frame.SampleRate,
			Bins:         frame.Bins,
		}
	}
	out.DetectedCount = len(cands)
	if len(cands) == 0 {
		return out, nil
	}

	// 4. Per-carrier extraction via a tuner Bank (reuse dsp/tuner).
	bank, strategy := newBank(cfg.TunerStrategy, len(cands), cfg.SampleRateHz, chRate, guard)
	out.Strategy = strategy
	bufs := make([][]complex64, len(cands))
	taps := make([]int, 0, len(cands)) // cand index per registered tap
	for i, c := range cands {
		i := i
		err := bank.AddTap(c.offsetHz, func(o []complex64) {
			if len(o) > 0 {
				bufs[i] = append(bufs[i], o...) // copy: the bank reuses o
			}
		})
		if err != nil {
			log.Debug("siglab/wideband: drop carrier", "offset_hz", c.offsetHz, "err", err)
			continue
		}
		taps = append(taps, i)
	}
	chOut := bank.OutputRateHz()
	out.ChannelRateHz = chOut

	const chunk = 8192
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		bank.Process(iq[off:end])
	}

	// 5+6. Identify each carrier; the winner's full Run carries topology/grants.
	// Carriers that don't decode to any real protocol (leakage / noise peaks
	// the spectral detector picked up) are dropped here so the reported list is
	// only genuine carriers.
	for _, i := range taps {
		c := cands[i]
		buf := bufs[i]
		if len(buf) == 0 {
			continue
		}
		cr := CarrierResult{OffsetHz: c.offsetHz, FreqHz: c.freqHz, SNRDb: c.snr}
		idr, err := IdentifyIQ(buf, fmt.Sprintf("%s@%+.0fkHz", baseName(source), c.offsetHz/1000), IdentifyConfig{
			SampleRateHz: chOut,
			Format:       FormatF32,
			Candidates:   cfg.Candidates,
			MaxSamples:   cfg.IdentifyMaxSamples,
			Log:          cfg.Log,
		})
		if err != nil {
			log.Debug("siglab/wideband: identify failed", "offset_hz", c.offsetHz, "err", err)
			continue
		}
		cr.identify = idr
		cr.Protocol = idr.Winner
		cr.Confidence = idr.Confidence
		cr.Inconclusive = idr.Inconclusive
		if res := idr.WinnerResult(); res != nil {
			cr.result = res
			cr.Locked = res.Locked
			cr.Grants = res.Grants
			cr.ColorCode, cr.SystemID = carrierIdentity(res)
		}
		if win := winnerScore(idr); win != nil {
			cr.Score = win.Score
		}
		cr.Role = roleForCarrier(cr)
		if !keepCarrier(cr) {
			continue
		}
		out.Carriers = append(out.Carriers, cr)
	}

	out.Systems = clusterSystems(out.Carriers, cfg.CenterHz, cfg.SampleRateHz, chRate)
	return out, nil
}

// keepCarrier filters the spectral detections down to genuine carriers: a DMR
// member, anything that locked a decoder, or a confident non-DMR
// identification. Inconclusive noise/leakage peaks are dropped.
func keepCarrier(c CarrierResult) bool {
	if isDMRMember(c) || c.Locked {
		return true
	}
	return !c.Inconclusive && c.Confidence >= dmrMemberConfidence
}

// detectCarriers runs the spectrum → peak → centroid-refine → grid-snap front
// end and returns the in-band snapped carrier offsets, strongest first.
//
// Digital land-mobile carriers (C4FM/π4-DQPSK) are 6.25–25 kHz wide with a
// rounded, noisy top, so the single strongest FFT bin can sit hundreds of Hz
// off the true centre — enough to break the per-carrier decode. The detector
// therefore (a) smooths the averaged spectrum over ~half a channel so each
// carrier collapses to one bump, (b) finds the bumps with the shared peak
// detector, (c) refines each centre to the power-weighted centroid of its hump,
// and only then snaps to the channel raster.
// detectCarriers returns the Welch-averaged band spectrum (for optional
// display) alongside the snapped carrier candidates found in it.
func detectCarriers(iq []complex64, cfg WidebandConfig, fftSize int, guard float64) (spectrum.Frame, []detectedCarrier) {
	frame := spectrum.AverageDB(iq, cfg.CenterHz, cfg.SampleRateHz, fftSize)
	binHzF := cfg.SampleRateHz / float64(fftSize)

	// Smooth over ~half the tightest channel so a wide carrier reads as a
	// single bump instead of many noisy local maxima.
	smoothBins := int(math.Round(6_250.0 / 2.0 / binHzF))
	smoothed := frame
	smoothed.Bins = spectrum.Smooth(frame.Bins, smoothBins)

	peaks := carriers.DetectPeaks(smoothed, carriers.PeakOptions{
		ThresholdDb:  cfg.PeakThresholdDb,
		MinSpacingHz: cfg.MinSpacingHz,
	})
	if len(peaks) == 0 {
		return frame, nil
	}

	floor := spectrum.Percentile(smoothed.Bins, 0.25)
	base := float64(cfg.CenterHz) - cfg.SampleRateHz/2
	halfWidthBins := int(math.Round(12_500.0 / binHzF)) // ±one channel
	freqs := make([]uint32, len(peaks))
	centers := make([]uint32, len(peaks))
	for i, p := range peaks {
		bin := int(math.Round((float64(p.FreqHz) - base) / binHzF))
		centerHz := spectrum.WeightedCentroidHz(smoothed.Bins, bin, halfWidthBins, floor, base, binHzF)
		centers[i] = uint32(math.Round(centerHz))
		freqs[i] = centers[i]
	}

	step, phase := carriers.InferGrid(freqs)
	binHz := uint32(math.Round(binHzF))
	half := cfg.SampleRateHz * (0.5 - guard)

	// Centroid refinement collapses every smoothed sub-maximum of one wide
	// carrier toward its centre, so re-dedup the refined centres: a second
	// detection within one channel width (12.5 kHz) of an already-kept,
	// stronger carrier is the same carrier's shoulder, not a new one.
	const mergeHz = 12_500
	var cands []detectedCarrier
	for i := range peaks {
		snapped := carriers.SnapHz(centers[i], step, phase, binHz)
		offsetHz := float64(snapped) - float64(cfg.CenterHz)
		if math.Abs(offsetHz) > half {
			continue // outside the usable (guard-banded) IQ window
		}
		dup := false
		for _, k := range cands {
			if absU32(snapped, k.freqHz) < mergeHz {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		cands = append(cands, detectedCarrier{offsetHz: offsetHz, freqHz: snapped, snr: peaks[i].SNRDb})
		if cfg.MaxCarriers > 0 && len(cands) >= cfg.MaxCarriers {
			break
		}
	}
	return frame, cands
}

// newBank builds the per-carrier channelizer, mirroring widebandt2.pickStrategy
// (DDC ≤ 6 taps, polyphase above; explicit override honored).
func newBank(strategy string, nTaps int, inRateHz, outRateHz, guardFrac float64) (tuner.Bank, string) {
	usePoly := false
	switch strategy {
	case "polyphase":
		usePoly = true
	case "ddc":
		usePoly = false
	default: // "" | "auto"
		usePoly = nTaps > wbStrategyAutoThreshold
	}
	if usePoly {
		return tuner.NewChannelizerBank(inRateHz, outRateHz, guardFrac,
			wbChannelizerBins, wbChannelizerTaps, wbChannelizerKaiserBeta), "polyphase"
	}
	return tuner.NewDDCBank(inRateHz, outRateHz, guardFrac), "ddc"
}

// carrierIdentity pulls (colorCode, systemID) from a DMR run Result: the lock
// payload (tier3 Aloha lock carries both) first, then any grant's channel-id
// (color code) as a fallback for voice carriers.
func carrierIdentity(res *Result) (colorCode uint8, systemID uint32) {
	if res.Lock != nil {
		if v, ok := res.Lock.Fields["ColorCode"]; ok {
			colorCode = uint8(toUint32(v))
		}
		if v, ok := res.Lock.Fields["SystemID"]; ok {
			systemID = toUint32(v)
		}
	}
	if colorCode == 0 && len(res.Grants) > 0 {
		colorCode = res.Grants[0].ChannelID
	}
	return colorCode, systemID
}

// winnerScore returns the winning candidate's score record.
func winnerScore(idr *IdentifyResult) *CandidateScore {
	for i := range idr.Candidates {
		if idr.Candidates[i].Protocol == idr.Winner {
			return &idr.Candidates[i]
		}
	}
	return nil
}

// baseName returns the path's final element without importing path/filepath in
// the hot survey loop (labels only).
func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}
