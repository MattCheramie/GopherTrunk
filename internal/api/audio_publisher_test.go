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

// Issue #598 regression: PCM must reach an unfiltered subscriber even
// when no CallStart grant has been observed for the device. The grant
// cache rides a lossy bus subscription; gating audio on it left the
// WebUI live stream silent while disk recordings worked. The frame
// carries a zero Grant in that case.
func TestAudioPublisher_FansUnfilteredWithoutGrant(t *testing.T) {
	pub, _ := mkPublisher(t)
	sub := pub.Subscribe(AudioSubFilter{})
	defer pub.Unsubscribe(sub)

	// No CallStart published — the publisher's grant map is empty.
	if err := pub.WritePCM("VOICE-1", []int16{1, -2, 3}); err != nil {
		t.Fatalf("WritePCM: %v", err)
	}
	select {
	case frame := <-sub.ch:
		if frame.DeviceSerial != "VOICE-1" {
			t.Errorf("device_serial = %q, want VOICE-1", frame.DeviceSerial)
		}
		if pcm := frame.GetPcm(); pcm == nil || len(pcm.Samples) != 6 {
			t.Errorf("samples = %v, want 6 bytes (3 int16)", pcm.GetSamples())
		}
		if gid := frame.GetGrant().GetGroupId(); gid != 0 {
			t.Errorf("grant.group_id = %d, want 0 (zero grant)", gid)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("subscriber received no frame despite open stream")
	}
}

// A device-serial-only filter still works without a grant: matching
// serials are delivered, non-matching serials are not.
func TestAudioPublisher_DeviceFilterFansWithoutGrant(t *testing.T) {
	pub, _ := mkPublisher(t)
	sub := pub.Subscribe(AudioSubFilter{DeviceSerials: []string{"VOICE-1"}})
	defer pub.Unsubscribe(sub)

	pub.WritePCM("VOICE-2", []int16{9, 9}) // filtered out
	pub.WritePCM("VOICE-1", []int16{1, 2}) // delivered

	select {
	case frame := <-sub.ch:
		if frame.DeviceSerial != "VOICE-1" {
			t.Errorf("got serial %q, want VOICE-1", frame.DeviceSerial)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("device-filtered subscriber received no frame")
	}
	select {
	case f := <-sub.ch:
		t.Errorf("got unexpected frame for %q", f.DeviceSerial)
	case <-time.After(50 * time.Millisecond):
		// pass
	}
}

// A talkgroup filter cannot be evaluated without a grant, so frames are
// withheld until a matching CallStart lands — then they flow.
func TestAudioPublisher_TalkgroupFilterSkippedWithoutGrant(t *testing.T) {
	pub, bus := mkPublisher(t)
	sub := pub.Subscribe(AudioSubFilter{TalkgroupIDs: []uint32{100}})
	defer pub.Unsubscribe(sub)

	pub.WritePCM("VOICE-1", []int16{1, 2})
	select {
	case f := <-sub.ch:
		t.Fatalf("TG-filtered subscriber got frame %v before any grant", f)
	case <-time.After(50 * time.Millisecond):
		// pass — no grant, can't match talkgroup
	}

	publishCallStart(t, pub, bus, "VOICE-1", trunking.Grant{GroupID: 100})
	pub.WritePCM("VOICE-1", []int16{3, 4})
	select {
	case <-sub.ch:
		// pass — grant now known, talkgroup matches
	case <-time.After(500 * time.Millisecond):
		t.Fatal("TG-filtered subscriber got no frame after matching grant")
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
	// without draining, so drop-on-full kicks in.
	for range audioSubChanCap * 2 {
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

// waitForCallID blocks until the publisher's grant for serial has the given
// CallID — used to sequence a tap-serial reuse so the test drives WritePCMForCall
// only after the serial's bound call has actually flipped.
func waitForCallID(t *testing.T, pub *AudioPublisher, serial string, callID uint64) {
	t.Helper()
	waitFor(t, 200*time.Millisecond, func() bool {
		pub.mu.RLock()
		defer pub.mu.RUnlock()
		return pub.grants[serial].CallID == callID
	})
}

// TestAudioPublisher_WritePCMForCallFencesStaleCall reproduces the voice-tap
// audio bleed: a wideband tap serial is reused for a new call, so the
// publisher's grant for that serial flips to the new call (TG 200, CallID 2)
// while the old call's PCM (CallID 1) is still draining. The stale frame must be
// dropped — not fanned to a TG-200 subscriber labelled as the new call.
func TestAudioPublisher_WritePCMForCallFencesStaleCall(t *testing.T) {
	pub, bus := mkPublisher(t)
	publishCallStart(t, pub, bus, "tap-0", trunking.Grant{GroupID: 100, CallID: 1})

	sub := pub.Subscribe(AudioSubFilter{TalkgroupIDs: []uint32{200}})
	defer pub.Unsubscribe(sub)

	// The new call binds the same tap serial.
	bus.Publish(events.Event{Kind: events.KindCallStart, Payload: trunking.CallStart{
		Grant: trunking.Grant{GroupID: 200, CallID: 2}, DeviceSerial: "tap-0",
	}})
	waitForCallID(t, pub, "tap-0", 2)

	// Old call's tail (CallID 1) must NOT reach the TG-200 subscriber.
	if err := pub.WritePCMForCall("tap-0", 1, []int16{1, 2, 3}); err != nil {
		t.Fatalf("WritePCMForCall stale: %v", err)
	}
	select {
	case f := <-sub.ch:
		t.Fatalf("TG-200 subscriber received stale CallID-1 audio (group_id=%d) — bleed not fenced", f.GetGrant().GroupId)
	case <-time.After(100 * time.Millisecond):
	}

	// The current call's audio (CallID 2) flows to the matching subscriber.
	if err := pub.WritePCMForCall("tap-0", 2, []int16{4, 5, 6}); err != nil {
		t.Fatalf("WritePCMForCall current: %v", err)
	}
	select {
	case f := <-sub.ch:
		if f.GetGrant().GroupId != 200 {
			t.Errorf("group_id = %d, want 200", f.GetGrant().GroupId)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("TG-200 subscriber received no current-call frame")
	}
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
