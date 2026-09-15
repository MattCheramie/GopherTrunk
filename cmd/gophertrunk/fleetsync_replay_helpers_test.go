package main

import (
	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// Thin wrappers so the FleetSync harness self-check can build its
// synthetic capture with the SAME modulator/demodulator the DSP unit tests
// use (kept out of the harness file so its env-driven surface stays easy
// to read).
func demodModulateFFSK(bits []byte, rate float64) []complex64 {
	return demod.ModulateFFSK(bits, rate, 1200, 1200, 1800)
}

func demodFM(iq []complex64) []float32 {
	return demod.NewFM().Process(nil, iq)
}

// interpolateIQ rationally resamples iq from inRate to outRate with the
// production polyphase resampler (anti-imaged), so a synthetic narrowband
// burst can be planted inside a wideband capture at its native deviation.
func interpolateIQ(iq []complex64, inRate, outRate float64) []complex64 {
	in, out := int(inRate), int(outRate)
	g := gcdInt(in, out)
	l, m := out/g, in/g
	r := dsp.NewResampler(l, m, 24, 7.0)
	return r.Process(nil, iq)
}

func gcdInt(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
