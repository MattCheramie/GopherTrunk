package phase2

import (
	"errors"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

func TestIdentifierUpdateRoundTrip(t *testing.T) {
	in := IdentifierUpdate{
		ChannelID:   3,
		BandwidthHz: 12_500,
		SpacingHz:   12_500,
		TxOffsetHz:  -45_000_000,
		BaseHz:      851_000_000,
	}
	p := AssembleIdentifierUpdate(in)
	got, ok := ParseIdentifierUpdate(p[:])
	if !ok {
		t.Fatal("ParseIdentifierUpdate returned !ok for an 8-byte field")
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestParseIdentifierUpdateShortInput(t *testing.T) {
	if _, ok := ParseIdentifierUpdate(make([]byte, 7)); ok {
		t.Error("ParseIdentifierUpdate returned ok for a 7-byte field")
	}
}

func TestBandPlanResolvesFrequency(t *testing.T) {
	var bp BandPlan
	bp.Apply(IdentifierUpdate{ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000})

	got, err := bp.Frequency(1, 10)
	if err != nil {
		t.Fatalf("Frequency: %v", err)
	}
	if want := uint32(851_125_000); got != want {
		t.Errorf("Frequency(1,10) = %d, want %d", got, want)
	}
	if !bp.Known(1) {
		t.Error("Known(1) = false after Apply")
	}
}

func TestBandPlanUnknownChannel(t *testing.T) {
	var bp BandPlan
	if _, err := bp.Frequency(5, 0); !errors.Is(err, ErrUnknownChannelID) {
		t.Errorf("Frequency on unknown ID err = %v, want ErrUnknownChannelID", err)
	}
}

// TestControlChannelResolvesGrantFrequency confirms that once an
// IdentifierUpdate MAC PDU has been ingested, a later voice grant is
// published with a resolved downlink frequency rather than 0.
func TestControlChannelResolvesGrantFrequency(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_012_500})

	idu := AssembleIdentifierUpdate(IdentifierUpdate{
		ChannelID: 1, SpacingHz: 12_500, BaseHz: 851_000_000,
	})
	cc.Ingest(MACPDU{Opcode: OpIdentifierUpdate, Payload: idu[:]})
	cc.Ingest(grantPDU(42, 0x000123, 0x1, 10))

	// Drain: the IdentifierUpdate ingest locks the channel (cc.locked),
	// then the grant publishes the resolved grant.
	var grant trunking.Grant
	gotGrant := false
	for i := 0; i < 2; i++ {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				grant = ev.Payload.(trunking.Grant)
				gotGrant = true
			}
		case <-time.After(time.Second):
			t.Fatal("timed out draining events")
		}
	}
	if !gotGrant {
		t.Fatal("no grant event published")
	}
	if want := uint32(851_125_000); grant.FrequencyHz != want {
		t.Errorf("grant FrequencyHz = %d, want %d", grant.FrequencyHz, want)
	}
	if grant.ChannelID != 1 || grant.ChannelNum != 10 {
		t.Errorf("grant channel = (%d,%d), want (1,10)", grant.ChannelID, grant.ChannelNum)
	}
}

// TestControlChannelGrantWithoutBandPlan confirms a grant that arrives
// before any IdentifierUpdate is still published (FrequencyHz 0) — the
// pre-band-plan behaviour, preserved for event-surface compatibility.
func TestControlChannelGrantWithoutBandPlan(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, SystemName: "p2", FrequencyHz: 851_012_500})
	cc.Ingest(grantPDU(7, 0x000456, 0x2, 5))

	gotGrant := false
	for i := 0; i < 2; i++ {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				g := ev.Payload.(trunking.Grant)
				gotGrant = true
				if g.FrequencyHz != 0 {
					t.Errorf("grant FrequencyHz = %d, want 0 without a band plan", g.FrequencyHz)
				}
			}
		case <-time.After(time.Second):
			t.Fatal("timed out draining events")
		}
	}
	if !gotGrant {
		t.Fatal("grant event not published without a band plan")
	}
}
