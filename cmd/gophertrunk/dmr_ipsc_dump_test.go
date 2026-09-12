package main

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// TestDMRIPSCBurstDump is a burst-level instrument for a conventional DMR
// capture: it prints every data-sync burst's slot type (colour code + data
// type, BPTC validity) and every voice superframe (phase, embedded-LC
// talkgroup/source, EMB colour code) against stream time, so a missed
// transmission or a second colour code can be found by inspection rather than
// inferred from grant counts. GT_DMR_IQ=<capture> GT_DMR_DUMP=<out file>.
func TestDMRIPSCBurstDump(t *testing.T) {
	out := os.Getenv("GT_DMR_DUMP")
	path := os.Getenv("GT_DMR_IQ")
	if out == "" || path == "" {
		t.Skip("set GT_DMR_IQ and GT_DMR_DUMP to run the burst dump")
	}
	inRate := 50000.0
	if v := os.Getenv("GT_DMR_IQ_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatal(err)
		}
		inRate = f
	}
	iq, rate, err := siglab.DecodeContainerFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if rate > 0 && os.Getenv("GT_DMR_IQ_RATE") == "" {
		inRate = float64(rate)
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ddc := ccdecoder.NewDownconverter(inRate, 48000)
	var allDibits []uint8
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: ddc.OutRateHz(),
		DeviationHz:  1944.0,
		ClockGain:    0.015,
		DibitSink: func(d []uint8, _ int) {
			allDibits = append(allDibits, d...)
		},
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

	// Data bursts: every sync match, sliced like the Tier II adapter, slot type
	// parsed at identity polarity when the sync is a data sync.
	det := dmr.NewSyncDetector(nil, 2)
	matches, _ := det.Process(nil, allDibits, 0)
	const lookback = dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1
	for _, m := range matches {
		start := m.Index - lookback
		if start < 0 || start+dmr.BurstDibits > len(allDibits) {
			continue
		}
		sec := float64(start) / 4800
		if !dmr.SyncIsDataAtPolarity(m.Pattern, 0) {
			fmt.Fprintf(f, "%9.3f sync %s pos=%d\n", sec, m.Pattern.Name, start)
			continue
		}
		var b dmr.Burst
		copy(b.Dibits[:], allDibits[start:start+dmr.BurstDibits])
		slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
		if err != nil {
			fmt.Fprintf(f, "%9.3f data %s pos=%d slottype=bad\n", sec, m.Pattern.Name, start)
			continue
		}
		_, errs := framing.DecodeBPTC196_96(b.PayloadBits())
		info := ""
		if errs >= 0 {
			bits, _ := framing.DecodeBPTC196_96(b.PayloadBits())
			bytes := make([]byte, 12)
			for i := 0; i < 96; i++ {
				if bits[i]&1 != 0 {
					bytes[i>>3] |= 1 << uint(7-(i&7))
				}
			}
			if slot.DataType == dmr.DTVoiceLCHeader || slot.DataType == dmr.DTTerminatorWithLC {
				seed := framing.RS129SeedVoiceLCHeader
				if slot.DataType == dmr.DTTerminatorWithLC {
					seed = framing.RS129SeedTerminatorLC
				}
				if framing.VerifyRS12_9(bytes, seed) {
					if flc, err := dmr.ParseFLC(bytes); err == nil {
						if gv, ok := flc.AsGroupVoiceUser(); ok {
							info = fmt.Sprintf(" lc=group tg=%d src=%d", gv.GroupAddress, gv.SourceID)
						} else if uu, ok := flc.AsUnitToUnitVoice(); ok {
							info = fmt.Sprintf(" lc=unit dst=%d src=%d", uu.DestinationID, uu.SourceID)
						} else {
							info = fmt.Sprintf(" lc=flco%d", flc.FLCO)
						}
					}
				} else {
					info = " rs=fail"
				}
			}
		}
		fmt.Fprintf(f, "%9.3f data %s pos=%d cc=%d dt=%d bptc_errs=%d%s\n", sec, m.Pattern.Name, start, slot.ColorCode, slot.DataType, errs, info)
	}

	// Voice superframes through the interleaved decoder.
	vd := dmrvoice.NewInterleavedDecoder()
	const dchunk = 4096
	for i := 0; i < len(allDibits); i += dchunk {
		e := i + dchunk
		if e > len(allDibits) {
			e = len(allDibits)
		}
		for _, sf := range vd.Process(allDibits[i:e], i) {
			sec := float64(sf.StartDibit) / 4800
			errs := 0
			for k := range sf.Frames {
				_, n, _ := dmrvoice.DecodeAMBEFrame(sf.Frames[k])
				errs += n
			}
			lc := ""
			if sf.HasLC {
				if gv, ok := sf.LC.AsGroupVoiceUser(); ok {
					lc = fmt.Sprintf(" lc=group tg=%d src=%d embcc=%d", gv.GroupAddress, gv.SourceID, sf.EMBColorCode)
				} else if uu, ok := sf.LC.AsUnitToUnitVoice(); ok {
					lc = fmt.Sprintf(" lc=unit dst=%d src=%d embcc=%d", uu.DestinationID, uu.SourceID, sf.EMBColorCode)
				} else {
					lc = fmt.Sprintf(" lc=flco%d embcc=%d", sf.LC.FLCO, sf.EMBColorCode)
				}
			}
			fmt.Fprintf(f, "%9.3f sf %s pos=%d phase=%d ambe_errs=%d%s\n", sec, sf.SyncName, sf.StartDibit, sf.Phase, errs, lc)
		}
	}
	t.Logf("dibits=%d (%.1fs) matches=%d written to %s", len(allDibits), float64(len(allDibits))/4800, len(matches), out)
}
