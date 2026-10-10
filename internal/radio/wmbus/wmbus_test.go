package wmbus

import (
	"bytes"
	"encoding/hex"
	"math"
	"math/bits"
	"strings"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The reveng catalogue's check value for CRC-16/EN-13757 (poly 0x3D65,
// init 0, unreflected, xorout 0xFFFF) over "123456789".
func TestCRC16CatalogueCheck(t *testing.T) {
	if got := CRC16([]byte("123456789")); got != 0xC2B7 {
		t.Fatalf("CRC16(123456789) = %#04x, want 0xc2b7", got)
	}
}

func TestCode3of6(t *testing.T) {
	seen := map[uint8]bool{}
	for n, c := range code3of6 {
		if bits.OnesCount8(c) != 3 || c >= 64 || seen[c] {
			t.Errorf("nibble %X: code %06b is not a distinct 6-chip weight-3 word", n, c)
		}
		seen[c] = true
		if decode3of6[c] != uint8(n) {
			t.Errorf("decode3of6[%06b] = %d, want %d", c, decode3of6[c], n)
		}
	}
	valid := 0
	for _, v := range decode3of6 {
		if v != 0xFF {
			valid++
		}
	}
	if valid != 16 {
		t.Errorf("%d valid words, want 16", valid)
	}
}

// Format A frame lengths: L, plus the 9 header bytes in block 1, then up to
// 16 data bytes per block, each block with a 2-byte CRC. Literal values
// from rtl-wmbus's L-field length table.
func TestFrameLengthFormatA(t *testing.T) {
	for l, want := range map[byte]int{9: 12, 10: 15, 25: 30, 26: 33, 41: 48, 78: 91, 206: 235, 255: 290} {
		if got := frameLength(l, FormatA); got != want {
			t.Errorf("frameLength(%d, A) = %d, want %d", l, got, want)
		}
	}
	if frameLength(8, FormatA) != 0 {
		t.Error("an L-field too short for the header accepted")
	}
}

// Real-air frames as received (CRC bytes included) from the rtl-wmbus
// sample captures, and the CRC-stripped data rtl-wmbus itself prints for
// them: an independent decoder validated these CRCs.
var realAirFrames = []struct {
	id, raw, data string
}{
	{"71200023",
		"294468502300207172f03840a0009f21000000a07d220200010000005b94000000000000000004000302030200002b75",
		"294468502300207172f0a0009f21000000a07d2202000100000000000000000000000400030203020000"},
	{"20210116",
		"574468501601212094082bc78c0013900f002c2579610900e048398413bd8dad6f3c7a7900300710719e84d90bd1125a44364d15579431a00ee6e761e308e774cd78232e8e6b2501c9753ec028b273bddee5e73a20d299996eb76edd6b9f0f4866695d1a",
		"574468501601212094088c0013900f002c2579610900e04839848dad6f3c7a7900300710719e84d90bd144364d15579431a00ee6e761e308e774232e8e6b2501c9753ec028b273bddee520d299996eb76edd6b9f0f486669"},
	{"19131290",
		"4e44b4099012131913077bb87a790040057eda852abb08c480d64db9c4fad22a3c1098b0873e3227280131a7a9523582005eee56a7ebdea93b2d2b93e9bdfd6692b7b35fcbc6d2c3842e323e98528acadf71d9d30d035dbc945f21",
		"4e44b4099012131913077a790040057eda852abb08c480d64db9d22a3c1098b0873e3227280131a7a952005eee56a7ebdea93b2d2b93e9bdfd66b35fcbc6d2c3842e323e98528acadf710d035dbc94"},
}

func TestStripCRCRealAirFrames(t *testing.T) {
	for _, f := range realAirFrames {
		raw := mustHex(t, f.raw)
		data, ok, err := stripCRC(raw, FormatA)
		if err != nil || !ok {
			t.Fatalf("%s: stripCRC ok=%v err=%v", f.id, ok, err)
		}
		if !bytes.Equal(data, mustHex(t, f.data)) {
			t.Errorf("%s: data\n got %x\nwant %s", f.id, data, f.data)
		}
		bad := append([]byte(nil), raw...)
		bad[len(bad)/2] ^= 0x10
		if _, ok, _ := stripCRC(bad, FormatA); ok {
			t.Errorf("%s: a flipped bit passed the CRC", f.id)
		}
		// The frame builder used by the synthesiser reproduces the
		// on-air CRCs exactly.
		if got := buildFrame(data, FormatA); !bytes.Equal(got, raw) {
			t.Errorf("%s: buildFrame\n got %x\nwant %x", f.id, got, raw)
		}
	}
}

func TestParseTelegramRealAir(t *testing.T) {
	parse := func(data string, m Mode) Telegram {
		tg, err := ParseTelegram(Frame{Mode: m, Format: FormatA, Data: mustHex(t, data), CRCOK: true})
		if err != nil {
			t.Fatal(err)
		}
		return tg
	}
	// Techem, manufacturer-specific application layer (CI 0xA0).
	tg := parse(realAirFrames[0].data, ModeT1)
	if tg.Manufacturer != "TCH" || tg.ID != "71200023" || tg.Version != 0x72 || tg.AppCI != 0xA0 || tg.Encrypted {
		t.Errorf("71200023: %+v", tg)
	}
	// Techem heat cost allocator over C1: extended link layer, then an
	// authentication layer, then a short TPL header in security mode 7.
	tg = parse(realAirFrames[1].data, ModeC1)
	if tg.Manufacturer != "TCH" || tg.ID != "20210116" || tg.DeviceType != 0x08 ||
		!tg.HasELL || !tg.HasAFL || tg.AppCI != 0x7A || tg.SecurityMode != 7 || !tg.Encrypted {
		t.Errorf("20210116: %+v", tg)
	}
	if tg.DeviceTypeName() != "heat cost allocator" {
		t.Errorf("device type name %q", tg.DeviceTypeName())
	}
	// BMeters water meter, short TPL header, security mode 5.
	tg = parse(realAirFrames[2].data, ModeT1)
	if tg.Manufacturer != "BMT" || tg.ID != "19131290" || tg.DeviceType != 0x07 ||
		tg.Access != 0x79 || tg.SecurityMode != 5 || !tg.Encrypted || len(tg.Records) != 0 {
		t.Errorf("19131290: %+v", tg)
	}
}

// Unencrypted telegrams from wmbusmeters' simulation files, checked against
// the values wmbusmeters' own drivers decode from them.
func TestParseRecordsWMBusmetersVectors(t *testing.T) {
	type want struct {
		quantity string
		storage  int
		value    float64
		text     string
	}
	for _, tc := range []struct {
		name, data, mfr, id string
		devType             byte
		want                []want
	}{
		{
			// qcaloric: current_consumption_hca 127, set_date 2018-12-31,
			// consumption_at_set_date_hca 145, set_date_17 2019-01-31,
			// consumption_at_set_date_17_hca 79, device_datetime
			// 2019-02-20 11:32.
			name: "qcaloric", mfr: "QDS", id: "78563412", devType: 0x08,
			data: "314493441234567835087a740000200b6e2701004b6e450100426c5f2ccb086e790000c2086c7f21326cffff046d200b7422",
			want: []want{
				{"heat cost allocation", 0, 127, ""},
				{"heat cost allocation", 1, 145, ""},
				{"date", 1, 0, "2018-12-31"},
				{"heat cost allocation", 17, 79, ""},
				{"date", 17, 0, "2019-01-31"},
				{"date", 0, 0, "invalid date FFFF"},
				{"date time", 0, 0, "2019-02-20 11:32"},
			},
		},
		{
			// iperl: total_m3 123.529, max_flow_m3h 0.
			name: "iperl", mfr: "SEN", id: "33225544", devType: 0x07,
			data: "1844AE4C4455223368077A55000000041389E20100023B0000",
			want: []want{
				{"volume", 0, 123.529, ""},
				{"volume flow", 0, 0, ""},
			},
		},
	} {
		tg, err := ParseTelegram(Frame{Mode: ModeC1, Format: FormatA, Data: mustHex(t, tc.data), CRCOK: true})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if tg.Manufacturer != tc.mfr || tg.ID != tc.id || tg.DeviceType != tc.devType || tg.Encrypted {
			t.Fatalf("%s: header %s %s type %#x encrypted=%v", tc.name, tg.Manufacturer, tg.ID, tg.DeviceType, tg.Encrypted)
		}
		if tg.RecordsErr != "" || len(tg.Records) != len(tc.want) {
			t.Fatalf("%s: %d records (err %q), want %d: %v", tc.name, len(tg.Records), tg.RecordsErr, len(tc.want), tg.Records)
		}
		for i, w := range tc.want {
			r := tg.Records[i]
			if r.Quantity != w.quantity || r.Storage != w.storage || r.Text != w.text ||
				(w.text == "" && (!r.HasValue || math.Abs(r.Value-w.value) > 1e-9)) {
				t.Errorf("%s record %d = %q (storage %d, value %v, text %q), want %+v", tc.name, i, r.String(), r.Storage, r.Value, r.Text, w)
			}
		}
	}
}

func TestManufacturerFlagAndBCD(t *testing.T) {
	if got := ManufacturerFlag(0x2C2D); got != "KAM" { // M-field bytes 2D 2C, little-endian
		t.Errorf("0x2C2D = %q, want KAM", got)
	}
	if got := bcdID([]byte{0x78, 0x56, 0x34, 0x12}); got != "12345678" {
		t.Errorf("bcdID = %q", got)
	}
}

func framesFromChips(chips []uint8) []Frame {
	var got []Frame
	fr := NewFramer(func(f Frame) { got = append(got, f) })
	for _, c := range chips {
		fr.Push(c)
	}
	return got
}

func TestFramerRoundTrip(t *testing.T) {
	short := mustHex(t, realAirFrames[0].data)
	// A long payload exercises format A's 16-byte blocks and format B's
	// third block (over 126 bytes).
	long := make([]byte, 160)
	copy(long, short)
	for i := len(short); i < len(long); i++ {
		long[i] = byte(i * 7)
	}
	long[0] = byte(len(long) - 1)
	for _, tc := range []struct {
		mode   Mode
		format Format
		data   []byte
	}{
		{ModeT1, FormatA, short}, {ModeT1, FormatA, long},
		{ModeC1, FormatA, short}, {ModeC1, FormatA, long},
		{ModeC1, FormatB, short}, {ModeC1, FormatB, long},
	} {
		raw := buildFrame(tc.data, tc.format)
		chips := append(make([]uint8, 37), encodeFrameChips(tc.mode, tc.format, raw, 48)...)
		got := framesFromChips(chips)
		if len(got) != 1 {
			t.Fatalf("%s-%s len %d: %d frames", tc.mode, tc.format, len(tc.data), len(got))
		}
		f := got[0]
		if f.Mode != tc.mode || f.Format != tc.format || !f.CRCOK || !bytes.Equal(f.Data, tc.data) {
			t.Errorf("%s-%s len %d: got %s-%s crc=%v data %x", tc.mode, tc.format, len(tc.data), f.Mode, f.Format, f.CRCOK, f.Data)
		}
	}
}

// The receiver decodes T1 and C1 frames across input rates, tuning
// error, chip-rate error (T mode allows several percent) and noise.
func TestReceiverSynthetic(t *testing.T) {
	data := mustHex(t, realAirFrames[1].data)
	for i, tc := range []struct {
		mode   Mode
		format Format
		rate   float64
		chip   float64
		dev    float64
		off    float64
	}{
		{ModeT1, FormatA, 1_600_000, 100_000, 50_000, 0},
		{ModeT1, FormatA, 2_400_000, 104_000, 50_000, 35_000},
		{ModeT1, FormatA, 2_048_000, 96_000, 60_000, -30_000},
		{ModeT1, FormatA, 1_024_000, 100_000, 50_000, 10_000},
		{ModeC1, FormatA, 1_600_000, 100_000, 45_000, 20_000},
		{ModeC1, FormatB, 2_400_000, 100_000, 45_000, -40_000},
	} {
		iq := Synthesize(data, SynthOptions{
			Mode: tc.mode, Format: tc.format, SampleRateHz: tc.rate, OffsetHz: tc.off,
			ChipRateHz: tc.chip, DeviationHz: tc.dev, NoiseRMS: 0.05, Seed: int64(i),
		})
		var got []Frame
		rx, err := NewReceiver(ReceiverOptions{SampleRateHz: tc.rate, OnFrame: func(f Frame, _ FrameInfo) {
			got = append(got, f)
		}})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < len(iq); i += 5000 {
			rx.Process(iq[i:min(i+5000, len(iq))])
		}
		rx.Flush()
		if len(got) != 1 || !got[0].CRCOK || !bytes.Equal(got[0].Data, data) || got[0].Mode != tc.mode || got[0].Format != tc.format {
			t.Errorf("%s-%s rate %.0f chip %.0f off %.0f: %d frames (stats %+v)", tc.mode, tc.format, tc.rate, tc.chip, tc.off, len(got), rx.Stats())
		}
	}
}
