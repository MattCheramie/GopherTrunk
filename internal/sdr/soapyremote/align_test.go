package soapyremote

import (
	"math"
	"math/rand"
	"testing"
)

// delayedCopy returns x delayed by an integer+fractional sample count via the
// same linear interpolation model the aligner inverts (adequate: the test
// signals are heavily oversampled, like a 25 kHz channel in a 200 kHz stream).
func delayedCopy(x []complex64, total float64) []complex64 {
	d := int(total)
	f := float32(total - float64(d))
	out := make([]complex64, len(x))
	for i := range out {
		j := i - d
		if j-1 >= 0 && j < len(x) {
			out[i] = x[j]*(1-complex(f, 0)) + x[j-1]*complex(f, 0)
		}
	}
	return out
}

// lowpassSignal is a random signal whose energy sits well inside the band a
// fractional linear interpolator handles transparently — a 4-sample moving
// average over random phases, mimicking an oversampled channel.
func lowpassSignal(rng *rand.Rand, n int, amp float64) []complex64 {
	raw := mrcSignal(rng, n+4, amp)
	out := make([]complex64, n)
	for i := range out {
		var s complex64
		for k := 0; k < 4; k++ {
			s += raw[i+k]
		}
		out[i] = s * 0.25
	}
	return out
}

func corrAtZero(a, b []complex64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var cr, ci, pa, pb float64
	for i := 0; i < n; i++ {
		cr += float64(real(a[i]))*float64(real(b[i])) + float64(imag(a[i]))*float64(imag(b[i]))
		ci += float64(real(a[i]))*float64(imag(b[i])) - float64(imag(a[i]))*float64(real(b[i]))
		pa += float64(real(a[i]))*float64(real(a[i])) + float64(imag(a[i]))*float64(imag(a[i]))
		pb += float64(real(b[i]))*float64(real(b[i])) + float64(imag(b[i]))*float64(imag(b[i]))
	}
	if pa == 0 || pb == 0 {
		return 0
	}
	return math.Hypot(cr, ci) / math.Sqrt(pa*pb)
}

// TestEstimateBranchDelay pins the measurement: sign convention (positive =
// branch 1 lags branch 0), integer + fractional recovery, and the DC-removal
// guard (two branches of independent noise sharing a DC offset must not read
// as coherent — the CrossStats lesson).
func TestEstimateBranchDelay(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	x := lowpassSignal(rng, alignMeasureSamples, 0.4)

	for _, want := range []float64{2.6, 3.0, -1.4} {
		b1 := delayedCopy(x, want)
		if want < 0 {
			b1 = delayedCopy(x, 0)
			x2 := delayedCopy(x, -want) // branch 0 lags instead
			lag, frac, rho := estimateBranchDelay(x2, b1)
			if got := float64(lag) + frac; math.Abs(got-want) > 0.15 || rho < 0.9 {
				t.Errorf("delay %.1f: measured %.2f (rho %.2f)", want, got, rho)
			}
			continue
		}
		lag, frac, rho := estimateBranchDelay(x, b1)
		if got := float64(lag) + frac; math.Abs(got-want) > 0.15 || rho < 0.9 {
			t.Errorf("delay %.1f: measured %.2f (rho %.2f)", want, got, rho)
		}
	}

	// Independent noise + common DC: must not clear the latch gate.
	n0 := mrcNoise(rng, make([]complex64, alignMeasureSamples), 0.1)
	n1 := mrcNoise(rng, make([]complex64, alignMeasureSamples), 0.1)
	for i := range n0 {
		n0[i] += complex(0.3, 0.3)
		n1[i] += complex(0.3, 0.3)
	}
	if _, _, rho := estimateBranchDelay(n0, n1); rho >= alignMinRho {
		t.Errorf("independent noise with common DC measured rho=%.2f — the DC guard failed", rho)
	}
}

// TestMRCCombinerAlignsSkewedBranches is the failing-first regression for the
// 19 Aug field capture: branch 1 is a 2.6-sample-delayed copy of branch 0
// (the measured X310 skew), which a scalar combine turns into a comb filter.
// The combiner must measure the skew, delay the early branch, recalibrate on
// the aligned stream, and emit output that matches the (aligned) signal —
// combining must stop being WORSE than the best branch alone.
func TestMRCCombinerAlignsSkewedBranches(t *testing.T) {
	rng := rand.New(rand.NewSource(12))
	const skew = 2.6
	const datagram = 184

	m := newMRCCombiner(formatCS16, diversityMRCTracking, 0)

	// Enough datagrams to fill the measurement buffer, latch, recalibrate and
	// emit a stretch of aligned output.
	total := alignMeasureSamples + 12*mrcTestWindow
	x := lowpassSignal(rng, total+8, 0.4)
	b1full := delayedCopy(x, skew)

	var outTail, refTail []complex64
	for pos := 0; pos+datagram <= total; pos += datagram {
		ch0 := x[pos : pos+datagram]
		ch1 := b1full[pos : pos+datagram]
		out := m.combine(twoBranchPayload(ch0, ch1), datagram)
		if pos > alignMeasureSamples+8*mrcTestWindow {
			outTail = append(outTail, out...)
			refTail = append(refTail, ch1...) // the DELAYED timeline both branches now share
		}
	}

	if !m.align.measured {
		t.Fatal("combiner never measured the inter-branch delay")
	}
	if m.align.delayIdx != 0 {
		t.Fatalf("delayed branch = %d, want 0 (the early branch)", m.align.delayIdx)
	}
	if got := float64(m.align.delaySamples) + m.align.delayFrac; math.Abs(got-skew) > 0.2 {
		t.Fatalf("measured delay %.2f samples, want ~%.1f", got, skew)
	}
	if !m.cal.Calibrated() {
		t.Fatal("combiner did not recalibrate after alignment")
	}
	// The aligned combine of a delayed-copy pair is the signal itself (up to
	// scale): correlation with the shared timeline must be near-perfect. The
	// UNALIGNED scalar combine of a 2.6-sample skew has a deep in-band comb
	// notch and measures well below this bar — the failing-first assertion.
	if rho := corrAtZero(outTail, refTail); rho < 0.995 {
		t.Fatalf("combined output correlation with the aligned signal = %.3f, want >= 0.995 (comb-filtered?)", rho)
	}
}
