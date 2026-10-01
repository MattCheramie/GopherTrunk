//go:build integration

package phase2_test

// Sweeps the Gardner loop gain on the continuous (no-watchdog) path.
//
// The re-acquisition loss attributes entirely to the Gardner timing loop:
// resetting it alone recovers everything a full receiver reset does, and
// resetting carrier recovery alone recovers nothing. The loop does not lose
// the symbol *rate* — it settles on a stable but wrong sampling phase and
// stays there. If that is the loop being dragged off the eye by a biased
// timing-error detector, the drift rate is proportional to the loop gain.
//
//	GT_P2_WAV=/path/capture.wav go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run GardnerGain -v

import (
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

func decodeGain(iq []complex64, rate float64, seq []byte, pdus map[string]int,
	gain float64, idleSF int) (bursts, decoded, resets int) {

	dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, 0)
	sfDec := p25p2.NewSuperframeDecoder()
	var wd *p25p2.CarrierWatchdog
	if idleSF > 0 {
		wd = p25p2.NewCarrierWatchdog(idleSF)
	}
	reacquire := false
	sink := func(dibits []uint8, baseIdx int) {
		sfs := sfDec.Process(dibits, baseIdx)
		if wd != nil && wd.Observe(len(dibits), len(sfs)) {
			reacquire = true
		}
		for _, sf := range sfs {
			for _, sub := range sf.Subframes {
				if len(sub.Dibits) < p25p2.BurstDibits || !p25p2.BurstTypeOf(sub.Dibits).IsACCH() {
					continue
				}
				bursts++
				for phase := 0; phase < 12; phase++ {
					if res, ok := p25p2.DecodeACCHBurst(sub.Dibits, phase, seq); ok {
						decoded++
						pdus[hexOf(res.Message)]++
						break
					}
				}
			}
		}
	}
	r := rx.New(rx.Options{SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner,
		GardnerGain: gain, DibitSink: sink})
	var buf []complex64
	const chunk = 8192
	for i := 0; i < len(iq); i += chunk {
		end := i + chunk
		if end > len(iq) {
			end = len(iq)
		}
		buf = dc.Process(buf[:0], iq[i:end])
		r.Process(buf)
		if reacquire {
			r.Reset()
			sfDec.Reset()
			reacquire = false
			resets++
		}
	}
	return bursts, decoded, resets
}

func TestGardnerGain(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))

	gains := []float64{0.05, 0.02, 0.01, 0.005, 0.002, 0.001, 5e-4, 1e-4, 1e-5, 1e-9}
	for _, idle := range []int{0, 1} {
		label := "continuous"
		if idle > 0 {
			label = "watchdog"
		}
		for _, g := range gains {
			pdus := map[string]int{}
			b, d, n := decodeGain(iq, rate, seq, pdus, g, idle)
			t.Logf("%-10s gain=%-8g: acch_bursts=%3d decoded=%3d distinct_pdus=%2d resets=%d",
				label, g, b, d, len(pdus), n)
		}
	}
}
