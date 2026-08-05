package main

import (
	"context"
	"encoding/binary"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDMRIPSCReplay is a real-air diagnostic harness for conventional DMR /
// IPSC captures (issue #1036). Skip unless GT_DMR_IQ points at a cs16
// (interleaved int16) IQ file; GT_DMR_IQ_RATE gives its sample rate (default
// 50000). It mirrors the daemon's dmr-tier2 decode: it downconverts to the DMR
// channel rate, runs the shared DMR receiver, and feeds the recovered dibits to
// BOTH the Tier II conventional state machine (which emits a grant on every
// Voice LC Header — "grants but no voice" comes from here) AND the DMR voice
// superframe/AMBE decoder (the audio the recorder would write). It prints grant,
// superframe, embedded-LC and AMBE-FEC yields so a capture that "detects frames
// but records no voice" can be triaged: 0 superframes ⇒ a receiver/DSP problem;
// superframes>0 with AMBE decoding ⇒ the audio is recoverable and the gap is in
// grant→voice-device binding, not the DSP.
//
// GT_DMR_INTERLEAVED=1 decodes the carrier as 2-slot interleaved (two talkers,
// one per timeslot) — the common IPSC case — and reports per-phase superframe
// counts so slot separation is visible.
func TestDMRIPSCReplay(t *testing.T) {
	path := os.Getenv("GT_DMR_IQ")
	if path == "" {
		t.Skip("set GT_DMR_IQ (cs16 IQ) [+ GT_DMR_IQ_RATE] to run the DMR IPSC replay")
	}
	inRate := 50000.0
	if v := os.Getenv("GT_DMR_IQ_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_DMR_IQ_RATE: %v", err)
		}
		inRate = f
	}
	interleaved := os.Getenv("GT_DMR_INTERLEAVED") == "1" || os.Getenv("GT_DMR_INTERLEAVED") == "true"

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

	// DMR is the 4800-baud C4FM family — normalise to the 48 kHz channel rate.
	ddc := ccdecoder.NewDownconverter(inRate, 48000)
	outRate := ddc.OutRateHz()

	// Collect grants off the bus (the tier2 state machine publishes KindGrant).
	bus := events.NewBus(256)
	defer bus.Close()
	sub := bus.Subscribe()
	type grantRec struct {
		tg, src uint32
		indiv   bool
	}
	var grantsMu sync.Mutex
	var grants []grantRec
	ctx, cancel := context.WithCancel(context.Background())
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.C:
				if !ok {
					return
				}
				if ev.Kind != events.KindGrant {
					continue
				}
				if g, ok := ev.Payload.(trunking.Grant); ok {
					grantsMu.Lock()
					grants = append(grants, grantRec{tg: g.GroupID, src: g.SourceID, indiv: g.Individual})
					grantsMu.Unlock()
				}
			}
		}
	}()

	cc := tier2.New(tier2.Options{Bus: bus, SystemName: "dmr-ipsc-replay", FrequencyHz: 0})

	var voiceDec *dmrvoice.Decoder
	if interleaved {
		voiceDec = dmrvoice.NewInterleavedDecoder()
	} else {
		voiceDec = dmrvoice.NewDecoder()
	}

	var (
		superframes   int
		lcSuperframes int
		ambeOK        int
		ambeUncorrect int
		phaseCounts   [2]int
		// lcGroups records the talkgroup each CRC-valid embedded LC named, so we
		// can see whether the (issue #644, unvalidated) embedded signalling names
		// the grant's talkgroup — the composer's interleaved slotRouter routes by
		// this, and a garbage LC-TG that never matches the grant would make the
		// router reject the call's own audio and record nothing.
		lcGroups = map[uint32]int{}
	)
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: outRate,
		DeviationHz:  1944.0,
		ClockGain:    0.015,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
			for _, sf := range voiceDec.Process(dibits, baseIdx) {
				superframes++
				phaseCounts[sf.Phase&1]++
				if sf.HasLC {
					lcSuperframes++
					if gv, ok := sf.LC.AsGroupVoiceUser(); ok {
						lcGroups[gv.GroupAddress]++
					}
				}
				for i := range sf.Frames {
					_, _, derr := dmrvoice.DecodeAMBEFrame(sf.Frames[i])
					if derr != nil {
						ambeUncorrect++
					} else {
						ambeOK++
					}
				}
			}
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
	cancel()
	<-drainDone

	grantsMu.Lock()
	defer grantsMu.Unlock()
	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs interleaved=%v",
		inRate, outRate, len(iq), float64(len(iq))/inRate, interleaved)
	t.Logf("grants=%d", len(grants))
	seen := map[grantRec]int{}
	for _, g := range grants {
		seen[grantRec{tg: g.tg, src: g.src, indiv: g.indiv}]++
	}
	for g, n := range seen {
		t.Logf("  grant tg=%d src=%d individual=%v count=%d", g.tg, g.src, g.indiv, n)
	}
	t.Logf("superframes=%d lc_superframes=%d phase0=%d phase1=%d ambe_ok=%d ambe_uncorrectable=%d",
		superframes, lcSuperframes, phaseCounts[0], phaseCounts[1], ambeOK, ambeUncorrect)
	for tg, n := range lcGroups {
		t.Logf("  embedded-LC group_address=%d count=%d", tg, n)
	}

	// The harness is a diagnostic: it must always surface *something* (a grant or
	// a decoded superframe) or the capture/rate/tune is wrong. A silent pass on a
	// live capture would hide exactly the failure #1036 is about.
	if len(grants) == 0 && superframes == 0 {
		t.Fatalf("no grants and no voice superframes decoded — check GT_DMR_IQ_RATE (%v) / tuning / capture", inRate)
	}
}
