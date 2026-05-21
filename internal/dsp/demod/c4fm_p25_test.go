package demod

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
)

func p25Conv(a, b []float32) []float32 {
	out := make([]float32, len(a)+len(b)-1)
	for i, av := range a {
		for j, bv := range b {
			out[i+j] += av * bv
		}
	}
	return out
}

func p25Argmax(s []float32) int {
	bi, bv := 0, float32(math.Inf(-1))
	for i, v := range s {
		if v > bv {
			bi, bv = i, v
		}
	}
	return bi
}

// worstISI returns the largest |tap| at the ±k·sps offsets from the
// peak — the inter-symbol interference a filter cascade leaves at the
// neighbouring symbol instants — as a fraction of the peak tap.
func worstISI(casc []float32, sps int) float64 {
	pk := p25Argmax(casc)
	var isi float64
	for k := 1; k <= 5; k++ {
		for _, idx := range []int{pk - k*sps, pk + k*sps} {
			if idx >= 0 && idx < len(casc) {
				if a := math.Abs(float64(casc[idx])); a > isi {
					isi = a
				}
			}
		}
	}
	return isi / math.Abs(float64(casc[pk]))
}

// TestP25C4FMCascadeISIFree is the regression guard for the issue #275
// C4FM filter fix. P25 Phase 1 C4FM is not a root-raised-cosine
// matched-pair system: the transmitter shapes with a raised-cosine
// cascaded with an inverse-sinc compensation, and the spec receive
// filter is a sinc that cancels it, leaving the transmit×receive
// cascade a plain raised-cosine — ISI-free at the symbol instants.
//
// The receiver previously used an RRC matched filter. Against a real
// (spec-shaped) P25 transmit signal an RRC leaves substantial residual
// ISI — the systematically-corrupted dibits (`errs=11`) that #275's
// C4FM path hit on-air. This test asserts the spec pair is ISI-free and
// that the old RRC pairing is not.
func TestP25C4FMCascadeISIFree(t *testing.T) {
	const sps = 10
	tx := P25C4FMTxTaps(48_000)
	rx := P25C4FMRxTaps(48_000)

	specISI := worstISI(p25Conv(tx, rx), sps)
	t.Logf("spec TX × spec RX: worst ISI = %.4f%% of peak", specISI*100)
	if specISI > 0.01 {
		t.Errorf("spec C4FM transmit×receive ISI = %.3f%% of peak, want <1%% — "+
			"the spec filter pair should be ISI-free at the symbol instants", specISI*100)
	}

	// The previous receiver matched filter — a root-raised-cosine — is
	// the wrong filter for a spec-shaped transmit signal.
	rrcISI := worstISI(p25Conv(tx, filter.RootRaisedCosine(sps, 8, 0.2)), sps)
	t.Logf("spec TX × RRC receive: worst ISI = %.4f%% of peak", rrcISI*100)
	if rrcISI < 0.05 {
		t.Errorf("spec-TX × RRC-receive ISI = %.3f%% of peak — expected substantial "+
			"residual ISI (the #275 root cause); the test premise may be wrong", rrcISI*100)
	}
}
