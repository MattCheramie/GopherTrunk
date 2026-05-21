package phase1

import (
	"bytes"
	"encoding/binary"
	"net/netip"
	"testing"
)

func TestPDUHeaderRoundTrip(t *testing.T) {
	in := PDUHeader{
		Format: PDUFmtConfirmedData, Confirmed: true,
		SAP: 0x04, MFID: 0x90, LLID: 0x00ABCD, BlockCount: 3,
	}
	got, err := ParsePDUHeader(AssemblePDUHeader(in))
	if err != nil {
		t.Fatalf("ParsePDUHeader: %v", err)
	}
	if got != in {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestReassemblePDU(t *testing.T) {
	h := PDUHeader{Format: PDUFmtUnconfirmedData, BlockCount: 2}
	b0 := bytes.Repeat([]byte{0xAA}, PDUBlockBytes)
	b1 := bytes.Repeat([]byte{0xBB}, PDUBlockBytes)
	pdu, err := ReassemblePDU(h, [][]byte{b0, b1})
	if err != nil {
		t.Fatalf("ReassemblePDU: %v", err)
	}
	if len(pdu.Payload) != 2*PDUBlockBytes {
		t.Errorf("payload len = %d, want %d", len(pdu.Payload), 2*PDUBlockBytes)
	}
	if _, err := ReassemblePDU(h, [][]byte{b0, b1[:5]}); err == nil {
		t.Error("ReassemblePDU accepted a short block")
	}
}

func TestSNDCPRoundTrip(t *testing.T) {
	in := SNDCPMessage{PDUType: 0x0A, NSAPI: 0x05, Payload: []byte("packet-data")}
	got, err := ParseSNDCP(AssembleSNDCP(in))
	if err != nil {
		t.Fatalf("ParseSNDCP: %v", err)
	}
	if got.PDUType != in.PDUType || got.NSAPI != in.NSAPI || !bytes.Equal(got.Payload, in.Payload) {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
	if _, err := ParseSNDCP([]byte{0x01}); err == nil {
		t.Error("ParseSNDCP accepted a 1-byte message")
	}
}

func TestParseIPv4(t *testing.T) {
	// Build a minimal 20-byte IPv4 header: 10.0.0.1 → 10.0.0.2, UDP.
	h := make([]byte, 20)
	h[0] = 0x45 // version 4, IHL 5 (20 bytes)
	binary.BigEndian.PutUint16(h[2:4], 48)
	h[9] = 17 // UDP
	copy(h[12:16], []byte{10, 0, 0, 1})
	copy(h[16:20], []byte{10, 0, 0, 2})

	got, err := ParseIPv4(h)
	if err != nil {
		t.Fatalf("ParseIPv4: %v", err)
	}
	if got.HeaderLen != 20 || got.TotalLength != 48 || got.Protocol != 17 {
		t.Errorf("header = %+v", got)
	}
	if got.Source != netip.AddrFrom4([4]byte{10, 0, 0, 1}) ||
		got.Dest != netip.AddrFrom4([4]byte{10, 0, 0, 2}) {
		t.Errorf("addrs = %v → %v", got.Source, got.Dest)
	}

	if _, err := ParseIPv4(h[:10]); err == nil {
		t.Error("ParseIPv4 accepted a truncated header")
	}
	bad := append([]byte(nil), h...)
	bad[0] = 0x65 // version 6 in an IPv4 parse
	if _, err := ParseIPv4(bad); err == nil {
		t.Error("ParseIPv4 accepted a non-IPv4 version")
	}
}

// TestPDUDataToIPEndToEnd walks the full data stack: an IPv4 packet
// wrapped in SNDCP, carried as a PDU payload, decoded back out.
func TestPDUDataToIPEndToEnd(t *testing.T) {
	ip := make([]byte, 20)
	ip[0] = 0x45
	binary.BigEndian.PutUint16(ip[2:4], 20)
	ip[9] = 6 // TCP
	copy(ip[12:16], []byte{172, 16, 0, 9})
	copy(ip[16:20], []byte{8, 8, 8, 8})

	sndcp := AssembleSNDCP(SNDCPMessage{PDUType: 1, NSAPI: 2, Payload: ip})
	// Pad the SNDCP message up to a whole number of 12-byte blocks.
	for len(sndcp)%PDUBlockBytes != 0 {
		sndcp = append(sndcp, 0)
	}
	var blocks [][]byte
	for off := 0; off < len(sndcp); off += PDUBlockBytes {
		blocks = append(blocks, sndcp[off:off+PDUBlockBytes])
	}
	h := PDUHeader{Format: PDUFmtUnconfirmedData, BlockCount: uint8(len(blocks))}

	pdu, err := ReassemblePDU(h, blocks)
	if err != nil {
		t.Fatalf("ReassemblePDU: %v", err)
	}
	msg, err := ParseSNDCP(pdu.Payload)
	if err != nil {
		t.Fatalf("ParseSNDCP: %v", err)
	}
	got, err := ParseIPv4(msg.Payload)
	if err != nil {
		t.Fatalf("ParseIPv4: %v", err)
	}
	if got.Protocol != 6 || got.Dest != netip.AddrFrom4([4]byte{8, 8, 8, 8}) {
		t.Errorf("recovered IPv4 = %+v, want TCP → 8.8.8.8", got)
	}
}
