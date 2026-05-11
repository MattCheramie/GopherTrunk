package motorola

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestStrictValidationDropsUnknownOpcode: an OSW whose 12-bit
// Opcode field is not in the recognised set must be dropped by
// Ingest under SetStrictValidation(true).
func TestStrictValidationDropsUnknownOpcode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	// Opcode 0xABC is not in the documented list.
	cc.Ingest(OSW{Address: 0xDEAD, Command: (0xABC << 4) | 0x1})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode OSW with unknown Opcode published %v", ev.Kind)
	default:
	}
}

func TestStrictValidationKeepsKnownOpcode(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cmd := (uint16(OpGroupVoiceChannelGrant) << 4) | 0x3
	cc.Ingest(OSW{Address: 0xCAFE, Command: cmd})

	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				got++
			}
		default:
			if got == 0 {
				t.Errorf("strict-mode OSW with known Opcode did not publish a Grant")
			}
			return
		}
	}
}

func TestOpcodeIsKnownRecognisesDocumentedConstants(t *testing.T) {
	known := []Opcode{
		OpGroupVoiceChannelGrant, OpGroupVoiceChannelGrantUpdate,
		OpPrivateCallGrant, OpAdjacentSiteStatus, OpSystemIDExtended,
		OpDataChannelGrant, OpAffiliationResponse,
		OpIdle1, OpIdle2, OpEncryption, OpEmergency,
	}
	for _, o := range known {
		if !o.IsKnown() {
			t.Errorf("Opcode %#x should be known", uint16(o))
		}
	}
	if OpUnknown.IsKnown() {
		t.Errorf("OpUnknown should NOT be known")
	}
}
