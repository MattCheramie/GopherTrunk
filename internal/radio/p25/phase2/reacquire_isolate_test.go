//go:build integration

package phase2_test

// Which piece of receiver state actually goes bad when a Phase 2 channel is
// lost?
//
// The carrier watchdog recovers a lot of signalling by resetting the whole
// receiver, but a full reset conflates several hypotheses: the coarse carrier
// seed, the Costas loop, the Gardner timing phase, the AGC and the
// differential reference all go at once. A fresh receiver on the same samples
// locks them, so it is state — but not which state.
//
// This runs the identical watchdog schedule with each subset in turn, so the
// yield difference attributes the loss.
//
//	GT_P2_WAV=/path/capture.wav go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run ReacquireIsolate -v

import (
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// decodeWatchdogWith is decodeWithWatchdog with the recovery action supplied,
// so the same trigger schedule can drive different subsets of a reset.
func decodeWatchdogWith(iq []complex64, rate float64, seq []byte, pdus map[string]int,
	idleSF int, recover func(*rx.Receiver, *p25p2.SuperframeDecoder)) (bursts, decoded, resets int) {

	dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, 0)
	sfDec := p25p2.NewSuperframeDecoder()
	wd := p25p2.NewCarrierWatchdog(idleSF)
	reacquire := false
	sink := func(dibits []uint8, baseIdx int) {
		sfs := sfDec.Process(dibits, baseIdx)
		if wd.Observe(len(dibits), len(sfs)) {
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
		GardnerGain: 0.005, DibitSink: sink})
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
			recover(r, sfDec)
			reacquire = false
			resets++
		}
	}
	return bursts, decoded, resets
}

func TestReacquireIsolate(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))

	cases := []struct {
		name string
		fn   func(*rx.Receiver, *p25p2.SuperframeDecoder)
	}{
		{"nothing (continuous)", func(r *rx.Receiver, d *p25p2.SuperframeDecoder) {}},
		{"carrier only", func(r *rx.Receiver, d *p25p2.SuperframeDecoder) { r.ReacquireCarrier() }},
		{"timing only", func(r *rx.Receiver, d *p25p2.SuperframeDecoder) { r.ReacquireTiming() }},
		{"carrier + timing", func(r *rx.Receiver, d *p25p2.SuperframeDecoder) {
			r.ReacquireCarrier()
			r.ReacquireTiming()
		}},
		{"carrier + sfdec", func(r *rx.Receiver, d *p25p2.SuperframeDecoder) {
			r.ReacquireCarrier()
			d.Reset()
		}},
		{"full Reset (shipped)", func(r *rx.Receiver, d *p25p2.SuperframeDecoder) {
			r.Reset()
			d.Reset()
		}},
	}
	for _, part := range []string{"gardner", "agc", "dq", "pending", "eq", "costas", "nco"} {
		pdus := map[string]int{}
		pt := part
		b, d, n := decodeWatchdogWith(iq, rate, seq, pdus, 1,
			func(r *rx.Receiver, _ *p25p2.SuperframeDecoder) { r.ReacquirePart(pt) })
		t.Logf("part %-10s      : acch_bursts=%3d decoded=%3d distinct_pdus=%2d triggers=%d",
			part, b, d, len(pdus), n)
	}

	for _, c := range cases {
		pdus := map[string]int{}
		b, d, n := decodeWatchdogWith(iq, rate, seq, pdus, 1, c.fn)
		t.Logf("%-22s: acch_bursts=%3d decoded=%3d distinct_pdus=%2d triggers=%d",
			c.name, b, d, len(pdus), n)
	}
}
