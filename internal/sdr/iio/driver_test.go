package iio

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The command strings below are LITERALS from libiio v0.25's client
// (iiod-client.c iio_snprintf formats, values with their trailing NUL as
// iio_channel_attr_write sends them) and SoapyPlutoSDR's attribute names.
// Do not build them from the driver's constants.

func TestOpenSendsLiteralCommands(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{Serial: "plutosky"})

	want := []string{
		"VERSION\r\n",
		"PRINT\r\n",
		"READ iio:device0 OUTPUT altvoltage0 frequency_available\r\n",
		"WRITE iio:device0 OUTPUT altvoltage0 powerdown 2\r\n0\x00",
	}
	assertCommands(t, f.commands(), want)

	if got := dev.Info().Serial; got != "plutosky" {
		t.Errorf("serial = %q", got)
	}
	if dev.rx.ID != "iio:device3" || dev.phy.ID != "iio:device0" {
		t.Errorf("bound phy=%s rx=%s, want iio:device0 / iio:device3", dev.phy.ID, dev.rx.ID)
	}
	if dev.fullScale != 2048 {
		t.Errorf("full scale = %v, want 2048 (12-bit LSB-aligned, #1331)", dev.fullScale)
	}
	// The board's own frequency_available (the PlutoSky on tezuka firmware
	// reports an RX LO floor of 46 875 001 Hz; 6 GHz clamps to uint32).
	if lo, hi := dev.FreqRange(); lo != 46_875_001 || hi != 4_294_967_295 {
		t.Errorf("FreqRange = %d-%d, want 46875001-4294967295", lo, hi)
	}
	if !strings.Contains(dev.Info().Product, "PlutoSDR Rev.C") {
		t.Errorf("product = %q, want the board's hw_model", dev.Info().Product)
	}
}

func TestOpenWritesRFPortWhenConfigured(t *testing.T) {
	f := newFakeIIOD(t)
	openFake(t, f, Spec{RFPort: "A_BALANCED"})
	cmds := f.commands()
	if got, want := cmds[len(cmds)-1], "WRITE iio:device0 INPUT voltage0 rf_port_select 11\r\nA_BALANCED\x00"; got != want {
		t.Errorf("last command = %q, want %q", got, want)
	}
}

func TestSetCenterFreqLiteral(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{})
	n := len(f.commands())
	if err := dev.SetCenterFreq(851_012_500); err != nil {
		t.Fatal(err)
	}
	assertCommands(t, f.commands()[n:], []string{
		"WRITE iio:device0 OUTPUT altvoltage0 frequency 10\r\n851012500\x00",
	})
}

func TestSetSampleRateLiterals(t *testing.T) {
	cases := []struct {
		rate uint32
		bw   uint32
		want []string
	}{
		{2_400_000, 0, []string{
			"WRITE iio:device0 INPUT voltage0 sampling_frequency 8\r\n2400000\x00",
			"WRITE iio:device3 INPUT voltage0 sampling_frequency 8\r\n2400000\x00",
			"WRITE iio:device0 INPUT voltage0 rf_bandwidth 8\r\n2400000\x00",
		}},
		// Below 25 MHz/12 the PHY runs at 8x and the RX core decimates
		// (SoapyPlutoSDR setSampleRate).
		{1_000_000, 0, []string{
			"WRITE iio:device0 INPUT voltage0 sampling_frequency 8\r\n8000000\x00",
			"WRITE iio:device3 INPUT voltage0 sampling_frequency 8\r\n1000000\x00",
			"WRITE iio:device0 INPUT voltage0 rf_bandwidth 8\r\n1000000\x00",
		}},
		// The bandwidth floor is the AD9361's 200 kHz; an explicit one wins.
		{300_000, 0, []string{
			"WRITE iio:device0 INPUT voltage0 sampling_frequency 8\r\n2400000\x00",
			"WRITE iio:device3 INPUT voltage0 sampling_frequency 7\r\n300000\x00",
			"WRITE iio:device0 INPUT voltage0 rf_bandwidth 7\r\n300000\x00",
		}},
		{2_400_000, 1_500_000, []string{
			"WRITE iio:device0 INPUT voltage0 sampling_frequency 8\r\n2400000\x00",
			"WRITE iio:device3 INPUT voltage0 sampling_frequency 8\r\n2400000\x00",
			"WRITE iio:device0 INPUT voltage0 rf_bandwidth 8\r\n1500000\x00",
		}},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.rate, "/", tc.bw), func(t *testing.T) {
			f := newFakeIIOD(t)
			dev := openFake(t, f, Spec{RFBandwidthHz: tc.bw})
			n := len(f.commands())
			if err := dev.SetSampleRate(tc.rate); err != nil {
				t.Fatal(err)
			}
			assertCommands(t, f.commands()[n:], tc.want)
		})
	}
}

func TestSetSampleRateRejectsOutOfRange(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{})
	n := len(f.commands())
	for _, r := range []uint32{250_000, 62_000_000} {
		if err := dev.SetSampleRate(r); err == nil {
			t.Errorf("SetSampleRate(%d) = nil, want an out-of-range error", r)
		}
	}
	if got := f.commands()[n:]; len(got) != 0 {
		t.Errorf("an out-of-range rate still sent %q", got)
	}
}

func TestSetGainLiterals(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{})
	n := len(f.commands())
	if err := dev.SetGain(-1); err != nil {
		t.Fatal(err)
	}
	if err := dev.SetGain(405); err != nil { // 40.5 dB rounds to 41
		t.Fatal(err)
	}
	assertCommands(t, f.commands()[n:], []string{
		"WRITE iio:device0 INPUT voltage0 gain_control_mode 12\r\nslow_attack\x00",
		"WRITE iio:device0 INPUT voltage0 gain_control_mode 7\r\nmanual\x00",
		"WRITE iio:device0 INPUT voltage0 hardwaregain 3\r\n41\x00",
	})
}

func TestNegativeStatusIsAnErrno(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{})
	f.mu.Lock()
	f.failWith["WRITE iio:device0 OUTPUT altvoltage0 frequency"] = -22
	f.mu.Unlock()
	err := dev.SetCenterFreq(100_000_000)
	var en *ErrnoError
	if !errors.As(err, &en) || en.Code != int(syscall.EINVAL) {
		t.Fatalf("err = %v, want ErrnoError EINVAL", err)
	}
}

func TestActualSampleRateReadsRXCore(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{})
	f.mu.Lock()
	f.attrs["iio:device3 voltage0 sampling_frequency"] = "2399999\x00" // with NUL, as sysfs-backed reads can carry
	f.mu.Unlock()
	n := len(f.commands())
	got, err := dev.ActualSampleRate()
	if err != nil {
		t.Fatal(err)
	}
	if got != 2_399_999 {
		t.Errorf("ActualSampleRate = %d", got)
	}
	assertCommands(t, f.commands()[n:], []string{"READ iio:device3 INPUT voltage0 sampling_frequency\r\n"})
}

func TestOpenRefusesNonPlutoContext(t *testing.T) {
	f := newFakeIIOD(t)
	f.xml = []byte(`<?xml version="1.0"?><context name="x"><device id="iio:device0" name="adm1177"/></context>`)
	_, err := New([]Spec{{Addr: f.addr(), ConnectTimeout: time.Second}}, quietLogger()).Open(0)
	if err == nil || !strings.Contains(err.Error(), "ad9361-phy") {
		t.Fatalf("err = %v, want a not-a-Pluto error naming ad9361-phy", err)
	}
}

func TestOpenRefusesUnexpectedSampleFormat(t *testing.T) {
	f := newFakeIIOD(t)
	f.xml = []byte(strings.ReplaceAll(string(f.xml), `le:S12/16&gt;&gt;0`, `le:S24/32&gt;&gt;0`))
	_, err := New([]Spec{{Addr: f.addr(), ConnectTimeout: time.Second}}, quietLogger()).Open(0)
	if err == nil || !strings.Contains(err.Error(), "le:S24/32>>0") {
		t.Fatalf("err = %v, want a sample-format refusal", err)
	}
}

// writeChunk writes one READBUF chunk the way iiod's send_data does: the
// count line, the mask line on the first chunk of the call, then the bytes.
// It dribbles the bytes out in small writes so the client sees the chunk
// split across TCP reads.
func writeChunk(w io.Writer, data []byte, withMask bool) {
	fmt.Fprintf(w, "%d\n", len(data))
	if withMask {
		io.WriteString(w, "00000003\n")
	}
	for len(data) > 0 {
		n := 7
		if n > len(data) {
			n = len(data)
		}
		w.Write(data[:n])
		data = data[n:]
	}
}

func le16(vals ...int16) []byte {
	b := make([]byte, 2*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint16(b[2*i:], uint16(v))
	}
	return b
}

func TestStreamIQConvertsAndParsesChunks(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{})
	if err := dev.SetSampleRate(300_000); err != nil { // 300000/60 → 8192-sample buffers
		t.Fatal(err)
	}
	const samples = 8192
	f.mu.Lock()
	f.readbuf = func(w io.Writer, devID string, n int, call int) {
		if devID != "iio:device3" || n != samples*4 {
			fmt.Fprintf(w, "-22\n")
			return
		}
		// Two chunks per READBUF; the mask rides only the first of EACH call.
		half := make([]int16, samples) // I,Q pairs for samples/2 samples
		for i := range half {
			switch i % 4 {
			case 0:
				half[i] = 2047
			case 1:
				half[i] = -2048
			case 2:
				half[i] = 1024
			case 3:
				half[i] = int16(call)
			}
		}
		writeChunk(w, le16(half...), true)
		writeChunk(w, le16(half...), false)
	}
	f.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var got []complex64
	deadline := time.After(5 * time.Second)
	for len(got) < 3*samples {
		select {
		case c, ok := <-ch:
			if !ok {
				t.Fatalf("stream closed early: %v", dev.StreamErr())
			}
			got = append(got, c...)
		case <-deadline:
			t.Fatalf("only %d samples in 5 s", len(got))
		}
	}
	if got[0] != complex(2047.0/2048, -1) {
		t.Errorf("sample 0 = %v, want (2047/2048, -1)", got[0])
	}
	if got[1] != complex(0.5, 0) {
		t.Errorf("sample 1 = %v, want (0.5, 0)", got[1])
	}
	// Sample 1 of the second READBUF carries Q = call index 1.
	if got[samples+1] != complex(0.5, 1.0/2048) {
		t.Errorf("second READBUF sample 1 = %v, want (0.5, 1/2048)", got[samples+1])
	}
	var open string
	for _, c := range f.commands() {
		if strings.HasPrefix(c, "OPEN") {
			open = c
		}
	}
	if open != "OPEN iio:device3 8192 00000003\r\n" {
		t.Errorf("OPEN = %q", open)
	}
}

// A 12-bit word whose upper four storage bits are not a sign extension must
// still convert as 12-bit two's complement (libiio iio_channel_convert).
func TestConvertSignExtendsFromSignificantBits(t *testing.T) {
	d := &device{bits: 12, shift: 0, fullScale: 2048, iFirst: true}
	got := d.convert([]byte{0xFF, 0xFF & 0x0F, 0x00, 0x08}) // I = 0x0FFF (-1), Q = 0x0800 (-2048)
	if got[0] != complex(-1.0/2048, -1) {
		t.Errorf("got %v, want (-1/2048, -1)", got[0])
	}
}

func TestStreamIQCancelReturnsPromptly(t *testing.T) {
	f := newFakeIIOD(t) // readbuf nil: the board never fills a buffer
	dev := openFake(t, f, Spec{})
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	start := time.Now()
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unexpected chunk")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("StreamIQ did not end within 2 s of cancel")
	}
	if el := time.Since(start); el > time.Second {
		t.Errorf("teardown took %v", el)
	}
	if err := dev.StreamErr(); err != nil {
		t.Errorf("StreamErr after cancel = %v, want nil", err)
	}
}

func TestStreamIQReportsBoardHangup(t *testing.T) {
	f := newFakeIIOD(t)
	dev := openFake(t, f, Spec{})
	f.mu.Lock()
	f.readbuf = func(w io.Writer, _ string, _ int, _ int) {
		fmt.Fprintf(w, "-5\n") // EIO, as iiod answers a failed refill
	}
	f.mu.Unlock()
	ch, err := dev.StreamIQ(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unexpected chunk")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not end on an iiod error")
	}
	if err := dev.StreamErr(); err == nil || !strings.Contains(err.Error(), "errno 5") {
		t.Errorf("StreamErr = %v, want the EIO", err)
	}
}

func TestNormalizeAddr(t *testing.T) {
	for in, want := range map[string]string{
		"192.168.2.1":      "192.168.2.1:30431",
		"ip:192.168.2.1":   "192.168.2.1:30431",
		"pluto.local:1234": "pluto.local:1234",
		" 10.0.0.5 ":       "10.0.0.5:30431",
		"fe80::1":          "[fe80::1]:30431",
		"[fe80::1]:30431":  "[fe80::1]:30431",
	} {
		if got := NormalizeAddr(in); got != want {
			t.Errorf("NormalizeAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBufferSamplesAndDepth(t *testing.T) {
	if got := bufferSamples(2_400_000); got != 65536 { // 40000 → next power of two
		t.Errorf("bufferSamples(2.4M) = %d", got)
	}
	if got := bufferSamples(300_000); got != 8192 {
		t.Errorf("bufferSamples(300k) = %d", got)
	}
	// 400 ms at 2.4 MS/s is 960000 samples = 15 buffers of 65536.
	if got := streamDepth(2_400_000, 65536); got != 15 {
		t.Errorf("streamDepth = %d, want 15", got)
	}
}

func assertCommands(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("commands:\n got %q\nwant %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("command %d = %q, want %q", i, got[i], want[i])
		}
	}
}
