package siglab

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDemodMetricsPopulated confirms the demod-quality metrics are surfaced on
// the P25 deep path for both modulations, with the right estimator selected,
// and that injecting noise degrades EVM and lowers the SNR estimate — the
// end-to-end proof that the metrics package is wired into siglab correctly.
func TestDemodMetricsPopulated(t *testing.T) {
	cases := []struct {
		name string
		mod  string
		want string // expected DemodMetrics.Modulation
	}{
		{"c4fm", "c4fm", "c4fm"},
		{"cqpsk", "lsm", "cqpsk"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clean := analyzeDemod(t, tc.mod, 0)
			if clean == nil {
				t.Fatalf("%s: clean run produced no DemodMetrics", tc.name)
			}
			if clean.Modulation != tc.want {
				t.Errorf("Modulation = %q, want %q", clean.Modulation, tc.want)
			}
			if clean.SymbolsAnalyzed == 0 {
				t.Error("SymbolsAnalyzed = 0")
			}
			if clean.EVMPct < 0 || math.IsNaN(clean.EVMPct) {
				t.Errorf("EVM = %v, want finite non-negative", clean.EVMPct)
			}
			if math.IsInf(clean.SNREstimateDB, 0) {
				t.Errorf("SNR estimate is infinite; cap not applied: %v", clean.SNREstimateDB)
			}

			// A noisy run should disperse the constellation more and read a
			// lower SNR than the clean run.
			noisy := analyzeDemod(t, tc.mod, 8)
			if noisy == nil {
				t.Fatalf("%s: noisy run produced no DemodMetrics", tc.name)
			}
			t.Logf("%s: clean EVM=%.1f%% SNR=%.1f dB | noisy(8dB) EVM=%.1f%% SNR=%.1f dB",
				tc.name, clean.EVMPct, clean.SNREstimateDB, noisy.EVMPct, noisy.SNREstimateDB)
			if noisy.EVMPct <= clean.EVMPct {
				t.Errorf("noise did not raise EVM: clean=%.2f%% noisy=%.2f%%", clean.EVMPct, noisy.EVMPct)
			}
			if noisy.SNREstimateDB >= clean.SNREstimateDB {
				t.Errorf("noise did not lower SNR estimate: clean=%.2f noisy=%.2f", clean.SNREstimateDB, noisy.SNREstimateDB)
			}
		})
	}
}

// analyzeDemod synthesizes a P25 signal in the given modulation at the given
// injected SNR (0 = clean) and returns the DemodMetrics from the deep path.
func analyzeDemod(t *testing.T, mod string, snrDB float64) *DemodMetrics {
	t.Helper()
	synth := SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32, Modulation: mod}
	if snrDB > 0 {
		synth.Impairments = demod.Impairments{SNRdB: snrDB, Seed: 1}
	}
	decode := Config{CollectIQDiag: true}
	if mod == "lsm" || mod == "cqpsk" {
		decode.DemodMode = "cqpsk"
	}
	res, _, err := SynthesizeAndAnalyze(synth, decode, nil)
	if err != nil {
		t.Fatalf("SynthesizeAndAnalyze(%s, %g dB): %v", mod, snrDB, err)
	}
	if res.Signal == nil {
		t.Fatalf("%s: no SignalQuality", mod)
	}
	return res.Signal.Demod
}
