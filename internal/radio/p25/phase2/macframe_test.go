package phase2

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// opMACVectors are 180-dibit timeslot packets (ISCH + body, one dibit per
// character) for WACN 1 / System 0x018 / NAC 0x293, each with the MAC PDU OP25
// recovers from it. They were produced by EncodeAirMACPacket and then decoded by
// OP25's own p25p2_tdma code (handle_packet's DUID lookup and XOR mask from
// apps/tdma/lfsr.py, handle_acch_frame's bit layout, ezpwd RS<63,35> and its
// crc12_ok / crc16, copied verbatim into a C++ harness): every one decoded
// crc_ok=1 to these bytes. The LCCH's last bytes are its CRC-16, as OP25's are.
var opMACVectors = []struct {
	slot int
	duid DUID
	pkt  string
	pdu  string
}{
	{3, DUIDFACCHScrambled, "111311311113331333332133030113313330223201020300223023230131323303230102330122300301021230000000000000000000002303120120320103212233033310133121313201233221122231131221123320231322", "0724415e7b98b5d2ef0c294663809dbad7f4"},
	{4, DUIDSACCHScrambled, "010110122210103301030331321112312113232002031311103220210302323332211321000023310001320303321100210212233321012112321102132211031112130320301122321001120131201322312200201233113001", "14314e6b88a5c2dffc193653708daac7e4011e3b58"},
	{7, DUIDSACCHClear, "111311311113331333333020103321123132021112302303332300021002121003120013312122231331103301003202231020121103101100213321233330133213122003031312010013201133330100301111112000002332", "213e5b7895b2cfec092643607d9ab7d4f10e2b4865"},
	{2, DUIDFACCHClear, "111311311113331333333023210231220201122022333313033210112303031100123120222213301032013330000000000000000000002012301203030002322130333321033331013330032310130132333113100220111033", "2e4b6885a2bfdcf91633506d8aa7c4e1fe1b"},
	{6, DUIDLCCHClear, "111311311113331333333032311201311210222333030322100120203110001131132221132310310132320023022010111202133232130230131320011103110232222201332000321013133012002303203321220001233331", "3b587592afcce90623405d7a97b4d1ee0b2845627ee72c"},
}

func vectorPacket(t *testing.T, s string) []uint8 {
	t.Helper()
	if len(s) != AirPacketDibits {
		t.Fatalf("packet has %d dibits", len(s))
	}
	out := make([]uint8, len(s))
	for i, c := range s {
		out[i] = uint8(c - '0')
	}
	return out
}

func vectorMasks() *AirMasks {
	return NewAirMasks(framing.PN44SeedFromIdentity(0x1, 0x018, 0x293))
}

func TestDecodeAirMACMatchesOP25(t *testing.T) {
	masks := vectorMasks()
	for _, v := range opMACVectors {
		pkt := vectorPacket(t, v.pkt)
		b := AirBurst{Slot: v.slot, DUID: packetDUID(pkt), Packet: pkt}
		if b.DUID != v.duid {
			t.Fatalf("slot %d: DUID %v, want %v", v.slot, b.DUID, v.duid)
		}
		m, ok := DecodeAirMAC(b, masks)
		want, _ := hex.DecodeString(v.pdu)
		n := len(want)
		if v.duid == DUIDLCCHClear {
			n = 22 // the 23rd byte is OP25's read past the 180 bits
			want = want[:n]
		}
		if !ok || !bytes.Equal(m.Bytes[:n], want) || m.Corrected != 0 {
			t.Fatalf("slot %d %v: ok=%v corrected=%d bytes %x, OP25 %x", v.slot, v.duid, ok, m.Corrected, m.Bytes, want)
		}
	}
}

// The punctured RS(63,35) corrects 9 symbol errors on a FACCH (RS(45,26)) and
// 11 on a SACCH (RS(52,30)); a 10th / 12th must not decode to a wrong PDU.
func TestDecodeAirMACCorrectsSymbolErrors(t *testing.T) {
	masks := vectorMasks()
	for _, v := range opMACVectors[:2] {
		lay, _, _, _ := airMACFor(v.duid)
		tmax := (lay.present - (lay.pduBits+12+5)/6) / 2
		for _, nErr := range []int{tmax, tmax + 1} {
			pkt := vectorPacket(t, v.pkt)
			// Hit one dibit inside each of nErr distinct hexbits (3 dibits
			// per hexbit along the spans).
			var flat []int
			for _, sp := range lay.spans {
				for i := sp[0]; i < sp[1]; i++ {
					flat = append(flat, i)
				}
			}
			for k := 0; k < nErr; k++ {
				pkt[AirISCHDibits+flat[3*(k*2)]] ^= 1
			}
			m, ok := DecodeAirMAC(AirBurst{Slot: v.slot, DUID: v.duid, Packet: pkt}, masks)
			want, _ := hex.DecodeString(v.pdu)
			switch {
			case nErr <= tmax && (!ok || !bytes.Equal(m.Bytes, want)):
				t.Errorf("%v: %d symbol errors not corrected (ok=%v)", v.duid, nErr, ok)
			case nErr > tmax && ok && !bytes.Equal(m.Bytes, want):
				t.Errorf("%v: %d symbol errors decoded to a wrong PDU", v.duid, nErr)
			}
		}
	}
}

// The wrong scrambling seed fails the CRC instead of yielding a PDU.
func TestDecodeAirMACNeedsTheRightSeed(t *testing.T) {
	wrong := NewAirMasks(framing.PN44SeedFromIdentity(0x1, 0x018, 0x294))
	v := opMACVectors[0]
	if _, ok := DecodeAirMAC(AirBurst{Slot: v.slot, DUID: v.duid, Packet: vectorPacket(t, v.pkt)}, wrong); ok {
		t.Fatal("decoded a scrambled FACCH with the wrong seed")
	}
}

// Messages walks a MAC PDU the way OP25's decode_mac_msg does: lengths from the
// opcode table, Null fill skipped.
func TestAirMACMessages(t *testing.T) {
	pdu := make([]byte, 18)
	pdu[0] = byte(AirMACActive)<<5 | 2<<2
	copy(pdu[1:], []byte{0x01, 0x40, 0x12, 0x34, 0xAB, 0xCD, 0xEF}) // GroupVoiceChannelUser (abbrev), 7 bytes
	copy(pdu[8:], []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // opcode 0x03, 7 bytes per the table
	m := AirMAC{Type: AirMACActive, Offset: 2, Bytes: pdu}
	msgs := m.Messages()
	if len(msgs) != 2 || msgs[0].Opcode != OpGroupVoiceChannelUserAbbreviated || msgs[1].Opcode != 0x03 {
		t.Fatalf("messages %+v", msgs)
	}
	u, ok := msgs[0].AsGroupVoiceChannelUser()
	if !ok || u.GroupAddress != 0x1234 || u.SourceID != 0xABCDEF || u.ServiceOptions != 0x40 {
		t.Fatalf("group voice user %+v ok=%v", u, ok)
	}
}

// MAC_PTT carries the encryption sync and the talker in fixed fields (OP25
// handle_mac_ptt).
func TestAirMACPTTFields(t *testing.T) {
	pdu := make([]byte, 18)
	pdu[0] = byte(AirMACPTT) << 5
	copy(pdu[1:10], []byte{1, 2, 3, 4, 5, 6, 7, 8, 9})
	pdu[10] = 0x84
	pdu[11], pdu[12] = 0x12, 0x34
	pdu[13], pdu[14], pdu[15] = 0x0A, 0x0B, 0x0C
	pdu[16], pdu[17] = 0x56, 0x78
	pkt, err := EncodeAirMACPacket(4, DUIDFACCHScrambled, pdu, vectorMasks(), 0)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := DecodeAirMAC(AirBurst{Slot: 4, DUID: DUIDFACCHScrambled, Packet: pkt}, vectorMasks())
	if !ok {
		t.Fatal("no decode")
	}
	p, ok := m.PTT()
	if !ok || p.Encryption.AlgorithmID != 0x84 || p.Encryption.KeyID != 0x1234 || p.Source != 0x0A0B0C || p.Group != 0x5678 || p.Encryption.MessageIndicator[8] != 9 {
		t.Fatalf("ptt %+v ok=%v", p, ok)
	}
	if m.Messages() != nil {
		t.Fatal("MAC_PTT has no messages")
	}
}
