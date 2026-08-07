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
	for r := 0; r < 200; r++ {
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
	if withAFC < 80 {
		t.Errorf("AFC: exact sync hits = %d under %g Hz offset, want >=80", withAFC, offsetHz)
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
	lcg := uint32(98765)
	for r := 0; r < 200; r++ {
		want = append(want, sync...)
		for i := 0; i < 100; i++ {
			lcg = lcg*1664525 + 1013904223
			want = append(want, uint8((lcg>>16)&3))
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
	if n < 80 {
		t.Errorf("AFC noop: exact sync hits on centred stream = %d, want >=80", n)
	}
}

// TestAFCReportedOffsetStableUnderMultipath pins that a locked control channel
// reports a STABLE residual carrier offset under multipath/ISI — it must not
// swing by ~±1 kHz block-to-block. A real reporter saw carrier_off_hz jitter
// from −1492 to +616 Hz on a fully-locked, calm CC (bsch_ok 100%), and the
// symbol-scope constellation "drift" a continuous-tracking reference demod does
// not show. The cause is the feed-forward AFC re-deriving a noisy estimate every
// ~1024-symbol block: on an ISI-smeared constellation both the coarse lag-1 and
// the fine 4×Δφ stages jitter, so the reported offset (and the per-block
// derotation that spins the scope) swings even though the true offset is ~fixed.
//
// The across-block EMA smooths that jitter without moving the tracked mean. This
// test drives the production 144 kHz / Gardner / AFC + channel-filter path over
// a multipath+noise channel and asserts the per-block reported offset stays
// tightly bounded. On a clean carrier the AFC is already quiet, so the multipath
// is what makes this fail before the smoothing and pass after.
func TestAFCReportedOffsetStableUnderMultipath(t *testing.T) {
	const (
		sps   = 8
		span  = 8
		alpha = 0.35
		rate  = 144_000.0
	)

	// A long random π/4-DQPSK stream so the AFC processes many ~1024-symbol
	// blocks and we can measure the block-to-block spread of its estimate.
	sync := tetra.NormalSyncDibits()
	want := make([]uint8, 0, 40000)
	lcg := uint32(0xC0FFEE)
	for r := 0; r < 380; r++ {
		want = append(want, sync...)
		for i := 0; i < 100; i++ {
			lcg = lcg*1664525 + 1013904223
			want = append(want, uint8((lcg>>16)&3))
		}
	}

	// Gaussian-ish noise (sum-of-uniforms), deterministic per subtest.
	noise := func(seed uint32, n int) []complex64 {
		out := make([]complex64, n)
		r := seed
		g := func() float64 {
			var s float64
			for k := 0; k < 12; k++ {
				r = r*1664525 + 1013904223
				s += float64(r>>8) / float64(1<<24)
			}
			return s - 6
		}
		for i := range out {
			out[i] = complex(float32(g()), float32(g()))
		}
		return out
	}

	// A centred carrier plus a modest reflection — the field condition
	// (ISI-smeared but decodable). The exact reflection differs per case so the
	// smoothing is exercised against more than one channel.
	cases := []struct {
		name   string
		delay  int
		gr, gi float32
		snrDB  float64
		seed   uint32
	}{
		{"mildMultipath", 12, 0.55, -0.35, 16, 0x11},
		{"medMultipath", 8, -0.6, 0.5, 14, 0x29},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := demod.ModulatePiOver4DQPSK(want, sps, span, alpha, math.Pi/4)
			iq := make([]complex64, len(base))
			copy(iq, base)
			g := complex(c.gr, c.gi)
			for n := len(iq) - 1; n >= c.delay; n-- {
				iq[n] += g * base[n-c.delay]
			}
			var sp float64
			for _, s := range iq {
				sp += float64(real(s)*real(s) + imag(s)*imag(s))
			}
			sp /= float64(len(iq))
			sigma := math.Sqrt(sp / math.Pow(10, c.snrDB/10) / 2)
			nz := noise(c.seed, len(iq))
			for n := range iq {
				iq[n] += complex(float32(sigma)*real(nz[n]), float32(sigma)*imag(nz[n]))
			}

			rx := New(Options{
				SampleRateHz:        rate,
				ClockMode:           ClockGardner,
				GardnerGain:         0.005,
				EnableAFC:           true,
				EnableChannelFilter: true, // production CC config
				DibitSink:           func([]uint8, int) {},
			})

			// Record the reported offset after each chunk; the AFC updates it once
			// per ~1024-symbol block, so this samples every block at least once.
			var samples []float64
			const chunk = 4096
			for i := 0; i < len(iq); i += chunk {
				end := i + chunk
				if end > len(iq) {
					end = len(iq)
				}
				rx.Process(iq[i:end])
				samples = append(samples, rx.CarrierOffsetHz())
			}

			// Skip a short warm-up (filter + EMA priming), then measure the spread
			// of the remaining per-block estimates.
			if len(samples) < 12 {
				t.Fatalf("too few offset samples: %d", len(samples))
			}
			tail := samples[6:]
			min, max := tail[0], tail[0]
			for _, v := range tail {
				if v < min {
					min = v
				}
				if v > max {
					max = v
				}
			}
			spread := max - min

			// A locked carrier has no fast real drift, so a large block-to-block
			// swing is feed-forward estimate jitter. The raw per-block estimate
			// swings ~1 kHz+ on these channels; the smoothed report must hold to a
			// few hundred Hz.
			if spread > 400 {
				t.Errorf("%s: reported offset spread = %.0f Hz (min %.0f, max %.0f) — "+
					"want <=400 Hz; the feed-forward estimate is jittering the residual "+
					"and spinning the constellation", c.name, spread, min, max)
			}
		})
	}
}
