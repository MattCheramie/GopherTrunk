package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
)

// TestTETRADMOSeedScan is the #1003 instrument that reads the scramble seed
// straight off the traffic: every error-free DNB of a capture is solved for its
// 30-bit extended colour code (tetra.SolveTCHScrambleSeed — an exact GF(2)
// solve, no candidate search, no MNI assumption), and every CRC-valid DSB's
// SCH/S SYNC PDU and raw SCH/H bits are printed alongside, so the seed of each
// transmission can be correlated with the DM-SYNC fields that announce it.
//
// Run with GT_TETRA_DMO_IQ=<capture> [GT_TETRA_DMO_RATE=…] GT_TETRA_DMO_SEEDS=1.
func TestTETRADMOSeedScan(t *testing.T) {
	if os.Getenv("GT_TETRA_DMO_SEEDS") == "" {
		t.Skip("set GT_TETRA_DMO_SEEDS=1 (with GT_TETRA_DMO_IQ) to run the DMO seed scan")
	}
	bursts, _, _, _, _, _ := loadDMOReplayBursts(t)
	dibitRate := tetrarx.SymbolRate

	seedCounts := map[uint32]int{}
	type seedRun struct {
		seed        uint32
		first, last float64
		n           int
	}
	var runs []seedRun
	solved, dnbs := 0, 0
	for i := range bursts {
		b := bursts[i]
		sec := float64(b.Lead) / dibitRate
		switch b.Kind {
		case tetra.DMBurstSync:
			type1, ok := tetra.DecodeDMSCHS(b)
			schh, hok := tetra.DecodeDMSCHH(b)
			if !ok && !hok {
				continue
			}
			line := fmt.Sprintf("dsb  t=%6.2fs lead=%d rot=%d", sec, b.Lead, b.Rotation)
			if ok {
				pdu, _ := tetra.ParseSyncPDU(type1)
				line += fmt.Sprintf(" schs_ok=1 schs=%s colour=%d TN=%d FN=%d MN=%d MCC=%d MNC=%d",
					hex.EncodeToString(framing.PackBitsMSB(type1)), pdu.ColourCode, pdu.TN, pdu.FN, pdu.MN, pdu.MCC, pdu.MNC)
			} else {
				line += " schs_ok=0"
			}
			if hok {
				line += fmt.Sprintf(" schh_ok=1 schh=%s", hex.EncodeToString(framing.PackBitsMSB(schh)))
			} else {
				line += " schh_ok=0"
			}
			t.Log(line)
		case tetra.DMBurstNormal:
			dnbs++
			seed, ok := tetra.DMBurstScrambleSeed(b)
			if !ok {
				continue
			}
			solved++
			seedCounts[seed]++
			if n := len(runs); n > 0 && runs[n-1].seed == seed && sec-runs[n-1].last < 2 {
				runs[n-1].last = sec
				runs[n-1].n++
			} else {
				runs = append(runs, seedRun{seed: seed, first: sec, last: sec, n: 1})
			}
		}
	}
	t.Logf("dnbs=%d solved_exactly=%d distinct_seeds=%d", dnbs, solved, len(seedCounts))
	type kv struct {
		seed uint32
		n    int
	}
	var kvs []kv
	for s, n := range seedCounts {
		kvs = append(kvs, kv{s, n})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].n > kvs[j].n })
	for _, e := range kvs {
		t.Logf("seed %#010x = %030b  mcc=%d mnc=%d colour=%d  bursts=%d",
			e.seed, e.seed, e.seed>>20, (e.seed>>6)&0x3FFF, e.seed&0x3F, e.n)
	}
	for _, r := range runs {
		t.Logf("run seed=%#010x colour=%2d mcc=%d mnc=%d  %6.2fs .. %6.2fs  bursts=%d",
			r.seed, r.seed&0x3F, r.seed>>20, (r.seed>>6)&0x3FFF, r.first, r.last, r.n)
	}
	seed, votes, ok := tetra.RecoverDMScrambleSeed(bursts)
	t.Logf("RecoverDMScrambleSeed over the whole capture: seed=%#010x votes=%d confident=%v", seed, votes, ok)
}
