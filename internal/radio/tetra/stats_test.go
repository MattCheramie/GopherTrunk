package tetra

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// debugCC builds a ControlChannel whose logger writes to buf at the given
// level, with channel coding on — the configuration the live/replay connector
// uses. Returns the control channel and its event bus (caller closes the bus).
func debugCC(t *testing.T, buf io.Writer, level slog.Level) (*ControlChannel, *events.Bus) {
	t.Helper()
	bus := events.NewBus(8)
	cc := New(Options{
		Bus:         bus,
		Log:         slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})),
		SystemName:  "Sys",
		FrequencyHz: 412_000_000,
	})
	cc.SetChannelCoding(ChannelCodingOn)
	return cc, bus
}

// buildCleanSBStream assembles a complete, error-free synchronisation burst as
// a hard dibit stream: pad, BSCH(60) carrying the cell identity, STS(19),
// broadcast(15), BNCH(108) carrying a SYSINFO PDU scrambled with the cell's
// extended colour code, then trailing margin so the look-ahead is satisfied.
// The BNCH info bits go through pduToType1Bits (the same helper the SCH/HD
// round-trip test uses), so the recovered PDU parses back to a SystemBroadcast.
func buildCleanSBStream(t *testing.T, cc6 uint8, mcc, mnc uint16, sysinfo PDU) []uint8 {
	t.Helper()
	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, uint32(cc6))
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))
	bschDibits := TetraBitsToDibits(EncodeBSCH(syncInfo))

	ext := ExtendedColourCode(mcc, mnc, cc6)
	info := pduToType1Bits(sysinfo, 124)
	if info == nil {
		t.Fatalf("sysinfo PDU too large for SCH/HD 124 type-1 bits")
	}
	bnchDibits := TetraBitsToDibits(EncodeSCHHD(info, ext))

	var s []uint8
	s = append(s, make([]uint8, 50)...)                // pad before BSCH
	s = append(s, bschDibits...)                       // BSCH (60)
	s = append(s, SyncTrainingDibits()...)             // STS (19)
	s = append(s, make([]uint8, sbBroadcastDibits)...) // broadcast (15)
	s = append(s, bnchDibits...)                       // BNCH (108)
	s = append(s, make([]uint8, 300)...)               // trailing look-ahead margin
	return s
}

// buildCleanSBStreamMN is buildCleanSBStream with the BSCH SYNC PDU's multiframe
// number (MN, bits 17..22) set, so tests can drive the multiframe-keyed sync log
// across multiframe boundaries and gaps.
func buildCleanSBStreamMN(t *testing.T, cc6 uint8, mcc, mnc uint16, mn uint8, sysinfo PDU) []uint8 {
	t.Helper()
	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, uint32(cc6))
	putBits(syncInfo, 17, 6, uint32(mn))
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))
	bschDibits := TetraBitsToDibits(EncodeBSCH(syncInfo))

	ext := ExtendedColourCode(mcc, mnc, cc6)
	info := pduToType1Bits(sysinfo, 124)
	if info == nil {
		t.Fatalf("sysinfo PDU too large for SCH/HD 124 type-1 bits")
	}
	bnchDibits := TetraBitsToDibits(EncodeSCHHD(info, ext))

	var s []uint8
	s = append(s, make([]uint8, 50)...)
	s = append(s, bschDibits...)
	s = append(s, SyncTrainingDibits()...)
	s = append(s, make([]uint8, sbBroadcastDibits)...)
	s = append(s, bnchDibits...)
	s = append(s, make([]uint8, 300)...)
	return s
}

// cleanSysinfoPDU is a minimal valid MLE SYSINFO PDU (MCC=10b / MNC=14b /
// LA=14b), the same payload layout the topology / round-trip tests use.
func cleanSysinfoPDU() PDU {
	return PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: []byte{0x00, 0xC0, 0x05, 0x01, 0x00}, // MCC=3 MNC=5 LA=64
	}
}

// TestDrainStatsCountsSyncBurst confirms the debug decode-health counters move
// on a clean synchronisation burst (SB) and reset on drain. A clean SB decodes
// its BSCH (learning the colour code) and its BNCH SYSINFO, so one burst yields
// SBBursts>=1, BSCHOK>=1, BSCHFail=0 and SysInfo>=1, and Dibits equal to the
// stream length. A second drain must return to zero.
func TestDrainStatsCountsSyncBurst(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelDebug)
	defer bus.Close()

	stream := buildCleanSBStream(t, 0x2D, 3, 5, cleanSysinfoPDU())
	cc.Process(stream, 0)

	got := cc.DrainStats()
	if got.Dibits != int64(len(stream)) {
		t.Errorf("Dibits = %d, want %d (stream length)", got.Dibits, len(stream))
	}
	if got.SBBursts < 1 {
		t.Errorf("SBBursts = %d, want >= 1", got.SBBursts)
	}
	if got.BSCHOK < 1 {
		t.Errorf("BSCHOK = %d, want >= 1", got.BSCHOK)
	}
	if got.SysInfo < 1 {
		t.Errorf("SysInfo = %d, want >= 1 (clean BNCH should decode)", got.SysInfo)
	}
	if got.BSCHFail != 0 {
		t.Errorf("BSCHFail = %d, want 0 on a clean burst", got.BSCHFail)
	}

	// Draining resets: a second drain with no further input is all-zero.
	if again := cc.DrainStats(); again != (Stats{}) {
		t.Errorf("second DrainStats = %+v, want zero", again)
	}

	// The location area from the BNCH SYSINFO made it into the lock state.
	if topo := cc.Topology(); topo.LocationArea != 64 {
		t.Errorf("LocationArea = %d, want 64 (from decoded BNCH SYSINFO)", topo.LocationArea)
	}
}

// TestStatsInertAtInfoLevel confirms the counters are gated on debug: at
// info level the accumulation is a no-op, so DrainStats stays zero even
// after decoding a burst. This is the zero-overhead production path.
func TestStatsInertAtInfoLevel(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelInfo)
	defer bus.Close()

	stream := buildCleanSBStream(t, 0x2D, 3, 5, cleanSysinfoPDU())
	cc.Process(stream, 0)

	if got := cc.DrainStats(); got != (Stats{}) {
		t.Errorf("DrainStats at info level = %+v, want zero (gated on debug)", got)
	}
	// The lock still happened — gating stats must not disable decoding.
	if topo := cc.Topology(); topo.MCC == 0 {
		t.Errorf("burst did not decode at info level (MCC=0); stats gating must not affect decode")
	}
}

// TestSyncBurstDebugLines confirms the higher-level, multiframe-keyed sync log
// (ETSI EN 300 392-2 §4.5.2) replaces the old per-burst spam: the first burst
// logs "tetra sync acquired", further bursts in the same multiframe are silent,
// a new multiframe logs "tetra sync", a non-consecutive multiframe logs
// "tetra sync gap", and all of it is suppressed at info level. Streams are fed at
// consecutive baseIdx to keep the SB decoder's absolute-index buffer monotonic.
func TestSyncBurstDebugLines(t *testing.T) {
	t.Run("debug emits multiframe-keyed lines, not per-burst spam", func(t *testing.T) {
		var buf bytes.Buffer
		cc, bus := debugCC(t, &buf, slog.LevelDebug)
		defer bus.Close()

		base := 0
		feed := func(mn uint8) {
			s := buildCleanSBStreamMN(t, 0x2D, 3, 5, mn, cleanSysinfoPDU())
			cc.Process(s, base)
			base += len(s)
		}

		feed(1) // first burst → acquired
		feed(1) // same multiframe → silent
		feed(2) // next multiframe → sync
		feed(5) // skipped MN 3,4 → gap (missed 2)

		out := buf.String()
		for _, want := range []string{"tetra sync acquired", "tetra sync gap"} {
			if !strings.Contains(out, want) {
				t.Errorf("debug output missing %q\n---\n%s", want, out)
			}
		}
		// The old per-burst spam must be gone.
		for _, gone := range []string{"tetra: sync burst decoded", "tetra: sysinfo", "BSCH decode failed"} {
			if strings.Contains(out, gone) {
				t.Errorf("debug output still contains removed per-burst line %q\n---\n%s", gone, out)
			}
		}
		// Exactly one "acquired" line (same-MN second burst did not re-acquire).
		if n := strings.Count(out, "tetra sync acquired"); n != 1 {
			t.Errorf("acquired lines = %d, want 1 (same-MN burst must not re-log)\n%s", n, out)
		}
		// The gap collapses the missed multiframes into one "gap" line reporting 2.
		if !strings.Contains(out, "missed=2") {
			t.Errorf("gap line should report missed=2\n---\n%s", out)
		}
	})
	t.Run("info suppresses", func(t *testing.T) {
		var buf bytes.Buffer
		cc, bus := debugCC(t, &buf, slog.LevelInfo)
		defer bus.Close()
		cc.Process(buildCleanSBStreamMN(t, 0x2D, 3, 5, 1, cleanSysinfoPDU()), 0)
		if out := buf.String(); strings.Contains(out, "tetra sync") {
			t.Errorf("info output should not contain debug sync lines\n---\n%s", out)
		}
	})
}

// buildNCDBStream assembles one Normal Continuous Downlink Burst as a hard dibit
// stream with the real on-air geometry: BKN1 look-back headroom, BKN1(108),
// AACH-half-1(7), the normal training sequence, AACH-half-2(8), BKN2(108), then a
// look-ahead tail. Mirrors the layout in TestProcessDecodesGrantFromNCDB and the
// ndb* geometry constants. aach1/aach2 may be nil (left as fill, so the AACH does
// not decode and the slot falls back to trying all rotations).
func buildNCDBStream(bkn1, aach1, aach2, bkn2 []uint8) []uint8 {
	if aach1 == nil {
		aach1 = make([]uint8, ndbAACH1Len)
	}
	if aach2 == nil {
		aach2 = make([]uint8, ndbAACH2Len)
	}
	var s []uint8
	s = append(s, make([]uint8, 24)...) // priming / BKN1 look-back headroom
	s = append(s, bkn1...)
	s = append(s, aach1...)
	s = append(s, NormalSyncDibits()...)
	s = append(s, aach2...)
	s = append(s, bkn2...)
	s = append(s, make([]uint8, 8)...) // look-ahead tail
	return s
}

// TestDrainStatsCountsSCHPDUFromNCDB confirms SCHPDUs tracks CRC-clean signalling
// blocks recovered off a *real* NCDB slot (the decodeDownlinkSlot path), not the
// legacy fixed-slice dispatchSlice path — which never fired on air and left the
// counter stuck at 0. Fail-first: before SCHPDUs was re-homed onto the NCDB path
// this stayed 0.
func TestDrainStatsCountsSCHPDUFromNCDB(t *testing.T) {
	const colour = 0x1234
	cc, bus := debugCC(t, io.Discard, slog.LevelDebug)
	defer bus.Close()
	cc.SetColourCode(colour)

	// A real MAC-RESOURCE SCH/F type-1 payload (same capture as
	// TestProcessDecodesGrantFromNCDB) laid into BKN1+BKN2.
	info := hexToBits(t, "20760f572c3c83d538c90a1fe305009e0f92813c83d538c91e1fe301304a1eae5880")[:268]
	dibits := TetraBitsToDibits(EncodeSCHF(info, colour)) // 216 dibits
	cc.Process(buildNCDBStream(dibits[:108], nil, nil, dibits[108:]), 0)

	if got := cc.DrainStats(); got.SCHPDUs < 1 {
		t.Errorf("SCHPDUs = %d, want >= 1 (CRC-clean SCH/F off a real NCDB slot)", got.SCHPDUs)
	}
}

// TestDrainStatsCountsSCHPDUFail confirms SCHPDUsFail counts an AACH-confirmed
// control slot that yields no CRC-clean SCH block — a real signalling-payload
// decode failure — while leaving SCHPDUs at 0. The AACH decodes as common control
// (Header 0), but the BKN blocks are fill, so neither SCH/F nor SCH/HD passes CRC.
func TestDrainStatsCountsSCHPDUFail(t *testing.T) {
	const colour = 0x1234
	cc, bus := debugCC(t, io.Discard, slog.LevelDebug)
	defer bus.Close()
	cc.SetColourCode(colour)

	// A valid ACCESS-ASSIGN whose header (0) marks the downlink as common control.
	aachDibits := TetraBitsToDibits(EncodeAACH(make([]byte, 14), colour)) // 15 dibits
	aach1, aach2 := aachDibits[:ndbAACH1Len], aachDibits[ndbAACH1Len:]
	// Fill BKN blocks so no SCH/F or SCH/HD decodes CRC-clean.
	bkn1, bkn2 := make([]uint8, ndbBlockDibits), make([]uint8, ndbBlockDibits)
	cc.Process(buildNCDBStream(bkn1, aach1, aach2, bkn2), 0)

	got := cc.DrainStats()
	if got.SCHPDUsFail < 1 {
		t.Errorf("SCHPDUsFail = %d, want >= 1 (control slot with no CRC-clean SCH block)", got.SCHPDUsFail)
	}
	if got.SCHPDUs != 0 {
		t.Errorf("SCHPDUs = %d, want 0 (no block should have decoded)", got.SCHPDUs)
	}
}

// TestDrainStatsCountsSysInfo confirms SysInfo counts SYSINFO MAC broadcast PDUs
// decoded off the real NCDB/MAC path (learnSysInfo), not the byte-aligned L3
// AsSystemBroadcast gate that rarely matched real bit-packed MAC SYSINFO and left
// the counter at 0 on air. Fail-first: before the re-home this stayed 0.
func TestDrainStatsCountsSysInfo(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelDebug)
	defer bus.Close()
	// A real recovered SYSINFO MAC broadcast (same capture as
	// TestGrantPublishesResolvedFrequency).
	cc.ingestMAC(hexToBits(t, "8a9c4c0e928eec8bd0c0041cffffd700"))
	if got := cc.DrainStats(); got.SysInfo < 1 {
		t.Errorf("SysInfo = %d, want >= 1 (SYSINFO MAC broadcast via learnSysInfo)", got.SysInfo)
	}
}

// TestBSCHCountsAlwaysOn pins that BSCHCounts accumulates cumulative BSCH decode
// counts even when debug telemetry is OFF — the always-on quality counter the live
// signal indicator reads. DrainStats stays zero at info level (it is debug-gated),
// while BSCHCounts still moves. Fail-first: without the always-on counters
// BSCHCounts returns 0 and the ok>=1 assertion goes red.
func TestBSCHCountsAlwaysOn(t *testing.T) {
	cc, bus := debugCC(t, io.Discard, slog.LevelInfo) // debug OFF
	defer bus.Close()

	stream := buildCleanSBStream(t, 0x2D, 3, 5, cleanSysinfoPDU())
	cc.Process(stream, 1_000_000)

	ok, fail := cc.BSCHCounts()
	if ok < 1 {
		t.Fatalf("BSCHCounts ok = %d, want >= 1 (always-on, independent of debug)", ok)
	}
	if fail < 0 {
		t.Fatalf("BSCHCounts fail = %d, want >= 0", fail)
	}
	// The debug-gated drain stays zero at info level, proving BSCHCounts is a
	// separate always-on path.
	if drained := cc.DrainStats().BSCHOK; drained != 0 {
		t.Fatalf("DrainStats BSCHOK = %d at info level, want 0 (debug-gated)", drained)
	}
}
