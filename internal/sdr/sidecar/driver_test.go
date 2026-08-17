package sidecar

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"math"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// The draft this driver came from did not implement sdr.Device — SetPPM and
// SetBiasTee are required and were missing, so it would not have compiled into
// the pool. Pinned at compile time.
var _ sdr.Device = (*device)(nil)

var _ sdr.FreqRanger = (*device)(nil)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeSidecar stands in for the external process: a TCP listener that streams
// whatever the test feeds it, and a UDP socket that records the 5-byte
// commands GopherTrunk sends.
type fakeSidecar struct {
	t    *testing.T
	ln   net.Listener
	ctrl *net.UDPConn
	feed chan []byte
	mu   sync.Mutex
	cmds []command
	conn net.Conn
}

type command struct {
	op    uint8
	param uint32
}

func newFakeSidecar(t *testing.T) *fakeSidecar {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctrl, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSidecar{t: t, ln: ln, ctrl: ctrl, feed: make(chan []byte, 8)}
	go f.serve()
	go f.readCommands()
	t.Cleanup(func() {
		ln.Close()
		ctrl.Close()
		f.mu.Lock()
		if f.conn != nil {
			f.conn.Close()
		}
		f.mu.Unlock()
	})
	return f
}

func (f *fakeSidecar) DataAddr() string    { return f.ln.Addr().String() }
func (f *fakeSidecar) ControlAddr() string { return f.ctrl.LocalAddr().String() }

func (f *fakeSidecar) serve() {
	conn, err := f.ln.Accept()
	if err != nil {
		return
	}
	f.mu.Lock()
	f.conn = conn
	f.mu.Unlock()
	for b := range f.feed {
		if _, err := conn.Write(b); err != nil {
			return
		}
	}
}

func (f *fakeSidecar) readCommands() {
	buf := make([]byte, 64)
	for {
		n, _, err := f.ctrl.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if n != 5 {
			f.mu.Lock()
			f.cmds = append(f.cmds, command{op: 0xff, param: uint32(n)})
			f.mu.Unlock()
			continue
		}
		f.mu.Lock()
		f.cmds = append(f.cmds, command{op: buf[0], param: binary.BigEndian.Uint32(buf[1:5])})
		f.mu.Unlock()
	}
}

// Feed queues bytes for the data socket. Splitting one logical chunk across
// several Feed calls models a short read.
func (f *fakeSidecar) Feed(b []byte) { f.feed <- b }

// CloseData hangs up the data socket, as a sidecar exiting would.
func (f *fakeSidecar) CloseData() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.conn != nil {
		f.conn.Close()
	}
}

func (f *fakeSidecar) commands() []command {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]command(nil), f.cmds...)
}

func (f *fakeSidecar) waitCommands(n int, d time.Duration) []command {
	f.t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if c := f.commands(); len(c) >= n {
			return c
		}
		time.Sleep(5 * time.Millisecond)
	}
	return f.commands()
}

func openTestDevice(t *testing.T, f *fakeSidecar, spec Spec) sdr.Device {
	t.Helper()
	spec.DataAddr = f.DataAddr()
	if spec.ControlAddr == "" {
		spec.ControlAddr = f.ControlAddr()
	}
	if spec.SampleRateHz == 0 {
		spec.SampleRateHz = 2_400_000
	}
	drv := New([]Spec{spec}, testLogger())
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { dev.Close() })
	return dev
}

func cs16Bytes(samples ...complex64) []byte {
	buf := make([]byte, 0, len(samples)*4)
	for _, s := range samples {
		var b [4]byte
		binary.LittleEndian.PutUint16(b[0:], uint16(int16(real(s)*32768)))
		binary.LittleEndian.PutUint16(b[2:], uint16(int16(imag(s)*32768)))
		buf = append(buf, b[:]...)
	}
	return buf
}

func TestEnumerateReportsConfiguredSpecs(t *testing.T) {
	drv := New([]Spec{
		{DataAddr: "127.0.0.1:5001"},
		{DataAddr: "", Serial: "skipped"}, // no address: not a device
		{DataAddr: "/tmp/iq.fifo", Transport: "unix_pipe", Serial: "named"},
	}, testLogger())
	got, err := drv.Enumerate()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("enumerated %d devices, want 2: %+v", len(got), got)
	}
	if got[0].Serial == got[1].Serial {
		t.Errorf("serials collide: %q", got[0].Serial)
	}
	if got[1].Serial != "named" {
		t.Errorf("explicit serial not honoured: %q", got[1].Serial)
	}
	if got[0].TunerName != "tcp" || got[1].TunerName != "unix_pipe" {
		t.Errorf("transports = %q / %q", got[0].TunerName, got[1].TunerName)
	}
}

func TestOpenRejectsUnknownTransport(t *testing.T) {
	drv := New([]Spec{{DataAddr: "127.0.0.1:1", Transport: "carrier-pigeon"}}, testLogger())
	if _, err := drv.Open(0); err == nil {
		t.Fatal("Open accepted an unknown transport")
	}
}

func TestOpenRejectsUnknownFormat(t *testing.T) {
	drv := New([]Spec{{DataAddr: "127.0.0.1:1", Format: "u8"}}, testLogger())
	if _, err := drv.Open(0); err == nil {
		t.Fatal("Open accepted an unsupported format")
	}
}

// 192.0.2.0/24 is TEST-NET-1 (RFC 5737) and is not routed, so the dial hangs
// until the timeout rather than failing fast.
func TestOpenTCPDialTimeout(t *testing.T) {
	drv := New([]Spec{{
		DataAddr: "192.0.2.1:5001", SampleRateHz: 1, ConnectTimeout: 150 * time.Millisecond,
	}}, testLogger())
	start := time.Now()
	if _, err := drv.Open(0); err == nil {
		t.Fatal("Open succeeded against an unroutable address")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("dial took %v, want it bounded by the connect timeout", elapsed)
	}
}

// The control protocol is the contract with an external program, so the bytes
// are pinned exactly rather than round-tripped through this driver's own
// encoder.
func TestControlCommandsAreEncodedCorrectly(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})

	if err := dev.SetCenterFreq(467_912_500); err != nil {
		t.Fatal(err)
	}
	if err := dev.SetSampleRate(2_400_000); err != nil {
		t.Fatal(err)
	}
	cmds := f.waitCommands(2, time.Second)
	if len(cmds) < 2 {
		t.Fatalf("sidecar saw %d commands, want 2", len(cmds))
	}
	if cmds[0] != (command{op: 0x01, param: 467_912_500}) {
		t.Errorf("SET_CENTER_FREQ = %+v, want {0x01 467912500}", cmds[0])
	}
	if cmds[1] != (command{op: 0x02, param: 2_400_000}) {
		t.Errorf("SET_SAMPLE_RATE = %+v, want {0x02 2400000}", cmds[1])
	}
}

// sdr.Device's gain convention makes a NEGATIVE value mean AGC. Casting that
// straight to uint32 — what the proposal did — would send 0xFFFFFFF6 as a
// manual gain of 4.29 billion tenths of a dB. It goes out as a gain-mode
// change instead, so there is no signed-encoding ambiguity on the wire.
func TestSetGainNegativeSelectsAGCRatherThanWrappingAround(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})

	if err := dev.SetGain(-1); err != nil {
		t.Fatal(err)
	}
	cmds := f.waitCommands(1, time.Second)
	if len(cmds) != 1 {
		t.Fatalf("AGC sent %d commands, want 1: %+v", len(cmds), cmds)
	}
	if cmds[0] != (command{op: 0x03, param: 0}) {
		t.Errorf("AGC command = %+v, want SET_GAIN_MODE(0x03) 0", cmds[0])
	}
}

func TestSetGainManualSendsModeThenValue(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})

	if err := dev.SetGain(496); err != nil {
		t.Fatal(err)
	}
	cmds := f.waitCommands(2, time.Second)
	if len(cmds) != 2 {
		t.Fatalf("manual gain sent %d commands, want 2: %+v", len(cmds), cmds)
	}
	if cmds[0] != (command{op: 0x03, param: 1}) {
		t.Errorf("first = %+v, want SET_GAIN_MODE(0x03) 1", cmds[0])
	}
	if cmds[1] != (command{op: 0x04, param: 496}) {
		t.Errorf("second = %+v, want SET_GAIN(0x04) 496", cmds[1])
	}
}

// Without a control address the setters must succeed and do nothing: failing
// every tune would take the device out of the pool, and a fixed-frequency feed
// is a legitimate configuration.
func TestSettersAreNoOpsWithoutAControlAddress(t *testing.T) {
	f := newFakeSidecar(t)
	drv := New([]Spec{{DataAddr: f.DataAddr(), SampleRateHz: 1}}, testLogger())
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	if err := dev.SetCenterFreq(100_000_000); err != nil {
		t.Errorf("SetCenterFreq: %v", err)
	}
	if err := dev.SetGain(300); err != nil {
		t.Errorf("SetGain: %v", err)
	}
	if cmds := f.commands(); len(cmds) != 0 {
		t.Errorf("commands were sent with no control address: %+v", cmds)
	}
}

func TestStreamIQDeliversConvertedSamples(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}

	want := []complex64{complex(0.5, -0.5), complex(0.25, 0.25), complex(-0.75, 0)}
	f.Feed(cs16Bytes(want...))

	select {
	case got := <-ch:
		if len(got) != len(want) {
			t.Fatalf("chunk len = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if absDiff(got[i], want[i]) > 1e-3 {
				t.Errorf("sample %d = %v, want ~%v", i, got[i], want[i])
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no samples within 2s")
	}
}

// A short read is not a sample boundary. The proposal forwarded whatever
// arrived and truncated with len/4, so a single odd byte would swap I and Q
// for the entire rest of the stream. The remainder has to be carried.
func TestStreamIQPreservesSampleAlignmentAcrossShortReads(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}

	want := make([]complex64, 64)
	for i := range want {
		want[i] = complex(float32(i)/128, -float32(i)/128)
	}
	raw := cs16Bytes(want...)
	// Dribble it out in 3-byte pieces: never a sample boundary.
	go func() {
		for pos := 0; pos < len(raw); pos += 3 {
			end := pos + 3
			if end > len(raw) {
				end = len(raw)
			}
			f.Feed(append([]byte(nil), raw[pos:end]...))
			time.Sleep(time.Millisecond)
		}
	}()

	var got []complex64
	deadline := time.After(5 * time.Second)
	for len(got) < len(want) {
		select {
		case chunk := <-ch:
			got = append(got, chunk...)
		case <-deadline:
			t.Fatalf("only %d of %d samples arrived", len(got), len(want))
		}
	}
	for i := range want {
		if absDiff(got[i], want[i]) > 1e-3 {
			t.Fatalf("sample %d = %v, want ~%v — I/Q alignment was lost across a short read",
				i, got[i], want[i])
		}
	}
}

func TestStreamIQDecodesComplex64(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{Format: "complex64"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}

	want := []complex64{complex(0.25, -0.75), complex(-1, 0.5)}
	buf := make([]byte, 0, len(want)*8)
	for _, s := range want {
		var b [8]byte
		binary.LittleEndian.PutUint32(b[0:], math.Float32bits(real(s)))
		binary.LittleEndian.PutUint32(b[4:], math.Float32bits(imag(s)))
		buf = append(buf, b[:]...)
	}
	f.Feed(buf)

	select {
	case got := <-ch:
		if len(got) != len(want) {
			t.Fatalf("chunk len = %d, want %d", len(got), len(want))
		}
		for i := range want {
			// float32 round-trips exactly; this must be bit-identical, not close.
			if got[i] != want[i] {
				t.Errorf("sample %d = %v, want exactly %v", i, got[i], want[i])
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no samples within 2s")
	}
}

// A quiet sidecar is the normal state at shutdown. The read must not sit until
// its deadline expires — that is the bug class this repo already carries in two
// other drivers, and a new one should not reintroduce it.
func TestStreamIQStopsPromptlyOnContextCancel(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // settle into the blocking read

	start := time.Now()
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unexpected samples from a silent sidecar")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not close within 5s of ctx cancel")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("stream took %v to close after cancel", elapsed)
	}
}

func TestStreamIQEndsWhenTheSidecarHangsUp(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f.Feed(cs16Bytes(complex(0.1, 0.1)))
	<-ch
	f.CloseData()

	select {
	case _, ok := <-ch:
		if ok {
			// Drain any straggler, then require closure.
			select {
			case _, ok := <-ch:
				if ok {
					t.Fatal("stream kept delivering after hang-up")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("stream did not close after the sidecar hung up")
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stream did not close after the sidecar hung up")
	}
}

// sendOrDrop must shed the OLDEST queued chunk, keeping the freshest data and
// never blocking the reader — a blocked read stalls the socket and makes the
// sidecar's own buffers overflow.
func TestSendOrDropShedsTheOldestChunk(t *testing.T) {
	out := make(chan []complex64, 2)
	th := newDropThrottle(testLogger(), "test")
	ctx := context.Background()

	first := []complex64{complex(1, 0)}
	second := []complex64{complex(2, 0)}
	third := []complex64{complex(3, 0)}
	for _, c := range [][]complex64{first, second, third} {
		if !sendOrDrop(ctx, out, c, th) {
			t.Fatal("sendOrDrop reported cancellation")
		}
	}
	if th.drops.Load() != 1 {
		t.Errorf("drops = %d, want 1", th.drops.Load())
	}
	got := []complex64{(<-out)[0], (<-out)[0]}
	if got[0] != second[0] || got[1] != third[0] {
		t.Errorf("queue holds %v, want the two NEWEST chunks (2,3)", got)
	}
}

func TestSendOrDropReturnsFalseOnCancel(t *testing.T) {
	out := make(chan []complex64) // unbuffered, nobody reading
	th := newDropThrottle(testLogger(), "test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sendOrDrop(ctx, out, []complex64{complex(1, 0)}, th) {
		t.Error("sendOrDrop returned true on a cancelled context")
	}
}

func TestActualSampleRateFromSpecThenSetter(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{SampleRateHz: 2_400_000})
	rater, ok := dev.(interface{ ActualSampleRate() (uint32, error) })
	if !ok {
		t.Fatal("device does not report its sample rate")
	}
	got, err := rater.ActualSampleRate()
	if err != nil || got != 2_400_000 {
		t.Fatalf("ActualSampleRate = %d, %v; want 2400000", got, err)
	}
	if err := dev.SetSampleRate(1_000_000); err != nil {
		t.Fatal(err)
	}
	if got, _ := rater.ActualSampleRate(); got != 1_000_000 {
		t.Errorf("after SetSampleRate: %d, want 1000000", got)
	}
}

// An unconfigured tuning range must read as unknown, not as an invented
// 24 MHz–6 GHz: the hunt sweep derives its band from this and would drive the
// sidecar somewhere its hardware cannot go.
func TestFreqRangeIsUnknownUnlessConfigured(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})
	lo, hi := dev.(sdr.FreqRanger).FreqRange()
	if lo != 0 || hi != 0 {
		t.Errorf("unconfigured FreqRange = (%d, %d), want (0, 0)", lo, hi)
	}

	dev2 := openTestDevice(t, newFakeSidecar(t), Spec{FreqMinHz: 70_000_000, FreqMaxHz: 3_000_000_000})
	lo, hi = dev2.(sdr.FreqRanger).FreqRange()
	if lo != 70_000_000 || hi != 3_000_000_000 {
		t.Errorf("configured FreqRange = (%d, %d)", lo, hi)
	}
}

func TestSetterAfterCloseFailsCleanly(t *testing.T) {
	f := newFakeSidecar(t)
	dev := openTestDevice(t, f, Spec{})
	if err := dev.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := dev.SetCenterFreq(100_000_000); err == nil {
		t.Error("SetCenterFreq after Close: want an error")
	}
	if err := dev.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestConvertCS16Scaling(t *testing.T) {
	// Full-scale positive and negative, and the DC midpoint.
	buf := new(bytes.Buffer)
	for _, pair := range [][2]int16{{32767, -32768}, {0, 0}} {
		binary.Write(buf, binary.LittleEndian, pair[0])
		binary.Write(buf, binary.LittleEndian, pair[1])
	}
	got := convertCS16(buf.Bytes())
	if len(got) != 2 {
		t.Fatalf("decoded %d samples, want 2", len(got))
	}
	if real(got[0]) < 0.99 || imag(got[0]) != -1 {
		t.Errorf("full-scale sample = %v", got[0])
	}
	if got[1] != 0 {
		t.Errorf("zero sample = %v", got[1])
	}
}

// A trailing partial sample must be dropped, never decoded from whatever
// happens to follow it in the buffer.
func TestConvertIgnoresATrailingPartialSample(t *testing.T) {
	if got := convertCS16([]byte{1, 2, 3}); len(got) != 0 {
		t.Errorf("cs16 decoded %d samples from 3 bytes", len(got))
	}
	if got := convertComplex64([]byte{1, 2, 3, 4, 5}); len(got) != 0 {
		t.Errorf("complex64 decoded %d samples from 5 bytes", len(got))
	}
}

func absDiff(a, b complex64) float64 {
	dr := float64(real(a) - real(b))
	di := float64(imag(a) - imag(b))
	return math.Hypot(dr, di)
}
