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
// (Discriminator, Type) pair must still publish its event under
// strict mode. Uses a D-CONNECT carrying a voice grant.
func TestStrictValidationKeepsKnownPDU(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys"})
	cc.SetStrictValidation(true)

	// Build a minimal D-CONNECT payload: 14-bit CID + flags, source SSI,
	// dest SSI, flags byte, carrier+slot. Channel 7, slot 0, group call.
	payload := make([]byte, 11)
	payload[0], payload[1] = 0x00, 0x04 // CID = 1 in upper 14 bits
	payload[2], payload[3], payload[4] = 0xAA, 0xAA, 0xAA
	payload[5], payload[6], payload[7] = 0xBB, 0xBB, 0xBB
	payload[8] = 0x80                    // group flag set
	payload[9], payload[10] = 0x00, 0x70 // carrier 7 in upper 12 bits, slot 0
	cc.Ingest(PDU{Disc: DiscCMCE, Type: uint8(CMCEDConnect), Payload: payload})

	got := 0
	for {
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindGrant {
				got++
			}
		default:
			if got == 0 {
				t.Errorf("strict-mode D-CONNECT did not publish a Grant")
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
