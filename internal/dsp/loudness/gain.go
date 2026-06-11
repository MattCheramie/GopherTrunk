package loudness

import "math"

// GainToTarget returns the linear gain that moves a signal measured at
// integratedLUFS to targetLUFS. Loudness normalization is a single linear
// gain (no compression), so the gain is exact in LUFS terms before any
// true-peak limiting.
func GainToTarget(integratedLUFS, targetLUFS float64) float64 {
	return math.Pow(10, (targetLUFS-integratedLUFS)/20)
}

// DBToLinear converts a level in decibels to a linear amplitude ratio.
func DBToLinear(db float64) float64 {
	return math.Pow(10, db/20)
}

// Default normalization parameters, matching the EBU R128 / rdio-scanner
// convention.
const (
	DefaultTargetLUFS   = -16.0
	DefaultTruePeakDBTP = -1.5
	DefaultMaxBoostDB   = 12.0
)

// NormalizeParams configures a single linear loudness-normalization pass:
// a perceptual loudness target, a true-peak ceiling, and a cap on the
// applied gain so near-silent audio doesn't amplify hiss to the target.
type NormalizeParams struct {
	TargetLUFS   float64
	TruePeakDBTP float64
	MaxBoostDB   float64
}

// WithDefaults returns a copy of p with zero fields replaced by the
// package defaults.
func (p NormalizeParams) WithDefaults() NormalizeParams {
	if p.TargetLUFS == 0 {
		p.TargetLUFS = DefaultTargetLUFS
	}
	if p.TruePeakDBTP == 0 {
		p.TruePeakDBTP = DefaultTruePeakDBTP
	}
	if p.MaxBoostDB == 0 {
		p.MaxBoostDB = DefaultMaxBoostDB
	}
	return p
}

// NormalizeGain measures mono samples (float64 in [-1, 1] at sampleRate)
// and returns the single linear gain that normalizes them toward
// p.TargetLUFS, clamped to ±MaxBoostDB and then reduced if necessary so
// the true peak stays under p.TruePeakDBTP. This mirrors ffmpeg's two-pass
// linear loudnorm: pure gain, no compression.
//
// ok is false when the audio is too short or too quiet to measure
// (IntegratedLUFS reports !ok); the returned gain is then 1.0 and the
// caller should leave the audio unchanged.
func NormalizeGain(samples []float64, sampleRate int, p NormalizeParams) (gain float64, ok bool) {
	lufs, ok := IntegratedLUFS(samples, sampleRate)
	if !ok {
		return 1.0, false
	}
	gainDB := p.TargetLUFS - lufs
	if gainDB > p.MaxBoostDB {
		gainDB = p.MaxBoostDB
	} else if gainDB < -p.MaxBoostDB {
		gainDB = -p.MaxBoostDB
	}
	gain = DBToLinear(gainDB)

	ceiling := DBToLinear(p.TruePeakDBTP)
	if peak := TruePeak(samples, sampleRate); peak > 0 && peak*gain > ceiling {
		gain = ceiling / peak
	}
	return gain, true
}
