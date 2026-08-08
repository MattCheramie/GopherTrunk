package hackrf

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// withDevice returns a Device whose USB transport is a freshly-built
// MockTransport so each test can populate its own Script.
func withDevice(t *testing.T) (*Device, *usb.MockTransport) {
	t.Helper()
	mt := usb.NewMockTransport()
	return &Device{t: mt, info: sdr.Info{Driver: driverName, Serial: "test"}}, mt
}

func TestDriverEnumerateAndOpen(t *testing.T) {
	// Open now issues BOARD_ID_READ + VERSION_STRING_READ control
	// transfers, so the mock needs a scripted transport (not the
	// default blank one) — script both readbacks here.
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 7, VID: vidHackRF, PID: pidHackRFOne, Serial: "ABC", Product: "HackRF One", Path: "mock/1"},
		},
		OpenFunc: func(usb.Descriptor) (*usb.MockTransport, error) {
			mt := usb.NewMockTransport()
			mt.Script = []usb.CtrlExchange{
				{In: true, BRequest: reqBoardIDRead, Reply: []byte{2}, N: 1},
				{In: true, BRequest: reqVersionStringRead, Reply: []byte("git-2024.02.1\x00"), N: 255},
			}
			return mt, nil
		},
	}
	drv := New(enum)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	if len(infos) != 1 || infos[0].Serial != "ABC" || infos[0].Driver != driverName {
		t.Fatalf("Enumerate = %+v", infos)
	}
	if infos[0].Product != "HackRF One" {
		t.Errorf("Enumerate Product = %q, want %q", infos[0].Product, "HackRF One")
	}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if dev.Info().Serial != "ABC" {
		t.Fatalf("Info.Serial = %q, want ABC", dev.Info().Serial)
	}
	if dev.Info().Product != "HackRF One" {
		t.Errorf("Info.Product = %q, want %q", dev.Info().Product, "HackRF One")
	}
	if !strings.Contains(dev.Info().TunerName, "git-2024.02.1") {
		t.Errorf("Info.TunerName = %q, want firmware suffix", dev.Info().TunerName)
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

func TestEnumerateRenamesByPID(t *testing.T) {
	// Even when the USB descriptor Product is garbage, the canonical
	// name comes from the PID.
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 1, VID: vidHackRF, PID: pidHackRFOne, Serial: "S1", Product: "weird-descriptor"},
			{Bus: 1, Address: 2, VID: vidHackRF, PID: pidHackRFJawbrk, Serial: "S2", Product: ""},
			{Bus: 1, Address: 3, VID: vidHackRF, PID: pidHackRFRad1o, Serial: "S3", Product: "Rad1o badge"},
		},
	}
	drv := New(enum)
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	want := []string{"HackRF One", "HackRF Jawbreaker", "Rad1o"}
	if len(infos) != len(want) {
		t.Fatalf("got %d infos, want %d", len(infos), len(want))
	}
	for i, w := range want {
		if infos[i].Product != w {
			t.Errorf("infos[%d].Product = %q, want %q", i, infos[i].Product, w)
		}
	}
}

func TestProductForPIDFallback(t *testing.T) {
	cases := []struct {
		pid      uint16
		fallback string
		want     string
	}{
		{pidHackRFOne, "ignored", "HackRF One"},
		{0xbeef, "USB descriptor name", "USB descriptor name"},
		{0xbeef, "", "HackRF"},
	}
	for _, c := range cases {
		if got := productForPID(c.pid, c.fallback); got != c.want {
			t.Errorf("productForPID(%#x, %q) = %q, want %q", c.pid, c.fallback, got, c.want)
		}
	}
}

func TestOpenAppendsPortaPackTag(t *testing.T) {
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 7, VID: vidHackRF, PID: pidHackRFOne, Serial: "PP1", Product: "HackRF One"},
		},
		OpenFunc: func(usb.Descriptor) (*usb.MockTransport, error) {
			mt := usb.NewMockTransport()
			mt.Script = []usb.CtrlExchange{
				{In: true, BRequest: reqBoardIDRead, Reply: []byte{2}, N: 1},
				{In: true, BRequest: reqVersionStringRead,
					Reply: []byte("git-portapack-mayhem-v1.7.4\x00"), N: 255},
			}
			return mt, nil
		},
	}
	drv := New(enum)
	if _, err := drv.Enumerate(); err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()
	if !strings.Contains(dev.Info().Product, "PortaPack") {
		t.Errorf("Info.Product = %q, want PortaPack suffix", dev.Info().Product)
	}
}

func TestReadBoardIDUnknownFallsBackToPID(t *testing.T) {
	// Board-ID 99 isn't in boardIDNames — Product must fall back to
	// the PID-derived name.
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 7, VID: vidHackRF, PID: pidHackRFRad1o, Serial: "R1", Product: "Rad1o"},
		},
		OpenFunc: func(usb.Descriptor) (*usb.MockTransport, error) {
			mt := usb.NewMockTransport()
			mt.Script = []usb.CtrlExchange{
				{In: true, BRequest: reqBoardIDRead, Reply: []byte{99}, N: 1},
				{In: true, BRequest: reqVersionStringRead, Reply: []byte("custom-fw\x00"), N: 255},
			}
			return mt, nil
		},
	}
	drv := New(enum)
	if _, err := drv.Enumerate(); err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()
	if dev.Info().Product != "Rad1o" {
		t.Errorf("Info.Product = %q, want Rad1o (PID fallback)", dev.Info().Product)
	}
}

func TestReadVersionStringIgnoredOnError(t *testing.T) {
	// Older firmware that doesn't implement VERSION_STRING_READ
	// returns an error; Open must still succeed with a plain
	// TunerName.
	enum := &usb.MockEnumerator{
		Devices: []usb.Descriptor{
			{Bus: 1, Address: 7, VID: vidHackRF, PID: pidHackRFOne, Serial: "OLD", Product: "HackRF One"},
		},
		OpenFunc: func(usb.Descriptor) (*usb.MockTransport, error) {
			mt := usb.NewMockTransport()
			// Only the board-ID read is scripted — version-string
			// read has no matching entry, so MockTransport.recordErr
			// returns an error which our Open is supposed to ignore.
			mt.Script = []usb.CtrlExchange{
				{In: true, BRequest: reqBoardIDRead, Reply: []byte{2}, N: 1},
			}
			return mt, nil
		},
	}
	drv := New(enum)
	if _, err := drv.Enumerate(); err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer dev.Close()
	if dev.Info().TunerName != "MAX2839+MAX5864" {
		t.Errorf("Info.TunerName = %q, want plain (no fw suffix)", dev.Info().TunerName)
	}
}

func TestCleanVersionStringStripsPaddingAndNonPrintable(t *testing.T) {
	// Padding NUL, embedded NUL, control char, trailing whitespace.
	got := cleanVersionString([]byte("git-2024.02.1\x00garbage-after-nul"))
	if got != "git-2024.02.1" {
		t.Errorf("cleanVersionString = %q, want %q", got, "git-2024.02.1")
	}
	got = cleanVersionString([]byte(" git-foo\x07\x01 \x00"))
	if got != "git-foo" {
		t.Errorf("cleanVersionString (control chars) = %q, want %q", got, "git-foo")
	}
}

func TestIsPortaPack(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"git-2024.02.1", false},
		{"git-portapack-v1.7.4", true},
		{"git-PORTAPACK", true},
		{"mayhem-v1.7.4", true},
		{"Mayhem build", true},
		{"", false},
	}
	for _, c := range cases {
		if got := isPortaPack(c.in); got != c.want {
			t.Errorf("isPortaPack(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestDriverOpenRejectsBadIndex(t *testing.T) {
	enum := &usb.MockEnumerator{Devices: nil}
	drv := New(enum)
	if _, err := drv.Open(0); err == nil {
		t.Fatal("Open on empty driver should error")
	}
}

func TestSetCenterFreqEncoding(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{{
		BRequest: reqSetFreq, WValue: 0, WIndex: 0,
		Data: encodeFreq(851_012_500),
	}}
	if err := dev.SetCenterFreq(851_012_500); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport error: %v", mt.Err)
	}
}

func encodeFreq(hz uint32) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], hz/1_000_000)
	binary.LittleEndian.PutUint32(buf[4:8], hz%1_000_000)
	return buf
}

func TestSetSampleRateProgramsFilter(t *testing.T) {
	dev, mt := withDevice(t)
	rateBytes := make([]byte, 8)
	binary.LittleEndian.PutUint32(rateBytes[0:4], 8_000_000)
	binary.LittleEndian.PutUint32(rateBytes[4:8], 1)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSampleRateSet, Data: rateBytes},
		{BRequest: reqBasebandFilterBwSet, WValue: uint16(6_000_000 & 0xFFFF), WIndex: uint16(6_000_000 >> 16)},
	}
	if err := dev.SetSampleRate(8_000_000); err != nil {
		t.Fatalf("SetSampleRate: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport error: %v", mt.Err)
	}
}

func TestSplitGain(t *testing.T) {
	cases := []struct {
		tenthDB          int
		wantLNA, wantVGA int
		wantAmp          bool
	}{
		{-1, defaultLNAGainDB, defaultVGAGainDB, true}, // "auto" enables the front-end RF amp
		{0, 0, 0, false},
		{160, 16, 0, false},   // 16 dB → all in LNA
		{180, 16, 2, false},   // 16 dB LNA + 2 dB VGA
		{300, 24, 6, false},   // 24 + 6 = 30 dB
		{900, 40, 50, false},  // clamped: 40 dB LNA, 50 dB VGA
		{1500, 40, 62, false}, // both saturated
	}
	for _, c := range cases {
		lna, vga, amp := splitGain(c.tenthDB)
		if lna != c.wantLNA || vga != c.wantVGA || amp != c.wantAmp {
			t.Errorf("splitGain(%d) = (%d,%d,%v), want (%d,%d,%v)",
				c.tenthDB, lna, vga, amp, c.wantLNA, c.wantVGA, c.wantAmp)
		}
	}
}

func TestSetGainIssuesAMPLNAVGA(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqAmpEnable, WValue: 1}, // "auto" (-1) enables the front-end amp
		{In: true, BRequest: reqSetLNAGain, WIndex: 16, Reply: []byte{1}, N: 1},
		{In: true, BRequest: reqSetVGAGain, WIndex: 20, Reply: []byte{1}, N: 1},
	}
	if err := dev.SetGain(-1); err != nil {
		t.Fatalf("SetGain: %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport error: %v", mt.Err)
	}
}

func TestSetBiasTeeRoundTrips(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqAntennaEnable, WValue: 1},
		{BRequest: reqAntennaEnable, WValue: 0},
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

func TestSetNarrowbandFilterPro(t *testing.T) {
	dev, mt := withDevice(t)
	dev.isPro = true // board ID 5 (Praline) — Pro-only request is allowed
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetNarrowbandFilter, WValue: 1},
		{BRequest: reqSetNarrowbandFilter, WValue: 0},
	}
	if err := dev.SetNarrowbandFilter(true); err != nil {
		t.Fatalf("SetNarrowbandFilter(on): %v", err)
	}
	if err := dev.SetNarrowbandFilter(false); err != nil {
		t.Fatalf("SetNarrowbandFilter(off): %v", err)
	}
	if mt.Err != nil {
		t.Fatalf("transport: %v", mt.Err)
	}
}

func TestSetNarrowbandFilterNonProErrors(t *testing.T) {
	dev, mt := withDevice(t) // isPro defaults to false
	dev.info.Product = "HackRF One"
	if err := dev.SetNarrowbandFilter(true); err == nil {
		t.Fatal("SetNarrowbandFilter on a non-Pro board: want error, got nil")
	}
	// The request must never reach the wire on a board that would stall it.
	if mt.Step != 0 {
		t.Fatalf("non-Pro SetNarrowbandFilter issued %d control transfer(s); want 0", mt.Step)
	}
}

func TestDecodeInt8IQ(t *testing.T) {
	// Three samples: (+127, -127), (0, 0), (-128, +64).
	buf := []byte{127, 0xFF - 126, 0, 0, 0x80, 64}
	got := decodeInt8IQ(buf)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if real(got[0]) <= 0.9 || imag(got[0]) >= -0.9 {
		t.Errorf("sample 0 = (%f,%f); expected near (+1,-1)", real(got[0]), imag(got[0]))
	}
	if real(got[1]) != 0 || imag(got[1]) != 0 {
		t.Errorf("sample 1 = (%f,%f); expected (0,0)", real(got[1]), imag(got[1]))
	}
	if real(got[2]) >= -0.9 || imag(got[2]) < 0.4 || imag(got[2]) > 0.6 {
		t.Errorf("sample 2 = (%f,%f); expected near (-1, +0.5)", real(got[2]), imag(got[2]))
	}
}

func TestStreamIQFlipsModeAndStopsOnCancel(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetTransceiverMode, WValue: transceiverModeReceive},
		{BRequest: reqSetTransceiverMode, WValue: transceiverModeOff},
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}
	cancel()
	// Drain until the chain shuts down.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, ok := <-ch; !ok {
			break
		}
	}
	if mt.Err != nil {
		t.Fatalf("transport error: %v", mt.Err)
	}
}

// TestStreamIQReopenAfterCancelWaitsForTeardown is the regression for #686.
// A fast retune cancels the stream ctx and immediately re-opens. The
// previous stream's teardown clears `streaming` in a detached goroutine,
// so StreamIQ must wait that teardown out rather than fail with
// "hackrf: stream already active". We cancel and re-open without draining
// the first channel; the second call must block until cleanup completes
// and then succeed.
func TestStreamIQReopenAfterCancelWaitsForTeardown(t *testing.T) {
	dev, mt := withDevice(t)
	mt.Script = []usb.CtrlExchange{
		{BRequest: reqSetTransceiverMode, WValue: transceiverModeReceive},
		{BRequest: reqSetTransceiverMode, WValue: transceiverModeOff},
		{BRequest: reqSetTransceiverMode, WValue: transceiverModeReceive},
		{BRequest: reqSetTransceiverMode, WValue: transceiverModeOff},
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
		t.Fatalf("transport error: %v", mt.Err)
	}
}
