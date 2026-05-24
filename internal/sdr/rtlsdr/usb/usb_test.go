package usb

import (
	"bytes"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMockEnumerator_ListFiltersByVIDPID(t *testing.T) {
	e := &MockEnumerator{Devices: []Descriptor{
		{Bus: 1, Address: 4, VID: 0x0bda, PID: 0x2838, Serial: "00000001"},
		{Bus: 1, Address: 5, VID: 0x0bda, PID: 0x2832, Serial: "00000002"},
		{Bus: 1, Address: 6, VID: 0x1d50, PID: 0x6089, Serial: "hackrf"},
	}}

	all, err := e.List(0, 0)
	if err != nil {
		t.Fatalf("List(0,0): %v", err)
	}
	if got, want := len(all), 3; got != want {
		t.Fatalf("List(0,0) len = %d, want %d", got, want)
	}

	rtl, err := e.List(0x0bda, 0)
	if err != nil {
		t.Fatalf("List vendor: %v", err)
	}
	if got, want := len(rtl), 2; got != want {
		t.Fatalf("Realtek len = %d, want %d", got, want)
	}

	one, err := e.List(0x0bda, 0x2838)
	if err != nil {
		t.Fatalf("List exact: %v", err)
	}
	if got, want := len(one), 1; got != want {
		t.Fatalf("exact match len = %d, want %d", got, want)
	}
	if one[0].Serial != "00000001" {
		t.Errorf("matched device serial = %q, want 00000001", one[0].Serial)
	}
}

func TestMockEnumerator_OpenReturnsTransport(t *testing.T) {
	e := &MockEnumerator{Devices: []Descriptor{{VID: 1, PID: 2}}}
	got, err := e.Open(e.Devices[0])
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got == nil {
		t.Fatal("Open returned nil transport")
	}
	_ = got.Close()
}

func TestMockTransport_ScriptedControlIn(t *testing.T) {
	m := NewMockTransport()
	m.Script = []CtrlExchange{
		{In: true, BRequest: 0, WValue: 0x0010, WIndex: 0x0300, N: 2, Reply: []byte{0xab, 0xcd}},
	}
	got, err := m.ControlIn(0, 0x0010, 0x0300, 2, 1000)
	if err != nil {
		t.Fatalf("ControlIn: %v", err)
	}
	if !bytes.Equal(got, []byte{0xab, 0xcd}) {
		t.Errorf("ControlIn data = %x, want abcd", got)
	}
	if m.Err != nil {
		t.Errorf("m.Err = %v, want nil", m.Err)
	}
	if m.Remaining() != 0 {
		t.Errorf("Remaining = %d, want 0", m.Remaining())
	}
}

func TestMockTransport_ScriptedControlOut(t *testing.T) {
	m := NewMockTransport()
	m.Script = []CtrlExchange{
		{In: false, BRequest: 1, WValue: 0x4444, WIndex: 0x0100, Data: []byte{0x10, 0x20}},
	}
	if err := m.ControlOut(1, 0x4444, 0x0100, []byte{0x10, 0x20}, 0); err != nil {
		t.Fatalf("ControlOut: %v", err)
	}
	if m.Err != nil {
		t.Errorf("m.Err = %v", m.Err)
	}
}

func TestMockTransport_DirectionMismatch(t *testing.T) {
	m := NewMockTransport()
	m.Script = []CtrlExchange{{In: true, BRequest: 0, WValue: 0, WIndex: 0, Reply: []byte{0}}}
	if err := m.ControlOut(0, 0, 0, nil, 0); err == nil {
		t.Fatal("expected error for direction mismatch")
	}
	if m.Err == nil {
		t.Error("m.Err not populated on mismatch")
	}
}

func TestMockTransport_ParamMismatch(t *testing.T) {
	m := NewMockTransport()
	m.Script = []CtrlExchange{{In: true, BRequest: 5, WValue: 1, WIndex: 2, Reply: []byte{0}}}
	if _, err := m.ControlIn(6, 1, 2, 1, 0); err == nil {
		t.Fatal("expected error for bRequest mismatch")
	}
}

func TestMockTransport_DataMismatch(t *testing.T) {
	m := NewMockTransport()
	m.Script = []CtrlExchange{{In: false, BRequest: 0, WValue: 0, WIndex: 0, Data: []byte{1, 2, 3}}}
	if err := m.ControlOut(0, 0, 0, []byte{1, 2, 4}, 0); err == nil {
		t.Fatal("expected error for data mismatch")
	}
}

func TestMockTransport_ScriptExhausted(t *testing.T) {
	m := NewMockTransport()
	if _, err := m.ControlIn(0, 0, 0, 1, 0); err == nil {
		t.Fatal("expected error from empty script")
	}
}

func TestMockTransport_NParamCheck(t *testing.T) {
	m := NewMockTransport()
	m.Script = []CtrlExchange{{In: true, BRequest: 0, WValue: 0, WIndex: 0, N: 4, Reply: []byte{1, 2, 3, 4}}}
	if _, err := m.ControlIn(0, 0, 0, 2, 0); err == nil {
		t.Fatal("expected error for n mismatch")
	}
}

func TestMockTransport_ErrFieldReturned(t *testing.T) {
	want := errors.New("boom")
	m := NewMockTransport()
	m.Script = []CtrlExchange{{In: true, BRequest: 0, WValue: 0, WIndex: 0, Err: want}}
	if _, err := m.ControlIn(0, 0, 0, 1, 0); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestMockTransport_BulkInDispatch(t *testing.T) {
	m := NewMockTransport()
	m.BulkPackets = [][]byte{
		{0x01, 0x02},
		{0x03, 0x04, 0x05},
	}

	var mu sync.Mutex
	var got [][]byte
	if err := m.StartBulkIn(0x81, 4, 8, func(p []byte) {
		mu.Lock()
		c := make([]byte, len(p))
		copy(c, p)
		got = append(got, c)
		mu.Unlock()
	}, nil); err != nil {
		t.Fatalf("StartBulkIn: %v", err)
	}
	// Wait for the goroutine to dispatch both packets and park on bulkStop.
	deadline := time.Now().Add(time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for bulk packets")
		}
		time.Sleep(time.Millisecond)
	}
	if err := m.StopBulkIn(); err != nil {
		t.Fatalf("StopBulkIn: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(got[0], []byte{0x01, 0x02}) || !bytes.Equal(got[1], []byte{0x03, 0x04, 0x05}) {
		t.Errorf("got packets %x, want [[0102] [030405]]", got)
	}
}

func TestMockTransport_DoubleStartFails(t *testing.T) {
	m := NewMockTransport()
	if err := m.StartBulkIn(0x81, 1, 8, func([]byte) {}, nil); err != nil {
		t.Fatalf("first StartBulkIn: %v", err)
	}
	if err := m.StartBulkIn(0x81, 1, 8, func([]byte) {}, nil); !errors.Is(err, ErrBulkActive) {
		t.Errorf("second StartBulkIn err = %v, want ErrBulkActive", err)
	}
	_ = m.StopBulkIn()
}

func TestMockTransport_StopWithoutStart(t *testing.T) {
	m := NewMockTransport()
	if err := m.StopBulkIn(); !errors.Is(err, ErrBulkInactive) {
		t.Errorf("StopBulkIn without start = %v, want ErrBulkInactive", err)
	}
}

func TestMockTransport_ClaimReleaseTrackIfaces(t *testing.T) {
	m := NewMockTransport()
	if err := m.ClaimInterface(0); err != nil {
		t.Fatalf("Claim(0): %v", err)
	}
	if !m.ClaimedIfaces[0] {
		t.Error("interface 0 not tracked after Claim")
	}
	if err := m.ReleaseInterface(0); err != nil {
		t.Fatalf("Release(0): %v", err)
	}
	if m.ClaimedIfaces[0] {
		t.Error("interface 0 still tracked after Release")
	}
}

func TestMockTransport_CloseIdempotent(t *testing.T) {
	m := NewMockTransport()
	if err := m.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, err := m.ControlIn(0, 0, 0, 1, 0); !errors.Is(err, ErrClosed) {
		t.Errorf("ControlIn after Close = %v, want ErrClosed", err)
	}
	if err := m.ControlOut(0, 0, 0, nil, 0); !errors.Is(err, ErrClosed) {
		t.Errorf("ControlOut after Close = %v, want ErrClosed", err)
	}
}

func TestMockTransport_CloseStopsBulk(t *testing.T) {
	m := NewMockTransport()
	if err := m.StartBulkIn(0x81, 1, 8, func([]byte) {}, nil); err != nil {
		t.Fatalf("StartBulkIn: %v", err)
	}
	done := make(chan struct{})
	go func() {
		_ = m.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not return")
	}
}

// TestMockTransport_OnStreamDeadFiresWhenSimulated pins the issue-#345
// contract: when BulkSimulateDeath is set, the mock fires onStreamDead
// after dispatching all packets — drivers wire this into their consumer-
// channel teardown path so a real reaper death doesn't leave the IQ
// consumer (ccdecoder) blocked forever at 0% CPU.
func TestMockTransport_OnStreamDeadFiresWhenSimulated(t *testing.T) {
	m := NewMockTransport()
	m.BulkPackets = [][]byte{{0xaa}}
	m.BulkSimulateDeath = true
	m.BulkDeathErr = ErrDeviceGone

	deadCh := make(chan error, 1)
	if err := m.StartBulkIn(0x81, 1, 8, func([]byte) {}, func(err error) {
		deadCh <- err
	}); err != nil {
		t.Fatalf("StartBulkIn: %v", err)
	}
	select {
	case err := <-deadCh:
		if !errors.Is(err, ErrDeviceGone) {
			t.Errorf("onStreamDead err = %v, want ErrDeviceGone", err)
		}
	case <-time.After(time.Second):
		t.Fatal("onStreamDead did not fire after simulated death")
	}
	// onStreamDead must not fire a second time even if Stop is called.
	if err := m.StopBulkIn(); err != nil {
		t.Fatalf("StopBulkIn: %v", err)
	}
	select {
	case err := <-deadCh:
		t.Errorf("onStreamDead fired again after StopBulkIn: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}

// TestMockTransport_OnStreamDeadNotFiredOnNormalStop guards against
// false positives: a clean StopBulkIn (no simulated death) must not
// fire onStreamDead, or the daemon's restart loop would trigger on
// every normal shutdown.
func TestMockTransport_OnStreamDeadNotFiredOnNormalStop(t *testing.T) {
	m := NewMockTransport()
	m.BulkPackets = [][]byte{{0xaa}}
	called := make(chan struct{}, 1)
	if err := m.StartBulkIn(0x81, 1, 8, func([]byte) {}, func(error) { called <- struct{}{} }); err != nil {
		t.Fatalf("StartBulkIn: %v", err)
	}
	// Wait briefly so the goroutine dispatches the packet and parks
	// on bulkStop (rather than racing StopBulkIn).
	time.Sleep(20 * time.Millisecond)
	if err := m.StopBulkIn(); err != nil {
		t.Fatalf("StopBulkIn: %v", err)
	}
	select {
	case <-called:
		t.Fatal("onStreamDead fired on normal StopBulkIn")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDefaultEnumeratorNotNil(t *testing.T) {
	e := DefaultEnumerator()
	if e == nil {
		t.Fatal("DefaultEnumerator returned nil")
	}
	if e.Name() == "" {
		t.Error("backend Name() empty")
	}
}

func TestVendorConstants(t *testing.T) {
	if VendorIn != 0xC0 {
		t.Errorf("VendorIn = 0x%02x, want 0xC0", VendorIn)
	}
	if VendorOut != 0x40 {
		t.Errorf("VendorOut = 0x%02x, want 0x40", VendorOut)
	}
}
