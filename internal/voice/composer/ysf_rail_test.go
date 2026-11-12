package composer

import "testing"

// iqRailMeter reports the share of samples with I or Q at the ADC rail: an
// RTL-SDR u8 rail scales to ±1.0, a cs16 rail to ±0.99997, and a signal
// well inside the range counts nothing.
func TestIQRailMeterCountsRailSamples(t *testing.T) {
	var m iqRailMeter
	iq := make([]complex64, 64*100)
	for i := 0; i < len(iq); i += 64 {
		switch (i / 64) % 4 {
		case 0:
			iq[i] = complex(1, 0.2)
		case 1:
			iq[i] = complex(-0.3, 32767.0/32768)
		case 2:
			iq[i] = complex(0.5, -0.5)
		case 3:
			iq[i] = complex(-0.98, 0.98)
		}
	}
	m.observe(iq[:64*50])
	m.observe(iq[64*50:])
	if got := m.pct(); got != 50 {
		t.Fatalf("rail pct %d, want 50", got)
	}
	if (&iqRailMeter{}).pct() != 0 {
		t.Fatal("empty meter must report 0")
	}
}
