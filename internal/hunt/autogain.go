package hunt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// GainSettable is the optional gain-control capability the auto-gain sweep needs
// beyond IQSource. The standalone live-device source implements it; the daemon's
// shared-SDR broker source does not (changing gain would disrupt the daemon's
// other consumers), so auto-gain is a standalone-live-SDR feature.
type GainSettable interface {
	// SetGain sets the front-end gain in tenths of a dB (-1 ⇒ AGC).
	SetGain(tenthDB int) error
}

// defaultGainSet is a conservative spread (tenths of dB) used when the caller
// doesn't supply one. RTL-SDR and Airspy gain ladders differ, so an operator
// should pass a device-appropriate set; this spread is a reasonable blind probe.
var defaultGainSet = []int{0, 87, 166, 240, 297, 370, 440, 496}

// AutoGainOptions configures an auto-gain sweep.
type AutoGainOptions struct {
	Gains        []int             // tenths of dB to try; empty ⇒ defaultGainSet
	DwellSeconds float64           // capture per (freq, gain); 0 ⇒ 1.0
	Protocol     trunking.Protocol // force the decoder (skip identify) when known
	SampleRateHz float64
	AutoTune     bool
	Log          *slog.Logger
}

// GainScore is the decode outcome at one (frequency, gain).
type GainScore struct {
	GainTenthDB int     `json:"gain_tenth_db"`
	ErrorRate   float64 `json:"error_rate"`
	Locked      bool    `json:"locked"`
	Err         string  `json:"error,omitempty"`
}

// GainRecommendation is the best gain found for one control channel.
type GainRecommendation struct {
	FreqHz          uint32      `json:"freq_hz"`
	BestGainTenthDB int         `json:"best_gain_tenth_db"`
	BestErrorRate   float64     `json:"best_error_rate"`
	Locked          bool        `json:"locked"`
	PerGain         []GainScore `json:"per_gain"`
}

// AutoGainSweep sweeps each control channel across a set of gains and recommends
// the gain that minimises the decode error rate (preferring a gain that locks).
// It requires a gain-controllable live SDR and mutates the device gain as it
// runs — the caller restores the desired gain afterwards. Recommendations are
// advisory; nothing is applied automatically.
func AutoGainSweep(ctx context.Context, src IQSource, gc GainSettable, ccs []uint32, opts AutoGainOptions) ([]GainRecommendation, error) {
	if src == nil || gc == nil {
		return nil, fmt.Errorf("hunt: auto-gain requires a gain-controllable live SDR")
	}
	if opts.SampleRateHz <= 0 {
		return nil, fmt.Errorf("hunt: auto-gain needs a positive sample rate")
	}
	gains := opts.Gains
	if len(gains) == 0 {
		gains = defaultGainSet
	}
	dwell := opts.DwellSeconds
	if dwell <= 0 {
		dwell = 1.0
	}
	n := int(dwell * opts.SampleRateHz)

	recs := make([]GainRecommendation, 0, len(ccs))
	for _, freq := range ccs {
		if err := ctx.Err(); err != nil {
			return recs, err
		}
		rec := GainRecommendation{FreqHz: freq, BestGainTenthDB: -1, BestErrorRate: math.Inf(1)}
		for _, g := range gains {
			if err := ctx.Err(); err != nil {
				return recs, err
			}
			rec.PerGain = append(rec.PerGain, scoreGain(ctx, src, gc, freq, g, n, opts))
		}
		if i := pickGain(rec.PerGain); i >= 0 {
			rec.BestGainTenthDB = rec.PerGain[i].GainTenthDB
			rec.BestErrorRate = rec.PerGain[i].ErrorRate
			rec.Locked = rec.PerGain[i].Locked
		}
		recs = append(recs, rec)
	}
	return recs, nil
}

// scoreGain sets the gain, tunes the frequency (which drains the settle
// transient, including the gain change), captures a short dwell, and decodes it,
// returning the protocol-neutral decode error rate and whether it locked.
func scoreGain(ctx context.Context, src IQSource, gc GainSettable, freq uint32, gain, n int, opts AutoGainOptions) GainScore {
	sc := GainScore{GainTenthDB: gain}
	if err := gc.SetGain(gain); err != nil {
		sc.Err = fmt.Sprintf("set gain: %v", err)
		return sc
	}
	if err := src.Tune(freq); err != nil {
		sc.Err = fmt.Sprintf("tune: %v", err)
		return sc
	}
	iq, err := captureN(ctx, src, n)
	if err != nil {
		sc.Err = fmt.Sprintf("capture: %v", err)
		return sc
	}
	buf := siglab.EncodeCapture(iq, siglab.FormatF32)
	rep := decodeAndAccumulate(&DiscoveredSystem{}, bytes.NewReader(buf),
		fmt.Sprintf("%.4f MHz @ gain %.1f dB", float64(freq)/1e6, float64(gain)/10),
		decodeParams{
			Protocol:     opts.Protocol,
			Format:       siglab.FormatF32,
			SampleRateHz: opts.SampleRateHz,
			FrequencyHz:  freq,
			AutoTune:     opts.AutoTune,
			Log:          opts.Log,
		})
	sc.ErrorRate = rep.ErrorRate
	sc.Locked = rep.Locked
	if rep.Error != "" {
		sc.Err = rep.Error
	}
	return sc
}

// pickGain returns the index of the best score: prefer one that locked, then the
// lowest error rate, then the lower gain (less front-end strain). Scores whose
// capture/decode errored are ignored. Returns -1 when none are usable.
func pickGain(scores []GainScore) int {
	best := -1
	for i := range scores {
		s := scores[i]
		if s.Err != "" {
			continue
		}
		if best < 0 {
			best = i
			continue
		}
		b := scores[best]
		switch {
		case s.Locked != b.Locked:
			if s.Locked {
				best = i
			}
		case s.ErrorRate != b.ErrorRate:
			if s.ErrorRate < b.ErrorRate {
				best = i
			}
		case s.GainTenthDB < b.GainTenthDB:
			best = i
		}
	}
	return best
}
