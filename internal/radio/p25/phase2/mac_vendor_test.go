package phase2

import (
	"bytes"
	"testing"
)

func TestIsManufacturerSpecific(t *testing.T) {
	for _, o := range []Opcode{0x80, 0x81, 0xA0, 0xBF} {
		if !o.IsManufacturerSpecific() {
			t.Errorf("%#x should be manufacturer-specific", uint8(o))
		}
	}
	for _, o := range []Opcode{0x00, 0x44, 0x7D, 0xC0, 0xFB} {
		if o.IsManufacturerSpecific() {
			t.Errorf("%#x should not be manufacturer-specific", uint8(o))
		}
	}
}

// TestMACPDUVendorRoundTrip confirms a manufacturer-specific PDU carries
// its MFID through Assemble → Parse, and that a standard PDU does not
// grow an MFID octet.
func TestMACPDUVendorRoundTrip(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	vendor := MACPDU{Opcode: OpVendorGroupRegroup, MFID: MFIDMotorola, Payload: payload}
	b := AssembleMACPDU(vendor)
	if b[0] != byte(OpVendorGroupRegroup) || b[1] != MFIDMotorola {
		t.Fatalf("assembled vendor header = %02X %02X, want 81 90", b[0], b[1])
	}
	got, err := ParseMACPDU(b)
	if err != nil {
		t.Fatalf("ParseMACPDU: %v", err)
	}
	if got.Opcode != OpVendorGroupRegroup || got.MFID != MFIDMotorola {
		t.Errorf("parsed opcode/MFID = %v/%02X", got.Opcode, got.MFID)
	}
	if !bytes.Equal(got.Payload, payload) {
		t.Errorf("parsed payload = %x, want %x", got.Payload, payload)
	}

	std := MACPDU{Opcode: OpGroupVoiceChannelGrant, Payload: payload}
	sb := AssembleMACPDU(std)
	if len(sb) != 1+len(payload) {
		t.Errorf("standard PDU assembled length = %d, want %d (no MFID octet)",
			len(sb), 1+len(payload))
	}
	sgot, _ := ParseMACPDU(sb)
	if sgot.MFID != MFIDStandard {
		t.Errorf("standard PDU MFID = %02X, want 0", sgot.MFID)
	}
}

// The Motorola group regroup layouts are pinned in mac_reference_test.go,
// against SDRTrunk's field positions and a real MMR SACCH.
