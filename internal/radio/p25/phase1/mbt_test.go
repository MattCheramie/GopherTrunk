package phase1

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
)

// TestParseSecondaryControlChannelBroadcastLayout pins the 0x39 payload
// layout against SDRTrunk's SecondaryControlChannelBroadcast bit offsets
// (RFSS 16-23, SITE 24-31, CH A 32-47, SC A 48-55, CH B 56-71, SC B
// 72-79). The previous working model read channel B from payload bytes
// 4-5 — one byte early, splicing service class A into the channel field —
// and its round-trip test passed because the assembler encoded the same
// wrong layout. A literal byte vector cannot be fooled that way: this
// test fails against the old parser (channel B = 7-0x02D phantom).
func TestParseSecondaryControlChannelBroadcastLayout(t *testing.T) {
	// RFSS 1, Site 1, chan A 2-1692 (0x69C), SC A 0x70,
	// chan B 2-3342 (0xD0E), SC B 0x50.
	p := [8]byte{0x01, 0x01, 0x26, 0x9C, 0x70, 0x2D, 0x0E, 0x50}
	s := ParseSecondaryControlChannelBroadcast(p)
	if s.RFSS != 1 || s.Site != 1 {
		t.Errorf("RFSS/Site = %d/%d, want 1/1", s.RFSS, s.Site)
	}
	if s.ChannelAID != 2 || s.ChannelANumber != 1692 {
		t.Errorf("channel A = %d-%d, want 2-1692", s.ChannelAID, s.ChannelANumber)
	}
	if s.ServiceClassA != 0x70 {
		t.Errorf("service class A = %#x, want 0x70", s.ServiceClassA)
	}
	if s.ChannelBID != 2 || s.ChannelBNumber != 3342 {
		t.Errorf("channel B = %d-%d, want 2-3342 (old layout read bytes 4-5: 7-45)", s.ChannelBID, s.ChannelBNumber)
	}
	if s.ServiceClassB != 0x50 {
		t.Errorf("service class B = %#x, want 0x50", s.ServiceClassB)
	}
	if back := AssembleSecondaryControlChannelBroadcast(s); back != p {
		t.Errorf("assemble round-trip = %x, want %x", back, p)
	}
}

// TestParseAdjacentSiteStatusBroadcastCFVA pins that the 0x3C parser
// captures the CFVA flags (byte 1 high nibble) and system service class
// (byte 7) — previously read only for a diagnostic log line, never into
// the struct — without disturbing the identity/channel fields.
func TestParseAdjacentSiteStatusBroadcastCFVA(t *testing.T) {
	// LRA 0, CFVA valid+active (0x3), sysid 0x2C2, RFSS 1, Site 7,
	// channel 2-1754 (0x6DA), service class 0x70.
	p := [8]byte{0x00, 0x32, 0xC2, 0x01, 0x07, 0x26, 0xDA, 0x70}
	a := ParseAdjacentSiteStatusBroadcast(p)
	if a.SystemID != 0x2C2 || a.RFSS != 1 || a.Site != 7 {
		t.Errorf("identity = sysid %#x rfss %d site %d, want 0x2C2/1/7", a.SystemID, a.RFSS, a.Site)
	}
	if a.ChannelID != 2 || a.ChannelNumber != 1754 {
		t.Errorf("channel = %d-%d, want 2-1754", a.ChannelID, a.ChannelNumber)
	}
	if a.CFVA != CFVAValid|CFVAActive {
		t.Errorf("CFVA = %#x, want valid|active (0x3)", a.CFVA)
	}
	if a.ServiceClass != 0x70 {
		t.Errorf("service class = %#x, want 0x70", a.ServiceClass)
	}
	if back := AssembleAdjacentSiteStatusBroadcast(a); back != p {
		t.Errorf("assemble round-trip = %x, want %x", back, p)
	}
}

// buildAMBTAdjacentFrame builds one on-air FSW + NID(PDU) + header block +
// data block frame carrying an AMBT Adjacent Site Status Broadcast.
func buildAMBTAdjacentFrame(nac uint16, lra uint8, sysid uint16, rfss, site uint8, dlID uint8, dlNum uint16, ulID uint8, ulNum uint16) []uint8 {
	header := AssembleMBTHeader(MBTHeader{
		Format: MBTFormatAlternate, SAP: MBTSAPTrunkingControl,
		BlocksToFollow: 1, Opcode: OpAdjacentSiteStatusBroadcast,
	}, [3]byte{lra, byte(sysid >> 8 & 0x0F), byte(sysid)}, [2]byte{rfss, site})
	block := make([]byte, 12)
	block[0] = dlID<<4 | byte(dlNum>>8&0x0F)
	block[1] = byte(dlNum)
	block[2] = ulID<<4 | byte(ulNum>>8&0x0F)
	block[3] = byte(ulNum)
	appendMBTDataCRC32(block)

	frame := make([]uint8, 0, 24+32+2*98)
	frame = append(frame, FrameSyncWord[:]...)
	bits := EncodeNIDBits(nac, DUIDPacketDataUnit)
	for i := 0; i < 32; i++ {
		frame = append(frame, (bits[2*i]<<1)|bits[2*i+1])
	}
	frame = append(frame, EncodeTSBKChannel(header)...)
	frame = append(frame, EncodeTSBKChannel(block)...)
	return frame
}

// TestControlChannelDecodesAMBTAdjacentStatus pins the Multi-Block
// Trunking path end-to-end: a PDU (DUID 0xC) on the control channel
// carrying an AMBT Adjacent Site Status Broadcast must yield a neighbour
// — with its explicit uplink resolved — in the topology snapshot. Before
// the MBT path existed these frames were logged as "non-control DUID"
// and dropped, which is why systems that broadcast their neighbour list
// only in AMBT form showed one neighbour where SDRTrunk listed twelve.
// The stream is delivered in 19-dibit batches so the multi-block PDU
// continuation across Process calls is exercised too.
func TestControlChannelDecodesAMBTAdjacentStatus(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Ost", FrequencyHz: 450_125_000,
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }})
	// Band 2: base 440 MHz, 6.25 kHz spacing (the operator's system).
	cc.bandPlan.Apply(IdentifierUpdate{ChannelID: 2, BaseHz: 440_000_000, SpacingHz: 6_250, TxOffsetHz: -6_400_000})

	// Neighbour site 7: downlink 2-1754 → 450.9625 MHz, uplink 2-3430 →
	// 461.4375 MHz (the operator's SDRTrunk ground truth).
	onAir := InjectControlStatusSymbols(buildAMBTAdjacentFrame(
		0x2C1, 0, 0x2C2, 1, 7, 2, 1754, 2, 3430))
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)
	for i := 0; i < len(stream); i += 19 {
		end := min(i+19, len(stream))
		cc.Process(stream[i:end], i)
	}

	snap := cc.TopologySnapshot()
	if snap == nil {
		t.Fatal("TopologySnapshot nil after an AMBT adjacent broadcast")
	}
	if len(snap.Neighbors) != 1 {
		t.Fatalf("neighbors = %+v, want exactly the AMBT-announced site", snap.Neighbors)
	}
	n := snap.Neighbors[0]
	if n.RFSS != 1 || n.Site != 7 {
		t.Errorf("neighbor identity = rfss %d site %d, want 1/7", n.RFSS, n.Site)
	}
	if n.ChannelID != 2 || n.ChannelNumber != 1754 || n.FrequencyHz != 450_962_500 {
		t.Errorf("neighbor downlink = %d-%d @ %d, want 2-1754 @ 450962500", n.ChannelID, n.ChannelNumber, n.FrequencyHz)
	}
	if n.UplinkChannelID != 2 || n.UplinkChannelNumber != 3430 || n.UplinkHz != 461_437_500 {
		t.Errorf("neighbor uplink = %d-%d @ %d, want 2-3430 @ 461437500", n.UplinkChannelID, n.UplinkChannelNumber, n.UplinkHz)
	}
	// The AMBT names the system ID; it must vote into the camped identity.
	if snap.SystemID != 0x2C2 {
		t.Errorf("SystemID = %#x, want 0x2C2 (voted from the adjacent broadcast)", snap.SystemID)
	}
	if got := cc.Stats(); got.MBTDecoded != 1 || got.AdjacentSeen != 1 {
		t.Errorf("stats = MBTDecoded %d AdjacentSeen %d, want 1/1", got.MBTDecoded, got.AdjacentSeen)
	}
}

// TestControlChannelDecodesAMBTNetworkStatus pins the AMBT Network
// Status Broadcast (0x3B in a PDU): the WACN must reach the topology
// snapshot. This is the "No Network Status Broadcast yet" fix for
// systems that carry the WACN only in AMBT form.
func TestControlChannelDecodesAMBTNetworkStatus(t *testing.T) {
	bus := events.NewBus(32)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Ost", FrequencyHz: 450_125_000,
		Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }})

	header := AssembleMBTHeader(MBTHeader{
		Format: MBTFormatAlternate, SAP: MBTSAPTrunkingControl,
		BlocksToFollow: 1, Opcode: OpNetworkStatusBroadcast,
	}, [3]byte{0x00, 0x02, 0xC2}, [2]byte{})
	block := make([]byte, 12)
	// WACN 0xBEE00 in the top 20 bits of octets 0-2.
	block[0], block[1], block[2] = 0xBE, 0xE0, 0x00
	// Downlink 2-1620, uplink 2-3310.
	block[3], block[4] = 0x26, 0x54
	block[5], block[6] = 0x2C, 0xEE
	block[7] = 0x70
	appendMBTDataCRC32(block)

	frame := make([]uint8, 0, 24+32+2*98)
	frame = append(frame, FrameSyncWord[:]...)
	bits := EncodeNIDBits(0x2C1, DUIDPacketDataUnit)
	for i := 0; i < 32; i++ {
		frame = append(frame, (bits[2*i]<<1)|bits[2*i+1])
	}
	frame = append(frame, EncodeTSBKChannel(header)...)
	frame = append(frame, EncodeTSBKChannel(block)...)

	onAir := InjectControlStatusSymbols(frame)
	stream := make([]uint8, 10+len(onAir)+16)
	copy(stream[10:], onAir)
	cc.Process(stream, 0)

	snap := cc.TopologySnapshot()
	if snap == nil {
		t.Fatal("TopologySnapshot nil after an AMBT network status broadcast")
	}
	if snap.WACN != 0xBEE00 || snap.SystemID != 0x2C2 {
		t.Errorf("identity = WACN %#x SystemID %#x, want 0xBEE00/0x2C2", snap.WACN, snap.SystemID)
	}
	if got := cc.Stats(); got.MBTDecoded != 1 || got.NetStatusSeen != 1 {
		t.Errorf("stats = MBTDecoded %d NetStatusSeen %d, want 1/1", got.MBTDecoded, got.NetStatusSeen)
	}
}

// TestMBTDataCRCRejectsCorruption proves a corrupted data block cannot
// dispatch: the CRC-32 gate must reject it (a trellis-surviving wrong
// payload must not inject a phantom neighbour).
func TestMBTDataCRCRejectsCorruption(t *testing.T) {
	block := make([]byte, 12)
	block[0], block[1] = 0x26, 0xDA
	appendMBTDataCRC32(block)
	if err := ValidateMBTData(block); err != nil {
		t.Fatalf("valid block train rejected: %v", err)
	}
	block[1] ^= 0x01
	if err := ValidateMBTData(block); err == nil {
		t.Fatal("corrupted block train passed the CRC-32 gate")
	}
}

// TestNetworkModelMergesNeighborForms pins the TSBK/AMBT merge: the two
// adjacent-status forms carry complementary fields (CFVA + service class
// vs explicit uplink), and observing both must yield one neighbour with
// both halves, whichever order they arrive in.
func TestNetworkModelMergesNeighborForms(t *testing.T) {
	var m NetworkModel
	m.ApplyMBTAdjacentSite(MBTAdjacentSiteStatusBroadcast{
		SystemID: 0x2C2, RFSS: 1, Site: 7,
		ChannelID: 2, ChannelNumber: 1754, UplinkID: 2, UplinkNumber: 3430,
	})
	m.ApplyAdjacentSite(AdjacentSiteStatusBroadcast{
		SystemID: 0x2C2, RFSS: 1, Site: 7,
		ChannelID: 2, ChannelNumber: 1754,
		CFVA: CFVAValid | CFVAActive, ServiceClass: 0x70,
	})
	snap := m.Snapshot()
	if len(snap.Neighbors) != 1 {
		t.Fatalf("neighbors = %+v, want one merged entry", snap.Neighbors)
	}
	n := snap.Neighbors[0]
	if n.UplinkID != 2 || n.UplinkNumber != 3430 {
		t.Errorf("merged uplink = %d-%d, want 2-3430 (TSBK form must not clobber it)", n.UplinkID, n.UplinkNumber)
	}
	if !n.CFVAKnown || n.CFVA != CFVAValid|CFVAActive || n.ServiceClass != 0x70 {
		t.Errorf("merged flags = known %v cfva %#x sc %#x, want valid|active + 0x70", n.CFVAKnown, n.CFVA, n.ServiceClass)
	}
}

// TestQuietNonControlDUIDSuppressesLog pins the p25_quiet_noncontrol_duid
// knob: with it set, the per-frame "non-control DUID" debug line (fired for
// every TDU on a busy CC — the operator-reported debug-log flood) stays
// silent; without it the historical line still fires.
func TestQuietNonControlDUIDSuppressesLog(t *testing.T) {
	for _, quiet := range []bool{false, true} {
		var buf bytes.Buffer
		log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		bus := events.NewBus(8)
		cc := New(Options{Bus: bus, Log: log, SystemName: "S", QuietNonControlDUID: quiet})
		cc.Process(buildLockedStreamWithTSBK(10, 0x293, DUIDTerminator, TSBK{LB: true}), 0)
		bus.Close()
		got := strings.Contains(buf.String(), "non-control DUID")
		if got == quiet {
			t.Errorf("quiet=%v: non-control DUID logged=%v, want %v", quiet, got, !quiet)
		}
	}
}
