package main

import (
	"encoding/binary"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	dmrrx "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr/tier2"
	dmrvoice "github.com/MattCheramie/GopherTrunk/internal/radio/dmr/voice"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// The issue #836 real-air fixtures: two 48 kHz slices of the reporter's
// 446.500 MHz direct-mode captures (see testdata/dmr-directmode-README.md).
// Every synthetic direct-mode test in this repo is self-consistent by
// construction; these pin the three things only the reporter's air could:
//
//   - the carrier gate (#1186) decodes a real handheld's burst cadence and
//     the ungated receiver decodes nothing from it;
//   - the feed-forward timing acquisition makes a keyup decode at EVERY
//     sub-symbol start phase — the pre-fix receiver decoded the second PTT
//     0.02 s after its carrier at seven of ten phases and 1.3–2.8 s after it
//     at the other three (its whole header train lost);
//   - a ten-copy header train is ONE grant (the 0.25 s re-key rule measured
//     from the first copy used to release and re-grant every keyup).
func readDirectModeSlice(t *testing.T, name string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/4)
	for i := range iq {
		re := int16(binary.LittleEndian.Uint16(raw[i*4:]))
		im := int16(binary.LittleEndian.Uint16(raw[i*4+2:]))
		iq[i] = complex(float32(re)/32768, float32(im)/32768)
	}
	return iq
}

// directModeDibits runs a 48 kHz slice through the production receiver in
// RTL-sized chunks and returns its dibit stream.
func directModeDibits(iq []complex64, noGate bool) []uint8 {
	var all []uint8
	rx := dmrrx.New(dmrrx.Options{
		SampleRateHz: 48_000, DeviationHz: 1944.0, ClockGain: 0.015, NoCarrierGate: noGate,
		DibitSink: func(d []uint8, _ int) { all = append(all, d...) },
	})
	for i := 0; i < len(iq); i += 4096 {
		e := i + 4096
		if e > len(iq) {
			e = len(iq)
		}
		rx.Process(iq[i:e])
	}
	return all
}

func TestDMRDirectModeRealAirKeyup(t *testing.T) {
	iq := readDirectModeSlice(t, "dmr-directmode-446500-keyup-48k.cs16")
	if ungated := directModeDibits(iq, true); len(syncMatches(ungated)) != 0 {
		t.Fatalf("the ungated receiver found %d sync words in the keyup slice — the #836 gate is no longer load-bearing", len(syncMatches(ungated)))
	}
	dibits := directModeDibits(iq, false)
	var data, voice int
	for _, m := range syncMatches(dibits) {
		switch m.Pattern.Name {
		case "MS-Data":
			data++
		case "MS-Voice":
			voice++
		}
	}
	if data < 8 || voice < 2 {
		t.Fatalf("gated receiver: %d MS data syncs (want ≥ 8 header copies) and %d MS voice syncs (want ≥ 2)", data, voice)
	}

	// The Tier II conventional state machine must grant this keyup exactly
	// once, from the header train, as tg 99 / radio 3024109.
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	cc := tier2.New(tier2.Options{Bus: bus, Log: slog.New(slog.NewTextHandler(os.Stderr, nil)), SystemName: "dm-836", FrequencyHz: 446_500_000, InterleavedVoice: true})
	for i := 0; i < len(dibits); i += 4096 {
		e := i + 4096
		if e > len(dibits) {
			e = len(dibits)
		}
		cc.Process(dibits[i:e], i)
	}
	var grants []trunking.Grant
	deadline := time.After(200 * time.Millisecond)
drain:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				if g, ok := ev.Payload.(trunking.Grant); ok {
					grants = append(grants, g)
				}
			}
		case <-deadline:
			break drain
		}
	}
	if len(grants) != 1 {
		t.Fatalf("keyup granted %d times (rekeys=%d), want exactly 1: %+v", len(grants), cc.Counters().Rekeys, grants)
	}
	if g := grants[0]; g.GroupID != 99 || g.SourceID != 3024109 || g.ChannelID != 1 {
		t.Fatalf("grant = tg %d src %d cc %d, want tg 99 src 3024109 cc 1", g.GroupID, g.SourceID, g.ChannelID)
	}

	// The voice that follows slices at the direct-mode 288-dibit cadence
	// (the cadence-detecting decoder) with its embedded LC intact.
	dec := dmrvoice.NewInterleavedDecoder()
	var superframes, withLC int
	for i := 0; i < len(dibits); i += 4096 {
		e := i + 4096
		if e > len(dibits) {
			e = len(dibits)
		}
		for _, sf := range dec.Process(dibits[i:e], i) {
			superframes++
			if gv, ok := sf.LC.AsGroupVoiceUser(); sf.HasLC && ok && gv.GroupAddress == 99 {
				withLC++
			}
		}
	}
	if superframes < 2 || withLC < 1 {
		t.Fatalf("voice: %d superframes (%d with embedded LC tg 99), want ≥ 2 / ≥ 1", superframes, withLC)
	}
}

// TestDMRDirectModeRealAirOnsetAtEveryPhase: the second PTT's onset must be
// acquired within ~0.15 s of its carrier whatever sub-symbol phase it lands
// on — prepending k samples of the slice's own leading noise moves the
// burst grid by k/10 of a symbol. Measured on the pre-fix receiver: first
// sync 0.02 s after the carrier at k ∈ {0..5, 9}, 2.62 / 2.98 / 1.54 s at
// k = 6 / 7 / 8. The carrier starts 0.30 s into the slice.
func TestDMRDirectModeRealAirOnsetAtEveryPhase(t *testing.T) {
	iq := readDirectModeSlice(t, "dmr-directmode-446500-ptt2-48k.cs16")
	const carrierAt = 0.30
	for k := 0; k < 10; k++ {
		shifted := append(append([]complex64(nil), iq[:k]...), iq...)
		ms := syncMatches(directModeDibits(shifted, false))
		if len(ms) == 0 {
			t.Errorf("start phase %d/10: no sync word at all", k)
			continue
		}
		first := float64(ms[0].Index) / 4800
		data := 0
		for _, m := range ms {
			if m.Pattern.Name == "MS-Data" {
				data++
			}
		}
		t.Logf("start phase %d/10: first sync %.2f s after the carrier, %d header copies decoded", k, first-carrierAt, data)
		if first-carrierAt > 0.15 {
			t.Errorf("start phase %d/10: first sync %.2f s after the carrier (want ≤ 0.15 s) — the header train is being lost to symbol-timing pull-in", k, first-carrierAt)
		}
		if data < 7 {
			t.Errorf("start phase %d/10: only %d header copies decoded, want ≥ 7", k, data)
		}
	}
}

func syncMatches(dibits []uint8) []dmr.Match {
	det := dmr.NewSyncDetector(dmr.AllSyncs, 2)
	ms, _ := det.Process(nil, dibits, 0)
	return ms
}
