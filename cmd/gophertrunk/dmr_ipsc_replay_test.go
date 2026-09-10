package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDMRIPSCReplay is a real-air diagnostic harness for conventional DMR /
// IPSC captures (issue #1036). Skip unless GT_DMR_IQ points at an IQ file;
// GT_DMR_IQ_RATE gives its sample rate (default 50000) and GT_DMR_IQ_FORMAT the
// encoding (cs16 default, or f32 for a GNU Radio / gqrx cfile — the reporter's
// 25 kS/s f32 capture: GT_DMR_IQ_RATE=25000 GT_DMR_IQ_FORMAT=f32). Note an
// idle-beacon capture (a keyed repeater with no voice keyup) decodes 0 grants
// by design but a non-zero beacons count (the ETSI Idle bursts it fills both
// slots with) — set GT_DMR_ALLOW_EMPTY=1 for a capture that decodes nothing. It mirrors the daemon's
// dmr-tier2 decode: it downconverts to the DMR
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
//
// GT_DMR_DROP_HEADERS=1 scrubs every Voice LC Header burst out of the recovered
// dibit stream before the Tier II state machine sees it — the on-air model of
// a keyup lost to a fade on a marginal tap. With late entry the transmissions
// must still be granted from their embedded LC (grants ≈ the un-scrubbed run,
// late_entries > 0); without it the run decodes voice but grants nothing —
// exactly the "GT said the call ended but the conversation continued" report.
//
// GT_DMR_DROP_TERMINATORS=1 scrubs every Terminator-with-LC burst instead: the
// on-air model of the 10 Sep report where the control path never decoded the
// terminator while the voice path did. Each transmission after the first must
// still be granted (rekeys = grants − 1); before the re-key rule the run
// granted exactly once and every reply on the same talkgroup was missed.
func TestDMRIPSCReplay(t *testing.T) {
	path := os.Getenv("GT_DMR_IQ")
	if path == "" {
		t.Skip("set GT_DMR_IQ (IQ file) [+ GT_DMR_IQ_RATE, GT_DMR_IQ_FORMAT=cs16|f32] to run the DMR IPSC replay")
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
	dropHeaders := os.Getenv("GT_DMR_DROP_HEADERS") == "1"
	dropTerminators := os.Getenv("GT_DMR_DROP_TERMINATORS") == "1"

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// GT_DMR_IQ_FORMAT selects the sample encoding: cs16 (default, interleaved
	// int16) or f32 (interleaved float32 — the GNU Radio / gqrx cfile format the
	// #1036 reporter's 25 kS/s capture uses). A wav/flac container (the Signal
	// Lab "capture from tuner" output) is sniffed from its content and carries
	// its own sample rate, so GT_DMR_IQ_RATE / GT_DMR_IQ_FORMAT are not needed.
	format := strings.ToLower(os.Getenv("GT_DMR_IQ_FORMAT"))
	if format == "" {
		format = "cs16"
	}
	var iq []complex64
	if _, isContainer := siglab.SniffContainer(raw); isContainer {
		format = "container"
	}
	switch format {
	case "container":
		samples, rate, err := siglab.DecodeContainerFile(path)
		if err != nil {
			t.Fatal(err)
		}
		iq = samples
		if rate > 0 && os.Getenv("GT_DMR_IQ_RATE") == "" {
			inRate = float64(rate)
		}
	case "cs16", "sc16":
		iq = make([]complex64, len(raw)/4)
		for i := range iq {
			re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
			im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
			iq[i] = complex(float32(re)/32768, float32(im)/32768)
		}
	case "f32", "cf32", "fc32":
		iq = make([]complex64, len(raw)/8)
		for i := range iq {
			re := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8:]))
			im := math.Float32frombits(binary.LittleEndian.Uint32(raw[i*8+4:]))
			iq[i] = complex(re, im)
		}
	default:
		t.Fatalf("unknown GT_DMR_IQ_FORMAT %q (want cs16 or f32)", format)
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
	// timeline records every grant / release the Tier II state machine
	// published against the dibit position the harness was feeding at the
	// time (4800 dibits/s), next to the composer-side terminator detector's
	// own detections, so a terminator the control path decodes seconds after
	// the voice path (the 10 Sep "re-key within hangtime is missed" report)
	// is visible as a timing gap rather than inferred from a daemon log.
	var feedPos atomic.Int64
	var timeline []string
	termDet := dmrvoice.NewTerminatorDetector()
	stamp := func(kind string, tg uint32) {
		timeline = append(timeline, fmt.Sprintf("%8.2fs %-14s tg=%d", float64(feedPos.Load())/4800, kind, tg))
	}
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
				switch ev.Kind {
				case events.KindGrant:
					if g, ok := ev.Payload.(trunking.Grant); ok {
						grantsMu.Lock()
						grants = append(grants, grantRec{tg: g.GroupID, src: g.SourceID, indiv: g.Individual})
						stamp("grant", g.GroupID)
						grantsMu.Unlock()
					}
				case events.KindCallRelease:
					if r, ok := ev.Payload.(trunking.CallRelease); ok {
						grantsMu.Lock()
						stamp("tier2-release", r.GroupID)
						grantsMu.Unlock()
					}
				}
			}
		}
	}()

	// GT_DMR_DEBUG=1 prints the Tier II state machine's Debug lines (grants,
	// terminators, the parked CSBK-failure diagnostics with their info hex).
	var ccLog *slog.Logger
	if os.Getenv("GT_DMR_DEBUG") == "1" {
		ccLog = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	cc := tier2.New(tier2.Options{Bus: bus, Log: ccLog, SystemName: "dmr-ipsc-replay", FrequencyHz: 0, InterleavedVoice: interleaved})

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
	var allDibits []uint8
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: outRate,
		DeviationHz:  1944.0,
		ClockGain:    0.015,
		DibitSink: func(dibits []uint8, _ int) {
			allDibits = append(allDibits, dibits...)
		},
	})
	feed := func(dibits []uint8, baseIdx int) {
		{
			feedPos.Store(int64(baseIdx))
			cc.Process(dibits, baseIdx)
			// Wait for the bus to deliver this chunk's events before moving on
			// so the timeline stamps carry this chunk's position.
			for _, flc := range termDet.Process(dibits, baseIdx) {
				if dest, ok := lcCallDestinationForReplay(flc); ok {
					grantsMu.Lock()
					stamp("voice-term", dest)
					grantsMu.Unlock()
				}
			}
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
		}
	}

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
	scrubbed := 0
	if dropHeaders {
		scrubbed = scrubBursts(allDibits, dmr.DTVoiceLCHeader)
	}
	if dropTerminators {
		scrubbed += scrubBursts(allDibits, dmr.DTTerminatorWithLC)
	}
	// Re-feed the (possibly scrubbed) dibit stream in receiver-sized chunks.
	const dibitChunk = 4096
	for i := 0; i < len(allDibits); i += dibitChunk {
		e := i + dibitChunk
		if e > len(allDibits) {
			e = len(allDibits)
		}
		feed(allDibits[i:e], i)
	}
	cancel()
	<-drainDone

	grantsMu.Lock()
	defer grantsMu.Unlock()
	t.Logf("in=%.0fHz out=%.0fHz samples=%d dur=%.1fs interleaved=%v",
		inRate, outRate, len(iq), float64(len(iq))/inRate, interleaved)
	t.Logf("grants=%d late_entries=%d rekeys=%d fec_pass=%d fec_fail=%d csbk_crc_fail=%d beacons=%d locks=%d scrubbed_headers=%d",
		len(grants), cc.Counters().LateEntries, cc.Counters().Rekeys, cc.Counters().FECPass, cc.Counters().FECFail,
		cc.Counters().CSBKCRCFail, cc.Counters().Beacons, cc.Counters().Locks, scrubbed)
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
	for _, line := range timeline {
		t.Logf("  timeline %s", line)
	}

	// The harness is a diagnostic. On a signal-bearing capture it should surface
	// a grant or a superframe, and zero of both usually means a wrong rate/tune.
	// But a genuinely weak-signal capture that decodes NOTHING is itself a valid
	// baseline measurement (the 0/0 floor the equalizer work is measured
	// against), so GT_DMR_ALLOW_EMPTY=1 downgrades the hard failure to a logged
	// warning. Default stays strict so an accidental mistune is still caught.
	if len(grants) == 0 && superframes == 0 && cc.Counters().Beacons == 0 {
		msg := fmt.Sprintf("no grants, no voice superframes and no idle beacons decoded — check GT_DMR_IQ_RATE (%v) / tuning / capture", inRate)
		if os.Getenv("GT_DMR_ALLOW_EMPTY") == "1" {
			t.Logf("WARNING: %s (GT_DMR_ALLOW_EMPTY=1: treating as a weak-signal 0/0 baseline)", msg)
		} else {
			t.Fatalf("%s", msg)
		}
	}
}

// scrubBursts overwrites every data burst of slot type dt in dibits (found via
// the data-sync detector + slot type, exactly as the Tier II adapter slices
// them) with a pseudo-random dibit pattern that matches no sync word, and
// returns how many bursts were scrubbed. With DTVoiceLCHeader it models a
// keyup whose header bursts were lost on air; with DTTerminatorWithLC
// (GT_DMR_DROP_TERMINATORS=1) it models the 10 Sep field condition where the
// control path's copy of every terminator was BPTC-uncorrectable while the
// composer's voice path still released each call — every transmission after
// the first must then be granted by the re-key rule.
func scrubBursts(dibits []uint8, dt dmr.DataType) int {
	det := dmr.NewSyncDetector(nil, 2)
	matches, _ := det.Process(nil, dibits, 0)
	n := 0
	seed := uint32(0x9E3779B9)
	for _, m := range matches {
		start := m.Index - (dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1)
		end := start + dmr.BurstDibits
		if start < 0 || end > len(dibits) {
			continue
		}
		var b dmr.Burst
		copy(b.Dibits[:], dibits[start:end])
		slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
		if err != nil || slot.DataType != dt {
			continue
		}
		for i := start; i < end; i++ {
			seed = seed*1664525 + 1013904223
			dibits[i] = uint8(seed >> 30)
		}
		n++
	}
	return n
}

// lcCallDestinationForReplay mirrors the composer's lcCallDestination: the
// destination a Terminator-with-LC's Full LC names (group or unit-to-unit).
func lcCallDestinationForReplay(flc dmr.FLC) (uint32, bool) {
	if gv, ok := flc.AsGroupVoiceUser(); ok {
		return gv.GroupAddress, true
	}
	if uu, ok := flc.AsUnitToUnitVoice(); ok {
		return uu.DestinationID, true
	}
	return 0, false
}
