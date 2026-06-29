package metrics

import (
	"math"
	"math/cmplx"
	"testing"
)

// dqpskPoints builds n π/4-DQPSK points cycling the eight ideal phases, each
// scaled by radiusOf(i) and rotated by extraPhaseRad(i) — the knobs the tests
// use to inject pure magnitude vs pure phase error.
func dqpskPoints(n int, radiusOf func(int) float64, extraPhaseRad func(int) float64) []complex64 {
	pts := make([]complex64, n)
	for i := 0; i < n; i++ {
		ang := float64(i%dqpskPhases)*2*math.Pi/dqpskPhases + extraPhaseRad(i)
		pts[i] = complex64(cmplx.Rect(radiusOf(i), ang))
	}
	return pts
}

func TestVSAMetricsCleanConstellationIsTight(t *testing.T) {
	pts := dqpskPoints(64, func(int) float64 { return 1 }, func(int) float64 { return 0 })
	v := VSAMetricsConstellation(pts)
	if v.RMSEVMPct > 1 || v.PhaseErrDeg > 1 || v.MagErrPct > 1 {
		t.Errorf("clean: rms=%.3f phase=%.3f mag=%.3f, want all ≈0", v.RMSEVMPct, v.PhaseErrDeg, v.MagErrPct)
	}
	if v.PeakEVMPct < v.RMSEVMPct {
		t.Errorf("peak EVM %.3f < rms EVM %.3f", v.PeakEVMPct, v.RMSEVMPct)
	}
	if len(v.PerSymbolEVMPct) != len(pts) || len(v.ErrorVectors) != len(pts) {
		t.Errorf("per-symbol arrays len = %d/%d, want %d", len(v.PerSymbolEVMPct), len(v.ErrorVectors), len(pts))
	}
	// The scalar wrapper must match the decomposed RMS.
	if got := EVMConstellation(pts); math.Abs(got-v.RMSEVMPct) > 1e-9 {
		t.Errorf("EVMConstellation %.6f != VSA RMS %.6f", got, v.RMSEVMPct)
	}
}

func TestVSAMetricsPhaseErrorIsolated(t *testing.T) {
	// Alternating ±5° jitter (zero-mean, so the 8th-power carrier estimate is
	// unaffected): raises phase error, leaves magnitude error ≈0.
	jit := 5 * math.Pi / 180
	pts := dqpskPoints(64, func(int) float64 { return 1 }, func(i int) float64 {
		if i%2 == 0 {
			return jit
		}
		return -jit
	})
	v := VSAMetricsConstellation(pts)
	if v.PhaseErrDeg < 3 {
		t.Errorf("phase error = %.2f°, want ≈5°", v.PhaseErrDeg)
	}
	if v.MagErrPct > 1 {
		t.Errorf("magnitude error = %.2f%%, want ≈0 under pure phase jitter", v.MagErrPct)
	}
	if v.PhaseErrDeg <= v.MagErrPct {
		t.Errorf("phase (%.2f) should dominate magnitude (%.2f) under phase jitter", v.PhaseErrDeg, v.MagErrPct)
	}
}

func TestVSAMetricsMagnitudeErrorIsolated(t *testing.T) {
	// Alternating ±10% amplitude, exact phase: raises magnitude error, leaves
	// phase error ≈0.
	pts := dqpskPoints(64, func(i int) float64 {
		if i%2 == 0 {
			return 1.1
		}
		return 0.9
	}, func(int) float64 { return 0 })
	v := VSAMetricsConstellation(pts)
	if v.MagErrPct < 3 {
		t.Errorf("magnitude error = %.2f%%, want ≈10%%", v.MagErrPct)
	}
	if v.PhaseErrDeg > 1 {
		t.Errorf("phase error = %.2f°, want ≈0 under pure amplitude error", v.PhaseErrDeg)
	}
}

func TestVSAMetricsC4FMTracksEVM(t *testing.T) {
	soft := []float32{3, 1, -1, -3, 3, 1, -1, -3}
	v := VSAMetricsC4FM(soft, 3)
	if v.RMSEVMPct > 1 {
		t.Errorf("clean C4FM rails: rms EVM = %.3f, want ≈0", v.RMSEVMPct)
	}
	if v.PeakEVMPct < v.RMSEVMPct || len(v.PerSymbolEVMPct) != len(soft) {
		t.Errorf("peak %.3f rms %.3f trace %d", v.PeakEVMPct, v.RMSEVMPct, len(v.PerSymbolEVMPct))
	}
	// Phase/quadrature/origin are undefined on the 1-D soft axis.
	if v.PhaseErrDeg != 0 || v.OriginOffsetPct != 0 {
		t.Errorf("C4FM phase=%.3f origin=%.3f, want 0", v.PhaseErrDeg, v.OriginOffsetPct)
	}
}
