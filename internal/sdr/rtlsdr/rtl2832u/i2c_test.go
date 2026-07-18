package rtl2832u

import (
	"bytes"
	"syscall"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// demodWriteStep builds a single demod-register-write CtrlExchange (no commit
// read), optionally scripted to return err instead of completing — so a test
// can inject a control-pipe stall on the write itself.
func demodWriteStep(page uint8, addr, val uint16, n int, err error) usb.CtrlExchange {
	return usb.CtrlExchange{
		In: false, BRequest: 0,
		WValue: (addr << 8) | 0x20,
		WIndex: uint16(0x10) | uint16(page),
		Data:   encodeWriteVal(val, n),
		Err:    err,
	}
}

// TestSetI2CRepeater_RecoversFromControlPipeStall reproduces issue #753: the
// RTL-SDR Blog V4's R828D intermittently STALLs the SetI2CRepeater demod
// write mid-run — surfacing as a "broken pipe" (EPIPE) on Linux usbdevfs, or
// usb.ErrPipeStalled on Windows/WinUSB — which aborted the whole
// SetCenterFreq retune with `SetI2CRepeater(true): ... broken pipe`.
// rtl_test / SDR++ tune cleanly because libusb recovers a control-endpoint
// stall (it clears on the next SETUP); GopherTrunk's native demod control
// path did not retry, unlike the tuner burst path (writeBurstChunk). The
// write must settle-and-retry once and succeed.
func TestSetI2CRepeater_RecoversFromControlPipeStall(t *testing.T) {
	defer func(d time.Duration) { ctrlStallRetryDelay = d }(ctrlStallRetryDelay)
	ctrlStallRetryDelay = 0
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"linux-epipe", syscall.EPIPE},
		{"win-pipe-stalled", usb.ErrPipeStalled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := usb.NewMockTransport()
			m.Script = []usb.CtrlExchange{
				demodWriteStep(1, 0x01, 0x18, 1, tc.err), // stall
				demodWriteStep(1, 0x01, 0x18, 1, nil),    // retry succeeds
				commit,
			}
			d := New(m)
			d.EnableControlStallRetry() // runtime path (post bring-up)
			if err := d.SetI2CRepeater(true); err != nil {
				t.Fatalf("SetI2CRepeater(true) must recover from a single control-pipe stall, got: %v", err)
			}
			if m.Err != nil || m.Remaining() != 0 {
				t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
			}
		})
	}
}

// TestSetI2CRepeater_StallNotRetriedDuringBringup guards that the runtime
// recovery stays OFF until EnableControlStallRetry is called: during open-time
// bring-up a control-pipe stall must propagate raw so the driver's reset+retry
// envelope owns recovery (a full USBDEVFS_RESET, not an inline resubmit — the
// NESDR-v5 cold-boot latch only clears on reset, issue #248). This is what
// keeps the purego bring-up envelope tests intact.
func TestSetI2CRepeater_StallNotRetriedDuringBringup(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		demodWriteStep(1, 0x01, 0x18, 1, syscall.EPIPE), // stall — must NOT be retried
	}
	d := New(m) // retry not enabled: models bring-up
	if err := d.SetI2CRepeater(true); err == nil {
		t.Fatal("expected the stall to propagate during bring-up, got nil")
	}
	if m.Remaining() != 0 {
		t.Errorf("bring-up path must issue exactly one attempt (no inline retry), remaining=%d", m.Remaining())
	}
}

// TestSetI2CRepeater_PersistentStallSurfaces guards that the retry is bounded:
// a stall that persists across the retry surfaces the error (wrapped so the
// caller can still classify it) rather than looping.
func TestSetI2CRepeater_PersistentStallSurfaces(t *testing.T) {
	defer func(d time.Duration) { ctrlStallRetryDelay = d }(ctrlStallRetryDelay)
	ctrlStallRetryDelay = 0
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		demodWriteStep(1, 0x01, 0x18, 1, syscall.EPIPE), // stall
		demodWriteStep(1, 0x01, 0x18, 1, syscall.EPIPE), // retry also stalls
	}
	d := New(m)
	d.EnableControlStallRetry() // runtime path (post bring-up)
	err := d.SetI2CRepeater(true)
	if err == nil {
		t.Fatal("a persistent stall must surface an error, got nil")
	}
	if m.Remaining() != 0 {
		t.Errorf("expected exactly one retry (2 attempts), remaining=%d", m.Remaining())
	}
}

func TestSetI2CRepeater_On(t *testing.T) {
	// page=1 addr=0x01 val=0x18 (bit 3 + bit 4 = enable repeater).
	m := usb.NewMockTransport()
	m.Script = expectDemodWrite(1, 0x01, 0x18, 1)
	d := New(m)
	if err := d.SetI2CRepeater(true); err != nil {
		t.Fatalf("SetI2CRepeater(true): %v", err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}

func TestSetI2CRepeater_Off(t *testing.T) {
	// Default cached value is false; calling Off again is a no-op so we
	// need to flip on first, then back off.
	m := usb.NewMockTransport()
	script := []usb.CtrlExchange{}
	script = append(script, expectDemodWrite(1, 0x01, 0x18, 1)...)
	script = append(script, expectDemodWrite(1, 0x01, 0x10, 1)...)
	m.Script = script
	d := New(m)
	if err := d.SetI2CRepeater(true); err != nil {
		t.Fatalf("on: %v", err)
	}
	if err := d.SetI2CRepeater(false); err != nil {
		t.Fatalf("off: %v", err)
	}
	if m.Remaining() != 0 {
		t.Errorf("remaining=%d, want 0", m.Remaining())
	}
}

func TestSetI2CRepeater_CachesSameValue(t *testing.T) {
	// Second call with the same arg must not emit any transfer.
	m := usb.NewMockTransport()
	m.Script = expectDemodWrite(1, 0x01, 0x18, 1)
	d := New(m)
	if err := d.SetI2CRepeater(true); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := d.SetI2CRepeater(true); err != nil {
		t.Fatalf("redundant call: %v", err)
	}
	if m.Remaining() != 0 {
		t.Errorf("redundant call emitted a transfer (remaining=%d after first consumed)", m.Remaining())
	}
}

func TestI2CWriteReg_TwoBytePayload(t *testing.T) {
	// i2c_addr=0x34 (R820T2), reg=0x07, val=0x80.
	// On the wire: block=BlockIIC=6, wValue=0x0034, wIndex=0x0610,
	// data=[0x07, 0x80].
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: 0x0034, WIndex: uint16(BlockIIC)<<8 | 0x10, Data: []byte{0x07, 0x80}},
	}
	d := New(m)
	if err := d.I2CWriteReg(0x34, 0x07, 0x80); err != nil {
		t.Fatalf("I2CWriteReg: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestI2CReadReg_WriteThenRead(t *testing.T) {
	// write the register pointer, then read 1 byte.
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: 0x0034, WIndex: uint16(BlockIIC)<<8 | 0x10, Data: []byte{0x09}},
		{In: true, BRequest: 0, WValue: 0x0034, WIndex: uint16(BlockIIC) << 8, N: 1, Reply: []byte{0xC9}},
	}
	d := New(m)
	got, err := d.I2CReadReg(0x34, 0x09)
	if err != nil {
		t.Fatalf("I2CReadReg: %v", err)
	}
	if got != 0xC9 {
		t.Errorf("got 0x%02x, want 0xC9", got)
	}
}

func TestI2CWrite_BulkPayload(t *testing.T) {
	// Tuner driver might burst-write a whole register block.
	payload := []byte{0x05, 0x90, 0x00, 0x01}
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: 0x0034, WIndex: uint16(BlockIIC)<<8 | 0x10, Data: payload},
	}
	d := New(m)
	if err := d.I2CWrite(0x34, payload); err != nil {
		t.Fatalf("I2CWrite: %v", err)
	}
}

func TestI2CRead_BulkPayload(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: true, BRequest: 0, WValue: 0x0034, WIndex: uint16(BlockIIC) << 8, N: 4, Reply: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
	}
	d := New(m)
	got, err := d.I2CRead(0x34, 4)
	if err != nil {
		t.Fatalf("I2CRead: %v", err)
	}
	if !bytes.Equal(got, []byte{0xDE, 0xAD, 0xBE, 0xEF}) {
		t.Errorf("got %x, want DEADBEEF", got)
	}
}

func TestI2CRead_ZeroN(t *testing.T) {
	d := New(usb.NewMockTransport())
	got, err := d.I2CRead(0x34, 0)
	if err != nil {
		t.Fatalf("I2CRead(0): %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil for n=0", got)
	}
}
