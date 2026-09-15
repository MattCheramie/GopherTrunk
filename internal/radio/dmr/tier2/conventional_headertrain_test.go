package tier2

import (
	"log/slog"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
)

// directModeHeaderTrain lays n Voice LC Header copies on the direct-mode
// frame grid (one 132-dibit burst per 288-dibit / 60 ms frame, the gap
// filled with idle dibits the way the receiver's gate mutes it), preceded
// and followed by silence frames.
func directModeHeaderTrain(lc dmr.FLC, cc uint8, n, leadFrames, tailFrames int) []uint8 {
	var s []uint8
	frame := func(burst []uint8) {
		if burst == nil {
			s = append(s, make([]uint8, dmr.BurstDibits)...)
		} else {
			s = append(s, burst...)
		}
		s = append(s, make([]uint8, 288-dmr.BurstDibits)...)
	}
	for i := 0; i < leadFrames; i++ {
		frame(nil)
	}
	for i := 0; i < n; i++ {
		b := burstWithFLC(lc)
		stampSlotType(b, cc, dmr.DTVoiceLCHeader)
		// Direct-mode data bursts carry the MS data sync.
		copy(b.Dibits[dmr.HalfPayloadDibits+dmr.SlotTypeDibits:], dmr.MSData.Dibits[:])
		frame(b.Dibits[:])
	}
	for i := 0; i < tailFrames; i++ {
		frame(nil)
	}
	return s
}

// TestConventionalHeaderTrainIsOneKeyup is the #836 regression for the
// phantom re-grant: the reporter's direct-mode handheld repeats its Voice LC
// Header ten times, 60 ms apart (0.6 s in total), before the first voice
// burst. Measured from the FIRST copy, copy six was already past the 0.25 s
// re-key rule, so every keyup released the call and re-granted it — one
// zero-length recording plus a real one per PTT. The anchor now follows the
// last copy, so the train is one grant; a genuine re-key (a header well past
// the previous transmission's last header, with voice in between) still
// re-grants.
func TestConventionalHeaderTrainIsOneKeyup(t *testing.T) {
	bus := events.NewBus(64)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	c := New(Options{Bus: bus, Log: slog.Default(), SystemName: "dm", FrequencyHz: 446_500_000})
	lc := groupLC(99, 3024109)
	// One contiguous stream (the state machine's re-key clock is the
	// absolute dibit index): a ten-copy header train, 2 s of silence with
	// the terminator lost, then a second keyup's three-copy train.
	stream := directModeHeaderTrain(lc, 1, 10, 4, 4)
	stream = append(stream, make([]uint8, 2*4800)...)
	stream = append(stream, directModeHeaderTrain(lc, 1, 3, 0, 4)...)
	feedChunks(c, stream, 4096)
	grants := grantsOf(drainEvents(sub, 100*time.Millisecond))
	if len(grants) != 2 {
		t.Fatalf("got %d grants (rekeys=%d), want 2 — one per keyup, none for the copies inside a header train: %+v",
			len(grants), c.Counters().Rekeys, grants)
	}
	for _, g := range grants {
		if g.GroupID != 99 || g.SourceID != 3024109 {
			t.Fatalf("grant = %+v", g)
		}
	}
	if got := c.Counters().Rekeys; got != 1 {
		t.Fatalf("rekeys=%d, want 1 (the second keyup, whose predecessor's terminator was lost)", got)
	}
}
