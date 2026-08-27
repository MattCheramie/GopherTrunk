package receiver

import (
	"math"
	"testing"
)

// dibitToSymbol is the inverse of SymbolToDibit — used by the test
// modulator to turn a dibit stream back into C4FM symbol deviations.
func dibitToSymbol(d uint8) int {
	switch d {
	case 0:
		return +1
	case 1:
		return +3
	case 2:
		return -1
	case 3:
		return -3
	}
	return 0
}

// makeC4FMIQWithOffset modulates a dibit stream to 48 kHz / 10 sps C4FM IQ
// (unfiltered NRFM, same shape as makePhaseRampIQ but data-driven) at the
// NXDN spec 1800 Hz deviation, then rotates the whole buffer by a constant
// carrier offset in Hz — the exact impairment an uncorrected tuner ppm
// error imposes. The DMR receiver's issue #836 fixture, ported.
func makeC4FMIQWithOffset(dibits []uint8, offsetHz float64) []complex64 {
	const sampleRate = 48_000.0
	const sps = 10
	const deviation = 1800.0
	iq := make([]complex64, len(dibits)*sps)
	phase := 0.0
	for i, d := range dibits {
		dphi := 2 * math.Pi * float64(dibitToSymbol(d)) * deviation / 3.0 / sampleRate
		base := i * sps
		for k := 0; k < sps; k++ {
			iq[base+k] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
			phase += dphi
		}
	}
	if offsetHz != 0 {
		for n := range iq {
			th := 2 * math.Pi * offsetHz * float64(n) / sampleRate
			iq[n] *= complex(float32(math.Cos(th)), float32(math.Sin(th)))
		}
	}
	return iq
}

// balancedStream builds a deterministic, zero-mean dibit stream long enough
// for the CoarseAFC to converge and be measured over a steady-state tail.
func balancedStream(n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8((i%4 + i/7) % 4)
	}
	return out
}

// decodeDibits runs one IQ buffer through a DeviationHz-calibrated NXDN
// receiver (nxdn_afc on — the configuration this file exercises) and
// returns the recovered dibit stream.
func decodeDibits(iq []complex64) []uint8 {
	var got []uint8
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		EnableAFC:    true,
		DibitSink:    func(dibits []uint8, baseIdx int) { got = append(got, dibits...) },
	})
	r.Process(iq)
	return got
}

// TestReceiverIsCarrierOffsetInvariant is the NXDN port of the DMR issue
// #836 regression: a narrowband C4FM signal carrying an uncorrected tuner
// carrier offset far beyond the slicer's tolerance must decode to the SAME
// dibit stream as the zero-offset signal, because the receiver's CoarseAFC
// removes the offset's constant DC bias from the symbols before the slicer
// sees them. NXDN was the one 4800-baud C4FM receiver with no carrier
// correction at all — failing-first, this test collapses well below the
// threshold without the AFC stage (the DMR port measured ~0.99 → ~0.79).
//
// Unlike DMR, the stage is OPT-IN (nxdn_afc): NXDN's unwhitened CAC
// produces long constant-dibit runs the plain coarse tracker drifts onto
// (issue #402), collapsing CAC CRC yield on a clean signal — measured on
// the siglab SITE_INFO round-trip fixture, which pins the default-off
// path. This test pins the flag-on path.
func TestReceiverIsCarrierOffsetInvariant(t *testing.T) {
	const offsetHz = 400.0
	stream := balancedStream(600)

	ref := decodeDibits(makeC4FMIQWithOffset(stream, 0))
	off := decodeDibits(makeC4FMIQWithOffset(stream, offsetHz))

	n := len(ref)
	if len(off) < n {
		n = len(off)
	}
	// Skip a warm-up region so the measurement is over the AFC's
	// steady state, not its convergence transient.
	const warmup = 250
	if n <= warmup {
		t.Fatalf("recovered only %d dibits, need > %d", n, warmup)
	}
	agree := 0
	for i := warmup; i < n; i++ {
		if ref[i] == off[i] {
			agree++
		}
	}
	frac := float64(agree) / float64(n-warmup)
	if frac < 0.95 {
		t.Fatalf("offset-vs-baseline dibit agreement = %.3f over %d symbols, want >=0.95 — CoarseAFC should make decode invariant to a %.0f Hz carrier offset", frac, n-warmup, offsetHz)
	}
}
