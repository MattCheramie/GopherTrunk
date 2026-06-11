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
