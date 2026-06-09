package airspy

import (
	"context"
	"encoding/binary"
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

func TestDriverEnumerateOpenReadsSamplerates(t *testing.T) {
	// On Open the driver reads the samplerate table (count first, then
	// list). SET_SAMPLE_TYPE is deferred until StreamIQ starts.
	openCalled := false
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 4, VID: vidAirspy, PID: pidAirspy, Serial: "AS1", Product: "Airspy R2", Path: "mock/1"},
		},
		OpenFunc: func(d usb.Descriptor) (*usb.MockTransport, error) {
			openCalled = true
			mt := usb.NewMockTransport()
			count := make([]byte, 4)
			binary.LittleEndian.PutUint32(count, 2)
			list := make([]byte, 8)
			binary.LittleEndian.PutUint32(list[0:4], 10_000_000)
			binary.LittleEndian.PutUint32(list[4:8], 2_500_000)
			mt.Script = []usb.CtrlExchange{
				{BRequest: reqReceiverMode, WValue: receiverModeOff},
				{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 0, Reply: count, N: 4},
				{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 2, Reply: list, N: 8},
			}
			return mt, nil
		},
	}
	drv := New(enum)
	infos, err := drv.Enumerate()
	if err != nil || len(infos) != 1 || infos[0].Serial != "AS1" {
		t.Fatalf("Enumerate = %+v, err=%v", infos, err)
	}
	if infos[0].TunerName != "R820T (Airspy R2)" {
		t.Errorf("Enumerate TunerName = %q, want R820T (Airspy R2)", infos[0].TunerName)
	}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !openCalled {
		t.Fatal("OpenFunc not invoked")
	}
	asDev := dev.(*Device)
	if len(asDev.rates) != 2 || asDev.rates[0] != 10_000_000 {
		t.Fatalf("samplerate table = %v", asDev.rates)
	}
	if err := dev.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestDriverOpenRetriesTransientDeviceGoneOnOpen(t *testing.T) {
	prevBackoff := openRetryBackoff
	openRetryBackoff = 0
	t.Cleanup(func() { openRetryBackoff = prevBackoff })

	openCalls := 0
	var second *usb.MockTransport

	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 4, VID: vidAirspy, PID: pidAirspy, Serial: "AS1", Product: "Airspy R2", Path: "mock/1"},
		},
		OpenFunc: func(d usb.Descriptor) (*usb.MockTransport, error) {
			openCalls++
			switch openCalls {
			case 1:
				return nil, usb.ErrDeviceGone
			case 2:
				mt := usb.NewMockTransport()
				count := make([]byte, 4)
				binary.LittleEndian.PutUint32(count, 1)
				list := make([]byte, 4)
				binary.LittleEndian.PutUint32(list, 10_000_000)
				mt.Script = []usb.CtrlExchange{
					{BRequest: reqReceiverMode, WValue: receiverModeOff},
					{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 0, Reply: count, N: 4},
					{In: true, BRequest: reqGetSamplerates, WValue: 0, WIndex: 1, Reply: list, N: 4},
				}
				second = mt
				return mt, nil
			default:
				t.Fatalf("unexpected OpenFunc call #%d", openCalls)
			}
			return nil, nil
		},
	}

	drv := New(enum)
	if _, err := drv.Enumerate(); err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open after transient ErrDeviceGone: %v", err)
	}
	defer dev.Close()

	if openCalls < 2 {
		t.Fatalf("OpenFunc calls = %d, want at least 2", openCalls)
	}
	if second == nil {
		t.Fatalf("second transport not captured")
	}
}

func TestTunerNameDetectsMini(t *testing.T) {
	cases := []struct {
		product string
		want    string
	}{
		{"Airspy R2", "R820T (Airspy R2)"},
		{"Airspy Mini", "R820T (Airspy Mini)"},
		{"AIRSPY MINI", "R820T (Airspy Mini)"},
		{"airspy mini", "R820T (Airspy Mini)"},
		{"", "R820T (Airspy R2)"},
		{"Generic Airspy", "R820T (Airspy R2)"},
	}
	for _, c := range cases {
		if got := tunerNameFor(c.product); got != c.want {
			t.Errorf("tunerNameFor(%q) = %q, want %q", c.product, got, c.want)
		}
	}
}

func TestEnumerateTagsMini(t *testing.T) {
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 9, VID: vidAirspy, PID: pidAirspy, Serial: "MINI1", Product: "Airspy Mini"},
		},
	}
	drv := New(enum)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) != 1 || infos[0].TunerName != "R820T (Airspy Mini)" {
		t.Fatalf("Enumerate = %+v", infos)
	}
}

func TestSetCenterFreqEncoding(t *testing.T) {
	dev, mt := withDevice(t)
	payload := make([]byte, 4)
	binary.LittleEndian.PutUint32(payload, 162_550_000)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetFreq, Data: payload},
	}
	if err := dev.SetCenterFreq(162_550_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestClosestRateIndex(t *testing.T) {
	dev := &Device{rates: []uint32{10_000_000, 2_500_000}}
	cases := []struct {
		hz   uint32
		want int
	}{
		{10_000_000, 0},
		{9_000_000, 0},
		{4_000_000, 1},
		{2_500_000, 1},
	}
	for _, c := range cases {
		if got := dev.closestRateIndex(c.hz); got != c.want {
			t.Errorf("closestRateIndex(%d) = %d, want %d", c.hz, got, c.want)
		}
	}
	// No table → falls back to index 0.
	if (&Device{}).closestRateIndex(123) != 0 {
		t.Error("empty table should default to index 0")
	}
}

func TestSetSampleRateEncodesByValueWhenNotIndexed(t *testing.T) {
	dev := &Device{rates: []uint32{10_000_000, 2_500_000}}
	mt := usb.NewMockTransport()
	dev.t = mt
	mt.Script = []usb.CtrlExchange{
		{In: true, BRequest: reqSetSamplerate, WValue: 0, WIndex: 8000, Reply: []byte{0}, N: 1},
	}
	if err := dev.SetSampleRate(4_000_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetSampleRateUsesIndexWhenExactMatch(t *testing.T) {
	dev := &Device{rates: []uint32{10_000_000, 2_500_000}}
	mt := usb.NewMockTransport()
	dev.t = mt
	mt.Script = []usb.CtrlExchange{
		{In: true, BRequest: reqSetSamplerate, WValue: 0, WIndex: 1, Reply: []byte{0}, N: 1},
	}
	if err := dev.SetSampleRate(2_500_000); err != nil {
		t.Fatalf("SetSampleRate exact: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSplitAirspyGain(t *testing.T) {
	cases := []struct {
		tenthDB                   int
		wantLNA, wantMix, wantVGA int
	}{
		{0, 0, 0, 0},
		{30, 1, 0, 0},
		{90, 3, 0, 0},
		{600, 15, 5, 0},    // saturates LNA; remainder rolls to mixer
		{1500, 15, 15, 15}, // fully saturated
	}
	for _, c := range cases {
		l, m, v := splitAirspyGain(c.tenthDB)
		if l != c.wantLNA || m != c.wantMix || v != c.wantVGA {
			t.Errorf("splitAirspyGain(%d) = (%d,%d,%d), want (%d,%d,%d)",
				c.tenthDB, l, m, v, c.wantLNA, c.wantMix, c.wantVGA)
		}
	}
}

func TestSetGainAutoEnablesAGC(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{In: true, BRequest: reqSetLNAAGC, WIndex: 1, Reply: []byte{0}, N: 1},
		{In: true, BRequest: reqSetMixerAGC, WIndex: 1, Reply: []byte{0}, N: 1},
		{In: true, BRequest: reqSetVGAGain, WIndex: defaultVGAGain, Reply: []byte{0}, N: 1},
	}
	if err := dev.SetGain(-1); err != nil {
		t.Fatalf("SetGain(-1): %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetGainManualDisablesAGC(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{In: true, BRequest: reqSetLNAAGC, WIndex: 0, Reply: []byte{0}, N: 1},
		{In: true, BRequest: reqSetMixerAGC, WIndex: 0, Reply: []byte{0}, N: 1},
		{In: true, BRequest: reqSetLNAGain, WIndex: 3, Reply: []byte{0}, N: 1},
		{In: true, BRequest: reqSetMixerGain, WIndex: 0, Reply: []byte{0}, N: 1},
		{In: true, BRequest: reqSetVGAGain, WIndex: 0, Reply: []byte{0}, N: 1},
	}
	if err := dev.SetGain(90); err != nil { // 9 dB target → LNA=3, others 0
		t.Fatalf("SetGain(90): %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetBiasTeeRoundTrips(t *testing.T) {
	dev, mt := withDevice(t)
	portPin := (biasTeeGPIOPort << 5) | biasTeeGPIOPin
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqGPIOWrite, WValue: 1, WIndex: portPin},
		{BRequest: reqGPIOWrite, WValue: 0, WIndex: portPin},
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
	// Two samples: (+32767, -32768) → ~(1,-1); (0,0) → (0,0).
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
