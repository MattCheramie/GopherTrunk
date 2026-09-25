//go:build integration

package phase2_test

// Runs the continuous (no-watchdog) ACCH path over a whole directory of Phase
// 2 captures and prints the distinct MAC PDUs recovered from each, one hex per
// line, so the set can be diffed against SDRtrunk's decode of the same file.
// One capture is an anecdote; the corpus is the measurement.
//
//	GT_P2_DIR=/path/to/captures go test -tags integration \
//	  ./internal/radio/p25/phase2/ -run ACCHCorpus -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

func TestACCHCorpus(t *testing.T) {
	dir := os.Getenv("GT_P2_DIR")
	if dir == "" {
		t.Skip("set GT_P2_DIR")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.wav"))
	if err != nil || len(files) == 0 {
		t.Skipf("no wavs in %s", dir)
	}
	sort.Strings(files)
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))

	for _, f := range files {
		iq, rate := readWavIQ(t, f)
		dc := ccdecoder.NewDownconverterWithOffset(rate, 48000, 0)
		sfDec := p25p2.NewSuperframeDecoder()
		pdus := map[string]int{}
		bursts, decoded := 0, 0
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
		for i := 0; i < len(iq); i += chunk {
			end := i + chunk
			if end > len(iq) {
				end = len(iq)
			}
			buf = dc.Process(buf[:0], iq[i:end])
			r.Process(buf)
		}
		base := filepath.Base(f)
		tag := base
		if i := strings.LastIndex(base, "_"); i > 0 {
			if j := strings.LastIndex(base[:i], "_"); j > 0 {
				tag = base[j+1 : i]
			}
		}
		t.Logf("%-8s bursts=%4d decoded=%4d distinct=%3d", tag, bursts, decoded, len(pdus))
		keys := make([]string, 0, len(pdus))
		for k := range pdus {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("CORPUS %s %s\n", tag, k)
		}
	}
}
