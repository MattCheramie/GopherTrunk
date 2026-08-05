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

// TestTETRAMultiSlotReplay is a real-air validation harness for the per-timeslot
// demux. Skip unless GT_TETRA_IQ points at a cs16 (interleaved int16) IQ file
// and GT_TETRA_IQ_RATE gives its sample rate (GT_TETRA_OUT optionally names a
// dir for per-slot WAVs). It resamples to the 144 kHz TETRA channel rate, runs
// the receiver + slot-tagging TrafficExtractor, and for each TDMA timeslot
// decodes the CRC-valid TCH/S speech, printing a per-slot activity timeline to
// cross-check against the control channel's grant timeslots.
//
// It confirms the slot-tagging mechanism on real air (every burst anchors to the
// synchronisation burst and is tagged 1..4). It also surfaces a separate,
// pre-existing limitation: on a clean same-carrier capture the TCH/S channel
// decode (tch.go) passes the class-2 CRC only at the ~1/256 chance floor, so
// almost no real speech is recovered — the recovered bursts are spurious and
// scattered, not clustered on the grants. That is a tch.go channel-coding issue,
// not a demux one, and blocks audible output until fixed.
func TestTETRAMultiSlotReplay(t *testing.T) {
	path := os.Getenv("GT_TETRA_IQ")
	if path == "" {
		t.Skip("set GT_TETRA_IQ (cs16 IQ) + GT_TETRA_IQ_RATE to run the multi-slot replay")
	}
	inRate := 50000.0
	if v := os.Getenv("GT_TETRA_IQ_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_TETRA_IQ_RATE: %v", err)
		}
		inRate = f
	}
	const colourExt = 262144876 // the reporter's cell

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

	// Per-slot accumulators.
	type slotAcc struct {
		speechFrames  [][]byte
		crcBursts     int
		firstT, lastT float64
		activeSec     map[int]int
	}
	var slots [5]slotAcc // index 1..4
	var dibitsFed int
	dibitRate := tetrarx.SymbolRate // 18000 symbols(=dibits)/sec

	// Per-usage-marker accumulator: the AACH downlink usage marker is the demux
	// key the live voice chain routes by (concurrent same-carrier calls each carry
	// a distinct marker). Cross-check that CRC-valid speech clusters cleanly by
	// marker, and that each marker maps to a single physical slot.
	type usageAcc struct {
		speechFrames [][]byte
		crcBursts    int
		slotHits     map[uint8]int
	}
	usage := map[uint8]*usageAcc{}

	var totalBursts, anchoredBursts, trafficMarked, trafficMarkedCRC, trafficMarkedCRCSoft int
	errsHist := map[string]int{}
	extractor := tetra.NewTrafficExtractor(colourExt, func(frame []byte, softType5 []float32, slot, mark uint8) {
		totalBursts++
		if slot != 0 {
			anchoredBursts++
		}
		if mark >= tetra.DLUsageTraffic {
			trafficMarked++
			if len(tetra.TCHSpeechFrames(frame)) > 0 {
				trafficMarkedCRC++
			}
			// Soft-path CRC yield: this is the stream the LMS equalizer (GT_TETRA_LMS)
			// conditions, so A/B it via this counter (the hard trafficMarkedCRC above
			// is unaffected by the soft equalizer).
			if softType5 != nil && len(tetra.TCHSpeechFramesSoft(softType5)) > 0 {
				trafficMarkedCRCSoft++
			}
		}
		if _, _, _, errs, ok := tetra.DecodeTCHS(frame); ok {
			switch {
			case errs <= 5:
				errsHist["00-05"]++
			case errs <= 15:
				errsHist["06-15"]++
			case errs <= 30:
				errsHist["16-30"]++
			default:
				errsHist["31+"]++
			}
		}
		sfs := tetra.TCHSpeechFrames(frame)
		if len(sfs) == 0 {
			return
		}
		if mark >= tetra.DLUsageTraffic {
			u := usage[mark]
			if u == nil {
				u = &usageAcc{slotHits: map[uint8]int{}}
				usage[mark] = u
			}
			u.crcBursts++
			u.speechFrames = append(u.speechFrames, sfs...)
			u.slotHits[slot]++
		}
		if slot < 1 || slot > 4 {
			return
		}
		s := &slots[slot]
		tSec := float64(dibitsFed) / dibitRate
		if s.crcBursts == 0 {
			s.firstT = tSec
		}
		s.lastT = tSec
		s.crcBursts++
		s.speechFrames = append(s.speechFrames, sfs...)
		if s.activeSec == nil {
			s.activeSec = map[int]int{}
		}
		s.activeSec[int(tSec)]++
	})

	// GT_TETRA_EQ=1 enables the blind CMA channel equalizer in the receiver (issue
	// #1001); GT_TETRA_LMS=1 enables the per-burst training-sequence LMS equalizer
	// in the extractor's soft path. Either can be A/B'd against the same capture:
	// compare traffic_marked_crc (hard, moves with CMA) and traffic_marked_crc_soft
	// (soft, moves with LMS) plus the per-slot crc_bursts counts.
	enableEQ := os.Getenv("GT_TETRA_EQ") == "1" || os.Getenv("GT_TETRA_EQ") == "true"
	enableLMS := os.Getenv("GT_TETRA_LMS") == "1" || os.Getenv("GT_TETRA_LMS") == "true"
	if enableLMS {
		// Train on each burst's known midamble and equalize BKN1/BKN2 before the
		// soft TCH/S decode. Requires the raw symbols (SymbolSink → StashSymbols).
		extractor.EnableLMSEqualizer(0, 0)
	}
	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: outRate,
		DibitSink: func(d []uint8, base int) {
			dibitsFed = base + len(d)
			extractor.Process(d, base)
		},
		// Feed the per-symbol differentials so the extractor's soft-decision paths
		// (soft AACH usage-marker recovery, soft TCH/S) run — mirroring the live
		// voice chain, which always stashes soft. SoftSink fires just before the
		// matching DibitSink, so StashSoft lands before Process consumes it.
		SoftSink: func(diffs []complex64, base int) {
			extractor.StashSoft(diffs, base)
		},
		// Feed the raw pre-differential symbols the LMS equalizer trains/applies on.
		// Fires just before the matching DibitSink, so StashSymbols lands before
		// Process consumes the block. Harmless (buffered, never read) when LMS is off.
		SymbolSink: func(syms []complex64, base int) {
			extractor.StashSymbols(syms, base)
		},
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
		EnableEqualizer:     enableEQ,
	})
	t.Logf("cma_equalizer=%v (GT_TETRA_EQ=1) lms_equalizer=%v (GT_TETRA_LMS=1)", enableEQ, enableLMS)

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

	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs", inRate, outRate, len(iq), float64(len(iq))/inRate)
	t.Logf("total_bursts=%d anchored=%d traffic_marked=%d traffic_marked_crc=%d traffic_marked_crc_soft=%d errs_hist=%v", totalBursts, anchoredBursts, trafficMarked, trafficMarkedCRC, trafficMarkedCRCSoft, errsHist)
	outDir := os.Getenv("GT_TETRA_OUT")
	for tn := 1; tn <= 4; tn++ {
		s := slots[tn]
		if s.crcBursts == 0 {
			t.Logf("slot %d: no TCH/S speech", tn)
			continue
		}
		// Decode this slot's speech frames to PCM with a fresh ACELP vocoder.
		voc := acelp.NewVocoder()
		var pcm []int16
		for _, sf := range s.speechFrames {
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
		var secs []int
		for k := range s.activeSec {
			if s.activeSec[k] >= 2 { // >=2 bursts in the second = real activity, not a lone spurious
				secs = append(secs, k)
			}
		}
		sort.Ints(secs)
		t.Logf("slot %d: crc_bursts=%d speech_frames=%d pcm=%.1fs peak=%d active_secs(>=2)=%v",
			tn, s.crcBursts, len(s.speechFrames), float64(len(pcm))/8000, peak, secs)
		if outDir != "" {
			writeWav8kLocal(outDir+"/slot"+string(rune('0'+tn))+".wav", pcm)
		}
	}

	// Per-usage-marker view: the live demux key. Each traffic marker should
	// cluster on a single physical slot (one call = one marker = one slot).
	var marks []int
	for m := range usage {
		marks = append(marks, int(m))
	}
	sort.Ints(marks)
	for _, mi := range marks {
		u := usage[uint8(mi)]
		t.Logf("usage_marker %d: crc_bursts=%d speech_frames=%d slot_hits=%v",
			mi, u.crcBursts, len(u.speechFrames), u.slotHits)
	}
}

func writeWav8kLocal(path string, pcm []int16) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	const rate = 8000
	dataLen := len(pcm) * 2
	h := make([]byte, 44)
	copy(h[0:], "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+dataLen))
	copy(h[8:], "WAVE")
	copy(h[12:], "fmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1)
	binary.LittleEndian.PutUint16(h[22:], 1)
	binary.LittleEndian.PutUint32(h[24:], rate)
	binary.LittleEndian.PutUint32(h[28:], rate*2)
	binary.LittleEndian.PutUint16(h[32:], 2)
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(dataLen))
	f.Write(h)
	buf := make([]byte, dataLen)
	for i, s := range pcm {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
	}
	f.Write(buf)
}
