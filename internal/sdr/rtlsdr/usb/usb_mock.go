package usb

import (
	"fmt"
	"sync"
	"time"
)

// MockEnumerator returns a fixed set of [Descriptor] values and opens each
// of them as a [MockTransport]. It is the test fixture used by every
// higher layer (rtl2832u, tuners) so the RTL-SDR driver can be exercised
// without a real dongle in CI.
type MockEnumerator struct {
	BackendName string       // override "mock" if set
	Devices     []Descriptor // returned by List, filtered by vid/pid
	OpenFunc    func(Descriptor) (*MockTransport, error)
}

func (m *MockEnumerator) Name() string {
	if m.BackendName != "" {
		return m.BackendName
	}
	return "mock"
}

func (m *MockEnumerator) List(vid, pid uint16) ([]Descriptor, error) {
	out := make([]Descriptor, 0, len(m.Devices))
	for _, d := range m.Devices {
		if vid != 0 && d.VID != vid {
			continue
		}
		if pid != 0 && d.PID != pid {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (m *MockEnumerator) Open(d Descriptor) (Transport, error) {
	if m.OpenFunc != nil {
		return m.OpenFunc(d)
	}
	return NewMockTransport(), nil
}

// CtrlExchange is one step in a [MockTransport.Script]. ControlIn matches
// against (BRequest, WValue, WIndex) and returns Reply; ControlOut matches
// the same triple plus the data payload and returns Err. Direction is
// inferred from In: if In is true the SUT must call ControlIn, otherwise
// ControlOut.
type CtrlExchange struct {
	In        bool
	BRequest  uint8
	WValue    uint16
	WIndex    uint16
	Data      []byte // expected for OUT, ignored for IN
	Reply     []byte // returned for IN, ignored for OUT
	Err       error  // returned instead of completing the transfer
	N         int    // expected `n` for IN (0 = don't check)
	TimeoutOK bool   // when true, ignore the SUT's timeout arg
}

// MockTransport replays a [Script] of [CtrlExchange] entries in order.
// Mismatched calls fail with an error captured in [MockTransport.Err];
// tests inspect Err after running the SUT. Bulk-IN is supported by
// pushing pre-built byte slices into BulkPackets; the transport spaces
// them by BulkInterval (default 0 = back-to-back).
type MockTransport struct {
	Script        []CtrlExchange
	Step          int
	Err           error
	ClaimedIfaces map[int]bool
	Closed        bool

	BulkPackets  [][]byte
	BulkInterval time.Duration

	mu       sync.Mutex
	bulkActv bool
	bulkStop chan struct{}
	bulkDone chan struct{}
}

// NewMockTransport returns an empty mock that fails every transfer until
// Script is populated. Useful when a test only cares about lifecycle
// (Open / ClaimInterface / Close).
func NewMockTransport() *MockTransport {
	return &MockTransport{ClaimedIfaces: map[int]bool{}}
}

func (m *MockTransport) recordErr(err error) {
	if m.Err == nil {
		m.Err = err
	}
}

func (m *MockTransport) ControlIn(bRequest uint8, wValue, wIndex uint16, n int, timeoutMs int) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Closed {
		return nil, ErrClosed
	}
	if m.Step >= len(m.Script) {
		err := fmt.Errorf("mock: unexpected ControlIn req=0x%02x val=0x%04x idx=0x%04x n=%d (script exhausted at step %d)", bRequest, wValue, wIndex, n, m.Step)
		m.recordErr(err)
		return nil, err
	}
	want := m.Script[m.Step]
	m.Step++
	if !want.In {
		err := fmt.Errorf("mock step %d: expected ControlOut, got ControlIn(req=0x%02x,val=0x%04x,idx=0x%04x)", m.Step-1, bRequest, wValue, wIndex)
		m.recordErr(err)
		return nil, err
	}
	if want.BRequest != bRequest || want.WValue != wValue || want.WIndex != wIndex {
		err := fmt.Errorf("mock step %d: ControlIn mismatch: got (req=0x%02x,val=0x%04x,idx=0x%04x), want (req=0x%02x,val=0x%04x,idx=0x%04x)", m.Step-1, bRequest, wValue, wIndex, want.BRequest, want.WValue, want.WIndex)
		m.recordErr(err)
		return nil, err
	}
	if want.N != 0 && want.N != n {
		err := fmt.Errorf("mock step %d: ControlIn n mismatch: got %d, want %d", m.Step-1, n, want.N)
		m.recordErr(err)
		return nil, err
	}
	if want.Err != nil {
		return nil, want.Err
	}
	out := make([]byte, len(want.Reply))
	copy(out, want.Reply)
	_ = timeoutMs
	return out, nil
}

func (m *MockTransport) ControlOut(bRequest uint8, wValue, wIndex uint16, data []byte, timeoutMs int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Closed {
		return ErrClosed
	}
	if m.Step >= len(m.Script) {
		err := fmt.Errorf("mock: unexpected ControlOut req=0x%02x val=0x%04x idx=0x%04x len=%d (script exhausted at step %d)", bRequest, wValue, wIndex, len(data), m.Step)
		m.recordErr(err)
		return err
	}
	want := m.Script[m.Step]
	m.Step++
	if want.In {
		err := fmt.Errorf("mock step %d: expected ControlIn, got ControlOut(req=0x%02x,val=0x%04x,idx=0x%04x,len=%d)", m.Step-1, bRequest, wValue, wIndex, len(data))
		m.recordErr(err)
		return err
	}
	if want.BRequest != bRequest || want.WValue != wValue || want.WIndex != wIndex {
		err := fmt.Errorf("mock step %d: ControlOut mismatch: got (req=0x%02x,val=0x%04x,idx=0x%04x), want (req=0x%02x,val=0x%04x,idx=0x%04x)", m.Step-1, bRequest, wValue, wIndex, want.BRequest, want.WValue, want.WIndex)
		m.recordErr(err)
		return err
	}
	if !bytesEqual(want.Data, data) {
		err := fmt.Errorf("mock step %d: ControlOut data mismatch: got %#x, want %#x", m.Step-1, data, want.Data)
		m.recordErr(err)
		return err
	}
	_ = timeoutMs
	return want.Err
}

func (m *MockTransport) ClaimInterface(num int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Closed {
		return ErrClosed
	}
	m.ClaimedIfaces[num] = true
	return nil
}

func (m *MockTransport) ReleaseInterface(num int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.ClaimedIfaces, num)
	return nil
}

func (m *MockTransport) StartBulkIn(epAddr byte, ringBufs, bufLen int, onPacket func([]byte)) error {
	m.mu.Lock()
	if m.Closed {
		m.mu.Unlock()
		return ErrClosed
	}
	if m.bulkActv {
		m.mu.Unlock()
		return ErrBulkActive
	}
	m.bulkActv = true
	m.bulkStop = make(chan struct{})
	m.bulkDone = make(chan struct{})
	packets := append([][]byte(nil), m.BulkPackets...)
	interval := m.BulkInterval
	m.mu.Unlock()

	go func() {
		defer close(m.bulkDone)
		for _, pkt := range packets {
			select {
			case <-m.bulkStop:
				return
			default:
			}
			buf := make([]byte, len(pkt))
			copy(buf, pkt)
			onPacket(buf)
			if interval > 0 {
				select {
				case <-m.bulkStop:
					return
				case <-time.After(interval):
				}
			}
		}
		<-m.bulkStop
	}()
	_ = epAddr
	_ = ringBufs
	_ = bufLen
	return nil
}

func (m *MockTransport) StopBulkIn() error {
	m.mu.Lock()
	if !m.bulkActv {
		m.mu.Unlock()
		return ErrBulkInactive
	}
	m.bulkActv = false
	stop := m.bulkStop
	done := m.bulkDone
	m.mu.Unlock()
	close(stop)
	<-done
	return nil
}

func (m *MockTransport) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Closed {
		return ErrClosed
	}
	return nil
}

func (m *MockTransport) Close() error {
	m.mu.Lock()
	if m.Closed {
		m.mu.Unlock()
		return nil
	}
	m.Closed = true
	active := m.bulkActv
	stop := m.bulkStop
	done := m.bulkDone
	m.bulkActv = false
	m.mu.Unlock()
	if active {
		close(stop)
		<-done
	}
	return nil
}

// Remaining returns the number of unconsumed [CtrlExchange] entries in
// the script. Tests typically assert it is zero after the SUT runs.
func (m *MockTransport) Remaining() int { return len(m.Script) - m.Step }

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
