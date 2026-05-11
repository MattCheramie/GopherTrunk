package conventional

import (
	"math"
	"testing"
)

func TestPowerDbFS_EmptyIsNegInf(t *testing.T) {
	got := PowerDbFS(nil)
	if !math.IsInf(got, -1) {
		t.Errorf("PowerDbFS(nil) = %v, want -Inf", got)
	}
}

func TestPowerDbFS_UnitToneIsZeroDb(t *testing.T) {
	// A constant complex tone of magnitude 1 has mean(|s|²) = 1
	// → 0 dBFS. We allow a tiny tolerance for floating-point.
	iq := make([]complex64, 1024)
	for i := range iq {
		iq[i] = complex(1, 0)
	}
	got := PowerDbFS(iq)
	if math.Abs(got) > 0.01 {
		t.Errorf("PowerDbFS(unit) = %v, want ~0", got)
	}
}

func TestPowerDbFS_SilenceIsDeepNegative(t *testing.T) {
	// Near-zero IQ should yield a strongly negative dB value
	// — below any reasonable squelch threshold.
	iq := make([]complex64, 1024)
	for i := range iq {
		iq[i] = complex(1e-5, 1e-5)
	}
	got := PowerDbFS(iq)
	if got > -80 {
		t.Errorf("PowerDbFS(silence) = %v, want <= -80", got)
	}
}

func TestPowerDbFS_OrdersByAmplitude(t *testing.T) {
	// A louder carrier must measure higher dBFS than a quieter one.
	mk := func(amp float32) []complex64 {
		out := make([]complex64, 256)
		for i := range out {
			out[i] = complex(amp, 0)
		}
		return out
	}
	loud := PowerDbFS(mk(0.5))
	quiet := PowerDbFS(mk(0.05))
	if loud <= quiet {
		t.Errorf("loud (%v) should be > quiet (%v)", loud, quiet)
	}
}
