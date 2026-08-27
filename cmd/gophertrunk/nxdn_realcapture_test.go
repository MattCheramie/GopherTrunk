package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	nxdnrx "github.com/MattCheramie/GopherTrunk/internal/radio/nxdn/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestReplayNXDNRealCapture is the NXDN real-air diagnostic harness — the
// NXDN analog of TestDMRIPSCReplay. Skip unless GT_NXDN_IQ points at a cs16
// (interleaved int16) IQ file; GT_NXDN_IQ_RATE gives its sample rate
// (default 48000). It mirrors the daemon's nxdn decode: downconvert to the
// 48 kHz C4FM channel rate, run the production NXDN receiver, and feed the
// recovered dibits to BOTH the ControlChannel state machine (lock + VCALL
// grants come from here) AND a standalone FSW/CAC slicer that reports raw
// FSW hits and CAC channel-decode / CRC yields. GT_NXDN_SOFT=1 flips the
// receiver onto the nxdn_soft_decision path and reports hard AND soft CAC
// CRC yields off the same stream — the opt-in's A/B metric.
//
// NXDN captures are the blocker for the whole voice path (deinterleave
// placeholder, scramble model, CAC structure — all flagged unverified on
// air); this harness is how a contributed capture gets baselined. See
// samples/nxdn/README.md and docs/protocol-feature-parity.md.
func TestReplayNXDNRealCapture(t *testing.T) {
	path := os.Getenv("GT_NXDN_IQ")
	if path == "" {
		t.Skip("set GT_NXDN_IQ (cs16 IQ) [+ GT_NXDN_IQ_RATE] to run the NXDN replay")
	}
	inRate := 48000.0
	if v := os.Getenv("GT_NXDN_IQ_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("bad GT_NXDN_IQ_RATE: %v", err)
		}
		inRate = f
	}
	// GT_NXDN_SOFT=1 runs the nxdn_soft_decision path (per-bit soft Viterbi
	// on the CAC) and reports BOTH hard and soft CAC CRC yields off the same
	// dibit/LLR stream — the A/B metric for the opt-in.
	soft := os.Getenv("GT_NXDN_SOFT") == "1" || os.Getenv("GT_NXDN_SOFT") == "true"
	// GT_NXDN_AFC=1 enables the opt-in post-clock CoarseAFC (nxdn_afc) —
	// the A/B lever for a capture with a real tuner ppm error. Off by
	// default like production (NXDN's unwhitened CAC runs drift the plain
	// coarse tracker — see the receiver's Options.EnableAFC comment).
	afc := os.Getenv("GT_NXDN_AFC") == "1" || os.Getenv("GT_NXDN_AFC") == "true"

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

	// NXDN's 9600-baud variant is the 4800-baud C4FM family — normalise to
	// the 48 kHz channel rate, exactly as the production pipeline does.
	ddc := ccdecoder.NewDownconverter(inRate, 48000)
	outRate := ddc.OutRateHz()

	// Collect lock + grant events off the bus (the ControlChannel publishes
	// KindCCLocked on a corroborated SITE_INFO and KindGrant on VCALL_ASSGN).
	bus := events.NewBus(256)
	defer bus.Close()
	sub := bus.Subscribe()
	var evMu sync.Mutex
	locks, grants := 0, 0
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
				evMu.Lock()
				switch ev.Kind {
				case events.KindCCLocked:
					locks++
				case events.KindGrant:
					if _, ok := ev.Payload.(trunking.Grant); ok {
						grants++
					}
				}
				evMu.Unlock()
			}
		}
	}()

	cc := nxdn.NewControlChannel(bus, nil, 0, nxdn.Rate9600)
	cc.SetSystemName("nxdn-replay")
	// The bare constructor defaults to ViterbiOff (fixture behaviour); the
	// production pipeline runs the spec chain — mirror it.
	mode, _ := nxdn.ParseViterbiMode("")
	cc.SetViterbiMode(mode)

	// Standalone FSW/CAC slicer over the same dibit stream, mirroring the
	// ControlChannel's ViterbiSpec frame layout (8 LICH + 150 CAC dibits
	// after the FSW) via the public decode chain: DecodeLICHWire +
	// DecodeCACChannel (deinterleave + depuncture + K=5 Viterbi + CRC16) +
	// ParseCAC. Yields the per-stage counts the CRC-yield A/B needs.
	const postFSWDibits = 8 + 150
	det := nxdn.NewSyncDetector([][]uint8{nxdn.FSWDibitsOutbound}, 1)
	var (
		fswHits      int
		cacTotal     int
		cacCRCok     int
		cacCRCokSoft int
		cacParsed    int
		rcchCounts   = map[nxdn.RCCHType]int{}
		remaining    int
		frame        []uint8
		frameSoft    []float32
		matches      []nxdn.Match
	)
	slicer := func(dibits []uint8, llrs []float32, baseIdx int) {
		matches, _ = det.Process(matches[:0], dibits, baseIdx)
		matchIdx := 0
		for i, d := range dibits {
			absPos := baseIdx + i
			if remaining > 0 {
				frame = append(frame, d)
				if llrs != nil {
					frameSoft = append(frameSoft, llrs[2*i], llrs[2*i+1])
				}
				remaining--
				if remaining == 0 {
					cacTotal++
					channelBits := framing.DibitsToBits(frame[8:])
					if info, ok := nxdn.DecodeCACChannel(channelBits); ok {
						cacCRCok++
						// Same L3 repack the production path applies
						// (§4.5.1.1: drop the 8-bit SR, take 72 L3 bits).
						if len(info) >= 8+72 {
							l3 := framing.PackBitsMSB(info[8 : 8+72])
							block := make([]byte, 11)
							copy(block, l3[:9])
							binary.BigEndian.PutUint16(block[9:11], framing.CRCCCITT(block[:9]))
							if cac, err := nxdn.ParseCAC(block); err == nil {
								cacParsed++
								rcchCounts[cac.Type]++
							}
						}
					}
					if llrs != nil && len(frameSoft) == 2*len(frame) {
						if _, ok := nxdn.DecodeCACChannelSoft(frameSoft[8*2:]); ok {
							cacCRCokSoft++
						}
					}
					frame = frame[:0]
					frameSoft = frameSoft[:0]
				}
			}
			for matchIdx < len(matches) && matches[matchIdx].Index == absPos {
				if !matches[matchIdx].Inbound {
					fswHits++
					remaining = postFSWDibits
					frame = frame[:0]
					frameSoft = frameSoft[:0]
				}
				matchIdx++
			}
		}
	}

	rxOpts := nxdnrx.Options{
		SampleRateHz: outRate,
		DeviationHz:  1800.0,
		EnableAFC:    afc,
	}
	if soft {
		rxOpts.SoftDecision = true
		rxOpts.SoftDibitSink = func(dibits []uint8, llrs []float32, baseIdx int) {
			cc.ProcessSoft(dibits, llrs, baseIdx)
			slicer(dibits, llrs, baseIdx)
		}
	} else {
		rxOpts.DibitSink = func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
			slicer(dibits, nil, baseIdx)
		}
	}
	rx := nxdnrx.New(rxOpts)

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

	evMu.Lock()
	defer evMu.Unlock()
	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs",
		inRate, outRate, len(iq), float64(len(iq))/inRate)
	t.Logf("locks=%d grants=%d", locks, grants)
	if soft {
		t.Logf("fsw_hits=%d cac_total=%d cac_crc_ok=%d cac_crc_ok_soft=%d cac_parsed=%d",
			fswHits, cacTotal, cacCRCok, cacCRCokSoft, cacParsed)
	} else {
		t.Logf("fsw_hits=%d cac_total=%d cac_crc_ok=%d cac_parsed=%d", fswHits, cacTotal, cacCRCok, cacParsed)
	}
	for typ, n := range rcchCounts {
		t.Logf("  rcch %s count=%d", typ, n)
	}
	topo := cc.Topology()
	t.Logf("topology system_id=0x%04X site_id=0x%04X location=0x%06X",
		topo.SystemID, topo.SiteID, topo.LocationID)

	// Diagnostic posture (mirrors TestDMRIPSCReplay): a signal-bearing
	// capture should surface FSW matches; zero usually means a wrong
	// rate/tune. A genuinely weak capture that decodes NOTHING is itself a
	// valid 0/0 baseline for the soft-decision A/B, so GT_NXDN_ALLOW_EMPTY=1
	// downgrades the failure to a logged warning.
	if fswHits == 0 {
		msg := fmt.Sprintf("no FSW matches decoded — check GT_NXDN_IQ_RATE (%v) / tuning / capture", inRate)
		if os.Getenv("GT_NXDN_ALLOW_EMPTY") == "1" {
			t.Logf("WARNING: %s (GT_NXDN_ALLOW_EMPTY=1: treating as a weak-signal baseline)", msg)
		} else {
			t.Fatalf("%s", msg)
		}
	}
}
