package metrics

import (
	"math"
	"testing"
)

func approx(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %.6g, want %.6g (±%.2g)", what, got, want, tol)
	}
}

func TestQFunc(t *testing.T) {
	// Reference values for the standard-normal tail.
	approx(t, QFunc(0), 0.5, 1e-9, "Q(0)")
	approx(t, QFunc(1), 0.158655, 1e-5, "Q(1)")
	approx(t, QFunc(2), 0.0227501, 1e-6, "Q(2)")
	approx(t, QFunc(3), 0.00134990, 1e-7, "Q(3)")
}

func TestBERCoherentQPSK(t *testing.T) {
	// Textbook BPSK/QPSK reference points (BER = Q(√(2·Eb/N0))).
	approx(t, BERCoherentQPSK(0), 0.0786496, 1e-5, "BER @0dB")  // Q(√2)
	approx(t, BERCoherentQPSK(10), 3.872e-6, 1e-7, "BER @10dB") // Q(√20)
	// Monotone decreasing in Eb/N0.
	if BERCoherentQPSK(5) <= BERCoherentQPSK(6) {
		t.Error("BER must decrease with Eb/N0")
	}
}

func TestBER4PAM(t *testing.T) {
	// 4-PAM is worse than QPSK at equal Eb/N0 (more levels, tighter spacing).
	for _, ebn0 := range []float64{4, 8, 12} {
		if BER4PAM(ebn0) <= BERCoherentQPSK(ebn0) {
			t.Errorf("4-PAM BER should exceed QPSK at %g dB: 4PAM=%g QPSK=%g",
				ebn0, BER4PAM(ebn0), BERCoherentQPSK(ebn0))
		}
	}
	// Sanity: a recognizable value. At 8 dB, 0.75·Q(√(0.8·6.31)) = 0.75·Q(2.247).
	approx(t, BER4PAM(8), 0.75*QFunc(math.Sqrt(0.8*dBToLinear(8))), 1e-12, "BER4PAM @8dB form")
}

func TestEsN0EbN0Conversions(t *testing.T) {
	// sps=10 → +10 dB from SNR to Es/N0.
	approx(t, EsN0FromSNR(5, 10), 15, 1e-9, "Es/N0 from SNR sps=10")
	// 2 bits/symbol → −3.01 dB from Es/N0 to Eb/N0.
	approx(t, EbN0FromEsN0(15, 2), 15-10*math.Log10(2), 1e-9, "Eb/N0 from Es/N0")
}

func TestInverseBERForEbN0(t *testing.T) {
	// The Eb/N0 at which coherent QPSK hits 1e-2, recovered by bisection,
	// should reproduce the forward curve at that point.
	const target = 1e-2
	ebn0 := InverseBERForEbN0(BERCoherentQPSK, target, -10, 30)
	approx(t, BERCoherentQPSK(ebn0), target, 1e-4, "inverse BER round-trip")
	// And 4-PAM needs a higher Eb/N0 than QPSK to reach the same BER.
	ebn0PAM := InverseBERForEbN0(BER4PAM, target, -10, 40)
	if ebn0PAM <= ebn0 {
		t.Errorf("4-PAM should need more Eb/N0 for %g: 4PAM=%.2f QPSK=%.2f", target, ebn0PAM, ebn0)
	}
}
