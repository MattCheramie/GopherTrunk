//go:build integration

package phase2_test

// Traces the front end's state over a capture the receiver loses partway
// through: symbols emitted per unit time (the recovered symbol *rate*), the
// Gardner sub-sample phase, the AGC gain, and whether the outbound sync is
// still correlating. If the loss is a timing-loop drift the symbol rate
// departs from 6000/s and never comes back; if it is carrier, the rate holds
// and only the sync correlation dies.
//
//	GT_P2_WAV=/path/capture.wav go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run TimingTrace -v

import (
	"math/cmplx"
	"os"
	"testing"

	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

func TestTimingTrace(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)

	syncDibits := p25p2.OutboundSyncDibits()
	idealA := make([]complex128, len(syncDibits))
	for i, d := range syncDibits {
		idealA[i] = cmplx.Exp(complex(0, syncDiffPhaseA[d&3]))
	}
	const n = p25p2.SyncDibits
	ring := make([]complex128, n)
	pos, primed := 0, 0

	// Per-bucket accumulators, one bucket per 0.25 s of *input*.
	type bucket struct {
		symbols  int
		syncHits int
		bestCoh  float64
		mu       float64
		gain     float64
	}
	var buckets []bucket
	cur := -1

	sink := func(dibits []uint8, soft []complex64, baseIdx int) {
		if cur >= 0 {
			buckets[cur].symbols += len(dibits)
		}
		const c, s = 0.9238795325112867, -0.3826834323650898
		for i := range dibits {
			re, im := float64(real(soft[i])), float64(imag(soft[i]))
			ring[pos] = complex(re*c-im*s, re*s+im*c)
			pos = (pos + 1) % n
			if primed < n {
				primed++
				continue
			}
			var corr complex128
			var mag float64
			idx := pos
			for k := 0; k < n; k++ {
				corr += ring[idx] * cmplx.Conj(idealA[k])
				mag += cmplx.Abs(ring[idx])
				idx = (idx + 1) % n
			}
			if mag == 0 {
				continue
			}
			coh := cmplx.Abs(corr) / mag
			if cur >= 0 {
				if coh > buckets[cur].bestCoh {
					buckets[cur].bestCoh = coh
				}
				if coh >= 0.85 {
					buckets[cur].syncHits++
				}
			}
		}
	}

	dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, 0)
	r := rx.New(rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner,
		GardnerGain: 0.005, SoftDecision: true, SoftSink: sink})

	var buf []complex64
	chunk := int(rate * 0.25) // one bucket per Process call
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		buckets = append(buckets, bucket{})
		cur = len(buckets) - 1
		buf = dc.Process(buf[:0], iq[i:end])
		r.Process(buf)
		buckets[cur].mu = r.TimingMu()
		buckets[cur].gain = r.AGCGain()
	}

	t.Logf("expected %.0f symbols per 0.25 s bucket", 0.25*rx.SymbolRate)
	t.Logf("%5s %8s %8s %8s %8s %8s", "t(s)", "symbols", "rate", "sync>=.85", "bestCoh", "mu")
	for i, b := range buckets {
		t.Logf("%5.2f %8d %8.1f %8d %8.2f %8.4f  agc=%.3f",
			float64(i)*0.25, b.symbols, float64(b.symbols)/0.25, b.syncHits, b.bestCoh, b.mu, b.gain)
	}

	// Summary: is the average recovered symbol rate right?
	total := 0
	for _, b := range buckets {
		total += b.symbols
	}
	dur := float64(len(iq)) / rate
	t.Logf("total symbols=%d over %.2f s -> %.2f sym/s (nominal %.0f, error %+.3f%%)",
		total, dur, float64(total)/dur, rx.SymbolRate,
		100*(float64(total)/dur/rx.SymbolRate-1))
}
