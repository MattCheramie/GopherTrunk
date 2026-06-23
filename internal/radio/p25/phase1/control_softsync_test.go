package phase1

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestControlChannelSoftSyncFallback exercises the opt-in wider FSW-fallback
// tolerance (issue #771). A sync word carrying 6 symbol errors — above the
// strict 4-of-24 gate but within the fallback bound — must NOT lock the strict
// detector, must lock the fallback detector when the NID + TSBK still decode,
// and must NOT lock the fallback detector when the TSBK CRC is broken (the
// false-lock guard: the looser sync is admitted only through CRC corroboration).
func TestControlChannelSoftSyncFallback(t *testing.T) {
	const (
		nac    = 0x293
		fswOff = 30 // FSW occupies dibits [fswOff, fswOff+24): status symbols
		// start only after 35 dibits, so the 24-dibit FSW is contiguous here.
	)
	makeStream := func() []uint8 {
		// Status-interleaved on-air stream, so the TSBK-CRC corroboration path
		// sees the same status-symbol layout it does on a real capture.
		return buildLockedStreamWithTSBK(fswOff, nac, DUIDTrunkingSignaling,
			TSBK{LB: true, Opcode: OpRFSSStatusBroadcast})
	}
	// corruptFSW flips n FSW dibits (each +1 mod 4 is a guaranteed rot-0
	// mismatch), so the best correlation distance becomes exactly n.
	corruptFSW := func(s []uint8, n int) {
		for i := 0; i < n; i++ {
			s[fswOff+i] = (s[fswOff+i] + 1) & 3
		}
	}
	locked := func(t *testing.T, fallbackTol int, mutate func([]uint8)) bool {
		t.Helper()
		// A generous buffer so nothing is dropped: a wide-tolerance hit can
		// publish a few KindDecodeError events (from spurious near-miss FSW
		// positions) before the genuine KindCCLocked.
		bus := events.NewBus(256)
		defer bus.Close()
		sub := bus.Subscribe()
		defer sub.Close()
		cc := New(Options{Bus: bus, FrequencyHz: 851_000_000, FSWFallbackTolerance: fallbackTol})
		s := makeStream()
		if mutate != nil {
			mutate(s)
		}
		cc.Process(s, 0)
		// Scan the published events for a control-channel lock; ignore the
		// decode-error noise a marginal capture produces.
		for {
			select {
			case ev := <-sub.C:
				if ev.Kind == events.KindCCLocked {
					return ev.Payload.(LockState).NAC == nac
				}
			case <-time.After(250 * time.Millisecond):
				// Events are published synchronously during Process, so a lock
				// would already be buffered; a timeout here means none came.
				return false
			}
		}
	}

	t.Run("clean frame locks strict", func(t *testing.T) {
		if !locked(t, 0, nil) {
			t.Fatal("a clean FSW+NID+TSBK frame must lock the strict detector")
		}
	})
	t.Run("6-error FSW does not lock strict", func(t *testing.T) {
		if locked(t, 0, func(s []uint8) { corruptFSW(s, 6) }) {
			t.Fatal("a 6-error FSW (> strict tolerance 4) must not lock the strict detector")
		}
	})
	t.Run("6-error FSW locks soft-sync fallback", func(t *testing.T) {
		if !locked(t, FSWFallbackToleranceDefault, func(s []uint8) { corruptFSW(s, 6) }) {
			t.Fatal("the fallback must lock a 6-error FSW whose NID + TSBK still corroborate")
		}
	})
	t.Run("6-error FSW + broken TSBK does not false-lock", func(t *testing.T) {
		mutate := func(s []uint8) {
			corruptFSW(s, 6)
			// Wreck the TSBK channel block — well past the FSW(24)+NID(32)+status
			// region (~fswOff+57) — so the CRC cannot corroborate; the fallback
			// must then reject the hit.
			for i := 0; i < 40 && fswOff+62+i < len(s); i++ {
				s[fswOff+62+i] = (s[fswOff+62+i] + 1) & 3
			}
		}
		if locked(t, FSWFallbackToleranceDefault, mutate) {
			t.Fatal("a 6-error FSW with a broken TSBK CRC must not lock (false-lock guard)")
		}
	})
}
