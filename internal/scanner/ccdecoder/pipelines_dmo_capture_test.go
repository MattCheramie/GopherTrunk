package ccdecoder

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestTETRADMOPipelineCaptureReplay replays a real DMO IQ capture through the
// PRODUCTION control pipeline (newTETRADMOPipeline — the same receiver, stream
// extractor, slot grid, seed tracker and grant logic the daemon runs) and prints,
// per transmission, when the pipeline locked, learned the seed and GRANTED
// relative to the transmission's first burst — the instrument for the 13 Sep
// operator report "DMO works, but we are eating the first seconds of every
// transmission". The voice chain only receives IQ from the grant onward, so every
// CRC-valid TCH/S burst that precedes the grant is speech the daemon can never
// record; the harness counts those ("lost_before_grant") by running the
// DMSeedTracker over the pipeline's own burst stream as an ideal chain that saw
// everything would.
//
// It also audits the per-burst exact seed solver over EVERY DNB (qualified or
// not — the voice chain solves them all): a solve that disagrees with the
// transmission's seed is a false solve, the mechanism behind the 13 Sep
// "scramble seed changed … changed back" flip-flops. GT_TETRA_DMO_DUMP=<dir>
// writes each false-solving burst as a gob fixture so it can be pinned as a
// literal regression vector (tetra/testdata).
//
// Skip unless GT_TETRA_DMO_IQ names a capture: a wav/flac container (rate from
// the header) or headerless cs16 at GT_TETRA_DMO_RATE. Run:
//
//	GT_TETRA_DMO_IQ=<capture.flac> go test ./internal/scanner/ccdecoder -run TestTETRADMOPipelineCaptureReplay -v
func TestTETRADMOPipelineCaptureReplay(t *testing.T) {
	path := os.Getenv("GT_TETRA_DMO_IQ")
	if path == "" {
		t.Skip("set GT_TETRA_DMO_IQ to a DMO IQ capture (flac/wav/cs16) to run the pipeline replay")
	}
	iq, inRate := loadDMOCaptureIQ(t, path)
	if inRate != dmoTestSampleRate {
		ddc := NewDownconverter(inRate, dmoTestSampleRate)
		iq = ddc.Process(nil, iq)
	}
	t.Logf("capture: %s  in_rate=%.0f Hz  samples=%d (%.1f s at %.0f Hz)", path, inRate, len(iq), float64(len(iq))/dmoTestSampleRate, dmoTestSampleRate)

	bus := events.NewBus(1024)
	clock := time.Unix(1_760_000_000, 0)
	epoch := clock
	p := dmoTestPipeline(t, bus, &clock)
	p.debug = false

	type grantAt struct{ t float64 }
	var grants []grantAt
	sub := bus.Subscribe()
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for ev := range sub.C {
			if ev.Kind == events.KindGrant {
				if g, ok := ev.Payload.(trunking.Grant); ok {
					grants = append(grants, grantAt{g.At.Sub(epoch).Seconds()})
				}
			}
		}
	}()

	// Wrap the extractor callback to observe every burst with the pipeline's
	// verdicts on it (qualified? seed adopted? grant fired?). Stream time is the
	// burst's lead index at the 18 kdibit/s symbol rate — exact, unlike the
	// chunk-granular pipeline clock.
	type rec struct {
		b         tetra.DMBurst
		t         float64
		qualified bool
		lockedNow bool
		seedNow   bool
		seed      uint32
		grantNow  bool
	}
	var recs []rec
	p.ext = tetra.NewDMStreamExtractor(func(b tetra.DMBurst) {
		q0, l0, s0, g0 := p.dnbQualified, p.locked, p.colourKnown, p.grantActive
		p.onBurst(b)
		recs = append(recs, rec{
			b: b, t: float64(b.Lead) / 18000.0,
			qualified: p.dnbQualified > q0,
			lockedNow: !l0 && p.locked,
			seedNow:   !s0 && p.colourKnown,
			seed:      p.colour,
			grantNow:  !g0 && p.grantActive,
		})
	})

	// SoapyRemote-sized chunks so the pipeline's wall clock (which drives the
	// re-arm drought) advances at a realistic granularity.
	const chunk = 512
	for i := 0; i < len(iq); i += chunk {
		e := i + chunk
		if e > len(iq) {
			e = len(iq)
		}
		clock = epoch.Add(time.Duration(float64(e) / dmoTestSampleRate * float64(time.Second)))
		p.Process(iq[i:e])
	}
	p.ext.Flush()
	bus.Close()
	<-drained

	// Group into transmissions: a run of qualified DNBs separated by more than
	// the re-arm drought from the next.
	type tx struct {
		firstDSB, firstQual, lastQual, seedAt, grantAt float64
		seed                                           uint32
		qualified, crcTotal, crcBeforeGrant            int
		firstCRC                                       float64
		falseSolves, falseSolvesCRCPass                int
	}
	var txs []tx
	var cur *tx
	lastDSB := -1.0
	// An ideal voice chain that saw every burst: the production tracker over
	// the pipeline's whole stream, in order.
	ideal := tetra.NewDMSeedTracker()
	var dumped int
	dumpDir := os.Getenv("GT_TETRA_DMO_DUMP")
	for _, r := range recs {
		if r.b.Kind == tetra.DMBurstSync {
			ideal.ObserveDSB(r.b)
			if cur == nil || r.t-cur.lastQual > dmoGrantRearm.Seconds() {
				if lastDSB < 0 || r.t-lastDSB > dmoGrantRearm.Seconds() {
					lastDSB = r.t
				}
			}
			continue
		}
		if r.qualified {
			if cur == nil || r.t-cur.lastQual > dmoGrantRearm.Seconds() {
				txs = append(txs, tx{firstDSB: lastDSB, firstQual: r.t, seedAt: -1, grantAt: -1, firstCRC: -1})
				cur = &txs[len(txs)-1]
				lastDSB = -1
			}
			cur.lastQual = r.t
			cur.qualified++
			if r.seedNow && cur.seedAt < 0 {
				cur.seedAt, cur.seed = r.t, r.seed
			}
			if r.grantNow && cur.grantAt < 0 {
				cur.grantAt = r.t
			}
		}
		frames, _, _ := ideal.ObserveDNB(r.b, r.qualified)
		if cur != nil && len(frames) == 2 {
			cur.crcTotal++
			if cur.firstCRC < 0 {
				cur.firstCRC = r.t
			}
			if cur.grantAt < 0 || r.t < cur.grantAt {
				cur.crcBeforeGrant++
			}
		}
	}
	// False-solve audit against each transmission's pipeline-recovered seed.
	for i := range txs {
		x := &txs[i]
		if x.seedAt < 0 {
			continue
		}
		for _, r := range recs {
			if r.b.Kind != tetra.DMBurstNormal || r.t < x.firstQual-1 || r.t > x.lastQual+1 {
				continue
			}
			s, ok := tetra.DMBurstScrambleSeed(r.b)
			if !ok || s == x.seed {
				continue
			}
			x.falseSolves++
			crc := len(tetra.DMBurstTCHSpeechSoft(r.b, s)) == 2 || len(tetra.DMBurstTCHSpeech(r.b, s)) == 2
			if crc {
				x.falseSolvesCRCPass++
			}
			t.Logf("  false solve: t=%.3fs qualified=%v solved=%#010x (tx seed %#010x) crc_at_solved=%v", r.t, r.qualified, s, x.seed, crc)
			if dumpDir != "" {
				name := filepath.Join(dumpDir, fmt.Sprintf("false_solve_%02d_%#010x.gob", dumped, s))
				if err := dumpDMBurst(name, r.b); err != nil {
					t.Fatalf("dump %s: %v", name, err)
				}
				dumped++
			}
		}
	}

	t.Logf("pipeline: dsb=%d/%d crc, dnb_total=%d dnb_qualified=%d tch_crc=%d grants=%d",
		p.dsbCRC, p.dsbTotal, p.dnbTotal, p.dnbQualified, p.tchCRC, len(grants))
	lostBursts, lostSecs := 0, 0.0
	for i, x := range txs {
		lost := ""
		if x.grantAt >= 0 && x.firstCRC >= 0 && x.grantAt > x.firstCRC {
			lostSecs += x.grantAt - x.firstCRC
			lost = fmt.Sprintf("  LOST before grant: %d CRC-valid bursts, %.2f s of speech", x.crcBeforeGrant, x.grantAt-x.firstCRC)
		}
		lostBursts += x.crcBeforeGrant
		t.Logf("tx %d: first_dsb=%.2fs first_qualified=%.2fs (+%.2f) seed=%#010x at +%.2f grant at +%.2f (%.2f s after first CRC-valid burst) qualified=%d crc=%d false_solves=%d (crc-pass %d)%s",
			i+1, x.firstDSB, x.firstQual, x.firstQual-x.firstDSB, x.seed,
			rel(x.seedAt, x.firstDSB), rel(x.grantAt, x.firstDSB), rel(x.grantAt, x.firstCRC),
			x.qualified, x.crcTotal, x.falseSolves, x.falseSolvesCRCPass, lost)
	}
	t.Logf("SUMMARY: %d transmissions, %d grants, %d CRC-valid bursts (%.2f s) lost before the grant",
		len(txs), len(grants), lostBursts, lostSecs)

	// Phase 2 — what a VOICE CHAIN actually recovers. The chain starts a cold
	// receiver (Gardner / AFC / CMA equalizer all from scratch) on IQ that begins
	// at the grant, so the pipeline's own burst count from the grant onward is an
	// upper bound the chain cannot reach while its receiver acquires. Simulate the
	// chain (the shared DMO receiver + extractor + tracker seeded with the grant's
	// seed, as runTETRADMOVoiceChain is) starting at grant − preroll for several
	// pre-roll lengths and count the CRC-valid TCH/S bursts it yields per
	// transmission. Also count the transmission's decodable bursts that precede
	// the first slot-grid-qualified one (decoded post hoc at the recovered seed):
	// speech no grant-time start can ever include without a pre-roll.
	prerolls := []float64{0, 0.5, 1, 2, 3}
	for i := range txs {
		x := &txs[i]
		if x.grantAt < 0 {
			continue
		}
		preLatch := 0
		for _, r := range recs {
			if r.b.Kind == tetra.DMBurstNormal && r.t >= x.firstDSB && r.t < x.firstQual &&
				(len(tetra.DMBurstTCHSpeechSoft(r.b, x.seed)) == 2 || len(tetra.DMBurstTCHSpeech(r.b, x.seed)) == 2) {
				preLatch++
			}
		}
		end := x.lastQual + 0.5
		if i+1 < len(txs) {
			end = txs[i+1].firstDSB
		}
		line := fmt.Sprintf("tx %d: decodable before grid latch=%d; voice chain CRC-valid bursts by pre-roll:", i+1, preLatch)
		for _, pr := range prerolls {
			start := x.grantAt - pr
			if start < 0 {
				start = 0
			}
			n := simulateDMOVoiceChain(iq, start, end, x.seed)
			line += fmt.Sprintf("  %.1fs→%d", pr, n)
		}
		t.Log(line + fmt.Sprintf("  (pipeline saw %d from its warm receiver)", x.crcTotal))
	}
	if v := os.Getenv("GT_TETRA_DMO_MAX_LOST_SECONDS"); v != "" {
		max, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatal(err)
		}
		if lostSecs > max {
			t.Errorf("%.2f s of speech lost before the grant across the capture, over the %.2f s bound", lostSecs, max)
		}
	}
}

func rel(t, base float64) float64 {
	if t < 0 || base < 0 {
		return -1
	}
	return t - base
}

// loadDMOCaptureIQ reads a DMO IQ capture: a flac container (baseband
// ReadIQFLACSamples, rate from STREAMINFO) or headerless cs16 at
// GT_TETRA_DMO_RATE (default 144000).
func loadDMOCaptureIQ(t *testing.T, path string) ([]complex64, float64) {
	t.Helper()
	if baseband.IsFLACIQFile(path) {
		iq, rate, err := baseband.ReadIQFLACSamples(path)
		if err != nil {
			t.Fatal(err)
		}
		return iq, float64(rate)
	}
	rate := 144000.0
	if v := os.Getenv("GT_TETRA_DMO_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatal(err)
		}
		rate = f
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq, rate
}

// dumpDMBurst writes one burst as a gob file (a literal on-air vector).
func dumpDMBurst(path string, b tetra.DMBurst) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(b)
}

// simulateDMOVoiceChain runs a cold DMO receiver (tetrarx.DMOOptions, as the
// composer's runTETRADMOVoiceChain builds it) + stream extractor + seed tracker
// primed with the grant's seed over iq[startSec, endSec) and returns the number
// of DNBs that yield CRC-valid TCH/S.
func simulateDMOVoiceChain(iq []complex64, startSec, endSec float64, seed uint32) int {
	s0, s1 := int(startSec*dmoTestSampleRate), int(endSec*dmoTestSampleRate)
	if s1 > len(iq) {
		s1 = len(iq)
	}
	if s0 >= s1 {
		return 0
	}
	tr := tetra.NewDMSeedTracker()
	tr.Adopt(seed)
	grid := tetra.NewDMSlotGrid()
	crc := 0
	ext := tetra.NewDMStreamExtractor(func(b tetra.DMBurst) {
		switch b.Kind {
		case tetra.DMBurstSync:
			tr.ObserveDSB(b)
		case tetra.DMBurstNormal:
			q := grid.Observe(b.Lead)
			if frames, _, _ := tr.ObserveDNB(b, q); len(frames) == 2 {
				crc++
			}
		}
	})
	var pending []complex64
	opts := tetrarx.DMOOptions(dmoTestSampleRate)
	opts.DibitSink = func(d []uint8, base int) { ext.Process(d, pending, base); pending = nil }
	opts.SoftSink = func(diffs []complex64, _ int) { pending = diffs }
	rx := tetrarx.New(opts)
	const chunk = 512
	for i := s0; i < s1; i += chunk {
		e := i + chunk
		if e > s1 {
			e = s1
		}
		rx.Process(iq[i:e])
	}
	ext.Flush()
	return crc
}
