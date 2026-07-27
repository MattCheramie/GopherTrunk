package tetra

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestStrictValidationDropsUnknownPDU: a PDU whose (Discriminator,
// Type) pair is not in the documented ETSI EN 300 392-2 set must be
// dropped by Ingest under SetStrictValidation(true).
func TestStrictValidationDropsUnknownPDU(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	// CMCE Type 0xF is unallocated in the trunking sub-protocol.
	cc.Ingest(PDU{Disc: DiscCMCE, Type: 0xF, Payload: make([]byte, 11)})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode unknown CMCE Type published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationDropsUnknownDiscriminator: a PDU whose
// Discriminator selects a sub-protocol the state machine doesn't
// surface (MM, SDS) must be dropped under strict mode regardless of
// the Type field.
func TestStrictValidationDropsUnknownDiscriminator(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	cc.Ingest(PDU{Disc: DiscSDS, Type: 0x1, Payload: []byte{0x00}})

	select {
	case ev := <-sub.C:
		t.Errorf("strict-mode SDS PDU published %v", ev.Kind)
	default:
	}
}

// TestStrictValidationKeepsKnownPDU: a PDU with a recognised
// (Discriminator, Type) pair must still publish its event under strict mode.
// Uses an MLE SYSINFO, the L3 PDU Ingest still surfaces (grants/teardown are
// decoded from the MAC layer, not Ingest — see downlink.go).
func TestStrictValidationKeepsKnownPDU(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys", FrequencyHz: 410_000_000})
	cc.SetStrictValidation(true)

	cc.Ingest(PDU{
		Disc:    DiscMLE,
		Type:    uint8(MLESystemInfo),
		Payload: buildSysInfoPayload(206, 1, 5),
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
				t.Errorf("strict-mode MLE SYSINFO did not publish cc.locked")
			}
			return
		}
	}
}

func TestPDUIsKnownCoversDocumentedConstants(t *testing.T) {
	knownCMCE := []PDUType{
		CMCEDSetup, CMCEDConnect, CMCEDRelease, CMCEDTxCeased,
		CMCEDTxGranted, CMCEDInfo, CMCEDCallProceeding,
	}
	for _, ty := range knownCMCE {
		p := PDU{Disc: DiscCMCE, Type: uint8(ty)}
		if !p.IsKnown() {
			t.Errorf("CMCE Type %#x should be known", uint8(ty))
		}
	}
	if !(PDU{Disc: DiscMLE, Type: uint8(MLESystemInfo)}.IsKnown()) {
		t.Errorf("MLE SYSINFO should be known")
	}
	for _, ty := range []uint8{0x0, 0x3, 0x8, 0xB, 0xC, 0xD, 0xE, 0xF} {
		p := PDU{Disc: DiscCMCE, Type: ty}
		if p.IsKnown() {
			t.Errorf("CMCE Type %#x should NOT be known", ty)
		}
	}
	if (PDU{Disc: DiscMM, Type: 0x1}).IsKnown() {
		t.Errorf("MM PDUs should NOT be known")
	}
}
