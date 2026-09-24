package conventional

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
)

// loadRealAirAt2400k reads a 48 kHz cs16 slice of the #1184 reporter's
// on-air capture and interpolates it back to 2.4 MS/s — the RTL-SDR rate
// the conventional scanner actually feeds its tone detector — so the test
// exercises the production rate with real transmitter modulation (voice +
// CTCSS + receiver noise), not a clean synthetic tone.
func loadRealAirAt2400k(t *testing.T, name string) []complex64 {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	n := len(b) / 4
	iq := make([]complex64, n)
	for i := range iq {
		iq[i] = complex(
			float32(int16(binary.LittleEndian.Uint16(b[4*i:])))/32768,
			float32(int16(binary.LittleEndian.Uint16(b[4*i+2:])))/32768)
	}
	return dsp.NewResampler(50, 1, 16, 8.6).Process(nil, iq)
}

// ctcssPresentFraction runs the production detector over x in RTL-sized
// chunks, as the scanner's dwell loop does, skipping the first 0.25 s
// (the first Goertzel block is still filling).
func ctcssPresentFraction(x []complex64, targetHz float64) float64 {
	const rate, chunk = 2_400_000, 16384
	d := NewCTCSSDetector(CTCSSConfig{SampleHz: rate, TargetHz: targetHz})
	var n, present int
	for s := 0; s+chunk <= len(x); s += chunk {
		got := d.Process(x[s : s+chunk])
		if s < rate/4 {
			continue
		}
		n++
		if got {
			present++
		}
	}
	return float64(present) / float64(n)
}

// TestCTCSSDetectsRealAirToneAtSDRRate is the #1184 regression: on the
// reporter's 447.100 MHz Radtel transmission with a 100 Hz CTCSS tone, the
// tone gate never opened ("tones were not being detected"). The magnitude
// threshold was calibrated in radians-per-sample at 48 kHz, but the scanner
// runs the detector at the SDR's 2.4 MS/s, where the same deviation reads
// 50x smaller — a real tone sat ~3000x below the threshold. The detector now
// decimates to ~48 kHz and channel-filters before the discriminator.
// Old code: 0% present.
func TestCTCSSDetectsRealAirToneAtSDRRate(t *testing.T) {
	x := loadRealAirAt2400k(t, "ctcss100_radtel_447100k_48k.cs16")
	if f := ctcssPresentFraction(x, 100); f < 0.95 {
		t.Errorf("100 Hz CTCSS present in %.0f%% of chunks on a transmission carrying it, want >= 95%%", f*100)
	}
	// Adjacent EIA codes on the same real signal must stay closed.
	for _, other := range []float64{94.8, 103.5, 107.2} {
		if f := ctcssPresentFraction(x, other); f > 0 {
			t.Errorf("%.1f Hz gate opened in %.0f%% of chunks on a 100 Hz transmission", other, f*100)
		}
	}
}

// The same radio keyed WITHOUT a tone must never open a 100 Hz gate.
func TestCTCSSRejectsRealAirToneFreeCarrierAtSDRRate(t *testing.T) {
	x := loadRealAirAt2400k(t, "ctcss_none_radtel_447100k_48k.cs16")
	if f := ctcssPresentFraction(x, 100); f > 0 {
		t.Errorf("100 Hz gate opened in %.0f%% of chunks on a transmission with no CTCSS", f*100)
	}
}

// TestCTCSSRealAirOnlyTheSentToneOpens sweeps the whole EIA table over
// both real-air captures: on the 100 Hz transmission only the 100 Hz gate
// may open, and on the tone-free one none may. This is the no-harm pin
// for the lower NFM threshold (#1184).
func TestCTCSSRealAirOnlyTheSentToneOpens(t *testing.T) {
	if testing.Short() {
		t.Skip("sweeps 51 tones over two captures")
	}
	none := loadRealAirAt2400k(t, "ctcss_none_radtel_447100k_48k.cs16")
	tone := loadRealAirAt2400k(t, "ctcss100_radtel_447100k_48k.cs16")
	for _, hz := range EIACTCSSTones {
		if f := ctcssPresentFraction(none, hz); f > 0 {
			t.Errorf("%.1f Hz gate opened in %.0f%% of chunks on a tone-free transmission", hz, f*100)
		}
		if hz == 100 {
			continue
		}
		if f := ctcssPresentFraction(tone, hz); f > 0 {
			t.Errorf("%.1f Hz gate opened in %.0f%% of chunks on a 100 Hz transmission", hz, f*100)
		}
	}
}
