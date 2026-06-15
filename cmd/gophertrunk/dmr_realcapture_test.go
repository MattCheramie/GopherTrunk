package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier3"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// These two fixtures are channelised slices of a live 441 MHz, 2 MS/s DMR
// Tier III trunked system (GNU Radio f32 wideband capture). Each carrier was
// mixed to baseband and decimated to a 48 kHz centred stream so it decodes
// standalone through the production DMR receiver without a down-converter —
// same shape as the P25 testdata/mmr-*.cfile fixtures.
//
// Why these exist (issue #527 follow-up): a field report claimed live DMR
// emitted a *constant stream* of decode.error at the voiceheader-bptc /
// voiceheader-rs stages while the synthesized fixtures passed — i.e. the
// receiver's real-air dibit recovery and end-to-end bit ordering through
// BPTC(196,96) + RS(12,9) were unproven on air. Every prior test round-trips
// our own encoder, so a real-air-only mismatch would pass CI and fail on the
// air. These captures close that gap with genuine over-the-air bits:
//
//   - dmr-t3-cc.cfile exercises BPTC(196,96) + CSBK CRC (the control channel's
//     Aloha beacon the receiver locks on).
//   - dmr-voice-term.cfile exercises the *identical* Full-LC FEC stack the
//     Voice LC Header uses — BPTC(196,96) → RS(12,9) → FLC parse — via the
//     Terminator-with-LC burst that brackets a voice call (the only
//     difference from a Voice LC Header is the RS seed: 0x99 vs 0x96, both
//     pinned by TestRS129MatchesIndependentReferenceEncoder).
//
// Both decode cleanly here, so the "real-air BPTC/RS is broken" hypothesis is
// refuted: the symbol-AGC calibration in internal/radio/dmr/receiver (issue
// #275) is what fixed the original "BPTC uncorrectable on a real signal"
// failure, and these guards pin it against a regression.

// readCenteredIQ loads a GNU Radio f32 .cfile of pre-centred 48 kHz baseband.
func readCenteredIQ(t *testing.T, name string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	pairs := len(raw) / 8
	if pairs == 0 {
		t.Fatalf("fixture %s has no samples", name)
	}
	iq := make([]complex64, pairs)
	dec, _ := siglab.FormatF32.Decoder()
	dec(raw[:pairs*8], iq)
	return iq
}

// newDMRReceiver builds the production DMR IQ→dibit receiver at the on-air
// 1944 Hz deviation, forwarding dibits to sink.
func newDMRReceiver(clockGain float64, sink dmr.DibitSink) *dmrrx.Receiver {
	return dmrrx.New(dmrrx.Options{
		SampleRateHz: 48_000.0,
		DeviationHz:  1944.0,
		ClockGain:    clockGain,
		DibitSink:    sink,
	})
}

func feed(rx *dmrrx.Receiver, iq []complex64) {
	const chunk = 8192
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[off:end])
	}
}

// sliceBursts walks every DMR sync match in a dibit stream and yields the
// 132-dibit burst (at both discriminator polarities) with its decoded slot
// type, mirroring the production Process adapters.
func sliceBursts(dibits []uint8, fn func(b *dmr.Burst, slot dmr.SlotType)) {
	det := dmr.NewSyncDetector(nil, 2)
	matches, _ := det.Process(nil, dibits, 0)
	const lookback = dmr.HalfPayloadDibits + dmr.SlotTypeDibits + dmr.SyncDibits - 1 // 77
	const lookahead = dmr.SlotTypeDibits + dmr.HalfPayloadDibits                     // 54
	for _, m := range matches {
		start := m.Index - lookback
		if start < 0 || m.Index+lookahead+1 > len(dibits) {
			continue
		}
		for _, k := range dmr.CandidatePolarities {
			var b dmr.Burst
			copy(b.Dibits[:], dibits[start:start+dmr.BurstDibits])
			dmr.RotateBurstDibits(&b, k)
			slot, _, err := dmr.ParseSlotType(b.SlotTypeBitsAll())
			if err != nil {
				continue
			}
			fn(&b, slot)
		}
	}
}

// TestReplayDMRTier3ControlDecodesRealAir asserts the production receiver +
// Tier III ControlChannel lock on a real over-the-air DMR control channel,
// and that its CSBKs pass BPTC(196,96) + CRC. Lock requires an Aloha CSBK
// whose CRC validates, so a real-air BPTC/dibit-recovery regression would
// drop the lock and the CSBK-CRC count together.
func TestReplayDMRTier3ControlDecodesRealAir(t *testing.T) {
	iq := readCenteredIQ(t, "dmr-t3-cc.cfile")

	// Production end-to-end: receiver → Tier III ControlChannel.
	bus := events.NewBus(1024)
	sub := bus.Subscribe()
	locks := 0
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range sub.C {
			if ev.Kind == events.KindCCLocked {
				locks++
			}
		}
	}()
	cc := tier3.New(tier3.Options{
		Bus:         bus,
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName:  "dmr-t3-realair",
		FrequencyHz: 440_262_200,
	})
	rx := newDMRReceiver(0.025, func(dibits []uint8, baseIdx int) {
		cc.Process(dibits, baseIdx)
	})
	feed(rx, iq)
	bus.Close()
	<-done

	if locks == 0 {
		t.Errorf("Tier III ControlChannel never locked on the real-air control channel")
	}

	// FEC-rate guard: decode every CSBK burst through the same BPTC + CRC the
	// lock path uses and require a healthy pass rate. A real-air dibit-recovery
	// or bit-ordering regression shows up as BPTC/CRC failures here first.
	var dibits []uint8
	rx2 := newDMRReceiver(0.025, func(d []uint8, _ int) { dibits = append(dibits, d...) })
	feed(rx2, iq)

	csbkTotal, csbkCRCok, aloha := 0, 0, 0
	sliceBursts(dibits, func(b *dmr.Burst, slot dmr.SlotType) {
		if slot.DataType != dmr.DTCSBK {
			return
		}
		csbkTotal++
		bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
		if errs < 0 {
			return
		}
		if csbk, err := tier3.ParseCSBK(tier3.InfoBitsToBytes(bits)); err == nil {
			csbkCRCok++
			if csbk.Opcode == tier3.OpAloha {
				aloha++
			}
		}
	})
	t.Logf("Tier III CC: locks=%d CSBK total=%d CRC_ok=%d Aloha=%d", locks, csbkTotal, csbkCRCok, aloha)
	if csbkCRCok < 12 {
		t.Errorf("CSBK CRC_ok = %d, want >= 12 (real-air BPTC+CRC decode regressed)", csbkCRCok)
	}
	if aloha < 8 {
		t.Errorf("Aloha CSBKs = %d, want >= 8 (control-channel beacon decode regressed)", aloha)
	}
}

// TestReplayDMRVoiceLCFECDecodesRealAir is the direct issue #527 guard. It
// runs the production receiver over a real voice carrier and decodes the
// Terminator-with-LC bursts through the exact BPTC(196,96) → RS(12,9) → FLC
// stack the Voice LC Header uses, asserting the Full LC recovers a stable,
// real talkgroup + source ID. This is the on-air confirmation that was
// "blocking, not optional" per samples/dmr-tier2/README.md.
func TestReplayDMRVoiceLCFECDecodesRealAir(t *testing.T) {
	iq := readCenteredIQ(t, "dmr-voice-term.cfile")

	var dibits []uint8
	rx := newDMRReceiver(0.015, func(d []uint8, _ int) { dibits = append(dibits, d...) })
	feed(rx, iq)

	termTotal, bptcOK, rsOK := 0, 0, 0
	calls := map[[2]uint32]int{}
	sliceBursts(dibits, func(b *dmr.Burst, slot dmr.SlotType) {
		if slot.DataType != dmr.DTTerminatorWithLC {
			return
		}
		termTotal++
		bits, errs := framing.DecodeBPTC196_96(b.PayloadBits())
		if errs < 0 {
			return
		}
		bptcOK++
		info := make([]byte, 12)
		for i := 0; i < 96; i++ {
			if bits[i]&1 != 0 {
				info[i>>3] |= 1 << uint(7-(i&7))
			}
		}
		if !framing.VerifyRS12_9(info, framing.RS129SeedTerminatorLC) {
			return
		}
		rsOK++
		flc, err := dmr.ParseFLC(info)
		if err != nil {
			return
		}
		if gv, ok := flc.AsGroupVoiceUser(); ok {
			calls[[2]uint32{gv.GroupAddress, gv.SourceID}]++
		}
	})

	t.Logf("Voice Terminator-with-LC: total=%d BPTC_ok=%d RS_ok=%d calls=%v", termTotal, bptcOK, rsOK, calls)
	if rsOK < 8 {
		t.Errorf("RS(12,9)-validated Full LC count = %d, want >= 8 (real-air voiceheader-rs path regressed)", rsOK)
	}
	// The captured call is TG 24 from source 4209000 (cross-checked against the
	// CSBK voice grants on the same site's control channel).
	const wantTG, wantSrc = 24, 4209000
	if calls[[2]uint32{wantTG, wantSrc}] < 8 {
		t.Errorf("recovered FLC tg=%d src=%d count = %d, want >= 8; saw %v",
			wantTG, wantSrc, calls[[2]uint32{wantTG, wantSrc}], calls)
	}
}
