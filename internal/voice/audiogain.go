package voice

import "math"

// applyAudioGain scales decoded PCM in place by gainDB (a conventional
// channel's audio_gain_db, carried as Grant.AudioGainDB — issue #1239),
// clamping at full scale so a boost that overshoots saturates instead of
// wrapping. 0 dB is a no-op.
func applyAudioGain(samples []int16, gainDB float64) {
	if gainDB == 0 {
		return
	}
	g := math.Pow(10, gainDB/20)
	for i, s := range samples {
		v := math.Round(float64(s) * g)
		if v > math.MaxInt16 {
			v = math.MaxInt16
		} else if v < math.MinInt16 {
			v = math.MinInt16
		}
		samples[i] = int16(v)
	}
}
