package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/filter"
	dspsync "github.com/MattCheramie/GopherTrunk/internal/dsp/sync"
	"github.com/MattCheramie/GopherTrunk/internal/radio/ysf"
	ysfrx "github.com/MattCheramie/GopherTrunk/internal/radio/ysf/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

// TestYSFReplay is the System Fusion real-air diagnostic harness (issue
// #1261). Skip unless GT_YSF_IQ points at an IQ capture (cs16 by default,
// GT_DMR_IQ_FORMAT=f32 for a cfile, or a wav/flac container); GT_YSF_RATE
// gives a raw file's sample rate (default 2400000, the voice_iq_debug tap's).
//
//	GT_YSF_IQ=<capture> go test ./cmd/gophertrunk -run 'TestYSFReplay$' -v
//
// It mirrors the voice chain: downconvert to the 48 kHz C4FM channel rate,
// run the production YSF receiver (carrier correction + channel filter +
// feed-forward timing acquisition), cut frames at each FSW and decode the
// FICH. Then it says what the capture IS when nothing framed: the share of
// samples at the ADC rail, the carrier offset, the 4-level eye, and whether
// the discriminator output is a single tone — the reporter's first "YSF"
// capture was an analog FM carrier with a 1 kHz test tone, every sample at
// the rail, and no frame sync anywhere in it.
func TestYSFReplay(t *testing.T) {
	path := os.Getenv("GT_YSF_IQ")
	if path == "" {
		t.Skip("set GT_YSF_IQ (+ GT_YSF_RATE) to replay a System Fusion capture")
	}
	inRate := 2400000.0
	if v := os.Getenv("GT_YSF_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatal(err)
		}
		inRate = f
	}
	iq, inRate := readDMRCaptureIQ(t, path, inRate)
	t.Logf("capture: %d samples at %.0f Hz = %.2f s", len(iq), inRate, float64(len(iq))/inRate)

	// ADC rail share on the raw capture: an RTL-SDR's u8 rails scale to
	// ±1.0, a cs16's to ±0.99997.
	rail := 0
	for _, s := range iq {
		re, im := real(s), imag(s)
		if re >= 0.99 || re <= -0.99 || im >= 0.99 || im <= -0.99 {
			rail++
		}
	}
	railPct := 100 * float64(rail) / float64(len(iq))
	t.Logf("samples with I or Q at the ADC rail: %.1f%%", railPct)

	ddc := ccdecoder.NewDownconverter(inRate, 48000)
	outRate := ddc.OutRateHz()
	var nb []complex64
	for i := 0; i < len(iq); i += 8192 {
		nb = append(nb, ddc.Process(nil, iq[i:min(i+8192, len(iq))])...)
	}

	// Production chain.
	framer := ysf.NewAirFramer(2)
	var dibits, frames, fichOK, vd2, voiceFrames int
	fichs := map[string]int{}
	rx := ysfrx.New(ysfrx.Options{
		SampleRateHz: outRate, DeviationHz: 1800, EnableAFC: true, EnableChannelFilter: true,
		DibitSink: func(d []uint8, base int) {
			dibits += len(d)
			for _, af := range framer.Process(d, base) {
				frames++
				if !af.FICHOK {
					continue
				}
				fichOK++
				f := af.FICH
				fichs[fmt.Sprintf("FI=%d DT=%d CM=%d FN=%d/%d DGID=%d", f.FI, f.DT, f.CM, f.FN, f.FT, f.DGID)]++
				if f.FI == ysf.AirFIComms && f.DT == ysf.AirDTVDMode2 {
					vd2++
					voiceFrames += len(ysf.DecodeVD2Voice(af.Payload))
				}
			}
		},
	})
	rx.ArmTimingAcquisition()
	for i := 0; i < len(nb); i += 1024 {
		rx.Process(nb[i:min(i+1024, len(nb))])
	}
	t.Logf("production chain: dibits=%d frame_syncs=%d fich_ok=%d vd2_frames=%d ambe_frames=%d timing_seeds=%d",
		dibits, frames, fichOK, vd2, voiceFrames, rx.TimingSeeds())
	keys := make([]string, 0, len(fichs))
	for k := range fichs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("  FICH %s x%d", k, fichs[k])
	}

	// Diagnostic chain: the same stages with the discriminator and the
	// symbols exposed.
	sps := outRate / 4800
	cf := filter.NewFIR(filter.LowpassKaiser(2*9*int(sps+0.5)+1, 6250/outRate, 8.6))
	disc := demod.NewFM().Process(nil, cf.Process(nil, nb))
	var mean float64
	for _, v := range disc {
		mean += float64(v)
	}
	mean /= float64(len(disc))
	offsetHz := mean * outRate / (2 * math.Pi)
	t.Logf("carrier offset: %.0f Hz", offsetHz)

	// Dominant discriminator tone: a C4FM data signal has a continuous
	// spectrum; an analog test tone is one line far above everything else.
	const n = 4800
	acc := make([]float64, n/2+1)
	blocks := 0
	for i := 0; i+n <= len(disc); i += n {
		blocks++
		for k := range acc {
			var re, im float64
			for j := 0; j < n; j++ {
				w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(j)/n)
				x := (float64(disc[i+j]) - mean) * w
				a := 2 * math.Pi * float64(k) * float64(j) / n
				re += x * math.Cos(a)
				im -= x * math.Sin(a)
			}
			acc[k] += re*re + im*im
		}
		if blocks == 4 {
			break
		}
	}
	if blocks > 0 {
		db := make([]float64, len(acc))
		for k, p := range acc {
			db[k] = 10 * math.Log10(p/float64(blocks)+1e-12)
		}
		best := 1
		for k := 2; k < len(db); k++ {
			if db[k] > db[best] {
				best = k
			}
		}
		sorted := append([]float64(nil), db[1:]...)
		sort.Float64s(sorted)
		median := sorted[len(sorted)/2]
		toneHz := float64(best) * outRate / n
		t.Logf("discriminator spectrum: strongest line %.0f Hz, %.0f dB over the median bin", toneHz, db[best]-median)
		if db[best]-median > 40 && frames == 0 {
			t.Logf("VERDICT: not a System Fusion transmission — the discriminator is a single %.0f Hz tone (analog FM test tone / voice), with no YSF frame sync in %.2f s", toneHz, float64(len(nb))/outRate)
		}
	}

	afc := demod.NewCoarseAFC(sps)
	afc.Process(disc)
	syms := dspsync.NewMuellerMuller(sps, 0.05).Process(nil, demod.NewC4FM(int(sps+0.5), 8, 0.2, 1).MatchedFilter(nil, disc))
	scale := 2 * math.Pi * 1800 / outRate
	norm := make([]float64, len(syms))
	for i, s := range syms {
		norm[i] = float64(s) / scale
	}
	sort.Float64s(norm)
	pct := func(p float64) float64 { return norm[int(p*float64(len(norm)-1))] }
	t.Logf("4-level eye (±1 inner / ±3 outer): 5%%=%.2f 25%%=%.2f 50%%=%.2f 75%%=%.2f 95%%=%.2f", pct(0.05), pct(0.25), pct(0.5), pct(0.75), pct(0.95))

	switch {
	case railPct > 50:
		t.Logf("VERDICT: the front end is overloaded — %.0f%% of samples sit at the ADC rail; lower the gain or move the antenna away from the radio", railPct)
	case frames == 0:
		t.Logf("VERDICT: no YSF frame sync in the capture")
	case vd2 == 0:
		t.Logf("VERDICT: YSF frames decode but none is V/D mode 2 (DN); the radio is in another mode (VW / DN mode 1 / data)")
	default:
		t.Logf("VERDICT: %d V/D mode 2 frames (%d AMBE+2 frames) decoded", vd2, voiceFrames)
	}
}
