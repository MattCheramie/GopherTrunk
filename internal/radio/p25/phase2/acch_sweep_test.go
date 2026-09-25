//go:build integration

package phase2_test

// Parameter sweep for the Phase 2 traffic-channel receiver, scored against
// SDRtrunk's decode of the same captures.
//
// Every knob on this receiver — loop gain, equalizer, DC block, the watchdog
// idle constant — was tuned while the Gardner timing loop's feedback sign was
// inverted, i.e. against a front end that was walking off the eye. Those
// measurements are void. This re-takes them.
//
// Scoring counts PDUs *confirmed by the reference*, not raw distinct PDUs: the
// ACCH gate is a CRC-12, which false-accepts about one burst in 4096, so raw
// counts reward noise. `false` is the count this decoder produced that
// SDRtrunk did not; `cover` is the fraction of SDRtrunk's PDUs recovered.
//
//	GT_P2_DIR=/path/to/captures GT_P2_MACHEX=/path/to/sdrtrunk-machex \
//	  go test -tags integration ./internal/radio/p25/phase2/ -run ACCHSweep -v

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	p25p2 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2"
	rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
)

type sweepCfg struct {
	name     string
	gain     float64
	eq       bool
	eqTaps   int
	eqMu     float64
	dcBlock  bool
	soft     bool
	idleSF   int     // 0 disables the watchdog
	outRate  float64 // 0 uses 48000
	costasBW float64 // 0 uses the receiver default
}

type sweepScore struct {
	bursts, decoded, rsValid int
	distinct, confirm, bogus int
	oracle                   int
	rotLocks                 [4]int
}

var machexRe = regexp.MustCompile(`bits=[0-9]+ ([0-9A-Fa-f]+)`)

// loadOracles reads SDRtrunk's MACHEX dumps, keyed by the capture tag embedded
// in the filename (e.g. "232" from "232.machex.txt").
func loadOracles(t *testing.T, dir string) map[string]map[string]bool {
	t.Helper()
	out := map[string]map[string]bool{}
	files, _ := filepath.Glob(filepath.Join(dir, "*.machex.txt"))
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		set := map[string]bool{}
		for _, m := range machexRe.FindAllStringSubmatch(string(raw), -1) {
			set[strings.ToUpper(m[1])] = true
		}
		tag := strings.TrimSuffix(filepath.Base(f), ".machex.txt")
		out[tag] = set
	}
	return out
}

// captureTag pulls the trailing sequence number out of a capture filename,
// e.g. "232" from "..._T-Site57_232_baseband.wav" — the key the SDRtrunk
// MACHEX dumps are named by.
func captureTag(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".wav")
	parts := strings.Split(base, "_")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "baseband" || parts[i] == "" {
			continue
		}
		return parts[i]
	}
	return base
}

func runSweep(t *testing.T, files []string, oracles map[string]map[string]bool, c sweepCfg) sweepScore {
	seq := p25p2.ScrambleSequence(framing.PN44SeedFromIdentity(probeWACN, probeSysID, probeNAC))
	outRate := c.outRate
	if outRate == 0 {
		outRate = 48000
	}
	var s sweepScore
	for _, f := range files {
		iq, rate := readWavIQ(t, f)
		dc := ccdecoder.NewDownconverterWithOffset(rate, outRate, 0)
		sfDec := p25p2.NewSuperframeDecoder()
		var wd *p25p2.CarrierWatchdog
		if c.idleSF > 0 {
			wd = p25p2.NewCarrierWatchdog(c.idleSF)
		}
		reacquire := false
		pdus := map[string]bool{}

		harvest := func(nDibits int, sfs []p25p2.Superframe) {
			if wd != nil && wd.Observe(nDibits, len(sfs)) {
				reacquire = true
			}
			for _, sf := range sfs {
				for _, sub := range sf.Subframes {
					if len(sub.Dibits) < p25p2.BurstDibits || !p25p2.BurstTypeOf(sub.Dibits).IsACCH() {
						continue
					}
					s.bursts++
					for phase := 0; phase < 12; phase++ {
						if res, ok := p25p2.DecodeACCHBurst(sub.Dibits, phase, seq); ok {
							s.decoded++
							if res.RSValid {
								s.rsValid++
							}
							pdus[hexOf(res.Message)] = true
							break
						}
					}
				}
			}
		}

		opts := rx.Options{
			SampleRateHz: dc.OutRateHz(), ClockMode: rx.ClockGardner,
			GardnerGain: c.gain, Equalizer: c.eq, EqualizerTaps: c.eqTaps,
			EqualizerMu: c.eqMu, EnableDCBlock: c.dcBlock, CostasLoopBWHz: c.costasBW,
		}
		if c.soft {
			opts.SoftDecision = true
			opts.SoftSink = func(dibits []uint8, soft []complex64, baseIdx int) {
				harvest(len(dibits), sfDec.ProcessSoft(dibits, soft, baseIdx))
			}
		} else {
			opts.DibitSink = func(dibits []uint8, baseIdx int) {
				harvest(len(dibits), sfDec.Process(dibits, baseIdx))
			}
		}
		r := rx.New(opts)
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
			}
		}
		rl := sfDec.RotationLocks()
		for i := range rl {
			s.rotLocks[i] += rl[i]
		}
		oracle := oracles[captureTag(f)]
		s.oracle += len(oracle)
		for h := range pdus {
			s.distinct++
			if oracle[strings.ToUpper(h)] {
				s.confirm++
			} else {
				s.bogus++
			}
		}
	}
	return s
}

func TestACCHSweep(t *testing.T) {
	dir := os.Getenv("GT_P2_DIR")
	machex := os.Getenv("GT_P2_MACHEX")
	if dir == "" || machex == "" {
		t.Skip("set GT_P2_DIR and GT_P2_MACHEX")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.wav"))
	sort.Strings(files)
	if len(files) == 0 {
		t.Skip("no captures")
	}
	oracles := loadOracles(t, machex)

	prod := sweepCfg{name: "BASELINE (production, gain 0.005)", gain: 0.005}
	// Stage 2 re-takes every non-gain knob at the corrected gain, because a
	// knob measured against a too-slow loop is measuring the loop.
	base := sweepCfg{name: "base (gain 0.03)", gain: 0.03}

	cfgs := []sweepCfg{prod, base}
	for _, g := range []float64{0.10, 0.08, 0.05, 0.04, 0.02, 0.01, 0.005, 0.002} {
		c := base
		c.gain, c.name = g, "  gain="+trimF(g)
		cfgs = append(cfgs, c)
	}
	for _, taps := range []int{5, 11, 21} {
		c := base
		c.eq, c.eqTaps, c.name = true, taps, "  equalizer taps="+strconv.Itoa(taps)
		cfgs = append(cfgs, c)
	}
	for _, mu := range []float64{0.01, 0.05, 0.2} {
		c := base
		c.eq, c.eqMu, c.name = true, mu, "  equalizer mu="+trimF(mu)
		cfgs = append(cfgs, c)
	}
	c := base
	c.dcBlock, c.name = true, "  dc_block=on"
	cfgs = append(cfgs, c)
	c = base
	c.soft, c.name = true, "  soft_decision=on"
	cfgs = append(cfgs, c)
	for _, idle := range []int{1, 2, 3} {
		c := base
		c.idleSF, c.name = idle, "  watchdog idle="+strconv.Itoa(idle)+"sf"
		cfgs = append(cfgs, c)
	}
	for _, r := range []float64{24000, 96000} {
		c := base
		c.outRate, c.name = r, "  out_rate="+trimF(r)+" (sps="+strconv.Itoa(int(r/6000))+")"
		cfgs = append(cfgs, c)
	}
	for _, bw := range []float64{30, 60, 100, 250, 400, 600} {
		c := base
		c.costasBW, c.name = bw, "  costas_bw="+trimF(bw)+"Hz"
		cfgs = append(cfgs, c)
	}
	if os.Getenv("GT_SWEEP_ONLY") == "costas" {
		cfgs = cfgs[len(cfgs)-7:] // base is not in the tail; prepend it
		cfgs = append([]sweepCfg{base}, cfgs...)
	}

	t.Logf("%-28s %7s %7s %8s %8s %8s %6s %6s", "config", "bursts", "decoded", "rs_valid", "distinct", "confirm", "false", "cover")
	for _, c := range cfgs {
		s := runSweep(t, files, oracles, c)
		cover := 0.0
		if s.oracle > 0 {
			cover = 100 * float64(s.confirm) / float64(s.oracle)
		}
		t.Logf("%-28s %7d %7d %8d %8d %8d %6d %5.0f%%  rot_locks=%v",
			c.name, s.bursts, s.decoded, s.rsValid, s.distinct, s.confirm, s.bogus, cover, s.rotLocks)
	}
}

func trimF(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }
