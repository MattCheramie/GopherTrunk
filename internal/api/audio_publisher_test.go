package api

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	apiv1 "github.com/MattCheramie/GopherTrunk/internal/api/pb/v1"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// mkPublisher spins up a publisher with its Run loop attached to t.Cleanup
// so tests don't leak goroutines.
func mkPublisher(t *testing.T) (*AudioPublisher, *events.Bus) {
	t.Helper()
	bus := events.NewBus(16)
	pub, err := NewAudioPublisher(bus, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = pub.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
		_ = pub.Close()
		bus.Close()
	})
	return pub, bus
}

// publishCallStart fires a CallStart for the supplied (device, grant).
// Waits briefly so the publisher's Run loop has time to update the
// internal grant map before the caller drives WritePCM.
func publishCallStart(t *testing.T, pub *AudioPublisher, bus *events.Bus, serial string, grant trunking.Grant) {
	t.Helper()
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant:        grant,
		DeviceSerial: serial,
	}})
	waitFor(t, 200*time.Millisecond, func() bool {
		return pub.Stats().TrackedGrants > 0
	})
}

func TestAudioPublisher_WritePCMFansToMatchingSubs(t *testing.T) {
	pub, bus := mkPublisher(t)
	publishCallStart(t, pub, bus, "VOICE-1", trunking.Grant{GroupID: 42, System: "Sys"})

	sub := pub.Subscribe(AudioSubFilter{})
	defer pub.Unsubscribe(sub)

	if err := pub.WritePCM("VOICE-1", []int16{1, -2, 3, -4}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	select {
	case frame := <-sub.ch:
		pcm := frame.GetPcm()
		if pcm == nil {
			t.Fatal("frame missing PCM body")
		}
		if len(pcm.Samples) != 8 {
			t.Errorf("samples len = %d, want 8 (4 int16 → 8 bytes)", len(pcm.Samples))
		}
		if frame.GetGrant().GroupId != 42 {
			t.Errorf("grant.group_id = %d, want 42", frame.GetGrant().GroupId)
		}
		if frame.DeviceSerial != "VOICE-1" {
			t.Errorf("device_serial = %q, want VOICE-1", frame.DeviceSerial)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("subscriber received no frame")
	}
}

func TestAudioPublisher_DropsWhenNoGrant(t *testing.T) {
	pub, _ := mkPublisher(t)
	sub := pub.Subscribe(AudioSubFilter{})
	defer pub.Unsubscribe(sub)

	// No CallStart published — the publisher's grant map is empty.
	if err := pub.WritePCM("VOICE-1", []int16{1, 2, 3}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	select {
	case f := <-sub.ch:
		t.Errorf("got frame %v despite no Grant", f)
	case <-time.After(50 * time.Millisecond):
		// pass
	}
}

func TestAudioPublisher_FilterDeviceSerial(t *testing.T) {
	pub, bus := mkPublisher(t)
	publishCallStart(t, pub, bus, "VOICE-1", trunking.Grant{GroupID: 1})
	publishCallStart(t, pub, bus, "VOICE-2", trunking.Grant{GroupID: 2})
	waitFor(t, 200*time.Millisecond, func() bool {
		return pub.Stats().TrackedGrants == 2
	})

	sub := pub.Subscribe(AudioSubFilter{DeviceSerials: []string{"VOICE-2"}})
	defer pub.Unsubscribe(sub)

	pub.WritePCM("VOICE-1", []int16{1, 2})
	pub.WritePCM("VOICE-2", []int16{3, 4})

	select {
	case frame := <-sub.ch:
		if frame.DeviceSerial != "VOICE-2" {
			t.Errorf("got serial %q, want VOICE-2", frame.DeviceSerial)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("subscriber received no frame")
	}
	// VOICE-1 frame should NOT arrive.
	select {
	case f := <-sub.ch:
		t.Errorf("got unexpected frame for %q", f.DeviceSerial)
	case <-time.After(50 * time.Millisecond):
		// pass
	}
}

func TestAudioPublisher_FilterTalkgroupID(t *testing.T) {
	pub, bus := mkPublisher(t)
	publishCallStart(t, pub, bus, "VOICE-1", trunking.Grant{GroupID: 100})

	yes := pub.Subscribe(AudioSubFilter{TalkgroupIDs: []uint32{100}})
	no := pub.Subscribe(AudioSubFilter{TalkgroupIDs: []uint32{999}})
	defer pub.Unsubscribe(yes)
	defer pub.Unsubscribe(no)

	pub.WritePCM("VOICE-1", []int16{1, 2})

	select {
	case <-yes.ch:
		// pass
	case <-time.After(500 * time.Millisecond):
		t.Error("matching filter received no frame")
	}
	select {
	case f := <-no.ch:
		t.Errorf("non-matching filter received frame %v", f)
	case <-time.After(50 * time.Millisecond):
		// pass
	}
}

func TestAudioPublisher_SlowSubscriberDropsNotBlocks(t *testing.T) {
	pub, bus := mkPublisher(t)
	publishCallStart(t, pub, bus, "VOICE-1", trunking.Grant{GroupID: 1})

	sub := pub.Subscribe(AudioSubFilter{})
	defer pub.Unsubscribe(sub)

	// Fill the bounded channel by writing more than its capacity
	// without draining. Channel cap is 64; write 128 frames.
	for range 128 {
		pub.WritePCM("VOICE-1", []int16{1})
	}
	if pub.Stats().DroppedTotal == 0 {
		t.Error("expected dropped samples on a full subscriber channel")
	}
	// The slow subscriber didn't deadlock the publisher — that's
	// the whole point of the drop-on-full policy.
}

func TestAudioPublisher_CallEndClearsGrant(t *testing.T) {
	pub, bus := mkPublisher(t)
	publishCallStart(t, pub, bus, "VOICE-1", trunking.Grant{GroupID: 1})
	if pub.Stats().TrackedGrants != 1 {
		t.Fatalf("setup: TrackedGrants=%d, want 1", pub.Stats().TrackedGrants)
	}
	bus.Publish(events.Event{Kind: events.KindCallEnd, Payload: trunking.CallEnd{
		DeviceSerial: "VOICE-1",
	}})
	waitFor(t, 200*time.Millisecond, func() bool {
		return pub.Stats().TrackedGrants == 0
	})
}

func TestAudioPublisher_UnsubscribeIsIdempotent(t *testing.T) {
	pub, _ := mkPublisher(t)
	sub := pub.Subscribe(AudioSubFilter{})
	pub.Unsubscribe(sub)
	pub.Unsubscribe(sub) // must not panic
	pub.Unsubscribe(nil) // must not panic
}

func TestAudioPublisher_NilSafe(t *testing.T) {
	var p *AudioPublisher
	if err := p.WritePCM("x", []int16{1}); err != nil {
		t.Errorf("nil publisher WritePCM: %v", err)
	}
}

func TestAudioSubFilter_Matches(t *testing.T) {
	cases := []struct {
		name   string
		filter AudioSubFilter
		serial string
		group  uint32
		want   bool
	}{
		{"empty matches all", AudioSubFilter{}, "X", 1, true},
		{"serial allow-list hit", AudioSubFilter{DeviceSerials: []string{"X"}}, "X", 1, true},
		{"serial allow-list miss", AudioSubFilter{DeviceSerials: []string{"Y"}}, "X", 1, false},
		{"group allow-list hit", AudioSubFilter{TalkgroupIDs: []uint32{1}}, "X", 1, true},
		{"group allow-list miss", AudioSubFilter{TalkgroupIDs: []uint32{2}}, "X", 1, false},
		{"both must match (hit)", AudioSubFilter{DeviceSerials: []string{"X"}, TalkgroupIDs: []uint32{1}}, "X", 1, true},
		{"both must match (group miss)", AudioSubFilter{DeviceSerials: []string{"X"}, TalkgroupIDs: []uint32{2}}, "X", 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.filter.matches(tc.serial, tc.group); got != tc.want {
				t.Errorf("matches = %v, want %v", got, tc.want)
			}
		})
	}
}

// waitFor polls fn until it returns true or the deadline elapses.
func waitFor(t *testing.T, d time.Duration, fn func() bool) {
	t.Helper()
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("waitFor: condition not met within %s", d)
}

// statsSnapshot lets older test files compile (unused right now; kept
// to keep imports stable across the package).
var _ = atomic.Int32{}
var _ = (*apiv1.AudioFrame)(nil)
