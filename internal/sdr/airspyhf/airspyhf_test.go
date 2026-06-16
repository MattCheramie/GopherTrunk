package airspyhf

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func withDevice(t *testing.T) (*Device, *usb.MockTransport) {
	t.Helper()
	mt := usb.NewMockTransport()
	return &Device{t: mt, info: sdr.Info{Driver: driverName, Serial: "test"}}, mt
}

func TestDriverEnumerateOpenReadsVersionAndRates(t *testing.T) {
	// On Open the driver reads the firmware version string and the
	// sample-rate table (count then list).
	openCalled := false
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 4, VID: vidAirspyHF, PID: pidAirspyHF,
				Serial: "HF1", Product: "Airspy HF+ Discovery", Path: "mock/1"},
		},
		OpenFunc: func(d usb.Descriptor) (*usb.MockTransport, error) {
			openCalled = true
			mt := usb.NewMockTransport()
			count := make([]byte, 4)
			binary.LittleEndian.PutUint32(count, 4)
			list := make([]byte, 16)
			binary.LittleEndian.PutUint32(list[0:4], 768_000)
			binary.LittleEndian.PutUint32(list[4:8], 384_000)
			binary.LittleEndian.PutUint32(list[8:12], 256_000)
			binary.LittleEndian.PutUint32(list[12:16], 192_000)
			mt.Script = []usb.CtrlExchange{
				{In: true, BRequest: reqGetVersionString, Reply: []byte("R3.0.7\x00"), N: 255},
				{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 0, Reply: count, N: 4},
				{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 4, Reply: list, N: 16},
			}
			return mt, nil
		},
	}
	drv := New(enum)
	infos, err := drv.Enumerate()
	if err != nil || len(infos) != 1 || infos[0].Serial != "HF1" {
		t.Fatalf("Enumerate = %+v, err=%v", infos, err)
	}
	if infos[0].Product != "Airspy HF+ Discovery" {
		t.Errorf("Enumerate Product = %q, want Airspy HF+ Discovery", infos[0].Product)
	}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !openCalled {
		t.Fatal("OpenFunc not invoked")
	}
	asDev := dev.(*Device)
	if len(asDev.rates) != 4 || asDev.rates[0] != 768_000 {
		t.Fatalf("samplerate table = %v", asDev.rates)
	}
	if !strings.Contains(asDev.info.TunerName, "R3.0.7") {
		t.Errorf("TunerName = %q, want firmware suffix", asDev.info.TunerName)
	}
	if err := dev.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestVariantNameDiscoveryDualLegacy(t *testing.T) {
	cases := []struct {
		product string
		want    string
	}{
		{"Airspy HF+ Discovery", "Airspy HF+ Discovery"},
		{"AIRSPY HF+ DUAL PORT", "Airspy HF+ Dual Port"},
		{"airspy hf+ dual-port revB", "Airspy HF+ Dual Port"},
		{"Airspy HF+", "Airspy HF+"},
		{"", "Airspy HF+"},
		{"random-product", "Airspy HF+"},
	}
	for _, c := range cases {
		if got := variantName(c.product); got != c.want {
			t.Errorf("variantName(%q) = %q, want %q", c.product, got, c.want)
		}
	}
}

func TestSetCenterFreqEncoding(t *testing.T) {
	dev, mt := withDevice(t)
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, 14_070_000)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetFreq, Data: payload},
	}
	if err := dev.SetCenterFreq(14_070_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestClosestRateIndex(t *testing.T) {
	dev := &Device{rates: []uint32{768_000, 384_000, 256_000, 192_000}}
	cases := []struct {
		hz   uint32
		want int
	}{
		{768_000, 0},
		{700_000, 0},
		{500_000, 1}, // closer to 384k (116k) than 768k (268k)
		{384_000, 1},
		{300_000, 2}, // closer to 256k (44k) than 384k (84k)
		{256_000, 2},
		{192_000, 3},
		{100_000, 3},
	}
	for _, c := range cases {
		if got := dev.closestRateIndex(c.hz); got != c.want {
			t.Errorf("closestRateIndex(%d) = %d, want %d", c.hz, got, c.want)
		}
	}
	// Empty table → falls back to index 0.
	if (&Device{}).closestRateIndex(123) != 0 {
		t.Error("empty table should default to index 0")
	}
}

func TestSetSampleRateSelectsClosest(t *testing.T) {
	dev := &Device{rates: []uint32{768_000, 384_000}}
	mt := usb.NewMockTransport()
	dev.t = mt
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetSamplerate, WValue: 1}, // 300k is closer to 384k than 768k
	}
	if err := dev.SetSampleRate(300_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetGainAutoEnablesHFAGC(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetHFAGC, WValue: 1},
		{BRequest: reqSetHFLNA, WValue: 0},
		{BRequest: reqSetHFATT, WValue: 0},
	}
	if err := dev.SetGain(-1); err != nil {
		t.Fatalf("SetGain(-1): %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetGainManualSplitsATTLNA(t *testing.T) {
	// 420 tenth-dB → 42 dB → ATT step 7 (42 dB), remainder 0 → LNA off.
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetHFAGC, WValue: 0},
		{BRequest: reqSetHFATT, WValue: 7},
		{BRequest: reqSetHFLNA, WValue: 0},
	}
	if err := dev.SetGain(420); err != nil {
		t.Fatalf("SetGain(420): %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetGainManualLNAOnAboveMaxATT(t *testing.T) {
	// 540 tenth-dB → 54 dB → ATT step 9 → clamped to 8 (48 dB),
	// remainder 60 → LNA on (+6 dB preamp).
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetHFAGC, WValue: 0},
		{BRequest: reqSetHFATT, WValue: 8},
		{BRequest: reqSetHFLNA, WValue: 1},
	}
	if err := dev.SetGain(540); err != nil {
		t.Fatalf("SetGain(540): %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetBiasTeeRoundTrips(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetBiasTee, WValue: 1},
		{BRequest: reqSetBiasTee, WValue: 0},
	}
	if err := dev.SetBiasTee(true); err != nil {
		t.Fatalf("SetBiasTee(on): %v", err)
	}
	if err := dev.SetBiasTee(false); err != nil {
		t.Fatalf("SetBiasTee(off): %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestDecodeInt16IQ(t *testing.T) {
	// Two samples: (+32767, -32768) → ~(1, -1); (0, 0) → (0, 0).
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint16(buf[0:2], uint16(int16(32767)))
	var minI16 int16 = -32768
	binary.LittleEndian.PutUint16(buf[2:4], uint16(minI16))
	binary.LittleEndian.PutUint16(buf[4:6], 0)
	binary.LittleEndian.PutUint16(buf[6:8], 0)
	got := decodeInt16IQ(buf)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if real(got[0]) <= 0.99 || imag(got[0]) > -0.99 {
		t.Errorf("sample 0 = (%f,%f); want near (1,-1)", real(got[0]), imag(got[0]))
	}
	if got[1] != 0 {
		t.Errorf("sample 1 = %v; want 0", got[1])
	}
}

func TestStreamIQFlipsReceiverAndStops(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	cancel()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, ok := <-ch; !ok {
			break
		}
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

// TestStreamIQReopenAfterCancelWaitsForTeardown is the regression for #686.
// A fast retune cancels the stream ctx and immediately re-opens. The
// previous stream's teardown clears `streaming` in a detached goroutine,
// so StreamIQ must wait that teardown out rather than fail with
// "airspyhf: stream already active". We cancel and re-open without draining
// the first channel; the second call must block until cleanup completes
// and then succeed.
func TestStreamIQReopenAfterCancelWaitsForTeardown(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
		{BRequest: reqReceiverMode, WValue: receiverModeOn},
		{BRequest: reqReceiverMode, WValue: receiverModeOff},
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	if _, err := dev.StreamIQ(ctx1); err != nil {
		t.Fatalf("first StreamIQ: %v", err)
	}
	cancel1()

	// The bounded ctx makes a buggy wait loop fail loudly instead of hanging.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	ch, err := dev.StreamIQ(ctx2)
	if err != nil {
		t.Fatalf("reopen after cancel: %v", err)
	}
	cancel2()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, ok := <-ch; !ok {
			break
		}
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestCleanVersionStringStripsPadding(t *testing.T) {
	got := cleanVersionString([]byte("R3.0.7\x00garbage"))
	if got != "R3.0.7" {
		t.Errorf("cleanVersionString = %q, want R3.0.7", got)
	}
}
