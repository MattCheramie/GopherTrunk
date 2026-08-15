package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/tetra"
	tetrarx "github.com/MattCheramie/GopherTrunk/internal/radio/tetra/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/voice/acelp"
)

// TestTETRADMOReplay is the real-air validation gate for TETRA Direct Mode
// Operation (DMO) voice decode — the acceptance criterion of issue #1003. Skip
// unless GT_TETRA_DMO_IQ points at a cs16 (interleaved int16) IQ file of a DMO
// transmission; GT_TETRA_DMO_RATE gives its sample rate (default 150000, the
// reporter's Tetra_DMO_Two_TX_cs16_30sec_bw150.raw), GT_TETRA_DMO_COLOUR pins the
// DM colour code the TCH/S traffic is scrambled with — but when it is UNSET the
// harness now auto-recovers the colour via tetra.RecoverDMColourCode (the colour
// that maximises CRC-valid TCH/S), so a clear DMO call decodes with no manual
// override (on the 10aug capture it recovers colour 3). GT_TETRA_DMO_OUT
// optionally names a dir for the decoded WAV.
//
// It resamples to the 144 kHz TETRA channel rate, runs the shared π/4-DQPSK
// receiver, accumulates the whole dibit stream, then:
//   - ExtractDMBursts (dmo.go) slices every DSB / DNB (EN 300 396-2 Tables 15/16);
//   - each DSB's SCH/S is channel-decoded with colour 0 (§8.2.5.2) — a CRC-valid
//     SCH/S is the DMO sync/lock indicator, marking a PTT/transmission start;
//   - each DNB's TCH/S is descrambled + decoded (dmo_decode.go) into 137-bit
//     speech frames, gated on the class-2 CRC exactly like the TMO path —
//     soft-decision (DMBurstTCHSpeechSoft, off the receiver's differentials) with
//     a hard fallback, the ~2× same-carrier yield lever from #1001;
//   - the speech frames feed the clean-room ACELP vocoder to PCM.
//
// The pass signal is CRC-valid speech clustered in time on the transmissions
// (versus the ~1/256 chance floor a wrong scramble/geometry would give — the
// symptom the #1003 investigation measured before the spec constants were
// sourced). It prints a per-transmission timeline and the recovered PCM duration
// so the maintainer can A/B it and confirm the audio is intelligible.
func TestTETRADMOReplay(t *testing.T) {
	var colour uint32
	colourSet := false
	if v := os.Getenv("GT_TETRA_DMO_COLOUR"); v != "" {
		c, err := strconv.ParseUint(v, 0, 32)
		if err != nil {
			t.Fatalf("bad GT_TETRA_DMO_COLOUR: %v", err)
		}
		colour = uint32(c)
		colourSet = true
	}

	bursts, inRate, outRate, iqLen, enableEQ, enableLMS := loadDMOReplayBursts(t)
	dibitRate := tetrarx.SymbolRate // 18000 dibits/sec

	// Recover the DM colour code the TCH/S is scrambled with when it was not
	// pinned via GT_TETRA_DMO_COLOUR (#1003). The DSB SCH/S is always colour-0
	// scrambled and can't reveal the traffic colour, so RecoverDMColourCode
	// learns it by maximising CRC-valid TCH/S. On the 10aug clear capture this
	// auto-selects colour 3 (the value that lifts TCH/S off the chance floor)
	// with no manual override — the C1 acceptance signal.
	if !colourSet {
		if rc, n, ok := tetra.RecoverDMColourCode(bursts); ok {
			colour = rc
			t.Logf("recovered DM colour code=%d (crc-valid TCH/S=%d) — no GT_TETRA_DMO_COLOUR set", rc, n)
		} else {
			t.Logf("DM colour code not recovered (no colour cleared the chance floor); using default %d", colour)
		}
	}

	var dsbTotal, dsbCRC, dnbTotal, tchCRC, tchSoftOnly int
	var syncSecs []int           // seconds carrying a CRC-valid SCH/S (transmission starts)
	speechBySec := map[int]int{} // CRC-valid TCH/S bursts per second
	var speechFrames [][]byte    // ordered speech frames for the vocoder
	seenSyncSec := map[int]struct{}{}
	// SYNC-PDU state read from the CRC-valid SCH/S. The DMO DM-SYNC PDU reuses the
	// TMO SYNC-PDU field layout (osmo-tetra-dmo does the same), so ParseSyncPDU
	// decodes it: colour@4-9, TN@10-11, FN@12-16, MN@17-22. A monotonically
	// advancing frame counter across the transmission is the proof the sync is
	// real (not a lone chance-CRC hit).
	var firstSync tetra.SyncPDU
	haveSync := false
	fnSeen := map[uint8]struct{}{}

	for i := range bursts {
		b := bursts[i]
		sec := int(float64(b.Lead) / dibitRate)
		switch b.Kind {
		case tetra.DMBurstSync:
			dsbTotal++
			if type1, ok := tetra.DecodeDMSCHS(b); ok {
				dsbCRC++
				if pdu, pok := tetra.ParseSyncPDU(type1); pok {
					fnSeen[pdu.FN] = struct{}{}
					if !haveSync {
						firstSync, haveSync = pdu, true
					}
				}
				if _, dup := seenSyncSec[sec]; !dup {
					seenSyncSec[sec] = struct{}{}
					syncSecs = append(syncSecs, sec)
				}
			}
		case tetra.DMBurstNormal:
			dnbTotal++
			// Prefer soft-decision; fall back to hard when no differentials were
			// carried for this burst (e.g. an edge burst).
			frames := tetra.DMBurstTCHSpeechSoft(b, colour)
			if len(frames) != 2 {
				if hard := tetra.DMBurstTCHSpeech(b, colour); len(hard) == 2 {
					frames = hard
				}
			} else if tetra.DMBurstTCHSpeech(b, colour) == nil {
				tchSoftOnly++ // recovered by soft-decision that the hard path missed
			}
			if len(frames) == 2 {
				tchCRC++
				speechBySec[sec]++
				speechFrames = append(speechFrames, frames...)
			}
		}
	}

	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs colour=%#x eq=%v lms=%v",
		inRate, outRate, iqLen, float64(iqLen)/inRate, colour, enableEQ, enableLMS)
	t.Logf("bursts: dsb_total=%d dsb_schs_crc=%d dnb_total=%d tch_crc=%d (soft_only=%d)",
		dsbTotal, dsbCRC, dnbTotal, tchCRC, tchSoftOnly)
	sort.Ints(syncSecs)
	t.Logf("sync (SCH/S CRC-valid) at seconds: %v", syncSecs)
	if haveSync {
		// The SYNC PDU decoded from the (colour-0-scrambled) SCH/S. distinct_fn > 1
		// proves the frame counter advances across the transmission — a genuine DMO
		// lock, not a chance CRC.
		t.Logf("SYNC PDU: colour=%d TN=%d FN=%d MN=%d MCC=%d MNC=%d (distinct FN across DSBs=%d)",
			firstSync.ColourCode, firstSync.TN, firstSync.FN, firstSync.MN,
			firstSync.MCC, firstSync.MNC, len(fnSeen))
	}

	// Signalling-vs-traffic verdict. DMO SCH/S is scrambled with colour 0 (like the
	// TMO BSCH) and decodes clear whenever the RF is receivable; TCH/S traffic is
	// scrambled with the DM colour code (now descrambled UNCONDITIONALLY, incl. at
	// colour 0 — the issue #1003 fix in dmo_decode.go) and, only if the call is
	// protected, additionally air-interface encrypted.
	//
	// A capture that decodes SCH/S well (dsb_schs_crc high, distinct FN) but yields
	// TCH/S CRC only near the ~1/256 chance floor has TWO possible causes, no longer
	// just one:
	//   - the call is genuinely air-interface ENCRYPTED, or
	//   - a remaining clear-voice DECODE defect (geometry / interleave / colour code).
	// The reporter confirmed their radios are TEA0 (clear, colour 0), and the
	// colour-0 descramble asymmetry that produced this exact chance-floor signature
	// in #1003 is now fixed. So on a capture KNOWN to be clear, a persistent chance
	// floor is a decode bug to keep chasing — NOT proof of encryption. Set
	// GT_TETRA_DMO_CLEAR=1 to assert the capture is clear and flip the verdict.
	if dsbCRC >= 8 && dnbTotal > 0 && tchCRC*20 < dnbTotal {
		if os.Getenv("GT_TETRA_DMO_CLEAR") == "1" {
			t.Logf("VERDICT: DMO SIGNALLING decodes (dsb_schs_crc=%d, distinct FN=%d) but TCH/S is at the chance floor (tch_crc=%d/%d) on a capture asserted CLEAR (TEA0) — this is a clear-voice DECODE defect to keep chasing (geometry/interleave/colour), NOT encryption. The colour-0 descramble is already fixed (#1003); next suspects are DNB geometry and the DM colour code (GT_TETRA_DMO_COLOUR).",
				dsbCRC, len(fnSeen), tchCRC, dnbTotal)
		} else {
			t.Logf("VERDICT: DMO SIGNALLING decodes (dsb_schs_crc=%d, distinct FN=%d) but TCH/S is at the chance floor (tch_crc=%d/%d) — either air-interface ENCRYPTED voice or a remaining clear-voice decode defect. If the call is known CLEAR (TEA0), set GT_TETRA_DMO_CLEAR=1; otherwise validate against a known-CLEAR DMO call.",
				dsbCRC, len(fnSeen), tchCRC, dnbTotal)
		}
	}

	// Seconds with >=2 CRC-valid speech bursts = real activity, not a lone
	// chance-CRC hit. On the reporter's capture these should cluster on the two
	// transmissions (~0.9-13s and ~15.2-25.7s).
	var speechSecs []int
	for s, n := range speechBySec {
		if n >= 2 {
			speechSecs = append(speechSecs, s)
		}
	}
	sort.Ints(speechSecs)
	t.Logf("speech active seconds (>=2 CRC bursts): %v", speechSecs)

	if len(speechFrames) == 0 {
		t.Logf("no CRC-valid TCH/S speech recovered — if the capture is a known-good DMO voice call, try GT_TETRA_DMO_COLOUR (the DM colour code), GT_TETRA_DMO_EQ=1 (blind CMA) or GT_TETRA_DMO_LMS=1 (training-sequence equalizer)")
		return
	}

	voc := acelp.NewVocoder()
	var pcm []int16
	for _, sf := range speechFrames {
		out, _ := voc.Decode(sf)
		pcm = append(pcm, out...)
	}
	var peak int16
	for _, v := range pcm {
		if v > peak {
			peak = v
		} else if -v > peak {
			peak = -v
		}
	}
	t.Logf("decoded speech_frames=%d pcm=%.1fs peak=%d", len(speechFrames), float64(len(pcm))/8000, peak)

	if outDir := os.Getenv("GT_TETRA_DMO_OUT"); outDir != "" {
		writeWav8kLocal(outDir+"/dmo.wav", pcm)
		t.Logf("wrote %s/dmo.wav", outDir)
	}
}

// loadDMOReplayBursts reads GT_TETRA_DMO_IQ (cs16, GT_TETRA_DMO_RATE), resamples
// to the 144 kHz TETRA channel rate, runs the shared receiver (GT_TETRA_DMO_EQ /
// GT_TETRA_DMO_LMS as in TestTETRADMOReplay) and returns every extracted DM
// burst with soft info, plus the run parameters for logging. Skips the calling
// test when no capture is configured.
func loadDMOReplayBursts(t *testing.T) (bursts []tetra.DMBurst, inRate, outRate float64, iqLen int, enableEQ, enableLMS bool) {
	t.Helper()
	path := os.Getenv("GT_TETRA_DMO_IQ")
	if path == "" {
		t.Skip("set GT_TETRA_DMO_IQ (cs16 IQ) + GT_TETRA_DMO_RATE to run the DMO replay")
	}
	inRate = 150000.0
	if v := os.Getenv("GT_TETRA_DMO_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_TETRA_DMO_RATE: %v", err)
		}
		inRate = f
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
	iqLen = len(iq)

	ddc := ccdecoder.NewDownconverter(inRate, 144000)
	outRate = ddc.OutRateHz()

	// Accumulate the full demodulated dibit stream and the parallel per-symbol
	// differentials (soft info); ExtractDMBurstsSoft is stateless over a complete
	// slice, which keeps the offline framing deterministic. The SoftSink fires
	// just before the matching DibitSink with the same base, so appending both in
	// order keeps allDiffs strictly parallel to allDibits.
	var allDibits []uint8
	var allDiffs []complex64
	var allSymbols []complex64
	// The receiver blind-CMA equalizer is REQUIRED, not optional, for DMO: on the
	// reporter's 438.9 MHz capture it lifts CRC-valid SCH/S from 6 → 64 (the whole
	// transmission) by inverting the ISI/multipath that smears the π/4-DQPSK
	// constellation — the same lever the live TMO CC path already runs by default
	// (pipelines.go newTETRAPipeline). So it is ON by default here; set
	// GT_TETRA_DMO_EQ=0 to A/B the raw (un-equalized) receiver.
	enableEQ = os.Getenv("GT_TETRA_DMO_EQ") != "0" && os.Getenv("GT_TETRA_DMO_EQ") != "false"
	enableLMS = os.Getenv("GT_TETRA_DMO_LMS") == "1" || os.Getenv("GT_TETRA_DMO_LMS") == "true"
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: outRate,
		DibitSink: func(d []uint8, _ int) {
			allDibits = append(allDibits, d...)
		},
		SoftSink: func(diffs []complex64, _ int) {
			allDiffs = append(allDiffs, diffs...)
		},
		// Raw symbols for the per-burst training-sequence equalizer (GT_TETRA_DMO_LMS).
		SymbolSink: func(syms []complex64, _ int) {
			allSymbols = append(allSymbols, syms...)
		},
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     enableEQ,
	})

	const chunk = 65536
	var scratch []complex64
	for i := 0; i < len(iq); i += chunk {
		e := i + chunk
		if e > len(iq) {
			e = len(iq)
		}
		scratch = ddc.Process(scratch[:0], iq[i:e])
		rx.Process(scratch)
	}

	// Soft-capable extraction: carry the differentials so DMBurstTCHSpeechSoft can
	// recover corrupted TCH/S bursts the hard path drops (the #1001 ~2× lever).
	// GT_TETRA_DMO_LMS=1 additionally trains a per-burst equalizer on each
	// rotation-0 DNB's midamble and re-derives the soft blocks from the equalized
	// symbols (#1001's LMS lever for Direct Mode).
	if enableLMS {
		bursts = tetra.ExtractDMBurstsEqualized(allDibits, allDiffs, allSymbols, 0, 0, 0)
	} else {
		bursts = tetra.ExtractDMBurstsSoft(allDibits, allDiffs, 0)
	}
	return bursts, inRate, outRate, iqLen, enableEQ, enableLMS
}

// TestTETRADMOColourScan is the #1003 descramble-model diagnostic. The 15aug
// capture (25 s of clear, colour-0 silent PTT) decodes SCH/S at ~90% yet TCH/S
// sits near the chance floor at EVERY single colour — and a 64-colour sweep
// shows several colours rising modestly above the floor at once (28/57/30),
// which one radio scrambling with one label cannot produce. That signature
// means the DM descramble model is missing a per-burst component. This scan
// maps, for every DNB: which colours CRC-decode it, its slot-grid position,
// and its frame number estimated from the CRC-valid DSBs' SYNC PDU FN — so the
// burst→colour map reveals the rule (FN-dependent seed, cross-burst pairing,
// …) instead of guessing. It also tries the cross-burst pairing hypothesis
// (BKN2 of burst k + BKN1 of burst k+1 as one slot) across all colours.
//
// Diagnostic only: it asserts nothing and prints the map for analysis.
func TestTETRADMOColourScan(t *testing.T) {
	if os.Getenv("GT_TETRA_DMO_SCAN") == "" {
		t.Skip("set GT_TETRA_DMO_SCAN=1 (plus GT_TETRA_DMO_IQ/RATE) to run the colour-map scan")
	}
	bursts, _, _, _, _, _ := loadDMOReplayBursts(t)

	// FN anchors from CRC-valid DSBs: (lead, FN).
	type anchor struct {
		lead int
		fn   int
	}
	var anchors []anchor
	for i := range bursts {
		b := bursts[i]
		if b.Kind != tetra.DMBurstSync {
			continue
		}
		if type1, ok := tetra.DecodeDMSCHS(b); ok {
			if pdu, pok := tetra.ParseSyncPDU(type1); pok {
				anchors = append(anchors, anchor{lead: b.Lead, fn: int(pdu.FN)})
			}
		}
	}
	// estFN estimates a burst's TETRA frame number (1..18) from the nearest
	// anchor, assuming one occupied slot per frame (4 slots × 255 dibits between
	// consecutive frames — the spacing consecutive DNBs of a DMO call show).
	const frameDibits = 4 * 255
	estFN := func(lead int) int {
		if len(anchors) == 0 {
			return -1
		}
		best := anchors[0]
		for _, a := range anchors[1:] {
			if iabsDMO(lead-a.lead) < iabsDMO(lead-best.lead) {
				best = a
			}
		}
		off := int(math.Round(float64(lead-best.lead) / frameDibits))
		fn := (best.fn - 1 + off) % 18
		for fn < 0 {
			fn += 18
		}
		return fn + 1
	}

	decodesAt := func(b tetra.DMBurst, c uint32) bool {
		return len(tetra.DMBurstTCHSpeechSoft(b, c)) == 2 || len(tetra.DMBurstTCHSpeech(b, c)) == 2
	}

	// Per-burst colour map + aggregations.
	var dnbs []tetra.DMBurst
	for i := range bursts {
		if bursts[i].Kind == tetra.DMBurstNormal {
			dnbs = append(dnbs, bursts[i])
		}
	}
	colourTotals := map[int]int{}
	fnColour := map[[2]int]int{} // {fn, colour} -> count
	deltaHist := map[int]int{}   // consecutive-DNB lead deltas (slot units ×255)
	multi := 0
	prevLead := -1
	for _, b := range dnbs {
		if prevLead >= 0 {
			deltaHist[b.Lead-prevLead]++
		}
		prevLead = b.Lead
		var wins []int
		for c := 0; c < 64; c++ {
			if decodesAt(b, uint32(c)) {
				wins = append(wins, c)
			}
		}
		if len(wins) > 1 {
			multi++
		}
		fn := estFN(b.Lead)
		for _, c := range wins {
			colourTotals[c]++
			fnColour[[2]int{fn, c}]++
		}
		if len(wins) > 0 {
			t.Logf("dnb lead=%d sec=%.1f rot=%d fn=%d colours=%v",
				b.Lead, float64(b.Lead)/tetrarx.SymbolRate, b.Rotation, fn, wins)
		}
	}
	t.Logf("dnbs=%d multi-colour bursts=%d anchors=%d", len(dnbs), multi, len(anchors))

	type kv struct{ k, v int }
	var totals []kv
	for c, n := range colourTotals {
		totals = append(totals, kv{c, n})
	}
	sort.Slice(totals, func(i, j int) bool { return totals[i].v > totals[j].v })
	if len(totals) > 12 {
		totals = totals[:12]
	}
	t.Logf("top colour totals: %v", totals)
	var fnMap []string
	for k, n := range fnColour {
		if n >= 3 {
			fnMap = append(fnMap, fmt.Sprintf("fn%02d:c%02d=%d", k[0], k[1], n))
		}
	}
	sort.Strings(fnMap)
	t.Logf("fn→colour (n>=3): %v", fnMap)
	var deltas []kv
	for d, n := range deltaHist {
		deltas = append(deltas, kv{d, n})
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].v > deltas[j].v })
	if len(deltas) > 8 {
		deltas = deltas[:8]
	}
	t.Logf("top DNB lead deltas (dibits): %v", deltas)

	// Cross-burst pairing hypothesis: decode BKN2 of burst k + BKN1 of burst k+1
	// as one 432-bit slot, for consecutive same-rotation DNBs one slot-multiple
	// apart, across all colours.
	pairTotals := map[int]int{}
	pairs := 0
	for i := 0; i+1 < len(dnbs); i++ {
		a, b := dnbs[i], dnbs[i+1]
		gap := b.Lead - a.Lead
		if a.Rotation != b.Rotation || gap <= 0 || gap > 8*255 || gap%255 != 0 {
			continue
		}
		pairs++
		pb := tetra.DMBurst{
			Kind: tetra.DMBurstNormal, Lead: a.Lead, Rotation: a.Rotation,
			BKN1: a.BKN2, BKN2: b.BKN1,
			SoftBKN1: a.SoftBKN2, SoftBKN2: b.SoftBKN1,
		}
		for c := 0; c < 64; c++ {
			if decodesAt(pb, uint32(c)) {
				pairTotals[c]++
			}
		}
	}
	var ptotals []kv
	for c, n := range pairTotals {
		ptotals = append(ptotals, kv{c, n})
	}
	sort.Slice(ptotals, func(i, j int) bool { return ptotals[i].v > ptotals[j].v })
	if len(ptotals) > 12 {
		ptotals = ptotals[:12]
	}
	t.Logf("cross-burst pairing: pairs=%d top colour totals: %v", pairs, ptotals)

	// Swapped-block hypothesis: BKN2+BKN1 of the SAME burst as one slot.
	swapTotals := map[int]int{}
	for _, b := range dnbs {
		sb := tetra.DMBurst{
			Kind: tetra.DMBurstNormal, Lead: b.Lead, Rotation: b.Rotation,
			BKN1: b.BKN2, BKN2: b.BKN1,
			SoftBKN1: b.SoftBKN2, SoftBKN2: b.SoftBKN1,
		}
		for c := 0; c < 64; c++ {
			if decodesAt(sb, uint32(c)) {
				swapTotals[c]++
			}
		}
	}
	var stotals []kv
	for c, n := range swapTotals {
		stotals = append(stotals, kv{c, n})
	}
	sort.Slice(stotals, func(i, j int) bool { return stotals[i].v > stotals[j].v })
	if len(stotals) > 6 {
		stotals = stotals[:6]
	}
	t.Logf("same-burst swapped blocks: top colour totals: %v", stotals)
}

func iabsDMO(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
