package receiver

import "testing"

// TestReceiverEmitsSoftAndEyeTaps pins the diagnostic taps the symbol scope
// relies on: the soft track is emitted one sample per dibit (index-aligned
// with the DibitSink batch), the oversampled eye is emitted at 10 sps (48 kHz
// / 4800 baud), and the SymbolSink is never fired on the pure-C4FM NXDN path.
// Mirrors the DMR receiver's tap contract so the scope routes both
// identically.
func TestReceiverEmitsSoftAndEyeTaps(t *testing.T) {
	iq := makePhaseRampIQ(600)

	var softN, dibitN, eyeN, eyeSPS int
	symbolFired := false
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		DibitSink:    func(d []uint8, _ int) { dibitN += len(d) },
		SoftSink:     func(s []float32) { softN += len(s) },
		EyeSink:      func(e []float32, sps int) { eyeN += len(e); eyeSPS = sps },
		SymbolSink:   func(_ []complex64) { symbolFired = true },
	})
	r.Process(iq)

	if dibitN == 0 {
		t.Fatal("no dibits decoded")
	}
	if softN != dibitN {
		t.Errorf("soft samples %d != dibit count %d (soft must be index-aligned with dibits)", softN, dibitN)
	}
	if eyeN == 0 {
		t.Error("no eye samples emitted")
	}
	if eyeSPS != 10 {
		t.Errorf("eye sps = %d, want 10 (48 kHz / 4800 baud)", eyeSPS)
	}
	if symbolFired {
		t.Error("SymbolSink fired on the pure-C4FM NXDN path (it should never fire)")
	}
}

// TestReceiverTapsDoNotAlterDecode confirms the taps are opt-in and
// side-effect-free: enabling SoftSink/EyeSink must not change the decoded
// dibit stream versus a receiver with only a DibitSink.
func TestReceiverTapsDoNotAlterDecode(t *testing.T) {
	iq := makePhaseRampIQ(400)

	var plain []uint8
	rp := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		DibitSink:    func(d []uint8, _ int) { plain = append(plain, d...) },
	})
	rp.Process(iq)

	var withTaps []uint8
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		DibitSink:    func(d []uint8, _ int) { withTaps = append(withTaps, d...) },
		SoftSink:     func(_ []float32) {},
		EyeSink:      func(_ []float32, _ int) {},
	})
	r.Process(iq)

	if len(withTaps) != len(plain) {
		t.Fatalf("taps changed dibit count: %d with taps vs %d without", len(withTaps), len(plain))
	}
	for i := range plain {
		if withTaps[i] != plain[i] {
			t.Fatalf("dibit %d differs with taps enabled: %d vs %d", i, withTaps[i], plain[i])
		}
	}
}
