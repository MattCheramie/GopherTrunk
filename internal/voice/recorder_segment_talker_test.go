package voice

import (
	"context"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// The call log stamps each recording file with CallComplete.Grant.SourceID
// (#1379), so that field has to be the radio heard in THAT file. This replays
// a four-over call in the order a P25 Phase 1 traffic channel delivers it:
// the terminator rolls the file, the next over's audio opens the next file,
// and only then does that over's link control name its radio. So every file
// after the first OPENS still carrying the previous over's radio and is
// corrected while it is being written; a file's opening source (the session's
// srcs[0]) is therefore the wrong one to record, and the grant at finalize is
// the right one. A radio that keys twice in a row publishes no update the
// second time (the composer only publishes a change), and its file must keep
// the carried radio.
func TestRecorderSegmentCompleteCarriesItsOwnTalker(t *testing.T) {
	r, bus, _ := mkRecorder(t, false)
	defer r.Close()
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	sub := bus.Subscribe()
	defer sub.Close()
	nextComplete := func() trunking.CallComplete {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case ev := <-sub.C:
				if cc, ok := ev.Payload.(trunking.CallComplete); ok && ev.Kind == events.KindCallComplete {
					return cc
				}
			case <-deadline:
				t.Fatal("no CallComplete for the finished over")
			}
		}
	}

	const (
		serial = "V1"
		radioA = uint32(3200460)
		radioB = uint32(315529)
	)
	cs := trunking.CallStart{
		Grant: trunking.Grant{
			System: "S", Protocol: "fm", GroupID: 5, SourceID: radioA,
			FrequencyHz: 451_000_000,
		},
		DeviceSerial: serial,
		StartedAt:    time.Date(2026, 11, 18, 13, 18, 19, 0, time.UTC),
	}
	audio := func() {
		t.Helper()
		if err := r.WritePCM(serial, make([]int16, 800)); err != nil {
			t.Fatal(err)
		}
	}
	names := func(src uint32) {
		bus.Publish(events.Event{Kind: events.KindCallSourceUpdate, Payload: trunking.CallSourceUpdate{
			DeviceSerial: serial, SourceID: src, At: time.Now().UTC(),
		}})
	}
	// overEnds rolls the file and returns the CallComplete that announced
	// it. The recorder publishes that after parking the session, so the next
	// write is certain to open a new file.
	overEnds := func() trunking.CallComplete {
		t.Helper()
		bus.Publish(events.Event{Kind: events.KindCallSegment, Payload: trunking.CallSegment{
			DeviceSerial: serial, At: time.Now().UTC(),
		}})
		return nextComplete()
	}

	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: cs})
	waitSession(t, r, serial, true)

	var got []trunking.CallComplete
	// Over 1: radio A, named by the grant.
	audio()
	got = append(got, overEnds())
	// Over 2: radio B answers. Its file opens on the first audio, before
	// its link control has named it.
	audio()
	names(radioB)
	audio()
	got = append(got, overEnds())
	// Over 3: radio B again. Nothing is published for an unchanged radio.
	audio()
	got = append(got, overEnds())
	// Over 4: radio A comes back, and the call ends on it.
	audio()
	names(radioA)
	audio()
	end := cs.Grant
	end.SourceID = radioA
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		Grant: end, DeviceSerial: serial, StartedAt: cs.StartedAt,
		EndedAt: time.Now().UTC(), Reason: trunking.EndReasonNormal,
	}})
	got = append(got, nextComplete())

	want := []uint32{radioA, radioB, radioB, radioA}
	paths := map[string]bool{}
	for i, cc := range got {
		if cc.Segment != i {
			t.Errorf("file %d announced as segment %d", i, cc.Segment)
		}
		if cc.Grant.SourceID != want[i] {
			t.Errorf("file %d: source = %d, want %d", i, cc.Grant.SourceID, want[i])
		}
		paths[cc.AudioPath] = true
	}
	if len(paths) != len(want) {
		t.Errorf("%d distinct files for %d overs", len(paths), len(want))
	}
}
