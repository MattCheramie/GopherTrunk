package receiver

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// simulcastMultipath is a near-spectral-null two-ray channel at 8 sps: main
// path plus a half-symbol (4-sample) echo at ~0.95 amplitude rotated 70° —
// the same shape the Phase 1 CQPSK issue #492 regression uses. Two
// near-equal-power overlapping transmitters (the simulcast case) produce
// exactly this deep fade, and it badly biases the sub-symbol-lag
// autocorrelation the coarse carrier seed fits (measured: ~1 kHz of spurious
// "offset" on a 0 Hz carrier).
func simulcastMultipath() []complex64 {
	return []complex64{
		complex(1, 0),
		0, 0, 0,
		complex(float32(0.95*math.Cos(70*math.Pi/180)), float32(0.95*math.Sin(70*math.Pi/180))),
	}
}

// runSeedGateReceiver feeds iq through a ClockGardner receiver in
// production-sized 192-sample chunks (so the coarse seed accumulates across
// calls) and returns the decoded dibits.
func runSeedGateReceiver(iq []complex64, equalizer bool) []uint8 {
	var rx []uint8
	r := New(Options{
		SampleRateHz: 48_000.0,
		ClockMode:    ClockGardner,
		GardnerGain:  0.005,
		Equalizer:    equalizer,
		DibitSink:    func(d []uint8, _ int) { rx = append(rx, d...) },
	})
	for i := 0; i < len(iq); i += 192 {
		end := i + 192
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}
	return rx
}

// TestReceiverSeedRejectsMultipathBias is the failing-first regression for
// the coarse carrier seed's multipath gate, the issue #492 lesson ported
// from the Phase 1 CQPSK path. A simulcast-like two-ray channel biases
// estimateCarrierSeedHz into a spurious ~1 kHz offset on a ZERO-offset
// carrier; the ungated seed then tunes the NCO ~1 kHz off — beyond the
// Costas loop's ±SymbolRate/8 = ±750 Hz pull-in — so the differential
// decode never recovers (measured tail SER ~0.56 pre-fix). The modulus-CV
// gate detects the ISI the biased de-rotation leaves and keeps the NCO
// identity instead; the Costas loop then locks the true 0 Hz offset and the
// CMA equalizer (the opt-in issue #915 stage, exactly what this channel
// needs) opens the eye (tail SER ~0.01 post-fix).
func TestReceiverSeedRejectsMultipathBias(t *testing.T) {
	const fs = 48_000.0
	tx := pseudoRandomDibits(2400, 0xC0FFEE)
	clean, _ := makeP2HDQPSKIQWithOffset(tx, 0, fs)
	base := demod.ApplyImpairments(clean, fs, demod.Impairments{
		Multipath: simulcastMultipath(),
		Seed:      1,
	})

	rx := runSeedGateReceiver(base, true)
	if ser := bestTailSER(tx, rx, 900); ser > 0.05 {
		t.Errorf("multipath channel, 0 Hz offset: tail SER = %.3f, want <= 0.05 (spurious coarse seed mis-tuned the NCO — issue #492 gate missing)", ser)
	}
}

// TestReceiverSeedStillFiresOnCleanOffset is the no-harm pin for the
// multipath gate: on a clean channel carrying a genuine 1500 Hz offset —
// twice the Costas pull-in, so ONLY the coarse seed can recover it — the
// gate must accept the estimate (a true carrier offset just rotates the
// constant-modulus constellation, leaving the modulus CV low) and decode
// must stay clean. An over-tight gate would reject the seed and break
// exactly the issue #813 case the seed exists for.
func TestReceiverSeedStillFiresOnCleanOffset(t *testing.T) {
	const fs = 48_000.0
	tx := pseudoRandomDibits(2400, 0xC0FFEE)
	iq, _ := makeP2HDQPSKIQWithOffset(tx, 1500, fs)

	rx := runSeedGateReceiver(iq, true)
	if ser := bestTailSER(tx, rx, 900); ser > 0.02 {
		t.Errorf("clean channel, 1500 Hz offset: tail SER = %.3f, want <= 0.02 (multipath gate rejected a genuine carrier seed)", ser)
	}
}
