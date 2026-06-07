package receiver

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
)

// TestChannelFilterSelectivity checks the channel-select filter the
// receiver builds passes the wanted ±12 kHz channel flat while deeply
// rejecting an adjacent carrier ~25 kHz away — the leakage the wide
// 144 kHz channelised passband admits and the RRC matched filter alone
// does not remove well (issue #553).
func TestChannelFilterSelectivity(t *testing.T) {
	const rate = 144_000.0
	const sps = 8
	taps := filter.LowpassKaiser(2*channelFilterSpanSymbols*sps+1, ChannelCutoffHz/rate, channelFilterBeta)

	mag := func(fHz float64) float64 {
		var acc complex128
		for k, h := range taps {
			acc += complex(float64(h), 0) * cmplx.Exp(complex(0, -2*math.Pi*fHz*float64(k)/rate))
		}
		return cmplx.Abs(acc)
	}
	db := func(x float64) float64 { return 20 * math.Log10(x) }

	// Passband (within the ±12.15 kHz occupied band) must be ~flat.
	for _, f := range []float64{0, 5000, 10000, 12000} {
		if d := db(mag(f)); d < -1.0 {
			t.Errorf("passband %.0f Hz attenuated %.1f dB, want > -1 dB", f, d)
		}
	}
	// Adjacent carrier (≥ 25 kHz spacing) must be deeply rejected.
	for _, f := range []float64{25000, 30000, 40000} {
		if d := db(mag(f)); d > -40 {
			t.Errorf("stopband %.0f Hz only %.1f dB down, want < -40 dB", f, d)
		}
	}
}

// TestChannelFilterNoopOnCleanSignal confirms the channel filter does
// not harm a clean, centred single-carrier capture: the receiver still
// recovers the training sequence. Guards the integer-symbol group-delay
// requirement — a fractional-symbol delay would shift the decimator off
// the symbol centres and wreck a clean signal.
func TestChannelFilterNoopOnCleanSignal(t *testing.T) {
	const sps, span, alpha, rate = 8, 8, 0.35, 144_000.0
	sync := tetra.NormalSyncDibits()
	want := make([]uint8, 0)
	for i := 0; i < 600; i++ {
		want = append(want, uint8(i&3))
	}
	for r := 0; r < 40; r++ {
		want = append(want, sync...)
		for i := 0; i < 108; i++ {
			want = append(want, uint8((i*7+r*3)&3))
		}
	}
	iq := demod.ModulatePiOver4DQPSK(want, sps, span, alpha, math.Pi/4)

	var got []uint8
	rx := New(Options{
		SampleRateHz: rate, ClockMode: ClockGardner, GardnerGain: 0.005,
		EnableChannelFilter: true,
		DibitSink:           func(d []uint8, _ int) { got = append(got, d...) },
	})
	for i := 0; i < len(iq); i += 4096 {
		e := i + 4096
		if e > len(iq) {
			e = len(iq)
		}
		rx.Process(iq[i:e])
	}
	best := 0
	for rot := uint8(0); rot < 4; rot++ {
		c := 0
		for p := 0; p+len(sync) <= len(got); p++ {
			m := 0
			for k := range sync {
				if (got[p+k]+rot)&3 != sync[k] {
					m++
				}
			}
			if m == 0 {
				c++
			}
		}
		if c > best {
			best = c
		}
	}
	if best < 30 {
		t.Errorf("channel filter harmed a clean signal: recovered %d/40 sync words, want >=30", best)
	}
}
