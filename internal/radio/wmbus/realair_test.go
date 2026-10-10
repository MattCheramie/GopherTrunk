package wmbus

import (
	"encoding/hex"
	"os"
	"strconv"
	"testing"
)

// loadCU8 reads an rtl_sdr-style unsigned 8-bit IQ file.
func loadCU8(t *testing.T, path string) []complex64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	iq := make([]complex64, len(raw)/2)
	for i := range iq {
		iq[i] = complex((float32(raw[2*i])-127.5)/128, (float32(raw[2*i+1])-127.5)/128)
	}
	return iq
}

// TestWMBusRealAirCaptures replays a real-air rtl_sdr capture through the
// receiver and logs every frame. Skip-guarded:
//
//	GT_WMBUS_IQ=<file.cu8> [GT_WMBUS_RATE=1600000] [GT_WMBUS_OFFSET=<Hz>]
//	[GT_WMBUS_MIN=<CRC-valid frames expected>] go test ./internal/radio/wmbus -run RealAir -v
//
// Verified against rtl-wmbus's four sample captures
// (github.com/xaelsouth/rtl-wmbus, samples/): every telegram rtl-wmbus
// decodes with a valid CRC is decoded byte for byte (6 distinct), plus 4
// more CRC-valid telegrams rtl-wmbus misses, including the 2.4 MS/s
// issue48 capture centred on 868.625 MHz (GT_WMBUS_OFFSET=325000), whose
// one frame rtl-wmbus fails the CRC on. The captures are not committed;
// TestStripCRCRealAirFrames pins literal frames from them.
func TestWMBusRealAirCaptures(t *testing.T) {
	path := os.Getenv("GT_WMBUS_IQ")
	if path == "" {
		t.Skip("GT_WMBUS_IQ not set")
	}
	rate := 1_600_000.0
	if v := os.Getenv("GT_WMBUS_RATE"); v != "" {
		rate, _ = strconv.ParseFloat(v, 64)
	}
	off := 0.0
	if v := os.Getenv("GT_WMBUS_OFFSET"); v != "" {
		off, _ = strconv.ParseFloat(v, 64)
	}
	iq := loadCU8(t, path)
	rx, err := NewReceiver(ReceiverOptions{SampleRateHz: rate, OffsetHz: off, OnFrame: func(f Frame, info FrameInfo) {
		tg, err := ParseTelegram(f)
		t.Logf("%s-%s crc=%v t=%.3fs level=%.1fdBFS inv=%v id=%s %s err=%v data=%s raw=%s",
			f.Mode, f.Format, f.CRCOK, float64(info.Sample)/rate, info.LevelDBFS, info.Inverted, tg.ID, tg.Manufacturer, err, hex.EncodeToString(f.Data), hex.EncodeToString(f.Raw))
	}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(iq); i += 16384 {
		rx.Process(iq[i:min(i+16384, len(iq))])
	}
	rx.Flush()
	st := rx.Stats()
	t.Logf("stats %+v", st)
	if v := os.Getenv("GT_WMBUS_MIN"); v != "" {
		if want, _ := strconv.Atoi(v); int(st.Framer.CRCOK) < want {
			t.Errorf("%d CRC-valid frames, want at least %d", st.Framer.CRCOK, want)
		}
	}
}
