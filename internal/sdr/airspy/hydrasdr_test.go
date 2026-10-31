package airspy

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"math/cmplx"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// The constants below are the HydraSDR wire protocol as the vendor's
// hydrasdr-host library defines it (libhydrasdr/src/hydrasdr_commands.h,
// hydrasdr.c, hydrasdr_rfone.c). They are restated as LITERALS here, not
// read back from the driver's own constants, so a typo in the driver fails
// a test instead of moving both sides together.
func TestHydraSDRWireConstantsMatchReference(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"official VID (38af)", int(vidHydraSDR), 0x38af},
		{"official PID (0001)", int(pidHydraSDR), 0x0001},
		{"legacy VID (1d50)", int(vidAirspy), 0x1d50},
		{"legacy PID (60a1)", int(pidAirspy), 0x60a1},
		{"HYDRASDR_RECEIVER_MODE", int(reqReceiverMode), 1},
		{"HYDRASDR_SET_SAMPLERATE", int(reqSetSamplerate), 12},
		{"HYDRASDR_SET_FREQ", int(reqSetFreq), 13},
		{"HYDRASDR_SET_LNA_GAIN", int(reqSetLNAGain), 14},
		{"HYDRASDR_SET_MIXER_GAIN", int(reqSetMixerGain), 15},
		{"HYDRASDR_SET_VGA_GAIN", int(reqSetVGAGain), 16},
		{"HYDRASDR_SET_LNA_AGC", int(reqSetLNAAGC), 17},
		{"HYDRASDR_SET_MIXER_AGC", int(reqSetMixerAGC), 18},
		{"HYDRASDR_SET_RF_BIAS_CMD", int(reqSetRFBias), 20},
		{"HYDRASDR_GET_SAMPLERATES", int(reqGetSamplerates), 25},
		{"HYDRASDR_SET_PACKING", int(reqSetPacking), 26},
		{"HYDRASDR_GET_CAPABILITIES", int(reqGetCapabilities), 29},
		{"HYDRASDR_SET_BANDWIDTH", int(reqSetBandwidth), 30},
		{"HYDRASDR_GET_BANDWIDTHS", int(reqGetBandwidths), 31},
		{"HYDRASDR_CAP_EXTENDED_SAMPLERATES", hydraCapExtendedRates, 1 << 21},
		{"RFONE_LNA_MAX_GAIN", hydraLNAMaxGain, 14},
		{"sizeof(hydrasdr_samplerate_info_t)", hydraRateInfoLen, 8},
		{"RFONE_RX_ENDPOINT | IN", int(bulkInEP), 0x81},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %#x, want %#x", c.name, c.got, c.want)
		}
	}
}

// One device on each ID pair must land in exactly one driver: the official
// IDs and a legacy-ID unit naming itself HydraSDR in the HydraSDR driver, a
// real Airspy in the Airspy driver, never both.
func TestHydraSDRAndAirspySplitTheSharedIDs(t *testing.T) {
	enum := &usb.MockEnumerator{Devices: []usb.Descriptor{
		{Bus: 1, Address: 2, VID: 0x38af, PID: 0x0001, Serial: "HYDRASDR SN:00000000AAAA0001", Product: "HydraSDR RFOne"},
		{Bus: 1, Address: 3, VID: 0x1d50, PID: 0x60a1, Serial: "HYDRASDR SN:00000000AAAA0002", Product: "HydraSDR RFOne"},
		{Bus: 1, Address: 4, VID: 0x1d50, PID: 0x60a1, Serial: "AS1", Product: "Airspy R2"},
	}}

	hydra, err := NewHydraSDR(enum).Enumerate()
	if err != nil {
		t.Fatalf("hydrasdr Enumerate: %v", err)
	}
	if len(hydra) != 2 {
		t.Fatalf("hydrasdr listed %d devices, want 2: %+v", len(hydra), hydra)
	}
	for _, in := range hydra {
		if in.Driver != "hydrasdr" || in.TunerName != "R828D (HydraSDR RFOne)" {
			t.Errorf("hydrasdr info = %+v", in)
		}
	}

	air, err := New(enum).Enumerate()
	if err != nil {
		t.Fatalf("airspy Enumerate: %v", err)
	}
	if len(air) != 1 || air[0].Serial != "AS1" || air[0].Driver != "airspy" {
		t.Fatalf("airspy listed %+v, want only AS1", air)
	}
}

func TestHydraSDRRegistered(t *testing.T) {
	var found bool
	for _, d := range sdr.Drivers() {
		if d.Name() == "hydrasdr" {
			found = true
		}
	}
	if !found {
		t.Fatal(`no "hydrasdr" driver registered`)
	}
}

func u32le(vs ...uint32) []byte {
	b := make([]byte, 4*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[4*i:], v)
	}
	return b
}

// rateInfo builds one 8-byte hydrasdr_samplerate_info_t record.
func rateInfo(rate uint32, adcBits, format uint8) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b, rate)
	b[4], b[5] = adcBits, format
	return b
}

// Opening an RFOne on extended-rate firmware reads the rate table, the
// capability mask, the per-rate ADC record and the bandwidth table, then
// turns packing off, in that order.
func TestHydraSDROpenReadsExtendedRateInfo(t *testing.T) {
	var mt *usb.MockTransport
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{{Bus: 1, Address: 2, VID: 0x38af, PID: 0x0001, Serial: "HYDRASDR SN:00000000AAAA0001", Product: "HydraSDR RFOne", Path: "mock/2"}},
		OpenFunc: func(usb.Descriptor) (*usb.MockTransport, error) {
			mt = usb.NewMockTransport()
			ext := append(rateInfo(10_000_000, 12, 0), rateInfo(2_500_000, 8, 0)...)
			mt.Script = []usb.CtrlExchange{
				{BRequest: 1, WValue: 0},
				{In: true, BRequest: 25, WValue: 0, WIndex: 0, Reply: u32le(2), N: 4},
				{In: true, BRequest: 25, WValue: 0, WIndex: 2, Reply: u32le(10_000_000, 2_500_000), N: 8},
				{In: true, BRequest: 29, WValue: 0, WIndex: 0, Reply: u32le(1 << 21), N: 4},
				{In: true, BRequest: 25, WValue: 1, WIndex: 2, Reply: ext, N: 16},
				{In: true, BRequest: 31, WIndex: 0, Reply: u32le(2), N: 4},
				{In: true, BRequest: 31, WIndex: 2, Reply: u32le(8_000_000, 3_000_000), N: 8},
				{In: true, BRequest: 26, WValue: 0, WIndex: 0, Reply: []byte{1}, N: 1},
				// SET_SAMPLERATE on extended firmware is read with 4 bytes.
				{In: true, BRequest: 12, WValue: 0, WIndex: 1, Reply: []byte{1, 8, 0, 0}, N: 4},
			}
			return mt, nil
		},
	}
	dev, err := NewHydraSDR(enum).Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	d := dev.(*Device)
	if !d.hydra || len(d.rateADCBits) != 2 || d.rateADCBits[1] != 8 {
		t.Fatalf("hydra=%v rateADCBits=%v", d.hydra, d.rateADCBits)
	}
	if len(d.bandwidths) != 2 || d.bandwidths[1] != 3_000_000 {
		t.Fatalf("bandwidths=%v", d.bandwidths)
	}
	if err := dev.SetSampleRate(2_500_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
	if d.adcBits != 8 {
		t.Errorf("adcBits after 2.5 MS/s = %d, want 8", d.adcBits)
	}
}

// Firmware without GET_CAPABILITIES (a stalled request) stays on the 12-bit
// default and the 1-byte SET_SAMPLERATE read, as libhydrasdr does.
func TestHydraSDROpenOldFirmwareKeepsDefaults(t *testing.T) {
	var mt *usb.MockTransport
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{{Bus: 1, Address: 3, VID: 0x1d50, PID: 0x60a1, Product: "HydraSDR RFOne", Path: "mock/3"}},
		OpenFunc: func(usb.Descriptor) (*usb.MockTransport, error) {
			mt = usb.NewMockTransport()
			mt.Script = []usb.CtrlExchange{
				{BRequest: 1, WValue: 0},
				{In: true, BRequest: 25, WIndex: 0, Reply: u32le(1), N: 4},
				{In: true, BRequest: 25, WIndex: 1, Reply: u32le(10_000_000), N: 4},
				{In: true, BRequest: 29, Err: errors.New("stall")},
				{In: true, BRequest: 31, Err: errors.New("stall")},
				{In: true, BRequest: 26, WIndex: 0, Reply: []byte{1}, N: 1},
				{In: true, BRequest: 12, WIndex: 0, Reply: []byte{1}, N: 1},
			}
			return mt, nil
		},
	}
	dev, err := NewHydraSDR(enum).Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := dev.Info().Serial; got != "hydrasdr-00" {
		t.Errorf("fallback serial = %q, want hydrasdr-00", got)
	}
	if err := dev.SetSampleRate(10_000_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
	if d := dev.(*Device); d.rateADCBits != nil || d.adcBits != 0 || d.bandwidths != nil {
		t.Errorf("rateADCBits=%v adcBits=%d, want nil/0 (12-bit default)", d.rateADCBits, d.adcBits)
	}
}

func hydraDevice() (*Device, *usb.MockTransport) {
	mt := usb.NewMockTransport()
	return &Device{t: mt, hydra: true, info: sdr.Info{Driver: hydraDriverName, Serial: "test"}}, mt
}

// SET_FREQ is a uint64 on the RFOne (hydrasdr_generic_set_freq); the
// Airspy's 4-byte form would leave the firmware reading 4 bytes of garbage.
func TestHydraSDRSetCenterFreqSendsUint64(t *testing.T) {
	dev, mt := hydraDevice()
	// 162.55 MHz = 0x09B050F0, little-endian, zero-extended to 8 bytes.
	mt.Script = []usb.CtrlExchange{
		{BRequest: 13, Data: []byte{0xF0, 0x50, 0xB0, 0x09, 0, 0, 0, 0}},
	}
	if err := dev.SetCenterFreq(162_550_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

// The RFOne bias tee is SET_RF_BIAS_CMD (OUT, wIndex = on/off), not the
// Airspy's GPIO write.
func TestHydraSDRSetBiasTeeUsesRFBiasCommand(t *testing.T) {
	dev, mt := hydraDevice()
	mt.Script = []usb.CtrlExchange{
		{BRequest: 20, WValue: 0, WIndex: 1},
		{BRequest: 20, WValue: 0, WIndex: 0},
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

// The RFOne's LNA stops at 14; a gain that splits to LNA 15 is clamped.
func TestHydraSDRSetGainClampsLNA(t *testing.T) {
	dev, mt := hydraDevice()
	mt.Script = []usb.CtrlExchange{
		{In: true, BRequest: 17, WIndex: 0, Reply: []byte{1}},
		{In: true, BRequest: 18, WIndex: 0, Reply: []byte{1}},
		{In: true, BRequest: 14, WIndex: 14, Reply: []byte{1}},
		{In: true, BRequest: 15, WIndex: 15, Reply: []byte{1}},
		{In: true, BRequest: 16, WIndex: 15, Reply: []byte{1}},
	}
	if err := dev.SetGain(1500); err != nil {
		t.Fatalf("SetGain: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

// Without a rate table the by-value SET_SAMPLERATE encoding is the real ADC
// rate in kHz (libhydrasdr doubles the IQ rate first).
func TestHydraSDRSampleRateByValueIsRealRate(t *testing.T) {
	dev, _ := hydraDevice()
	if got := dev.sampleRateCommandParam(2_500_000); got != 5000 {
		t.Errorf("by-value param = %d, want 5000", got)
	}
}

// An 8-bit ADC streams one byte per sample (libhydrasdr's
// iqconverter_process_u8). The same tone through the 8-bit and 12-bit paths
// must come out at the same frequency and level.
func TestIQConverterEightBitMatchesTwelveBit(t *testing.T) {
	const (
		n      = 1 << 16
		hwRate = 5_000_000.0
		tone   = 700_000.0
	)
	buf8 := make([]byte, n)
	buf12 := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		v := math.Cos(2 * math.Pi * tone * float64(i) / hwRate)
		buf8[i] = uint8(math.Round(128 + 120*v))
		binary.LittleEndian.PutUint16(buf12[2*i:], uint16(math.Round(2048+1920*v)))
	}
	out8 := newIQConverterADC(8).processRaw(buf8)
	out12 := newIQConverterADC(12).processRaw(buf12)
	if len(out8) != n/2 || len(out12) != n/2 {
		t.Fatalf("lengths 8-bit=%d 12-bit=%d, want %d", len(out8), len(out12), n/2)
	}
	var p8, p12 float64
	for i := 1024; i < len(out8); i++ {
		p8 += math.Pow(cmplx.Abs(complex128(out8[i])), 2)
		p12 += math.Pow(cmplx.Abs(complex128(out12[i])), 2)
	}
	if r := 10 * math.Log10(p8/p12); math.Abs(r) > 0.2 {
		t.Errorf("8-bit vs 12-bit power differs by %.2f dB, want < 0.2", r)
	}
	// Same rotation per sample ⇒ same frequency.
	ph := func(o []complex64) float64 {
		var acc complex128
		for i := 1025; i < len(o); i++ {
			acc += complex128(o[i]) * cmplx.Conj(complex128(o[i-1]))
		}
		return cmplx.Phase(acc)
	}
	if d := math.Abs(ph(out8) - ph(out12)); d > 1e-3 {
		t.Errorf("8-bit vs 12-bit phase step differs by %.4f rad", d)
	}
}

// An unreported or unsupported width falls back to the 12-bit Airspy ADC.
func TestIQConverterADCDefaults(t *testing.T) {
	for _, bits := range []uint8{0, 9, 32} {
		c := newIQConverterADC(bits)
		if c.mid != 2048 || c.byteSamples {
			t.Errorf("newIQConverterADC(%d) mid=%v byte=%v, want 12-bit", bits, c.mid, c.byteSamples)
		}
	}
}

// libhydrasdr's auto-bandwidth rule: the narrowest advertised IF bandwidth at
// least as wide as the sample rate, whatever order the table is in.
func TestHydraSDRBandwidthIndex(t *testing.T) {
	bws := []uint32{8_000_000, 1_700_000, 3_000_000, 5_000_000}
	for _, c := range []struct {
		rate uint32
		want int
	}{
		{2_500_000, 2}, // 3 MHz
		{1_000_000, 1}, // 1.7 MHz
		{5_000_000, 3}, // exactly 5 MHz
		{6_000_000, 0}, // 8 MHz
		{10_000_000, -1},
	} {
		if got := hydraBandwidthIndex(bws, c.rate); got != c.want {
			t.Errorf("rate %d: index %d, want %d", c.rate, got, c.want)
		}
	}
	if got := hydraBandwidthIndex(nil, 2_500_000); got != -1 {
		t.Errorf("no table: index %d, want -1", got)
	}
}

// The firmware applies its own IF bandwidth when RX mode goes on, so the
// driver must set the one that fits the sample rate AFTER receiver-on, as
// libhydrasdr's start_rx does (here 2.5 MS/s -> the 3 MHz entry, index 1).
// Old firmware with no bandwidth table gets no SET_BANDWIDTH at all.
func TestHydraSDRStreamSetsBandwidthAfterReceiverOn(t *testing.T) {
	for _, c := range []struct {
		name   string
		bws    []uint32
		script []usb.CtrlExchange
	}{
		{"bandwidth table", []uint32{8_000_000, 3_000_000, 1_700_000}, []usb.CtrlExchange{
			{BRequest: reqReceiverMode, WValue: receiverModeOn},
			{In: true, BRequest: 30, WValue: 0, WIndex: 1, Reply: []byte{1}, N: 1},
			{BRequest: reqReceiverMode, WValue: receiverModeOff},
		}},
		{"old firmware", nil, []usb.CtrlExchange{
			{BRequest: reqReceiverMode, WValue: receiverModeOn},
			{BRequest: reqReceiverMode, WValue: receiverModeOff},
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			dev, mt := hydraDevice()
			dev.rates = []uint32{10_000_000, 2_500_000}
			dev.lastReqRate = 2_500_000
			dev.bandwidths = c.bws
			mt.Script = c.script
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
			if mt.Step != len(c.script) {
				t.Fatalf("ran %d of %d scripted transfers", mt.Step, len(c.script))
			}
		})
	}
}
