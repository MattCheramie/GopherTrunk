package receiver

import (
	"math/rand"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
)

// TestReceiverMatchedFilterLowSNRBER pins the symbol matched filter's effect:
// at a low SNR the integrate-and-dump over one symbol period must hold the
// recovered bit-error rate well below what the raw per-sample slicer achieves.
// FFSK sends unshaped tone bursts, so the post-discriminator symbol pulse is
// rectangular and a one-symbol boxcar is its matched filter; averaging the
// symbol's energy before slicing is the SNR the pre-matched-filter receiver
// threw away. Empirically (48 kHz, 4 dB AWGN, seed 1) the un-filtered chain
// sits near ~2.5% BER while the matched filter reaches <1%; assert a threshold
// between the two so the test fails on the pre-matched-filter receiver and
// passes with it.
func TestReceiverMatchedFilterLowSNRBER(t *testing.T) {
	rate := 48000.0
	rng := rand.New(rand.NewSource(9))
	src := make([]byte, 6000)
	for i := range src {
		src[i] = byte(rng.Intn(2))
	}
	iq := demod.ModulateFFSK(src, rate, SymbolRate, MarkHz, SpaceHz)
	iq = demod.ApplyImpairments(iq, rate, demod.Impairments{SNRdB: 4, Seed: 1})

	var rx []byte
	r := New(Options{
		SampleRateHz: rate,
		BitSink:      func(b []byte, _ int) { rx = append(rx, b...) },
	})
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		r.Process(iq[i:end])
	}

	ber := bestAlignBER(src, rx, int(rate/SymbolRate+0.5))
	t.Logf("low-SNR (4 dB) recovered BER = %.4f over %d bits", ber, len(rx))
	if ber >= 0.015 {
		t.Errorf("BER = %.4f, want < 0.015 — the symbol matched filter should hold "+
			"the error rate well below the ~2.5%% the raw per-sample slicer gives at 4 dB", ber)
	}
}

// bestAlignBER returns the minimum bit-error rate of rx against src over a small
// lag window (absorbing the demod chain's constant group delay).
func bestAlignBER(src, rx []byte, sps int) float64 {
	best := 1.0
	for lag := 0; lag < sps*4 && lag < len(rx); lag++ {
		errs, cnt := 0, 0
		for i := 0; i+lag < len(rx) && i < len(src); i++ {
			if rx[i+lag] != src[i] {
				errs++
			}
			cnt++
		}
		if cnt > 500 {
			if r := float64(errs) / float64(cnt); r < best {
				best = r
			}
		}
	}
	return best
}
