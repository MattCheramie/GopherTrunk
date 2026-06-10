package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// TestReplayMMRSite9DecodesRealP25 is a real-capture regression guard.
//
// testdata/mmr-s9-cc.cfile is a ~1.09 s slice of the MMR Site 9 P25 control
// channel (420.0500 MHz, NAC 0x167 / WACN BEE00 / SYS 164 — see issue #402),
// originally a 2.4 MSPS RTL-SDR recording. It has been channelised down to a
// 48 kHz centred baseband (GNU Radio f32) so it decodes standalone through
// the production C4FM receiver without a down-converter, and stays small
// enough to commit (~410 KB).
//
// The capture decodes to real trunking frames (NAC ≈ 0x164–0x167, DUID TSDU,
// TSBKs that pass CRC), so this asserts genuine decode — not the degenerate
// NAC=0 collapse an earlier (wrong-sample-rate) revision mistook for a lock.
//
// The ~14 frames in this slice carry roughly three TSBKs each (a TSDU packs
// up to three blocks per FSW+NID); decoding all of them — not just the first
// block of each, the pre-#402 behaviour — yields ~41 TSBKs. minTSBK guards
// that multi-block decode: a regression back to one-block-per-frame would
// drop to ~14 and trip it. Decode is still imperfect on this site (the NID
// occasionally lands a few bits off, e.g. 0x164 vs 0x167); the thresholds
// here are floors the remaining #402 demod-quality work should raise.
func TestReplayMMRSite9DecodesRealP25(t *testing.T) {
	const (
		sampleRateHz  = 48_000.0
		minNIDTrusted = 8
		minTSBK       = 24
	)

	path := filepath.Join("testdata", "mmr-s9-cc.cfile")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	pairs := len(raw) / 8
	if pairs == 0 {
		t.Fatalf("fixture %s has no samples", path)
	}
	iq := make([]complex64, pairs)
	dec, _ := siglab.FormatF32.Decoder()
	dec(raw[:pairs*8], iq)

	bus := events.NewBus(1024)
	sub := bus.Subscribe()
	nacs := make(map[uint16]int)
	doneEvents := make(chan struct{})
	go func() {
		defer close(doneEvents)
		for ev := range sub.C {
			if ev.Kind != events.KindCCLocked {
				continue
			}
			if ls, ok := ev.Payload.(p25phase1.LockState); ok {
				nacs[ls.NAC]++
			}
		}
	}()

	cc := p25phase1.New(p25phase1.Options{
		Bus:        bus,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName: "mmr-s9-test",
		Rotations:  p25phase1.RotationsC4FM,
	})
	rx := p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz: sampleRateHz,
		DeviationHz:  1800.0,
		DemodMode:    p25phase1rx.DemodC4FM,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
	})

	// Pre-centred 48 kHz baseband → feed the receiver directly (no DDC).
	const chunk = 8192
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[off:end])
	}

	bus.Close()
	<-doneEvents

	st := cc.Stats()
	t.Logf("Site 9: NIDTrusted=%d NIDFailed=%d TSBKDecoded=%d TSBKCRCFailed=%d lockedNACs=%v",
		st.NIDTrusted, st.NIDFailed, st.TSBKDecoded, st.TSBKCRCFailed, nacs)

	if st.NIDTrusted < minNIDTrusted {
		t.Errorf("NIDTrusted = %d, want >= %d (real-capture C4FM decode regressed)", st.NIDTrusted, minNIDTrusted)
	}
	if st.TSBKDecoded < minTSBK {
		t.Errorf("TSBKDecoded = %d, want >= %d (real-capture C4FM decode regressed)", st.TSBKDecoded, minTSBK)
	}
	// The real site NAC is 0x167; marginal decode also lands 0x164–0x166.
	// Require at least one lock to a plausible real NAC (never the NAC=0
	// all-zero collapse).
	realNAC := false
	for nac := range nacs {
		if nac >= 0x160 && nac <= 0x167 {
			realNAC = true
		}
	}
	if !realNAC {
		t.Errorf("no lock to a real Site 9 NAC (0x160–0x167); saw %v", nacs)
	}
}

// TestReplayMMRCityDecodesCleanP25 is a clean-decode regression guard for
// the MMR City control channel (420.0125 MHz, NAC 0x164 — see issue #402).
//
// testdata/mmr-city-cc.cfile is a 1.5 s slice of an off-channel 960 kSPS
// RTL-SDR capture (serial 76361606, gain 37), channelised down to a 48 kHz
// centred baseband (GNU Radio f32) so it decodes standalone through the
// production C4FM receiver without a down-converter (~576 KB).
//
// This site is the one the reporter saw elevated TSBK-CRC / NID failures on
// *live* while SDRTrunk decoded it better, which seeded a "multipath ISI →
// needs an adaptive equalizer" hypothesis. The capture refutes that for the
// decode chain: the current C4FM path decodes it essentially perfectly
// (EVM ≈ 7%, SNR ≈ 20 dB, zero TSBK-CRC failures), and both the CQPSK CMA
// equalizer and the adaptive C4FM slicer make it *worse* on this signal. So
// the assertions below pin the clean decode: a future change that regresses
// the C4FM path on a healthy real signal — e.g. switching a blind equalizer
// on by default — trips TSBKCRCFailed first.
func TestReplayMMRCityDecodesCleanP25(t *testing.T) {
	const (
		sampleRateHz  = 48_000.0
		minNIDTrusted = 12
		minTSBK       = 40
	)

	path := filepath.Join("testdata", "mmr-city-cc.cfile")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	pairs := len(raw) / 8
	if pairs == 0 {
		t.Fatalf("fixture %s has no samples", path)
	}
	iq := make([]complex64, pairs)
	dec, _ := siglab.FormatF32.Decoder()
	dec(raw[:pairs*8], iq)

	bus := events.NewBus(1024)
	sub := bus.Subscribe()
	nacs := make(map[uint16]int)
	doneEvents := make(chan struct{})
	go func() {
		defer close(doneEvents)
		for ev := range sub.C {
			if ev.Kind != events.KindCCLocked {
				continue
			}
			if ls, ok := ev.Payload.(p25phase1.LockState); ok {
				nacs[ls.NAC]++
			}
		}
	}()

	cc := p25phase1.New(p25phase1.Options{
		Bus:        bus,
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		SystemName: "mmr-city-test",
		Rotations:  p25phase1.RotationsC4FM,
	})
	rx := p25phase1rx.New(p25phase1rx.Options{
		SampleRateHz: sampleRateHz,
		DeviationHz:  1800.0,
		DemodMode:    p25phase1rx.DemodC4FM,
		DibitSink: func(dibits []uint8, baseIdx int) {
			cc.Process(dibits, baseIdx)
		},
	})

	// Pre-centred 48 kHz baseband → feed the receiver directly (no DDC).
	const chunk = 8192
	for off := 0; off < len(iq); off += chunk {
		end := off + chunk
		if end > len(iq) {
			end = len(iq)
		}
		rx.Process(iq[off:end])
	}

	bus.Close()
	<-doneEvents

	st := cc.Stats()
	t.Logf("MMR City: NIDTrusted=%d NIDFailed=%d TSBKDecoded=%d TSBKCRCFailed=%d lockedNACs=%v",
		st.NIDTrusted, st.NIDFailed, st.TSBKDecoded, st.TSBKCRCFailed, nacs)

	if st.NIDTrusted < minNIDTrusted {
		t.Errorf("NIDTrusted = %d, want >= %d (real-capture C4FM decode regressed)", st.NIDTrusted, minNIDTrusted)
	}
	if st.TSBKDecoded < minTSBK {
		t.Errorf("TSBKDecoded = %d, want >= %d (real-capture C4FM decode regressed)", st.TSBKDecoded, minTSBK)
	}
	// The signal is clean (EVM ≈ 7%, SNR ≈ 20 dB): a healthy C4FM path
	// decodes every frame. Any CRC/NID failure here is a real regression.
	if st.TSBKCRCFailed != 0 {
		t.Errorf("TSBKCRCFailed = %d, want 0 (C4FM decode regressed on a clean real signal)", st.TSBKCRCFailed)
	}
	if st.NIDFailed != 0 {
		t.Errorf("NIDFailed = %d, want 0 (C4FM decode regressed on a clean real signal)", st.NIDFailed)
	}
	// MMR City's control channel decodes to NAC 0x164.
	if nacs[0x164] == 0 {
		t.Errorf("no lock to the MMR City NAC 0x164; saw %v", nacs)
	}
}
