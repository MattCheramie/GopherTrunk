// Package toneout detects fire/EMS paging tones — Two-Tone Sequential
// (Motorola Quick Call II), single-tone, and DTMF — over the PCM stream
// produced by the voice composer, and emits events.KindToneAlert when a
// configured profile matches.
//
// Files:
//
//	goertzel.go   single-frequency power detector (the Goertzel
//	              algorithm — much cheaper than a full FFT for the
//	              small handful of tones each profile cares about)
//	profile.go    Profile + Tone configuration types
//	detector.go   per-device state machine; satisfies composer.PCMSink
//	              so the daemon can fan PCM into it alongside the
//	              recorder
package toneout

import "math"

// Goertzel computes the magnitude of a single frequency bin over a
// fixed-size block of real PCM samples. It is materially cheaper than
// running a full FFT when each profile only cares about one or two
// target frequencies.
//
// Usage:
//
//	g := NewGoertzel(1000.0, 8000.0, 800)
//	for each sample s in the block:
//	    if mag, ready := g.Process(s); ready {
//	        // mag is the squared magnitude of the 1 kHz bin over the
//	        // last 800 samples. Block-aligned; not a sliding window.
//	    }
type Goertzel struct {
	coeff     float64
	s1, s2    float64
	count     int
	blockSize int
	normalize float64
}

// NewGoertzel constructs a detector for targetHz over a sampleHz PCM
// stream, accumulating blockSize samples per output magnitude. The
// frequency bin is rounded to the nearest exact bin so the algorithm
// stays numerically stable; the effective resolution is sampleHz /
// blockSize Hz (e.g. 8000/800 = 10 Hz at the default settings).
func NewGoertzel(targetHz, sampleHz float64, blockSize int) *Goertzel {
	if blockSize <= 0 {
		blockSize = 1
	}
	k := math.Round(float64(blockSize) * targetHz / sampleHz)
	omega := 2 * math.Pi * k / float64(blockSize)
	return &Goertzel{
		coeff:     2 * math.Cos(omega),
		blockSize: blockSize,
		// Normalize by block size so magnitudes are comparable across
		// different block sizes and so the threshold has a stable scale.
		normalize: 1.0 / float64(blockSize) / float64(blockSize),
	}
}

// Reset clears internal state. The detector is also reset
// automatically each time a block completes.
func (g *Goertzel) Reset() {
	g.s1 = 0
	g.s2 = 0
	g.count = 0
}

// Process feeds one int16 PCM sample. When the configured block size
// has accumulated, returns (squaredMagnitude, true) and resets state.
// Between block boundaries returns (0, false). Magnitudes are
// normalized into the approximate range [0, 1] for a unit-amplitude
// sine at the target frequency.
func (g *Goertzel) Process(sample int16) (float64, bool) {
	x := float64(sample) / 32768.0
	s0 := x + g.coeff*g.s1 - g.s2
	g.s2 = g.s1
	g.s1 = s0
	g.count++
	if g.count < g.blockSize {
		return 0, false
	}
	mag2 := g.s1*g.s1 + g.s2*g.s2 - g.coeff*g.s1*g.s2
	g.Reset()
	return mag2 * g.normalize * 4, true // factor of 4 ≈ unit-amplitude sine → 1
}

// BlockSize returns the configured block size.
func (g *Goertzel) BlockSize() int { return g.blockSize }
