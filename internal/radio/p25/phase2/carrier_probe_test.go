//go:build integration

package phase2_test

// What does the outbound frame sync look like, in the complex differential
// domain, at the point where the receiver hands dibits out?
//
// Two questions this answers on real air, neither of which a synthesized
// fixture can:
//
//  1. Where do the ideal differential points actually sit? The dibit slicer's
//     decision regions are centred on π/8 + k·π/2 while the standard's sync
//     documentation puts the on-air phases at ±π/4 / ±3π/4 — a constant π/8
//     apart. A sync-driven AFC that assumes the wrong one injects a constant
//     375 Hz error, so this has to be measured rather than reasoned about.
//
//  2. Is the sync still *there* during the stretch the receiver has given up
//     on? The dibit-domain detector cannot see a sync whose symbols have all
//     rotated a quadrant, but a complex correlation is rotation-blind: it
//     finds the sync and reports the rotation as the residual carrier.
//
//	GT_P2_WAV=/path/capture.wav go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run CarrierSyncProbe -v

import (
	"math"
	"math/cmplx"
	"os"
	"testing"

	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// syncDiffPhaseA is the differential phase of each canonical dibit under the
// standard's sync documentation: {0:+π/4, 1:+3π/4, 2:−π/4, 3:−3π/4}.
var syncDiffPhaseA = [4]float64{math.Pi / 4, 3 * math.Pi / 4, -math.Pi / 4, -3 * math.Pi / 4}

func TestCarrierSyncProbe(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)

	syncDibits := p25p2.OutboundSyncDibits()
	// Ideal phasor per sync position, hypothesis A.
	idealA := make([]complex128, len(syncDibits))
	for i, d := range syncDibits {
		idealA[i] = cmplx.Exp(complex(0, syncDiffPhaseA[d&3]))
	}

	const n = p25p2.SyncDibits
	ring := make([]complex128, n)
	pos, primed := 0, 0

	det := p25p2.NewSyncDetector(syncDibits, 1)
	var detHits []int
	var detIdx []int

	type hit struct {
		idx  int
		coh  float64
		eps  float64 // residual per-symbol rotation, rad
		hard bool    // the dibit-domain detector fired here too
	}
	var hits []hit

	sink := func(dibits []uint8, soft []complex64, baseIdx int) {
		detIdx, _ = det.Process(detIdx[:0], dibits, baseIdx)
		detHits = append(detHits, detIdx...)
		hard := map[int]bool{}
		for _, h := range detIdx {
			hard[h] = true
		}
		const c, s = 0.9238795325112867, -0.3826834323650898 // cos/sin(−π/8)
		for i := range dibits {
			// soft = diff·e^{+jπ/8}; undo it to recover the raw differential.
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
			if coh >= 0.55 {
				hits = append(hits, hit{idx: baseIdx + i, coh: coh,
					eps: cmplx.Phase(corr), hard: hard[baseIdx+i]})
			}
		}
	}

	dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, 0)
	r := rx.New(rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner,
		GardnerGain: 0.005, SoftDecision: true, SoftSink: sink})
	var buf []complex64
	const chunk = 8192
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		buf = dc.Process(buf[:0], iq[i:end])
		r.Process(buf)
	}

	t.Logf("dibit-domain sync hits (tol 1): %d", len(detHits))
	t.Logf("capture %.2f s = %d dibits; one burst every %d dibits -> %.0f syncs expected",
		float64(len(iq))/rate, int(float64(len(iq))/rate*rx.SymbolRate), 180,
		float64(len(iq))/rate*rx.SymbolRate/180)

	// Hypothesis check on the hits the dibit detector also found (receiver
	// locked, so the true residual is ~0). ~0 => map A; ~-pi/8 => map B.
	var sum complex128
	nLocked := 0
	for _, h := range hits {
		if h.hard {
			sum += cmplx.Exp(complex(0, h.eps))
			nLocked++
		}
	}
	if nLocked > 0 {
		mean := cmplx.Phase(sum)
		t.Logf("locked hits=%d  mean eps=%+.4f rad (%.0f Hz)  |R|=%.3f  [A -> ~0, B -> ~%.4f]",
			nLocked, mean, mean*rx.SymbolRate/(2*math.Pi), cmplx.Abs(sum)/float64(nLocked),
			-math.Pi/8)
	}

	// Are these real bursts? Real outbound syncs land 180 dibits apart. Cluster
	// the raw hits (keep the local maximum within a burst) and histogram the
	// spacing between consecutive clusters, per coherence threshold.
	for _, thr := range []float64{0.55, 0.65, 0.75, 0.85} {
		var cl []hit
		for _, h := range hits {
			if h.coh < thr {
				continue
			}
			if n := len(cl); n > 0 && h.idx-cl[n-1].idx <= 8 {
				if h.coh > cl[n-1].coh {
					cl[n-1] = h
				}
				continue
			}
			cl = append(cl, h)
		}
		gaps := map[int]int{}
		on180 := 0
		for i := 1; i < len(cl); i++ {
			g := cl[i].idx - cl[i-1].idx
			gaps[g]++
			if g%180 == 0 || (g+1)%180 == 0 || (g-1)%180 == 0 {
				on180++
			}
		}
		t.Logf("thr=%.2f clusters=%4d  gaps on a 180-dibit lattice: %d/%d (%.0f%%)",
			thr, len(cl), on180, len(cl)-1, 100*float64(on180)/math.Max(1, float64(len(cl)-1)))
		if thr == 0.75 {
			// Residual trace: where is the carrier over time?
			t.Logf("  residual trace (thr 0.75, every 8th cluster):")
			for i := 0; i < len(cl); i += 8 {
				t.Logf("    t=%5.2fs  coh=%.2f  residual=%+7.0f Hz  hard=%v",
					float64(cl[i].idx)/rx.SymbolRate, cl[i].coh,
					cl[i].eps*rx.SymbolRate/(2*math.Pi), cl[i].hard)
			}
		}
	}
}
