package airspy

import (
	"context"
	"encoding/binary"
	"math"
	"math/cmplx"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
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
	// The opened device must carry the gain ladder so `sdr list --probe`
	// (which reads dev.Info() post-open) doesn't report an empty list.
	if got := dev.Info().Gains; len(got) == 0 {
		t.Errorf("opened device Info().Gains is empty; want the gain ladder")
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
	// Table holds DEVICE rates; closestRateIndex takes an IQ rate and
	// matches 2×IQ against the table (the converter decimates by two).
	dev := &Device{rates: []uint32{10_000_000, 2_500_000}}
	cases := []struct {
		iqHz uint32
		want int
	}{
		{5_000_000, 0}, // device 10M — exact
		{4_500_000, 0}, // device 9M — nearer 10M than 2.5M
		{2_000_000, 1}, // device 4M — nearer 2.5M
		{1_250_000, 1}, // device 2.5M — exact
	}
	for _, c := range cases {
		if got := dev.closestRateIndex(c.iqHz); got != c.want {
			t.Errorf("closestRateIndex(%d) = %d, want %d", c.iqHz, got, c.want)
		}
	}
	// No table → falls back to index 0.
	if (&Device{}).closestRateIndex(123) != 0 {
		t.Error("empty table should default to index 0")
	}
}

func TestSetSampleRateEncodesByValueWhenNoTable(t *testing.T) {
	// With no device-rate table the param is a by-value encoding of the
	// device rate (2×IQ) in kHz: IQ 4 MHz → device 8 MHz → 8000.
	dev := &Device{rates: nil}
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
	// Device table {10 MSPS, 2.5 MSPS}; requesting IQ 1.25 MHz selects
	// the 2.5 MSPS device rate (index 1).
	dev := &Device{rates: []uint32{10_000_000, 2_500_000}}
	mt := usb.NewMockTransport()
	dev.t = mt
	mt.Script = []usb.CtrlExchange{
		{In: true, BRequest: reqSetSamplerate, WValue: 0, WIndex: 1, Reply: []byte{0}, N: 1},
	}
	if err := dev.SetSampleRate(1_250_000); err != nil {
		t.Fatalf("SetSampleRate exact: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

// TestSetSampleRateSelectsDeviceRateAtTwiceIQ is the regression gate for
// the real-sampling 2× rate bug: a requested IQ rate must program the
// device at twice that rate so the host converter delivers the IQ rate
// the rest of the pipeline assumes.
func TestSetSampleRateSelectsDeviceRateAtTwiceIQ(t *testing.T) {
	// A real Airspy R2/Mini advertises 6 MSPS and 3 MSPS device rates.
	cases := []struct {
		iqHz      uint32
		wantIndex uint16
	}{
		{3_000_000, 0}, // → device 6 MSPS (index 0)
		{1_500_000, 1}, // → device 3 MSPS (index 1)
	}
	for _, c := range cases {
		dev := &Device{rates: []uint32{6_000_000, 3_000_000}}
		mt := usb.NewMockTransport()
		dev.t = mt
		mt.Script = []usb.CtrlExchange{
			{In: true, BRequest: reqSetSamplerate, WValue: 0, WIndex: c.wantIndex, Reply: []byte{0}, N: 1},
		}
		if err := dev.SetSampleRate(c.iqHz); err != nil {
			t.Fatalf("SetSampleRate(%d): %v", c.iqHz, err)
		}
		if mt.Err != nil {
			t.Fatalf("SetSampleRate(%d) transport: %v (wanted device index %d)", c.iqHz, mt.Err, c.wantIndex)
		}
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

// realToneBuf renders n real ADC samples (unpacked little-endian uint16,
// 12-bit, DC at 2048) of a cosine at fHz given sampleHz, as the device would
// stream them.
func realToneBuf(n int, fHz, sampleHz float64) []byte {
	buf := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		v := math.Cos(2 * math.Pi * fHz * float64(i) / sampleHz)
		s := uint16(2048 + 2000*v) // stay inside the 12-bit range
		binary.LittleEndian.PutUint16(buf[2*i:], s)
	}
	return buf
}

func TestIQConverterDecimatesByTwo(t *testing.T) {
	// 4096 real samples (8192 bytes) → 2048 complex samples.
	buf := realToneBuf(4096, 1_000_000, 5_000_000)
	got := newIQConverter().processRaw(buf)
	if len(got) != 2048 {
		t.Fatalf("processRaw len = %d, want 2048", len(got))
	}
}

// TestIQConverterRejectsImage is the regression gate for issue #454: feeding a
// real tone through the converter must yield a clean analytic (single-sided)
// complex stream. Before the fix the driver read real samples as interleaved
// I/Q, which the reporter saw as ~+78° phase imbalance and ~3.3 dB image
// rejection. We measure the same quantities here and require quadrature.
func TestIQConverterRejectsImage(t *testing.T) {
	const (
		hwRate = 5_000_000 // hardware rate (2× the 2.5 MSPS IQ rate)
		tone   = 1_000_000 // real tone inside the half-band passband
		nReal  = 1 << 18
	)
	out := newIQConverter().processRaw(realToneBuf(nReal, tone, hwRate))

	// Skip the filter warm-up so transient settling doesn't skew the stats.
	out = out[hbTaps:]

	var st rtlsdr.IQImbalanceStats
	st.Observe(out)

	if phase := math.Abs(st.PhaseImbalanceDeg()); phase > 1.0 {
		t.Errorf("phase imbalance = %.3f°, want < 1° (issue #454 saw +78°)", phase)
	}
	if rej := st.ImageRejectionDB(); rej < 40 {
		t.Errorf("image rejection = %.1f dB, want > 40 dB (issue #454 saw 3.3 dB)", rej)
	}
}

// TestIQConverterContinuousAcrossPackets confirms the converter's filter state
// carries across packet boundaries: streaming a tone in many small packets
// must match converting it in one shot.
func TestIQConverterContinuousAcrossPackets(t *testing.T) {
	buf := realToneBuf(1<<16, 700_000, 5_000_000)

	oneShot := newIQConverter().processRaw(buf)

	split := newIQConverter()
	var chunked []complex64
	const step = 1024 // bytes per packet (512 real samples, multiple of 4)
	for off := 0; off < len(buf); off += step {
		end := off + step
		if end > len(buf) {
			end = len(buf)
		}
		chunked = append(chunked, split.processRaw(buf[off:end])...)
	}

	if len(chunked) != len(oneShot) {
		t.Fatalf("chunked len = %d, one-shot len = %d", len(chunked), len(oneShot))
	}
	for i := range oneShot {
		if d := cmplx.Abs(complex128(oneShot[i] - chunked[i])); d > 1e-6 {
			t.Fatalf("sample %d differs by %g: one-shot=%v chunked=%v", i, d, oneShot[i], chunked[i])
		}
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
// In scanner mode a fast retune cancels the stream ctx and immediately
// re-opens. The previous stream's teardown clears `streaming` in a detached
// goroutine, so StreamIQ must wait that teardown out rather than fail with
// "airspy: stream already active". Here we cancel and re-open without
// draining the first channel; the second call must block until cleanup
// completes and then succeed.
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
