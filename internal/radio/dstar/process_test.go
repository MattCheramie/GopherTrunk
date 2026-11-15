package dstar

import (
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestProcess_DecodesHeaderAfterFrameSync feeds the Process adapter
// the start of a transmission (preamble, frame sync, FEC-encoded
// group-call header) under the default FEC mode. The state machine
// should publish events.KindCCLocked + an events.KindGrant on the bus.
func TestProcess_DecodesHeaderAfterFrameSync(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "TestRepeater",
		FrequencyHz: 145_670_000,
	})

	stream := buildDStarHeaderStream("CQCQCQ  ", "MYCALL  ", "RPT2CALL", "RPT1CALL")
	if got := cc.Process(stream, 0); got != len(stream) {
		t.Errorf("Process returned %d, want %d", got, len(stream))
	}

	var sawLock, sawGrant bool
	deadline := time.After(200 * time.Millisecond)
DrainLoop:
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				sawLock = true
			case events.KindGrant:
				sawGrant = true
			}
			if sawLock && sawGrant {
				break DrainLoop
			}
		case <-deadline:
			break DrainLoop
		}
	}
	if !sawLock {
		t.Errorf("no KindCCLocked event observed")
	}
	if !sawGrant {
		t.Errorf("no KindGrant event observed (UR=CQCQCQ should fire)")
	}
}

// TestProcess_RejectsBadCRC feeds the Process adapter a header with a
// deliberately corrupted CRC. The state machine should silently drop
// the frame.
func TestProcess_RejectsBadCRC(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "TestRepeater",
		FrequencyHz: 145_670_000,
	})

	h := SealHeader(Header{RPT2: "RPT2CALL", RPT1: "RPT1CALL", UR: "CQCQCQ  ", MY1: "MYCALL  ", MY2: "SUFX"})
	h.CRC ^= 1 // a CRC the header does not match
	stream := append(BitSyncBits(64), EncodeHeaderBurst(h)...)
	cc.Process(stream, 0)

	select {
	case ev := <-sub.C:
		t.Errorf("expected no events for bad-CRC frame, got %v", ev.Kind)
	default:
	}
}

// buildDStarHeaderStream assembles the start of a transmission as a
// radio sends it: bit-sync preamble, frame sync, and the FEC-encoded
// header (UR/MY1/RPT2/RPT1) with a valid CRC.
func buildDStarHeaderStream(ur, my1, rpt2, rpt1 string) []byte {
	hdr := SealHeader(Header{RPT2: rpt2, RPT1: rpt1, UR: ur, MY1: my1, MY2: "SUFX"})
	out := BitSyncBits(64)
	return append(out, EncodeHeaderBurst(hdr)...)
}

// TestProcess_DecodesHeaderUnderFECOn feeds the Process adapter a bit
// stream with the 24-bit Frame Sync followed by the 660-bit FEC-
// encoded header (conv + scramble + interleave). Under FECOn the
// adapter should run framing.DecodeDStarHeaderFEC, recover the
// 41-byte information field, verify the CRC, and publish a Lock +
// Grant on the bus.
func TestProcess_DecodesHeaderUnderFECOn(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "TestRepeater",
		FrequencyHz: 145_670_000,
	})
	cc.SetFECMode(FECOn)

	stream := buildDStarHeaderStreamFECOn("CQCQCQ  ", "MYCALL  ", "RPT2CALL", "RPT1CALL")
	cc.Process(stream, 0)

	var sawLock, sawGrant bool
	deadline := time.After(200 * time.Millisecond)
DrainLoop:
	for {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				sawLock = true
			case events.KindGrant:
				sawGrant = true
			}
			if sawLock && sawGrant {
				break DrainLoop
			}
		case <-deadline:
			break DrainLoop
		}
	}
	if !sawLock {
		t.Errorf("no KindCCLocked event observed under FECOn")
	}
	if !sawGrant {
		t.Errorf("no KindGrant event observed under FECOn (UR=CQCQCQ should fire)")
	}
}

// TestProcess_FECOnSurvivesSingleBitError mirrors the framing-package
// test: a single bit error in the 660-bit FEC-encoded channel stream
// should still decode cleanly thanks to the K=3 rate-1/2 code.
func TestProcess_FECOnSurvivesSingleBitError(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{
		Bus:         bus,
		Log:         slog.Default(),
		SystemName:  "TestRepeater",
		FrequencyHz: 145_670_000,
	})
	cc.SetFECMode(FECOn)

	stream := buildDStarHeaderStreamFECOn("CQCQCQ  ", "MYCALL  ", "RPT2CALL", "RPT1CALL")
	// Flip a single bit inside the FEC section, which runs from bit
	// 64+24 = 88 to 747.
	stream[200] ^= 1
	cc.Process(stream, 0)

	var sawLock bool
	deadline := time.After(200 * time.Millisecond)
DrainLoop:
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				sawLock = true
				break DrainLoop
			}
		case <-deadline:
			break DrainLoop
		}
	}
	if !sawLock {
		t.Errorf("Viterbi failed to absorb a 1-bit error")
	}
}

// buildDStarHeaderStreamFECOn is buildDStarHeaderStream: FECOn is the
// default and the on-air format.
func buildDStarHeaderStreamFECOn(ur, my1, rpt2, rpt1 string) []byte {
	return buildDStarHeaderStream(ur, my1, rpt2, rpt1)
}
