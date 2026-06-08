package hunt

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/survey"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fmNoise builds a constant-envelope analog-FM carrier modulated by deterministic
// voice-like (low-pass noise) audio — a stand-in for a conventional FM channel.
func fmNoise(n int, rateHz, deviationHz float64, seed int64) []complex64 {
	r := rand.New(rand.NewSource(seed))
	raw := make([]float64, n)
	for i := range raw {
		raw[i] = r.NormFloat64()
	}
	const win = 12
	msg := make([]float64, n)
	var acc float64
	for i := 0; i < n; i++ {
		acc += raw[i]
		if i >= win {
			acc -= raw[i-win]
		}
		msg[i] = acc / win
	}
	out := make([]complex64, n)
	var phase float64
	k := 2 * math.Pi * deviationHz / rateHz
	for i, m := range msg {
		phase += k * m
		out[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	return out
}

// TestRunLiveSurvey_MixedTraffic probes two carriers — a synthesized P25 control
// channel and an analog FM voice channel — and asserts the survey classifies and
// routes each correctly: the P25 carrier folds into the discovered system, the
// analog carrier is reported as active NBFM.
func TestRunLiveSurvey_MixedTraffic(t *testing.T) {
	p25, meta, err := siglab.Synthesize(siglab.SynthOptions{
		Protocol: trunking.ProtocolP25,
		Format:   siglab.FormatF32,
	})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	rate := uint32(meta.SampleRateHz)

	const p25Hz = 851_000_000
	const fmHz = 154_000_000
	analog := fmNoise(len(p25), float64(rate), 3000, 7)

	src := NewMappedIQSource(map[uint32][]complex64{
		p25Hz: p25,
		fmHz:  analog,
	}, rate)

	sv, reports, err := RunLiveSurvey(context.Background(), LiveHuntOptions{
		Source:        src,
		Candidates:    []uint32{p25Hz, fmHz},
		DwellSeconds:  float64(len(p25)) / float64(rate),
		MinConfidence: 0.3,
		Name:          "Survey Test",
	})
	if err != nil {
		t.Fatalf("RunLiveSurvey: %v", err)
	}
	if len(sv.Signals) != 2 {
		t.Fatalf("len(Signals) = %d, want 2", len(sv.Signals))
	}

	byFreq := map[uint32]DetectedSignal{}
	for _, s := range sv.Signals {
		byFreq[s.FreqHz] = s
	}

	// P25 carrier → trunk control, folded into the system.
	p25Sig := byFreq[p25Hz]
	if p25Sig.Class != survey.ClassTrunkControl {
		t.Errorf("p25 carrier class = %q, want trunk-control\n  features: %+v", p25Sig.Class, p25Sig.Features)
	}
	if p25Sig.Trunking == nil || !p25Sig.Trunking.Locked {
		t.Errorf("p25 carrier: expected a locked trunking ref, got %+v", p25Sig.Trunking)
	}
	if sv.System == nil || len(sv.System.Sites) == 0 {
		t.Fatalf("expected a discovered system with at least one site, got %+v", sv.System)
	}

	// Analog carrier → active NBFM.
	fmSig := byFreq[fmHz]
	if fmSig.Class != survey.ClassNBFM {
		t.Errorf("fm carrier class = %q, want nbfm\n  features: %+v", fmSig.Class, fmSig.Features)
	}
	if fmSig.Analog == nil || !fmSig.Analog.Active {
		t.Errorf("fm carrier: expected an active analog report, got %+v", fmSig.Analog)
	}

	// The trunking branch reports through the shared export tail.
	if len(reports) == 0 {
		t.Errorf("expected at least one trunking CaptureReport")
	}
}
