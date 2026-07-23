package main

import (
	"encoding/binary"
	"os"
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
	}
	var slots [5]slotAcc // index 1..4
	var dibitsFed int
	dibitRate := tetrarx.SymbolRate // 18000 symbols(=dibits)/sec

	var totalBursts, anchoredBursts int
	errsHist := map[string]int{}
	extractor := tetra.NewTrafficExtractor(colourExt, func(frame []byte, slot uint8) {
		totalBursts++
		if slot != 0 {
			anchoredBursts++
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
		if slot < 1 || slot > 4 {
			return
		}
		sfs := tetra.TCHSpeechFrames(frame)
		if len(sfs) == 0 {
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
	})

	rx := tetrarx.New(tetrarx.Options{
		SampleRateHz: outRate,
		DibitSink: func(d []uint8, base int) {
			dibitsFed = base + len(d)
			extractor.Process(d, base)
		},
		ClockMode:           tetrarx.ClockGardner,
		GardnerGain:         0.005,
		EnableAFC:           true,
		EnableChannelFilter: true,
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

	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs", inRate, outRate, len(iq), float64(len(iq))/inRate)
	t.Logf("total_bursts=%d anchored=%d errs_hist=%v", totalBursts, anchoredBursts, errsHist)
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
		t.Logf("slot %d: crc_bursts=%d speech_frames=%d active=%.1f..%.1fs pcm=%.1fs peak=%d",
			tn, s.crcBursts, len(s.speechFrames), s.firstT, s.lastT, float64(len(pcm))/8000, peak)
		if outDir != "" {
			writeWav8kLocal(outDir+"/slot"+string(rune('0'+tn))+".wav", pcm)
		}
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
