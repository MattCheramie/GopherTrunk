package purego

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/tuners"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func TestConvertU8IQ_BitIdenticalWithCGO(t *testing.T) {
	// The conversion math is the bit-identical port of
	// rtlsdr_cgo.go:225-240. Reproduce it inline as the oracle so
	// any drift in the production code (e.g. someone "optimizing"
	// /127.5 to *0.0078431) shows up as a failed assertion.
	cases := []struct {
		buf []byte
	}{
		{buf: []byte{}},
		{buf: []byte{127, 128}}, // ~zero IQ
		{buf: []byte{0, 0}},     // negative max
		{buf: []byte{255, 255}}, // positive max
		{buf: []byte{127, 128, 0, 0, 255, 255, 64, 192}},
	}
	for _, c := range cases {
		got := convertU8IQ(c.buf)
		want := referenceConvertU8IQ(c.buf)
		if len(got) != len(want) {
			t.Errorf("len mismatch on %v: got %d, want %d", c.buf, len(got), len(want))
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("convertU8IQ(%v)[%d] = %v, want %v", c.buf, i, got[i], want[i])
			}
		}
	}
}

// referenceConvertU8IQ is the C-side math written out longhand.
// MUST stay byte-identical with rtlsdr_cgo.go:225-240.
func referenceConvertU8IQ(buf []byte) []complex64 {
	n := len(buf) / 2
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		i8 := float32(buf[2*i]) - 127.5
		q8 := float32(buf[2*i+1]) - 127.5
		out[i] = complex(i8/127.5, q8/127.5)
	}
	return out
}

func TestConvertU8IQ_ZeroBufferYieldsNoSamples(t *testing.T) {
	if got := convertU8IQ(nil); len(got) != 0 {
		t.Errorf("convertU8IQ(nil) returned %d samples, want 0", len(got))
	}
	if got := convertU8IQ([]byte{1}); len(got) != 0 {
		t.Errorf("convertU8IQ(odd length) returned %d samples, want 0 (truncated by /2)", len(got))
	}
}

func TestConvertU8IQ_DCBiasMidScaleIsNearZero(t *testing.T) {
	// 127, 128 brackets the chip's DC bias of 127.5; the resulting
	// complex value should have |real|, |imag| ≤ 1/127.5 ≈ 0.00785.
	const tol = 0.01
	got := convertU8IQ([]byte{127, 128})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if math.Abs(float64(real(got[0]))) > tol || math.Abs(float64(imag(got[0]))) > tol {
		t.Errorf("expected near-zero, got %v", got[0])
	}
}

func TestConvertU8IQ_ExtremesScaleToUnit(t *testing.T) {
	// 0 and 255 map to (0-127.5)/127.5 = -1 and (255-127.5)/127.5 ≈ 1.0.
	const tol = 0.005
	gotMin := convertU8IQ([]byte{0, 0})
	if math.Abs(float64(real(gotMin[0]))-(-1.0)) > tol {
		t.Errorf("min real = %v, want ~-1", real(gotMin[0]))
	}
	gotMax := convertU8IQ([]byte{255, 255})
	if math.Abs(float64(real(gotMax[0]))-1.0) > tol {
		t.Errorf("max real = %v, want ~1", real(gotMax[0]))
	}
}

// fakeTuner is an in-memory [tuners.Tuner] that records dispatch
// calls. Used by Device-level tests that care about the interaction
// between Device methods and the tuner without needing the full
// R820T register-script setup.
type fakeTuner struct {
	mu               sync.Mutex
	freqCalls        []uint32
	bwCalls          []uint32
	gainCalls        []int
	gainModeCalls    []bool
	standbyCalls     int
	freqErr, bwErr   error
	gainErr, modeErr error
}

func (f *fakeTuner) Type() tuners.Type { return tuners.TypeR820T2 }
func (f *fakeTuner) IFFreqHz() uint32  { return 3_570_000 }
func (f *fakeTuner) Init() error       { return nil }
func (f *fakeTuner) Standby() error    { f.standbyCalls++; return nil }
func (f *fakeTuner) Close() error      { return f.Standby() }
func (f *fakeTuner) Gains() []int      { return []int{0, 100, 200} }
func (f *fakeTuner) SetFreq(hz uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.freqCalls = append(f.freqCalls, hz)
	return f.freqErr
}
func (f *fakeTuner) SetBandwidth(hz uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bwCalls = append(f.bwCalls, hz)
	return f.bwErr
}
func (f *fakeTuner) SetGain(t int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gainCalls = append(f.gainCalls, t)
	return f.gainErr
}
func (f *fakeTuner) SetGainMode(m bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gainModeCalls = append(f.gainModeCalls, m)
	return f.modeErr
}

func newDeviceWithFakeTuner(transport usb.Transport, fake *fakeTuner) *Device {
	demod := rtl2832u.New(transport)
	return &Device{
		transport: transport,
		demod:     demod,
		tuner:     fake,
		info:      sdr.Info{Driver: DriverName, Index: 0},
	}
}

func TestDevice_SetCenterFreqDispatchesToTuner(t *testing.T) {
	mock := usb.NewMockTransport()
	fake := &fakeTuner{}
	d := newDeviceWithFakeTuner(mock, fake)
	if err := d.SetCenterFreq(100_000_000); err != nil {
		t.Fatalf("SetCenterFreq: %v", err)
	}
	if len(fake.freqCalls) != 1 || fake.freqCalls[0] != 100_000_000 {
		t.Errorf("freqCalls = %v, want [100_000_000]", fake.freqCalls)
	}
}

func TestDevice_SetCenterFreqAfterCloseFails(t *testing.T) {
	mock := usb.NewMockTransport()
	d := newDeviceWithFakeTuner(mock, &fakeTuner{})
	d.closed.Store(true)
	if err := d.SetCenterFreq(100_000_000); !errors.Is(err, ErrClosed) {
		t.Errorf("got %v, want ErrClosed", err)
	}
}

func TestDevice_SetGainAutoFlipsModeOff(t *testing.T) {
	fake := &fakeTuner{}
	d := newDeviceWithFakeTuner(usb.NewMockTransport(), fake)
	if err := d.SetGain(-1); err != nil {
		t.Fatalf("SetGain(-1): %v", err)
	}
	if len(fake.gainModeCalls) != 1 || fake.gainModeCalls[0] != false {
		t.Errorf("gainModeCalls = %v, want [false]", fake.gainModeCalls)
	}
	if len(fake.gainCalls) != 0 {
		t.Errorf("gainCalls = %v, want empty (AGC mode)", fake.gainCalls)
	}
}

func TestDevice_SetGainManualSetsModeAndValue(t *testing.T) {
	fake := &fakeTuner{}
	d := newDeviceWithFakeTuner(usb.NewMockTransport(), fake)
	if err := d.SetGain(250); err != nil {
		t.Fatalf("SetGain(250): %v", err)
	}
	if len(fake.gainModeCalls) != 1 || fake.gainModeCalls[0] != true {
		t.Errorf("gainModeCalls = %v, want [true]", fake.gainModeCalls)
	}
	if len(fake.gainCalls) != 1 || fake.gainCalls[0] != 250 {
		t.Errorf("gainCalls = %v, want [250]", fake.gainCalls)
	}
}

func TestStreamIQ_ChannelClosesOnContextCancel(t *testing.T) {
	mock := usb.NewMockTransport()
	// Script: ResetBuffer = 2 block writes (0x1002, 0x0000) at
	// USBEpaCtl. Then bulk packets dispatched by mock.
	mock.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: rtl2832u.USBEpaCtl, WIndex: uint16(rtl2832u.BlockUSB)<<8 | 0x10, Data: []byte{0x10, 0x02}},
		{In: false, BRequest: 0, WValue: rtl2832u.USBEpaCtl, WIndex: uint16(rtl2832u.BlockUSB)<<8 | 0x10, Data: []byte{0x00, 0x00}},
	}
	mock.BulkPackets = [][]byte{
		{127, 128, 127, 128}, // 2 IQ samples near zero
	}
	d := newDeviceWithFakeTuner(mock, &fakeTuner{})

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := d.StreamIQ(ctx)
	if err != nil {
		t.Fatalf("StreamIQ: %v", err)
	}

	// Drain at least one sample (the mock dispatches the packet
	// immediately after StartBulkIn).
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first packet")
	}

	cancel()
	// Channel must close after cancel.
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
loop:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break loop
			}
		case <-deadline.C:
			t.Fatal("channel did not close after ctx cancel")
		}
	}
}

func TestStreamIQ_DoubleStartFails(t *testing.T) {
	mock := usb.NewMockTransport()
	mock.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: rtl2832u.USBEpaCtl, WIndex: uint16(rtl2832u.BlockUSB)<<8 | 0x10, Data: []byte{0x10, 0x02}},
		{In: false, BRequest: 0, WValue: rtl2832u.USBEpaCtl, WIndex: uint16(rtl2832u.BlockUSB)<<8 | 0x10, Data: []byte{0x00, 0x00}},
	}
	d := newDeviceWithFakeTuner(mock, &fakeTuner{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := d.StreamIQ(ctx); err != nil {
		t.Fatalf("first StreamIQ: %v", err)
	}
	if _, err := d.StreamIQ(ctx); err == nil {
		t.Error("second StreamIQ returned nil, want error")
	}
}

func TestStreamIQ_AfterCloseFails(t *testing.T) {
	d := newDeviceWithFakeTuner(usb.NewMockTransport(), &fakeTuner{})
	d.closed.Store(true)
	if _, err := d.StreamIQ(context.Background()); !errors.Is(err, ErrClosed) {
		t.Errorf("StreamIQ after close = %v, want ErrClosed", err)
	}
}

func TestDevice_CloseIdempotent(t *testing.T) {
	d := newDeviceWithFakeTuner(usb.NewMockTransport(), &fakeTuner{})
	// First Close — DeinitBaseband writes 1 block reg, transport.Close
	// is a no-op on the mock. Tuner standby increments the counter.
	mock := d.transport.(*usb.MockTransport)
	mock.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: rtl2832u.SysDemodCtl, WIndex: uint16(rtl2832u.BlockSys)<<8 | 0x10, Data: []byte{0x20}},
	}
	if err := d.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	fake := d.tuner.(*fakeTuner)
	if fake.standbyCalls != 1 {
		t.Errorf("standby called %d times across two Close() calls, want exactly 1", fake.standbyCalls)
	}
}

func TestStreamConstants_MatchCGOGeometry(t *testing.T) {
	// Pin the buffer geometry so any drift from the CGO driver
	// (which still ships in rtlsdr_cgo.go until PR-09) shows up
	// as a test failure.
	if asyncBufCount != 32 {
		t.Errorf("asyncBufCount = %d, want 32 (matches rtlsdr_cgo.go:43)", asyncBufCount)
	}
	if asyncBufLen != 16*1024 {
		t.Errorf("asyncBufLen = %d, want 16384 (matches rtlsdr_cgo.go:44)", asyncBufLen)
	}
	if streamChanDepth != 8 {
		t.Errorf("streamChanDepth = %d, want 8 (matches rtlsdr_cgo.go:190)", streamChanDepth)
	}
	if bulkInEndpoint != 0x81 {
		t.Errorf("bulkInEndpoint = 0x%02x, want 0x81", bulkInEndpoint)
	}
}
