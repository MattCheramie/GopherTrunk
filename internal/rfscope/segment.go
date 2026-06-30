package rfscope

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/carriers"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/spectrum"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
)

// SegmentConfig tunes the segmentation pre-pass. The zero value is valid; every
// field falls back to a sensible default via withDefaults.
type SegmentConfig struct {
	// FFTSize is the wideband FFT used for carrier discovery (power of two).
	FFTSize int
	// ChannelTargetRateHz is the per-channel decimated baseband rate the
	// downconverter targets. ~50 kHz comfortably holds any land-mobile / IoT
	// narrowband channel while keeping per-burst buffers small.
	ChannelTargetRateHz float64
	// PeakThresholdDb / MinSpacingHz tune carrier discovery (carriers.DetectPeaks).
	PeakThresholdDb float32
	MinSpacingHz    uint32
	// MaxChannels caps how many discovered carriers are analyzed (strongest first).
	MaxChannels int
	// EnvelopeRateHz is the per-channel power-envelope frame rate the onset state
	// machine runs at. Higher resolves shorter bursts at more CPU.
	EnvelopeRateHz float64
	// OnThreshDb/OffThreshDb/Debounce/NoiseAlpha are the onset state-machine knobs,
	// lifted from the DMR onset detector.
	OnThreshDb  float64
	OffThreshDb float64
	Debounce    int
	NoiseAlpha  float64
	// MinBurstSec drops bursts shorter than this (debounce noise).
	MinBurstSec float64
	// MaxSamples caps how many wideband samples are read (0 = whole capture). The
	// CLI sets this from a -window/-duration flag.
	MaxSamples int
	// Now stamps the scene; nil ⇒ time.Now.
	Now func() time.Time
}

func (c SegmentConfig) withDefaults() SegmentConfig {
	if c.FFTSize <= 0 || c.FFTSize&(c.FFTSize-1) != 0 {
		c.FFTSize = 4096
	}
	if c.ChannelTargetRateHz <= 0 {
		c.ChannelTargetRateHz = 50_000
	}
	if c.PeakThresholdDb <= 0 {
		c.PeakThresholdDb = 10
	}
	if c.MinSpacingHz == 0 {
		c.MinSpacingHz = 12_500
	}
	if c.MaxChannels <= 0 {
		c.MaxChannels = 64
	}
	if c.EnvelopeRateHz <= 0 {
		c.EnvelopeRateHz = 200
	}
	if c.OnThreshDb <= 0 {
		c.OnThreshDb = 10
	}
	if c.OffThreshDb <= 0 {
		c.OffThreshDb = 6
	}
	if c.Debounce <= 0 {
		c.Debounce = 3
	}
	if c.NoiseAlpha <= 0 {
		c.NoiseAlpha = 0.02
	}
	if c.MinBurstSec <= 0 {
		c.MinBurstSec = 0.005
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

// occupancyFFT is the FFT size used to measure per-burst spectral occupancy on
// the (already narrow) decimated baseband.
const occupancyFFT = 1024

// Segment reads the wideband span from src, discovers carriers, and slices each
// into onset→offset bursts — the protocol-agnostic atoms every analyzer
// consumes. It returns a Scene populated with Bursts and header fields, and an
// Input whose BurstIQ accessor serves each burst's decimated baseband. It is the
// producer pre-pass; rfscope.Run then layers the derived views on top.
func Segment(ctx context.Context, src Source, cfg SegmentConfig) (*Scene, *Input, error) {
	cfg = cfg.withDefaults()
	if src == nil {
		return nil, nil, fmt.Errorf("rfscope: nil source")
	}
	rate := src.SampleRateHz()
	center := src.CenterHz()
	if rate <= 0 {
		return nil, nil, fmt.Errorf("rfscope: source sample rate must be positive")
	}

	raw, err := Drain(ctx, src, 1<<18, cfg.MaxSamples)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) < cfg.FFTSize {
		return nil, nil, fmt.Errorf("rfscope: capture too short (%d samples) for FFT size %d", len(raw), cfg.FFTSize)
	}

	window := float64(len(raw)) / rate
	avg := spectrum.AverageDB(raw, center, rate, cfg.FFTSize)
	noiseFloor := spectrum.Percentile(avg.Bins, 0.25)

	peaks := carriers.DetectPeaks(avg, carriers.PeakOptions{
		ThresholdDb:  cfg.PeakThresholdDb,
		MinSpacingHz: cfg.MinSpacingHz,
	})
	// DetectPeaks drops the DC bin (a front-end artifact for live SDRs), but a
	// real carrier can sit exactly at the tuned centre — common for a
	// single-channel capture. Always consider the band centre as a candidate; a
	// static DC spike produces no burst (the onset floor initializes to it), so
	// this is safe.
	dcBin := len(avg.Bins) / 2
	cands := []carriers.Peak{{FreqHz: center, PowerDbFS: avg.Bins[dcBin], SNRDb: avg.Bins[dcBin] - noiseFloor}}
	for _, p := range peaks {
		if absU32(p.FreqHz, center) >= cfg.MinSpacingHz {
			cands = append(cands, p)
		}
	}
	if len(cands) > cfg.MaxChannels {
		cands = cands[:cfg.MaxChannels]
	}

	now := cfg.Now()
	scene := &Scene{
		Source:         "",
		StartedAt:      now,
		FinishedAt:     now,
		CenterHz:       center,
		SpanHz:         uint32(math.Round(rate)),
		WindowSec:      window,
		NoiseFloorDbFS: noiseFloor,
	}

	store := map[uint64]storedIQ{}
	var nextID uint64

	for _, pk := range cands {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		// Only segment confirmed carriers. DetectPeaks results clear the
		// threshold by construction; this gates the always-added DC candidate so
		// a dead band centre (or noise rising under an adjacent transmission)
		// produces no phantom burst.
		if pk.SNRDb < cfg.PeakThresholdDb {
			continue
		}
		offset := float64(pk.FreqHz) - float64(center)
		ddc := ccdecoder.NewDownconverterWithOffset(rate, cfg.ChannelTargetRateHz, offset)
		base := ddc.Process(nil, raw)
		outRate := ddc.OutRateHz()
		if len(base) == 0 || outRate <= 0 {
			continue
		}

		carrierConfirmed := pk.SNRDb >= cfg.PeakThresholdDb
		spans := detectBursts(base, outRate, cfg, carrierConfirmed)
		for _, sp := range spans {
			slice := base[sp.startSample:sp.endSample]
			cls := survey.Classify(slice, outRate)
			occ := spectrum.ComputeOccupancy(
				spectrum.AverageDB(slice, 0, outRate, occFFTFor(len(slice))).Bins,
				outRate, float64(cfg.MinSpacingHz), float64(cfg.MinSpacingHz), float64(cfg.MinSpacingHz),
			)
			// Prefer survey's occupied bandwidth (an above-noise-floor outward walk)
			// over ComputeOccupancy's 99%-power span, which under-measures FM-family
			// carriers whose strong DC carrier component concentrates the power.
			occBwHz := cls.OccupiedBwHz
			if occBwHz == 0 {
				occBwHz = uint32(math.Round(occ.OccupiedBandwidthHz))
			}

			nextID++
			id := nextID
			b := Burst{
				ID:               id,
				FreqHz:           pk.FreqHz,
				StartSec:         float64(sp.startSample) / outRate,
				EndSec:           float64(sp.endSample) / outRate,
				PeakDbFS:         pk.PowerDbFS,
				SNRDb:            pk.SNRDb,
				OccupiedBwHz:     occBwHz,
				ChannelPowerDBFS: occ.ChannelPowerDBFS,
				ACPRUpperDB:      occ.ACPRUpperDB,
				ACPRLowerDB:      occ.ACPRLowerDB,
				SpectralFlatness: occ.SpectralFlatness,
				Class:            cls.Class,
				Features:         cls.Features,
			}
			scene.Bursts = append(scene.Bursts, b)
			// Copy the slice so a later resize of base never aliases it.
			cp := make([]complex64, len(slice))
			copy(cp, slice)
			store[id] = storedIQ{iq: cp, rateHz: outRate}
		}
	}

	// Deterministic ordering: by start time, then frequency.
	sort.Slice(scene.Bursts, func(i, j int) bool {
		a, b := scene.Bursts[i], scene.Bursts[j]
		if a.StartSec != b.StartSec {
			return a.StartSec < b.StartSec
		}
		return a.FreqHz < b.FreqHz
	})

	in := &Input{
		SampleRateHz: rate,
		Window:       window,
		BurstIQ: func(b Burst) ([]complex64, float64, error) {
			s, ok := store[b.ID]
			if !ok {
				return nil, 0, fmt.Errorf("rfscope: no stored IQ for burst %d", b.ID)
			}
			return s.iq, s.rateHz, nil
		},
	}
	return scene, in, nil
}

type storedIQ struct {
	iq     []complex64
	rateHz float64
}

// burstSpan is one onset→offset run on a channel's decimated baseband, in
// baseband-sample indices.
type burstSpan struct {
	startSample int
	endSample   int
}

// detectBursts slices a channel's decimated baseband into onset→offset spans.
//
// It applies the hysteresis + debounce logic lifted from the DMR onset detector,
// but anchors the threshold on a fixed quiet floor (a low percentile of the
// channel's own power envelope over the whole capture) rather than an EWMA seeded
// from the first frame — so a transmission that is already keyed up at the start
// of the capture is still found. When the channel has too little dynamic range to
// segment (a continuous carrier on for the whole window, or dead air),
// carrierConfirmed (from the wideband carrier SNR) decides between one full-span
// burst and none.
func detectBursts(base []complex64, outRate float64, cfg SegmentConfig, carrierConfirmed bool) []burstSpan {
	hop := int(math.Round(outRate / cfg.EnvelopeRateHz))
	if hop < 1 {
		hop = 1
	}
	minSamples := int(math.Round(cfg.MinBurstSec * outRate))

	// Power envelope, one dB value per frame.
	var env []float64
	for i := 0; i+hop <= len(base); i += hop {
		env = append(env, frameDb(base[i:i+hop]))
	}
	if len(env) == 0 {
		return nil
	}

	floor := percentileFloat(env, 0.10)
	peak := percentileFloat(env, 0.95)
	// Low dynamic range ⇒ the channel is either continuously on or dead. Defer to
	// the wideband carrier confirmation.
	if peak-floor < cfg.OnThreshDb {
		if carrierConfirmed && len(base) >= minSamples {
			return []burstSpan{{startSample: 0, endSample: len(base)}}
		}
		return nil
	}

	onLevel := floor + cfg.OnThreshDb
	offLevel := floor + cfg.OffThreshDb
	var spans []burstSpan
	active := false
	above, below := 0, 0
	curStart := -1
	for i, db := range env {
		switch {
		case db >= onLevel:
			above++
			below = 0
			if !active && above >= cfg.Debounce {
				active = true
				curStart = (i - (cfg.Debounce - 1)) * hop
				if curStart < 0 {
					curStart = 0
				}
			}
		case db <= offLevel:
			below++
			above = 0
			if active && below >= cfg.Debounce {
				active = false
				end := i * hop
				if end-curStart >= minSamples {
					spans = append(spans, burstSpan{startSample: curStart, endSample: end})
				}
				curStart = -1
			}
		default:
			above, below = 0, 0
		}
	}
	if active && curStart >= 0 && len(base)-curStart >= minSamples {
		spans = append(spans, burstSpan{startSample: curStart, endSample: len(base)})
	}
	return spans
}

// frameDb is the mean linear power of a baseband frame, in dB (floored).
func frameDb(frame []complex64) float64 {
	if len(frame) == 0 {
		return -300
	}
	var sum float64
	for _, z := range frame {
		sum += float64(real(z))*float64(real(z)) + float64(imag(z))*float64(imag(z))
	}
	p := sum / float64(len(frame))
	if p < 1e-30 {
		p = 1e-30
	}
	return 10 * math.Log10(p)
}

// percentileFloat returns the p-quantile (0..1) of xs without mutating it.
func percentileFloat(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	idx := int(p * float64(len(cp)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

// absU32 is the absolute difference of two unsigned frequencies.
func absU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// occFFTFor picks a power-of-two FFT size no larger than the slice for occupancy
// measurement, capped at occupancyFFT.
func occFFTFor(n int) int {
	size := occupancyFFT
	for size > n && size > 64 {
		size >>= 1
	}
	return size
}
