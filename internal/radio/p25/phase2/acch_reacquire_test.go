//go:build integration

package phase2_test

// Does the Phase 2 receiver recover on its own once it loses carrier?
//
// A continuous run over a capture stops finding the outbound sync a fraction
// of a second in and never finds it again, yet a receiver started fresh on the
// later samples locks them without trouble. That points at receiver state
// rather than the channel. This measures the gap directly: the same capture
// decoded in one continuous pass, versus decoded in independent windows, each
// with a fresh receiver.
//
// Windowing throws away every burst that straddles a boundary, so it is a
// lower bound on what re-acquisition would recover.

import (
	"fmt"
	"os"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// decodeWindow runs one fresh receiver + superframe decoder over seg and
// returns the ACCH burst count, the decoded count and the distinct PDUs.
func decodeWindow(seg []complex64, rate float64, seq []byte, pdus map[string]int) (bursts, decoded int) {
	dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, 0)
	sfDec := p25p2.NewSuperframeDecoder()
	sink := func(dibits []uint8, baseIdx int) {
		for _, sf := range sfDec.Process(dibits, baseIdx) {
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
	for i := 0; i < len(seg); i += chunk {
		end := i + chunk
		if end > len(seg) {
			end = len(seg)
		}
		buf = dc.Process(buf[:0], seg[i:end])
		r.Process(buf)
	}
	return bursts, decoded
}

// decodeWithWatchdog is the continuous path plus the production
// CarrierWatchdog: one receiver, reset whenever it has gone
// ReacquireIdleSuperframes without locking a superframe. The reset is applied
// between IQ chunks, never from inside the receiver's own sink.
func decodeWithWatchdog(iq []complex64, rate float64, seq []byte, pdus map[string]int, idleSF int) (bursts, decoded, resets int) {
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
			r.Reset()
			sfDec.Reset()
			reacquire = false
			resets++
		}
	}
	return bursts, decoded, resets
}

func TestACCHReacquire(t *testing.T) {
	path := os.Getenv("GT_P2_WAV")
	if path == "" {
		t.Skip("set GT_P2_WAV")
	}
	iq, rate := readWavIQ(t, path)
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))

	cont := map[string]int{}
	cb, cd := decodeWindow(iq, rate, seq, cont)
	t.Logf("continuous          : acch_bursts=%3d decoded=%3d distinct_pdus=%d", cb, cd, len(cont))

	for _, idleSF := range []int{1, 2, 3} {
		wdPDUs := map[string]int{}
		wb, wdd, nres := decodeWithWatchdog(iq, rate, seq, wdPDUs, idleSF)
		t.Logf("watchdog idle=%dsf   : acch_bursts=%3d decoded=%3d distinct_pdus=%d resets=%d",
			idleSF, wb, wdd, len(wdPDUs), nres)
		if idleSF == 1 {
			for h := range wdPDUs {
				fmt.Printf("WDMAC %s\n", h)
			}
		}
	}

	for _, winS := range []float64{2.0, 1.0, 0.5, 0.25} {
		win := int(winS * rate)
		pdus := map[string]int{}
		var wb, wd int
		for i := 0; i < len(iq); i += win {
			end := i + win
			if end > len(iq) {
				end = len(iq)
			}
			b, d := decodeWindow(iq[i:end], rate, seq, pdus)
			wb += b
			wd += d
		}
		t.Logf("restart every %4.2fs : acch_bursts=%3d decoded=%3d distinct_pdus=%d", winS, wb, wd, len(pdus))
		if winS == 1.0 {
			for h := range pdus {
				fmt.Printf("WINMAC %s\n", h)
			}
		}
	}
}
