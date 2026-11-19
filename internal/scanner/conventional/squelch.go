// Package conventional is the fixed-frequency analog FM scanner.
//
// State machine cycles through operator-configured channels, measures
// IQ-domain RMS power on each tune-and-dwell, and on squelch break
// hands off to the trunking engine (via Engine.HandleSyntheticCall) so
// the recorder writes a WAV like any other call. After hangtime
// silence the scanner publishes CallEnd through EndSyntheticCall and
// resumes hopping.
//
// The first squelch test is IQ-domain power: measurable before any
// demod chain spins up (cheap blind-channel visits during scanning),
// and FM carriers are constant-envelope so the same metric drives
// hangtime detection while a call is active. It is an absolute level,
// though, so each mode pairs it with a test that does not depend on
// gain: an analog FM channel's noise squelch (fm_noise_squelch.go), an
// AM channel's carrier-to-noise (am_squelch.go), a digital channel's
// call gate (the *_gate.go files, nac.go).
package conventional

import "math"

// PowerDbFS measures the RMS power of a complex64 IQ chunk in dB
// relative to full-scale (where a unity-amplitude tone is 0 dBFS).
// Returns -infinity for an empty buffer; callers compare against
// SquelchDbFS and treat -Inf as well below any threshold.
//
// This is the load-bearing primitive for both squelch-open detection
// (during SCANNING) and hangtime-silence detection (during DWELL).
func PowerDbFS(iq []complex64) float64 {
	if len(iq) == 0 {
		return math.Inf(-1)
	}
	var acc float64
	for _, s := range iq {
		i := float64(real(s))
		q := float64(imag(s))
		acc += i*i + q*q
	}
	mean := acc / float64(len(iq))
	if mean <= 0 {
		return math.Inf(-1)
	}
	// IQ power is already an envelope-squared metric; convert
	// directly to dB. A unit-amplitude tone (|s|=1) yields mean=1
	// → 0 dB.
	return 10 * math.Log10(mean)
}
