package mpt1327

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestStrictValidationDropsDataCodeword: a Data codeword (Type =
// TypeData) must be dropped by Ingest under strict mode. (Data
// codewords aren't followed at the trunking layer regardless,
// but strict mode now drops them BEFORE the existing IsIdle /
// Kind dispatch instead of relying on the per-kind handlers'
// implicit rejection.)
func TestStrictValidationDropsDataCodeword(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(Codeword{
		Type:     TypeData,
		Function: uint32(KindGoToChan) << 13,
	})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode Data codeword published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationDropsUnknownKind: an Address codeword whose
// Kind() is KindUnknown (0x0) must be dropped under strict mode.
func TestStrictValidationDropsUnknownKind(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(Codeword{Type: TypeAddress, Function: 0})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode unknown-Kind codeword published %v", ev.Kind)
	default:
	}
}

func TestStrictValidationKeepsAlohaCodeword(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(Codeword{
		Type:     TypeAddress,
		Prefix:   0x3,
		Function: uint32(KindAloha) << 13,
	})

	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				got++
			}
		default:
			if got == 0 {
				t.Errorf("strict-mode Aloha codeword did not publish a CCLocked")
			}
			return
		}
	}
}
