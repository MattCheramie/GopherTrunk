package dpmr

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestStrictValidationDropsUnknownMessageType: a CSBK whose 5-bit
// MessageType field is not in the documented ETSI TS 102 658 §6.5.2
// set must be dropped by Ingest under SetStrictValidation(true).
func TestStrictValidationDropsUnknownMessageType(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	// MessageType 0x10 is in the unallocated range.
	cc.Ingest(CSBK{Type: MessageType(0x10), SourceID: 0x111111, DestID: 0x222222})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode CSBK with unknown MessageType published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationKeepsKnownMessageType: a CSBK with a recognised
// MessageType must still publish its event under strict mode.
func TestStrictValidationKeepsKnownMessageType(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(CSBK{
		Type:     MsgVoiceServiceAllocation,
		Flags:    FlagGroupCall,
		SourceID: 0xAAAAAA,
		DestID:   0xBBBBBB,
		Extra:    7,
	})

	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				got++
			}
		default:
			if got == 0 {
				t.Errorf("strict-mode CSBK with known MessageType did not publish a Grant")
			}
			return
		}
	}
}

func TestMessageTypeIsKnownCoversDocumentedConstants(t *testing.T) {
	known := []MessageType{
		MsgRegistrationRequest, MsgRegistrationResponse,
		MsgVoiceServiceAllocation, MsgIndividualVoiceAllocation,
		MsgDataServiceAllocation, MsgServiceRequest,
		MsgStandingServiceStatus, MsgRelease, MsgIdle,
	}
	for _, m := range known {
		if !m.IsKnown() {
			t.Errorf("MessageType %#x should be known", uint8(m))
		}
	}
	if MsgUnknown.IsKnown() {
		t.Errorf("MsgUnknown should NOT be known")
	}
	if MessageType(0x10).IsKnown() {
		t.Errorf("MessageType 0x10 should NOT be known")
	}
}
