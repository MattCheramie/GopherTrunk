package tetra

import "testing"

// sysInfoBits assembles a MAC broadcast SYSINFO PDU header (through the reverse
// operation field) as a one-bit-per-byte slice, per EN 300 392-2 §21.4.4.1
// Table 21.65. Trailing SYSINFO fields are padded with zeros — ParseSysInfo only
// reads through reverse operation.
func sysInfoBits(mainCarrier uint16, band, offset, duplex uint8, reverse bool) []byte {
	w := &cmceBitWriter{}
	w.u(uint64(MACPDUBroadcast), 2)
	w.u(uint64(TETRAMACBcastSysInfo), 2)
	w.u(uint64(mainCarrier), 12)
	w.u(uint64(band), 4)
	w.u(uint64(offset), 2)
	w.u(uint64(duplex), 3)
	if reverse {
		w.u(1, 1)
	} else {
		w.u(0, 1)
	}
	w.u(0, 32) // padding for the remaining SYSINFO fields
	return w.bits
}

// TestParseSysInfoFields decodes all five frequency parameters from a SYSINFO
// broadcast for the reported "fat" cell: main carrier 2795, band 4, offset 01
// (+6.25 kHz). This is the field that was previously dropped — SysInfoMainCarrier
// stopped after the 12-bit main carrier.
func TestParseSysInfoFields(t *testing.T) {
	bits := sysInfoBits(2795, 4, 1, 0, false)
	si, ok := ParseSysInfo(bits)
	if !ok {
		t.Fatal("ParseSysInfo ok=false on a valid SYSINFO PDU")
	}
	if si.MainCarrier != 2795 || si.FreqBand != 4 || si.Offset != 1 || si.DuplexSpacing != 0 || si.ReverseOper {
		t.Fatalf("ParseSysInfo = %+v, want {2795 4 1 0 false}", si)
	}
	// A MAC-RESOURCE PDU (leading type 00) is not a SYSINFO broadcast.
	if _, ok := ParseSysInfo(hexToBits(t, "20760f572c3c83d538c90a1fe305009e")); ok {
		t.Error("ParseSysInfo accepted a non-broadcast PDU")
	}
}

// TestTetraDLCarrierHz pins the offset field's effect on the true downlink
// carrier: the cell broadcasts main carrier 2795 in band 4, which is nominally
// 469.875 MHz, but offset 01 (+6.25 kHz) puts the real carrier at 469.88125 MHz.
// This ±6.25 kHz is exactly the gap between the broadcast frequency and where a
// receiver must actually tune to lock.
func TestTetraDLCarrierHz(t *testing.T) {
	cases := []struct {
		band    uint8
		carrier uint16
		offset  uint8
		want    int64
	}{
		{4, 2795, 0, 469_875_000},   // no offset: nominal
		{4, 2795, 1, 469_881_250},   // +6.25 kHz — the reported cell
		{4, 2795, 2, 469_868_750},   // -6.25 kHz
		{4, 2795, 3, 469_887_500},   // +12.5 kHz
		{4, 2716, 3, 467_912_500},   // the 467.913 capture (offset field 3)
	}
	for _, tc := range cases {
		if got := tetraDLCarrierHz(tc.band, tc.carrier, tc.offset); got != tc.want {
			t.Errorf("tetraDLCarrierHz(band=%d, carrier=%d, offset=%d) = %d, want %d",
				tc.band, tc.carrier, tc.offset, got, tc.want)
		}
	}
}

// TestTetraULCarrierHz checks the uplink derivation from duplex spacing and
// reverse operation. The 400 MHz band uses a 10 MHz duplex, so DL 469.881250 →
// UL 459.881250 (normal) or 479.881250 (reverse).
func TestTetraULCarrierHz(t *testing.T) {
	const dl = 469_881_250
	if ul, ok := tetraULCarrierHz(dl, 4, 0, false); !ok || ul != 459_881_250 {
		t.Errorf("normal UL = %d (ok=%v), want 459881250", ul, ok)
	}
	if ul, ok := tetraULCarrierHz(dl, 4, 0, true); !ok || ul != 479_881_250 {
		t.Errorf("reverse UL = %d (ok=%v), want 479881250", ul, ok)
	}
	// An unmapped band yields ok=false (uplink reported unknown, not guessed).
	if _, ok := tetraULCarrierHz(dl, 7, 0, false); ok {
		t.Error("tetraULCarrierHz returned ok=true for an unmapped band")
	}
}

// TestControlChannelLearnsSysInfoFrequencies drives the full ingest: a SYSINFO
// broadcast teaches band 4 / carrier 2795 / offset +6.25 kHz, and the topology
// then reports the offset-corrected downlink and the derived uplink.
func TestControlChannelLearnsSysInfoFrequencies(t *testing.T) {
	cc := New(Options{FrequencyHz: 469_880_000}) // tuned near the real carrier
	cc.ingestMAC(sysInfoBits(2795, 4, 1, 0, false))
	topo := cc.Topology()
	if topo.MainCarrier != 2795 {
		t.Errorf("MainCarrier = %d, want 2795", topo.MainCarrier)
	}
	if topo.DownlinkHz != 469_881_250 {
		t.Errorf("DownlinkHz = %d, want 469881250 (offset-corrected)", topo.DownlinkHz)
	}
	if topo.UplinkHz != 459_881_250 {
		t.Errorf("UplinkHz = %d, want 459881250", topo.UplinkHz)
	}
	// A grant on the cell's own carrier resolves to the true downlink carrier.
	if hz, ok := cc.carrierFrequency(2795); !ok || hz != 469_881_250 {
		t.Errorf("carrierFrequency(2795) = %d (ok=%v), want 469881250", hz, ok)
	}
}
