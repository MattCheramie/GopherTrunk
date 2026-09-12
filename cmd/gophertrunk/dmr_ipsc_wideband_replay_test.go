package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/tuner"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDMRIPSCWidebandReplay re-creates the WIDEBAND geometry a narrowband IPSC
// capture was received through and A/Bs the two tuner banks on it: the
// capture (a Signal Lab 25 kS/s wav/flac slice, GT_DMR_IQ) is interpolated to
// a wideband rate of GT_DMR_WB_BINS channelizer bins of GT_DMR_WB_BIN_HZ each,
// mixed to GT_DMR_WB_OFFSET_HZ from centre, and fed to a ChannelizerBank (the
// daemon's tuner_strategy: polyphase) and a DDCBank with one tap each at that
// offset, each driving its own DMR receiver + Tier II state machine. Per-window
// idle-beacon and FEC counts show whether the polyphase tap goes deaf where
// the DDC does not.
//
// Defaults reproduce the 10 Sep field report: a 6.25 MS/s X310 plan gets 32
// bins of 195.3125 kHz, and the repeater at +687.5 kHz lands 0.48 of a bin
// from bin 4's centre — only the bin width and residual matter to the tap, so
// 8 bins of the same width keep the replay affordable (and the tap in band).
// GT_DMR_WB_START / GT_DMR_WB_END (seconds) select an excerpt.
func TestDMRIPSCWidebandReplay(t *testing.T) {
	path := os.Getenv("GT_DMR_IQ")
	if path == "" || os.Getenv("GT_DMR_WB") == "" {
		t.Skip("set GT_DMR_IQ (container capture) and GT_DMR_WB=1")
	}
	iq, rate, err := siglab.DecodeContainerFile(path)
	if err != nil {
		t.Fatal(err)
	}
	inRate := float64(rate)
	bins := envInt("GT_DMR_WB_BINS", 8)
	binHz := envFloat("GT_DMR_WB_BIN_HZ", 6_250_000.0/32)
	offsetHz := envFloat("GT_DMR_WB_OFFSET_HZ", 687_500)
	startS := envFloat("GT_DMR_WB_START", 0)
	endS := envFloat("GT_DMR_WB_END", float64(len(iq))/inRate)
	iq = iq[int(startS*inRate):int(math.Min(endS*inRate, float64(len(iq))))]
	wbRate := binHz * float64(bins)
	l, m := rationalRatioTest(wbRate, inRate)
	up := dsp.NewResampler(l, m, 24, 9.0)
	t.Logf("capture %.0f Hz %.1fs → wideband %.1f Hz (L=%d M=%d), %d bins of %.1f Hz, tap at %+.0f Hz = bin %.2f",
		inRate, float64(len(iq))/inRate, wbRate, l, m, bins, binHz, offsetHz, offsetHz/binHz)

	type arm struct {
		name  string
		bank  tuner.Bank
		rx    *dmrrx.Receiver
		cc    *tier2.ConventionalChannel
		bus   *events.Bus
		grant []string
		lastB uint64
		lastF uint64
		rows  []string
	}
	mk := func(name string, bank tuner.Bank) *arm {
		a := &arm{name: name, bank: bank, bus: events.NewBus(256)}
		a.cc = tier2.New(tier2.Options{Bus: a.bus, SystemName: name, FrequencyHz: 0})
		a.rx = dmrrx.New(dmrrx.Options{SampleRateHz: bank.OutputRateHz(), DeviationHz: 1944.0, ClockGain: 0.015,
			DibitSink: dmr.DibitSink(func(d []uint8, b int) { a.cc.Process(d, b) })})
		if err := bank.AddTap(offsetHz, func(out []complex64) { a.rx.Process(out) }); err != nil {
			t.Fatal(err)
		}
		sub := a.bus.Subscribe()
		go func() {
			for ev := range sub.C {
				if ev.Kind == events.KindGrant {
					if g, ok := ev.Payload.(trunking.Grant); ok {
						a.grant = append(a.grant, fmt.Sprintf("tg=%d src=%d", g.GroupID, g.SourceID))
					}
				}
			}
		}()
		return a
	}
	arms := []*arm{
		mk("polyphase", tuner.NewChannelizerBank(wbRate, 48_000, 0.05, bins, 16, 9.0)),
		mk("ddc", tuner.NewDDCBank(wbRate, 48_000, 0.05)),
	}
	const chunk = 25_000
	window := int(10 * inRate)
	var wide []complex64
	var nco complex128 = 1
	step := complex(math.Cos(2*math.Pi*offsetHz/wbRate), math.Sin(2*math.Pi*offsetHz/wbRate))
	for i := 0; i < len(iq); i += chunk {
		e := i + chunk
		if e > len(iq) {
			e = len(iq)
		}
		wide = up.Process(wide[:0], iq[i:e])
		for j := range wide {
			wide[j] = complex(float32(real(nco))*real(wide[j])-float32(imag(nco))*imag(wide[j]),
				float32(real(nco))*imag(wide[j])+float32(imag(nco))*real(wide[j]))
			nco *= step
		}
		if i%(chunk*40) == 0 {
			nco /= complex(math.Hypot(real(nco), imag(nco)), 0)
		}
		// GT_DMR_WB_CHUNK feeds the banks in wideband chunks of this many samples
		// (the live daemon's SoapyRemote datagram is ~2000 samples at 6.25 MS/s,
		// i.e. ~125 samples per channelizer bin per call); 0 = one call per
		// resampled block.
		if wbChunk := envInt("GT_DMR_WB_CHUNK", 0); wbChunk > 0 {
			for k := 0; k < len(wide); k += wbChunk {
				ke := k + wbChunk
				if ke > len(wide) {
					ke = len(wide)
				}
				for _, a := range arms {
					a.bank.Process(wide[k:ke])
				}
			}
		} else {
			for _, a := range arms {
				a.bank.Process(wide)
			}
		}
		if (e/window) != (i/window) || e == len(iq) {
			sec := startS + float64(e)/inRate
			for _, a := range arms {
				c := a.cc.Counters()
				a.rows = append(a.rows, fmt.Sprintf("%6.0fs beacons %4d fec %3d", sec, c.Beacons-a.lastB, c.FECPass-a.lastF))
				a.lastB, a.lastF = c.Beacons, c.FECPass
			}
		}
	}
	for _, a := range arms {
		c := a.cc.Counters()
		t.Logf("%-9s out_rate=%.1f beacons=%d fec_pass=%d fec_fail=%d late_entries=%d rekeys=%d grants=%d", a.name, a.bank.OutputRateHz(), c.Beacons, c.FECPass, c.FECFail, c.LateEntries, c.Rekeys, len(a.grant))
	}
	for i := range arms[0].rows {
		t.Logf("  %s | %s", arms[0].rows[i], arms[1].rows[i])
	}
	for _, a := range arms {
		a.bus.Close()
	}
}

func envInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}

func envFloat(k string, d float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return d
}

// rationalRatioTest reduces out/in to a small L/M for the interpolator.
func rationalRatioTest(out, in float64) (int, int) {
	for m := 1; m <= 64; m++ {
		l := out * float64(m) / in
		if math.Abs(l-math.Round(l)) < 1e-6 {
			return int(math.Round(l)), m
		}
	}
	return int(math.Round(out / in)), 1
}
