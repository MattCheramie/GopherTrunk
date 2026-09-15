package voice

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/dmr"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildPIHeaderBurstDibits builds a full on-air Privacy Indicator header
// burst: BPTC(196,96) over the 12-octet header, the slot-type Hamming
// codeword for DTPIHeader, and a BS-Data sync.
func buildPIHeaderBurstDibits(t *testing.T, h dmr.PIHeader, colorCode uint8) []uint8 {
	t.Helper()
	return buildDataBurstDibits(t, dmr.AssemblePIHeader(h), dmr.DTPIHeader, colorCode)
}

// buildDataBurstDibits frames a 12-octet BPTC info block as a data burst
// of the given slot type with a BS-Data sync.
func buildDataBurstDibits(t *testing.T, info []byte, dt dmr.DataType, colorCode uint8) []uint8 {
	t.Helper()
	bits := make([]byte, 96)
	for i := 0; i < 96; i++ {
		bits[i] = (info[i>>3] >> uint(7-(i&7))) & 1
	}
	payloadDibits := framing.BitsToDibits(framing.EncodeBPTC196_96(bits))
	slotDibits := framing.BitsToDibits(dmr.AssembleSlotType(dmr.SlotType{ColorCode: colorCode, DataType: dt}))
	burst := make([]uint8, 0, dmr.BurstDibits)
	burst = append(burst, payloadDibits[:dmr.HalfPayloadDibits]...)
	burst = append(burst, slotDibits[:dmr.SlotTypeDibits]...)
	burst = append(burst, dmr.BSData.Dibits[:]...)
	burst = append(burst, slotDibits[dmr.SlotTypeDibits:]...)
	burst = append(burst, payloadDibits[dmr.HalfPayloadDibits:]...)
	return burst
}

func TestPIHeaderDetectorFindsHeader(t *testing.T) {
	want := dmr.PIHeader{AlgID: dmr.PIAlgRC4, FID: dmr.PIFIDDMRA, KeyID: 7, MI: [4]byte{0xDE, 0xAD, 0xBE, 0xEF}, DstAddr: 0x64}
	burst := buildPIHeaderBurstDibits(t, want, 5)
	stream := append(make([]uint8, 60), burst...)
	stream = append(stream, make([]uint8, 40)...)

	d := NewPIHeaderDetector()
	got := d.Process(stream, 0)
	if len(got) != 1 {
		t.Fatalf("expected 1 PI header, got %d", len(got))
	}
	if got[0].AlgID != want.AlgID || got[0].KeyID != 7 || got[0].MI32() != 0xDEADBEEF || got[0].DstAddr != 0x64 {
		t.Fatalf("recovered header wrong: %+v", got[0])
	}
	if n, _ := d.Rejects(); n != 0 {
		t.Fatalf("rejects = %d, want 0", n)
	}
}

// The detector must be indifferent to how the stream is chunked (the live
// chain hands it receiver-sized windows and a burst straddles them).
func TestPIHeaderDetectorChunked(t *testing.T) {
	want := dmr.PIHeader{AlgID: dmr.PIAlgRC4, FID: dmr.PIFIDDMRA, KeyID: 1, MI: [4]byte{1, 2, 3, 4}, DstAddr: 9}
	burst := buildPIHeaderBurstDibits(t, want, 1)
	stream := append(make([]uint8, 33), burst...)
	stream = append(stream, make([]uint8, 70)...)
	d := NewPIHeaderDetector()
	var got []dmr.PIHeader
	for i := 0; i < len(stream); i += 17 {
		e := i + 17
		if e > len(stream) {
			e = len(stream)
		}
		got = append(got, d.Process(stream[i:e], i)...)
	}
	if len(got) != 1 || got[0].MI32() != 0x01020304 {
		t.Fatalf("got %d headers (%+v), want 1 with MI 0x01020304", len(got), got)
	}
}

// A Terminator-with-LC (another data burst) and a voice stream must not
// read as PI headers; a PI-typed burst with a corrupt CRC is counted as a
// reject with its octets kept, never reported.
func TestPIHeaderDetectorIgnoresOtherBursts(t *testing.T) {
	stream, _ := buildStream(t, 1, 40)
	stream = append(stream, buildTerminatorBurstDibits(t, dmr.FLC{FLCO: dmr.FLCOGroupVoiceUser, DstAddr: 1, SrcAddr: 2}, 1)...)
	info := dmr.AssemblePIHeader(dmr.PIHeader{AlgID: dmr.PIAlgRC4, FID: dmr.PIFIDDMRA, KeyID: 3, MI: [4]byte{5, 6, 7, 8}})
	info[11] ^= 0x10 // corrupt the CRC
	stream = append(stream, buildDataBurstDibits(t, info, dmr.DTPIHeader, 1)...)
	stream = append(stream, make([]uint8, 80)...)

	d := NewPIHeaderDetector()
	if got := d.Process(stream, 0); len(got) != 0 {
		t.Fatalf("expected no PI headers, got %d: %+v", len(got), got)
	}
	n, hexs := d.Rejects()
	if n != 1 || hexs == "" {
		t.Fatalf("rejects = %d (%q), want 1 with the raw octets", n, hexs)
	}
}
