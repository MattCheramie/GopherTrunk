package tier2

import (
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// burstWithTerminatorFLC builds a Terminator-with-LC-shaped burst carrying the
// supplied FLC, using the terminator RS(12,9) parity seed (0x99…) — what a real
// transmitter puts on air at end-of-call, and what terminatorDest decodes.
func burstWithTerminatorFLC(f dmr.FLC) *dmr.Burst {
	flcBytes := dmr.AssembleFLC(f)
	var data [9]byte
	copy(data[:], flcBytes)
	cw := framing.EncodeRS12_9(data)
	for i := 0; i < 3; i++ {
		cw[9+i] ^= framing.RS129SeedTerminatorLC[i]
	}
	info := cw[:]

	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	channel := framing.EncodeBPTC196_96(bits)

	var b dmr.Burst
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[i] = (channel[2*i] << 1) | channel[2*i+1]
	}
	for i := 0; i < dmr.HalfPayloadDibits; i++ {
		b.Dibits[dmr.BurstDibits-dmr.HalfPayloadDibits+i] =
			(channel[2*(dmr.HalfPayloadDibits+i)] << 1) | channel[2*(dmr.HalfPayloadDibits+i)+1]
	}
	return &b
}

// drainEvents collects every event published within d.
func drainEvents(sub *events.Subscription, d time.Duration) []events.Event {
	var out []events.Event
	deadline := time.After(d)
	for {
		select {
		case ev := <-sub.C:
			out = append(out, ev)
		case <-deadline:
			return out
		}
	}
}

func grantsOf(evs []events.Event) []trunking.Grant {
	var out []trunking.Grant
	for _, ev := range evs {
		if ev.Kind == events.KindGrant {
			out = append(out, ev.Payload.(trunking.Grant))
		}
	}
	return out
}

// TestConventionalStampsInterleavedVoice pins the audio fix: Options.Interleaved-
// Voice is stamped onto the grant as DMRInterleavedVoice so the composer runs
// the two-slot interleaved decoder. Default (Tier I direct mode) stays false.
func TestConventionalStampsInterleavedVoice(t *testing.T) {
	for _, tc := range []struct {
		name        string
		interleaved bool
	}{
		{"interleaved on (Tier II conventional)", true},
		{"interleaved off (Tier I direct)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bus := events.NewBus(8)
			defer bus.Close()
			sub := bus.Subscribe()
			defer sub.Close()

			cc := New(Options{Bus: bus, SystemName: "S", FrequencyHz: 1, InterleavedVoice: tc.interleaved})
			flc := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x11, SrcAddr: 0x100}
			cc.IngestBurst(burstWithFLC(flc), dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader})

			grants := grantsOf(drainEvents(sub, 100*time.Millisecond))
			if len(grants) != 1 {
				t.Fatalf("got %d grants, want 1", len(grants))
			}
			if grants[0].DMRInterleavedVoice != tc.interleaved {
				t.Errorf("DMRInterleavedVoice = %v, want %v", grants[0].DMRInterleavedVoice, tc.interleaved)
			}
		})
	}
}

// TestConventionalTwoTalkgroupsGetDistinctSlots pins the two-slot identity fix:
// two concurrent talkgroups on one repeater (TS1/TS2) must produce two grants
// with distinct synthetic timeslots, and their interleaved repeat headers must
// not ping-pong out fresh grants. Before the fix both collapsed to Timeslot 0
// and re-granted on every alternating header.
func TestConventionalTwoTalkgroupsGetDistinctSlots(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Fire2", FrequencyHz: 442_387_500})
	slot := dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader}
	tgA := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x11, SrcAddr: 0x13000013}
	tgB := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x12, SrcAddr: 0x13000042}

	// Interleaved on air: A, B, A, B, A, B (each slot's Voice LC Header repeats
	// every superframe).
	for i := 0; i < 3; i++ {
		cc.IngestBurst(burstWithFLC(tgA), slot)
		cc.IngestBurst(burstWithFLC(tgB), slot)
	}

	grants := grantsOf(drainEvents(sub, 150*time.Millisecond))
	if len(grants) != 2 {
		t.Fatalf("got %d grants, want exactly 2 (one per talkgroup, no ping-pong)", len(grants))
	}
	byTG := map[uint32]trunking.Grant{}
	for _, g := range grants {
		byTG[g.GroupID] = g
	}
	a, okA := byTG[0x11]
	b, okB := byTG[0x12]
	if !okA || !okB {
		t.Fatalf("expected grants for both TG 0x11 and 0x12, got %v", byTG)
	}
	if a.Timeslot == 0 || b.Timeslot == 0 {
		t.Errorf("timeslots must be non-zero synthetic slots, got A=%d B=%d", a.Timeslot, b.Timeslot)
	}
	if a.Timeslot == b.Timeslot {
		t.Errorf("the two concurrent talkgroups share timeslot %d; they must differ", a.Timeslot)
	}
}

// TestConventionalTerminatorReleasesOnlyItsTalkgroup pins that a Terminator for
// one slot's talkgroup does not tear down the other slot's active call: only
// TG-A is released, and a later TG-A header opens a fresh call while TG-B's
// repeat header still dedupes (it was never released).
func TestConventionalTerminatorReleasesOnlyItsTalkgroup(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "Fire2", FrequencyHz: 442_387_500})
	hdr := dmr.SlotType{ColorCode: 1, DataType: dmr.DTVoiceLCHeader}
	term := dmr.SlotType{ColorCode: 1, DataType: dmr.DTTerminatorWithLC}
	tgA := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x11, SrcAddr: 0x13000013}
	tgB := dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 0x12, SrcAddr: 0x13000042}

	// Both slots in call.
	cc.IngestBurst(burstWithFLC(tgA), hdr)
	cc.IngestBurst(burstWithFLC(tgB), hdr)
	// Terminator for TG-A only.
	cc.IngestBurst(burstWithTerminatorFLC(tgA), term)
	// TG-B repeats (still in call → dedupe, no new grant); TG-A opens again.
	cc.IngestBurst(burstWithFLC(tgB), hdr)
	cc.IngestBurst(burstWithFLC(tgA), hdr)

	evs := drainEvents(sub, 200*time.Millisecond)

	var releases []trunking.CallRelease
	for _, ev := range evs {
		if ev.Kind == events.KindCallRelease {
			releases = append(releases, ev.Payload.(trunking.CallRelease))
		}
	}
	if len(releases) != 1 {
		t.Fatalf("got %d releases, want exactly 1 (only TG-A)", len(releases))
	}
	if releases[0].GroupID != 0x11 {
		t.Errorf("released TG = %X, want 0x11 (TG-A); TG-B must not be torn down", releases[0].GroupID)
	}

	// Grants: TG-A opens (1), TG-B opens (1), TG-B repeat dedupes (0), TG-A
	// re-opens after its terminator (1) → 3 total, and TG-A appears twice.
	grants := grantsOf(evs)
	if len(grants) != 3 {
		t.Fatalf("got %d grants, want 3 (A, B, A-reopened; B-repeat deduped)", len(grants))
	}
	var aCount, bCount int
	for _, g := range grants {
		switch g.GroupID {
		case 0x11:
			aCount++
		case 0x12:
			bCount++
		}
	}
	if aCount != 2 || bCount != 1 {
		t.Errorf("grant counts A=%d B=%d, want A=2 B=1", aCount, bCount)
	}
}
