package edacs

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestStrictValidationDropsUnknownCommand: a CCW with Command =
// 0xA (in the unallocated 0xA..0xE range) must be dropped by
// Ingest when SetStrictValidation(true). The same CCW without
// strict mode should still publish a Grant.
func TestStrictValidationDropsUnknownCommand(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(CCW{Command: Command(0xA), Address: 0x1234})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode CCW with unknown Command published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationKeepsKnownCommand: a CCW with a recognised
// Command must still publish its event under strict mode.
func TestStrictValidationKeepsKnownCommand(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(CCW{Command: CmdGroupVoiceGrant, Address: 0xCAFE, LCN: 1})

	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				got++
			}
		default:
			if got == 0 {
				t.Errorf("strict-mode CCW with known Command did not publish a Grant")
			}
			return
		}
	}
}

func TestCommandIsKnownCoversAllConstants(t *testing.T) {
	known := []Command{
		CmdIdle, CmdGroupVoiceGrant, CmdProVoiceGrant,
		CmdIndividualCall, CmdDataGrant, CmdSystemID,
		CmdAdjacentSite, CmdEmergency, CmdAffiliation,
		CmdEncryption, CmdReserved,
	}
	for _, c := range known {
		if !c.IsKnown() {
			t.Errorf("Command %#x should be known", uint8(c))
		}
	}
	for _, c := range []Command{0xA, 0xB, 0xC, 0xD, 0xE} {
		if c.IsKnown() {
			t.Errorf("Command %#x should NOT be known", uint8(c))
		}
	}
}
