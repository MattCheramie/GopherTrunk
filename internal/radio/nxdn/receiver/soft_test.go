package receiver

import "testing"

// TestReceiverSoftDibitSinkContract pins the SoftDibitSink contract the CC's
// ProcessSoft relies on: two LLRs per dibit, index-aligned, sign-consistent
// with the hard decisions (bit = 1 ⇔ its LLR < 0 — the framing convention),
// and the hard DibitSink never fires on the soft path.
func TestReceiverSoftDibitSinkContract(t *testing.T) {
	iq := makePhaseRampIQ(600)

	hardFired := false
	var dibits []uint8
	var llrs []float32
	r := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		SoftDecision: true,
		DibitSink:    func(_ []uint8, _ int) { hardFired = true },
		SoftDibitSink: func(d []uint8, soft []float32, _ int) {
			if len(soft) != 2*len(d) {
				t.Fatalf("soft len %d != 2*dibit len %d", len(soft), len(d))
			}
			dibits = append(dibits, d...)
			llrs = append(llrs, soft...)
		},
	})
	r.Process(iq)

	if hardFired {
		t.Error("hard DibitSink fired on the soft path")
	}
	if len(dibits) == 0 {
		t.Fatal("no dibits decoded")
	}
	for i, d := range dibits {
		msb := (d >> 1) & 1
		lsb := d & 1
		// An LLR of exactly 0 is an erasure (a symbol landing exactly on a
		// slicer boundary, e.g. during AGC warm-up) — it carries no sign to
		// check.
		if l := llrs[2*i]; l != 0 {
			if wantNeg := msb == 1; (l < 0) != wantNeg {
				t.Fatalf("dibit %d (=%d): MSB LLR %.4f sign inconsistent with hard decision", i, d, l)
			}
		}
		if l := llrs[2*i+1]; l != 0 {
			if wantNeg := lsb == 1; (l < 0) != wantNeg {
				t.Fatalf("dibit %d (=%d): LSB LLR %.4f sign inconsistent with hard decision", i, d, l)
			}
		}
	}
}

// TestReceiverSoftPathMatchesHardDibits: the soft path's hard decisions must
// be identical to the plain hard path's on the same IQ — SoftDecision adds
// LLRs, it never changes the dibit stream.
func TestReceiverSoftPathMatchesHardDibits(t *testing.T) {
	iq := makePhaseRampIQ(400)

	var hard []uint8
	rh := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		DibitSink:    func(d []uint8, _ int) { hard = append(hard, d...) },
	})
	rh.Process(iq)

	var soft []uint8
	rs := New(Options{
		SampleRateHz: 48_000,
		DeviationHz:  1800.0,
		SoftDecision: true,
		SoftDibitSink: func(d []uint8, _ []float32, _ int) {
			soft = append(soft, d...)
		},
	})
	rs.Process(iq)

	if len(soft) != len(hard) {
		t.Fatalf("soft path dibit count %d != hard path %d", len(soft), len(hard))
	}
	for i := range hard {
		if soft[i] != hard[i] {
			t.Fatalf("dibit %d differs between soft and hard paths", i)
		}
	}
}
