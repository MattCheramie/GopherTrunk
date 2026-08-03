package main

import (
	"encoding/binary"
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
// reporter's Tetra_DMO_Two_TX_cs16_30sec_bw150.raw), GT_TETRA_DMO_COLOUR sets the
// DM colour code the TCH/S traffic is scrambled with (default 0 — the spec
// default for DSB signalling and the common radio-to-radio value; the SYNC-PDU
// parser that would learn it live is EN 300 396-3, a follow-up), and
// GT_TETRA_DMO_OUT optionally names a dir for the decoded WAV.
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
	path := os.Getenv("GT_TETRA_DMO_IQ")
	if path == "" {
		t.Skip("set GT_TETRA_DMO_IQ (cs16 IQ) + GT_TETRA_DMO_RATE to run the DMO replay")
	}
	inRate := 150000.0
	if v := os.Getenv("GT_TETRA_DMO_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_TETRA_DMO_RATE: %v", err)
		}
		inRate = f
	}
	var colour uint32
	if v := os.Getenv("GT_TETRA_DMO_COLOUR"); v != "" {
		c, err := strconv.ParseUint(v, 0, 32)
		if err != nil {
			t.Fatalf("bad GT_TETRA_DMO_COLOUR: %v", err)
		}
		colour = uint32(c)
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

	ddc := ccdecoder.NewDownconverter(inRate, 144000)
	outRate := ddc.OutRateHz()

	// Accumulate the full demodulated dibit stream and the parallel per-symbol
	// differentials (soft info); ExtractDMBurstsSoft is stateless over a complete
	// slice, which keeps the offline framing deterministic. The SoftSink fires
	// just before the matching DibitSink with the same base, so appending both in
	// order keeps allDiffs strictly parallel to allDibits.
	var allDibits []uint8
	var allDiffs []complex64
	enableEQ := os.Getenv("GT_TETRA_DMO_EQ") == "1" || os.Getenv("GT_TETRA_DMO_EQ") == "true"
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: outRate,
		DibitSink: func(d []uint8, _ int) {
			allDibits = append(allDibits, d...)
		},
		SoftSink: func(diffs []complex64, _ int) {
			allDiffs = append(allDiffs, diffs...)
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

	dibitRate := tetrarx.SymbolRate // 18000 dibits/sec
	// Soft-capable extraction: carry the differentials so DMBurstTCHSpeechSoft can
	// recover corrupted TCH/S bursts the hard path drops (the #1001 ~2× lever).
	// Falls back to the hard ExtractDMBursts geometry if the soft stream didn't
	// stay parallel (length mismatch).
	bursts := tetra.ExtractDMBurstsSoft(allDibits, allDiffs, 0)

	var dsbTotal, dsbCRC, dnbTotal, tchCRC, tchSoftOnly int
	var syncSecs []int           // seconds carrying a CRC-valid SCH/S (transmission starts)
	speechBySec := map[int]int{} // CRC-valid TCH/S bursts per second
	var speechFrames [][]byte    // ordered speech frames for the vocoder
	seenSyncSec := map[int]struct{}{}

	for i := range bursts {
		b := bursts[i]
		sec := int(float64(b.Lead) / dibitRate)
		switch b.Kind {
		case tetra.DMBurstSync:
			dsbTotal++
			if _, ok := tetra.DecodeDMSCHS(b); ok {
				dsbCRC++
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

	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs colour=%#x eq=%v",
		inRate, outRate, len(iq), float64(len(iq))/inRate, colour, enableEQ)
	t.Logf("bursts: dsb_total=%d dsb_schs_crc=%d dnb_total=%d tch_crc=%d (soft_only=%d)",
		dsbTotal, dsbCRC, dnbTotal, tchCRC, tchSoftOnly)
	sort.Ints(syncSecs)
	t.Logf("sync (SCH/S CRC-valid) at seconds: %v", syncSecs)

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
		t.Logf("no CRC-valid TCH/S speech recovered — if the capture is a known-good DMO voice call, try GT_TETRA_DMO_COLOUR (the DM colour code) or GT_TETRA_DMO_EQ=1")
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
