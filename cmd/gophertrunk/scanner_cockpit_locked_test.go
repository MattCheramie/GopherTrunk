package main

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/cchunt"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/widebandt2"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeHealth stands in for a wideband engine's live channel report.
type fakeHealth []widebandt2.ChannelHealth

func (f fakeHealth) ChannelHealth() []widebandt2.ChannelHealth { return f }

// TestLockedControlChannelsAreTheOnesDecodingNow: the symbol panels' locked
// set is what is decoding right now, from the same two sources the decode
// chip reads (#1386): a wideband channel decoding within the last 15 s, and
// a system the hunter holds locked. A wideband channel that stopped decoding
// drops out, and the hunter's lock leaves the set when the lock is lost.
func TestLockedControlChannelsAreTheOnesDecodingNow(t *testing.T) {
	t.Run("wideband: only channels decoding now", func(t *testing.T) {
		ck := scannerCockpit{wideband: []channelHealthSource{fakeHealth{
			{System: "metro", FreqHz: lockedTestFarCC, Decoding: false, HasSignal: true},
			{System: "metro", FreqHz: lockedTestNearCC, Decoding: true},
		}}}
		if got := ck.LockedControlChannels(); !slices.Equal(got, []uint32{lockedTestNearCC}) {
			t.Fatalf("LockedControlChannels = %v, want [%d]", got, lockedTestNearCC)
		}
		// And the provider rests on it, over a configured channel nearer the
		// centre.
		systems := []trunking.System{{Name: "metro", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{lockedTestCentreHz, lockedTestFarCC}}}
		if got := controlChannelFor(systems, ck.LockedControlChannels(), lockedTestCentreHz, lockedTestRateHz); got != lockedTestNearCC {
			t.Fatalf("controlChannelFor = %d, want the decoding channel %d", got, lockedTestNearCC)
		}
	})

	t.Run("hunter: locked until the lock is lost", func(t *testing.T) {
		bus := events.NewBus(64)
		defer bus.Close()
		sup, err := cchunt.New(cchunt.Options{
			Bus: bus, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Tuner: nopTuner{},
			Systems: []trunking.System{{Name: "county", Protocol: trunking.ProtocolP25,
				ControlChannels: []uint32{lockedTestNearCC}}},
			Dwell: 50 * time.Millisecond, InitialBackoff: time.Hour, MaxBackoff: time.Hour,
		})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() { _ = sup.Run(ctx) }()
		ck := scannerCockpit{cchunt: sup}

		waitFor := func(what string, cond func([]uint32) bool) {
			t.Helper()
			deadline := time.Now().Add(2 * time.Second)
			for !cond(ck.LockedControlChannels()) {
				if time.Now().After(deadline) {
					t.Fatalf("%s: LockedControlChannels = %v", what, ck.LockedControlChannels())
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
		// Publish until the supervisor's listener has subscribed and taken it.
		lock := events.Event{Kind: events.KindCCLocked, Payload: testLock{freq: lockedTestNearCC}}
		deadline := time.Now().Add(2 * time.Second)
		for !slices.Equal(ck.LockedControlChannels(), []uint32{lockedTestNearCC}) {
			if time.Now().After(deadline) {
				t.Fatalf("hunter lock never reported: %v", ck.LockedControlChannels())
			}
			bus.Publish(lock)
			time.Sleep(10 * time.Millisecond)
		}
		bus.Publish(events.Event{Kind: events.KindCCLost, Payload: testLock{freq: lockedTestNearCC}})
		waitFor("after cc.lost", func(got []uint32) bool { return len(got) == 0 })
	})
}

type nopTuner struct{}

func (nopTuner) SetCenterFreq(uint32) error { return nil }

type testLock struct{ freq uint32 }

func (l testLock) LockedFrequencyHz() uint32 { return l.freq }
func (l testLock) LockedNAC() uint16         { return 0 }
