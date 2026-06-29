package siglab

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// gen4FSK builds a clean 4-FSK carrier with no valid protocol structure, so no
// decoder locks — exercising the inconclusive → reference-DB fallback path.
func gen4FSK(rateHz, baudHz, devHz float64, nSym int) []complex64 {
	sps := int(math.Round(rateHz / baudHz))
	lv := []float64{-1, -1.0 / 3, 1.0 / 3, 1}
	out := make([]complex64, 0, sps*nSym)
	phase, seed := 0.0, 987654321
	for s := 0; s < nSym; s++ {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		dphi := 2 * math.Pi * (lv[(seed>>16)%4] * devHz) / rateHz
		for k := 0; k < sps; k++ {
			phase += dphi
			out = append(out, complex64(complex(math.Cos(phase), math.Sin(phase))))
		}
	}
	return out
}

// TestIdentifyReferenceFallback feeds a structureless 4-FSK carrier that no
// decoder can lock and asserts the offline reference DB still names it from its
// blind symbol-rate / modulation estimate.
func TestIdentifyReferenceFallback(t *testing.T) {
	iq := gen4FSK(48000, 4800, 1800, 8000)
	res, err := IdentifyIQ(iq, "synthetic-4fsk", IdentifyConfig{
		SampleRateHz: 48000,
		Format:       FormatF32,
		// Restrict to a couple of decoders that cannot lock structureless
		// noise, guaranteeing the inconclusive fallback path.
		Candidates: []trunking.Protocol{trunking.ProtocolP25, trunking.ProtocolDMR},
	})
	if err != nil {
		t.Fatalf("IdentifyIQ: %v", err)
	}
	if !res.Inconclusive {
		t.Fatalf("expected inconclusive identification, got winner=%q conf=%.2f", res.Winner, res.Confidence)
	}
	if res.Blind == nil {
		t.Fatal("no blind estimate attached")
	}
	if rel := math.Abs(res.Blind.SymbolRateHz-4800) / 4800; rel > 0.05 {
		t.Errorf("blind symbol rate = %.0f Hz, want ≈ 4800 Hz", res.Blind.SymbolRateHz)
	}
	if len(res.ReferenceMatches) == 0 {
		t.Fatal("no reference matches for an undecodable carrier")
	}
	// The top match should be a 4800-baud 4FSK family protocol.
	top := res.ReferenceMatches[0].Entry
	if math.Abs(top.SymbolRateHz-4800) > 100 {
		t.Errorf("top reference match = %s (%.0f sym/s), want a ~4800-baud signal; why=%q",
			top.DisplayName, top.SymbolRateHz, res.ReferenceMatches[0].Why)
	}
	t.Logf("named undecodable carrier as %q (%.2f): %s", top.DisplayName, res.ReferenceMatches[0].Score, res.ReferenceMatches[0].Why)
}
