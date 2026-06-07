package receiver

import (
	"math"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// TestAFCRecoversDibitsUnderCarrierOffset modulates a known dibit
// sequence, applies a residual carrier-frequency offset to the IQ
// (the failure mode the live DDC can't correct), and confirms the
// AFC-enabled receiver recovers the original dibits while a receiver
// without AFC does not.
func TestAFCRecoversDibitsUnderCarrierOffset(t *testing.T) {
	const (
		sps   = 8
		span  = 8
		alpha = 0.35
		rate  = 144_000.0
		// 3500 Hz ≈ 70° per-symbol differential rotation — well past the
		// ±45° decision margin, so the plain differential decoder produces
		// garbage. (Below ~45° the plain decoder copes without AFC.)
		offsetHz = 3500.0
	)

	// A long pseudo-random dibit stream + a known training sequence
	// repeated so the AFC has data to acquire/track on.
	want := make([]uint8, 0, 4000)
	for i := 0; i < 600; i++ { // warmup
		want = append(want, uint8(i&3))
	}
	sync := tetra.NormalSyncDibits()
	lcg := uint32(12345)
	for r := 0; r < 30; r++ {
		want = append(want, sync...)
		for i := 0; i < 100; i++ {
			lcg = lcg*1664525 + 1013904223
			want = append(want, uint8((lcg>>16)&3))
		}
	}

	iq := demod.ModulatePiOver4DQPSK(want, sps, span, alpha, math.Pi/4)
	// Apply the carrier offset.
	for n := range iq {
		ph := 2 * math.Pi * offsetHz * float64(n) / rate
		rot := complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
		iq[n] *= rot
	}

	run := func(enableAFC bool) []uint8 {
		var got []uint8
		rx := New(Options{
			SampleRateHz: rate,
			ClockMode:    ClockGardner,
			GardnerGain:  0.005,
			EnableAFC:    enableAFC,
			DibitSink:    func(d []uint8, _ int) { got = append(got, d...) },
		})
		const chunk = 4096
		for i := 0; i < len(iq); i += chunk {
			end := i + chunk
			if end > len(iq) {
				end = len(iq)
			}
			rx.Process(iq[i:end])
		}
		return got
	}

	// Best sync-correlation hit count for the training sequence, tried
	// under every constant constellation rotation (π/4-DQPSK differential
	// decode has an inherent 90°/dibit ambiguity the framing layer
	// resolves). A proxy for "did the demod recover the real symbols".
	hits := func(stream []uint8) int {
		best := 0
		for rot := uint8(0); rot < 4; rot++ {
			n := 0
			for pos := 0; pos+len(sync) <= len(stream); pos++ {
				mism := 0
				for k := range sync {
					if (stream[pos+k]+rot)&3 != sync[k] {
						mism++
					}
				}
				if mism == 0 {
					n++
				}
			}
			if n > best {
				best = n
			}
		}
		return best
	}

	withAFC := hits(run(true))
	without := hits(run(false))
	if withAFC < 20 {
		t.Errorf("AFC: exact sync hits = %d under %g Hz offset, want >=20", withAFC, offsetHz)
	}
	if without >= withAFC {
		t.Errorf("AFC made no difference: with=%d without=%d (AFC should rescue an offset that defeats the plain decoder)", withAFC, without)
	}
}

// TestAFCNoopWithoutOffset confirms the AFC is ~harmless on a
// perfectly-centred stream: a receiver with AFC recovers the training
// sequence just as a plain receiver does.
func TestAFCNoopWithoutOffset(t *testing.T) {
	const (
		sps   = 8
		span  = 8
		alpha = 0.35
		rate  = 144_000.0
	)
	sync := tetra.NormalSyncDibits()
	want := make([]uint8, 0, 2000)
	for i := 0; i < 600; i++ {
		want = append(want, uint8(i&3))
	}
	for r := 0; r < 20; r++ {
		want = append(want, sync...)
		for i := 0; i < 100; i++ {
			want = append(want, uint8((i*7+r)&3))
		}
	}
	iq := demod.ModulatePiOver4DQPSK(want, sps, span, alpha, math.Pi/4)
	var got []uint8
	rx := New(Options{
		SampleRateHz: rate, ClockMode: ClockGardner, GardnerGain: 0.005, EnableAFC: true,
		DibitSink: func(d []uint8, _ int) { got = append(got, d...) },
	})
	const chunk = 4096
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[i:end])
	}
	n := 0
	for pos := 0; pos+len(sync) <= len(got); pos++ {
		mism := 0
		for k := range sync {
			if got[pos+k] != sync[k] {
				mism++
			}
		}
		if mism == 0 {
			n++
		}
	}
	if n < 15 {
		t.Errorf("AFC noop: exact sync hits on centred stream = %d, want >=15", n)
	}
}
