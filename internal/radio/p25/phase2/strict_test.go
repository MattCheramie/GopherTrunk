package phase2

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestStrictValidationDropsUnknownOpcode: a MAC PDU whose 8-bit
// Opcode field is not in the documented TIA-102.AABF / BBAB set must
// be dropped by Ingest under SetStrictValidation(true).
func TestStrictValidationDropsUnknownOpcode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	// Opcode 0x7E is not in the documented list.
	cc.Ingest(MACPDU{Opcode: Opcode(0x7E), Payload: make([]byte, 8)})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode MAC PDU with unknown Opcode published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationKeepsKnownOpcode: a MAC PDU with a recognised
// Opcode must still publish its event under strict mode.
func TestStrictValidationKeepsKnownOpcode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	// GroupVoiceChannelGrant payload: service options, channel ID+num,
	// group address, source ID. Channel ID 1, channel 7, talkgroup
	// 0xCAFE, source 0x123456.
	payload := []byte{
		0x00,             // service options
		0x10, 0x07,       // channel ID 1 (upper 4 bits) + channel 7
		0xCA, 0xFE,       // group address
		0x12, 0x34, 0x56, // source ID
	}
	cc.Ingest(MACPDU{Opcode: OpGroupVoiceChannelGrant, Payload: payload})

	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				got++
			}
		default:
			if got == 0 {
				t.Errorf("strict-mode MAC PDU with known Opcode did not publish a Grant")
			}
			return
		}
	}
}

func TestOpcodeIsKnownCoversDocumentedConstants(t *testing.T) {
	known := []Opcode{
		OpMACPTT, OpMACEnd, OpMACIdle, OpMACHangtime, OpMACActive,
		OpGroupVoiceChannelGrant, OpGroupVoiceChannelGrantUpdate,
		OpGroupVoiceChannelUserExt, OpUnitToUnitVoiceChannelGrant,
		OpNetworkStatusBroadcastUpdate, OpRFSSStatusBroadcastUpdate,
	}
	for _, o := range known {
		if !o.IsKnown() {
			t.Errorf("Opcode %#x should be known", uint8(o))
		}
	}
	if OpUnknown.IsKnown() {
		t.Errorf("OpUnknown should NOT be known")
	}
	for _, o := range []Opcode{0x10, 0x7E, 0x80, 0xC0} {
		if o.IsKnown() {
			t.Errorf("Opcode %#x should NOT be known", uint8(o))
		}
	}
}
