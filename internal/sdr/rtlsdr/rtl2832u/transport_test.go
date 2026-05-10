package rtl2832u

import (
	"bytes"
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// commit is the dummy demod read every demod write triggers (page 0x0A,
// addr 0x01, 1 byte) — defined once so the test scripts read cleanly.
var commit = usb.CtrlExchange{In: true, BRequest: 0, WValue: (0x01 << 8) | 0x20, WIndex: 0x0A, N: 1, Reply: []byte{0x00}}

func TestReadBlockReg_EncodesUSBControlTransfer(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: true, BRequest: 0, WValue: USBSysctl, WIndex: uint16(BlockUSB) << 8, N: 1, Reply: []byte{0xAB}},
	}
	d := New(m)
	got, err := d.ReadBlockReg(BlockUSB, USBSysctl, 1)
	if err != nil {
		t.Fatalf("ReadBlockReg: %v", err)
	}
	if !bytes.Equal(got, []byte{0xAB}) {
		t.Errorf("got %x, want AB", got)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestWriteBlockReg_OneByte(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: USBSysctl, WIndex: uint16(BlockUSB)<<8 | 0x10, Data: []byte{0x09}},
	}
	d := New(m)
	if err := d.WriteBlockReg(BlockUSB, USBSysctl, 0x09, 1); err != nil {
		t.Fatalf("WriteBlockReg: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestWriteBlockReg_TwoBytesBigEndian(t *testing.T) {
	// librtlsdr emits 2-byte writes big-endian: data[0]=high, data[1]=low.
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: USBEpaCtl, WIndex: uint16(BlockUSB)<<8 | 0x10, Data: []byte{0x10, 0x02}},
	}
	d := New(m)
	if err := d.WriteBlockReg(BlockUSB, USBEpaCtl, 0x1002, 2); err != nil {
		t.Fatalf("WriteBlockReg: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestReadDemodReg_EncodesPageInIndex(t *testing.T) {
	m := usb.NewMockTransport()
	// page=1, addr=0x01 → wValue=(0x01<<8)|0x20=0x0120, wIndex=page=1
	m.Script = []usb.CtrlExchange{
		{In: true, BRequest: 0, WValue: 0x0120, WIndex: 1, N: 1, Reply: []byte{0xCD}},
	}
	d := New(m)
	got, err := d.ReadDemodReg(1, 0x01, 1)
	if err != nil {
		t.Fatalf("ReadDemodReg: %v", err)
	}
	if !bytes.Equal(got, []byte{0xCD}) {
		t.Errorf("got %x, want CD", got)
	}
}

func TestWriteDemodReg_IncludesCommitRead(t *testing.T) {
	// Each demod write must be followed by a 1-byte read of page 0x0A,
	// addr 0x01 — without it the chip doesn't latch the write.
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		// The write itself: page=1, addr=0x01, val=0x14, wValue=(0x01<<8)|0x20=0x0120,
		// wIndex=0x10|1=0x11, data=[0x14] (low byte for n=1).
		{In: false, BRequest: 0, WValue: 0x0120, WIndex: 0x11, Data: []byte{0x14}},
		// The commit read: page=0x0A, addr=0x01, wValue=0x0120, wIndex=0x0A, n=1.
		commit,
	}
	d := New(m)
	if err := d.WriteDemodReg(1, 0x01, 0x14, 1); err != nil {
		t.Fatalf("WriteDemodReg: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestEncodeWriteVal(t *testing.T) {
	cases := []struct {
		val  uint16
		n    int
		want []byte
	}{
		{val: 0x09, n: 1, want: []byte{0x09}},
		{val: 0x42, n: 1, want: []byte{0x42}},
		{val: 0xABCD, n: 1, want: []byte{0xCD}}, // n=1 truncates to low byte
		{val: 0x0000, n: 2, want: []byte{0x00, 0x00}},
		{val: 0x1002, n: 2, want: []byte{0x10, 0x02}}, // big-endian
		{val: 0xABCD, n: 2, want: []byte{0xAB, 0xCD}}, // big-endian
	}
	for _, tc := range cases {
		got := encodeWriteVal(tc.val, tc.n)
		if !bytes.Equal(got, tc.want) {
			t.Errorf("encodeWriteVal(0x%04x, %d) = %x, want %x", tc.val, tc.n, got, tc.want)
		}
	}
}

func TestEncodeWriteVal_BadN(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("encodeWriteVal with n=3 should panic")
		}
	}()
	encodeWriteVal(0, 3)
}

func TestWriteBlockReg_PropagatesTransportError(t *testing.T) {
	wantErr := errors.New("boom")
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: USBSysctl, WIndex: uint16(BlockUSB)<<8 | 0x10, Data: []byte{0x09}, Err: wantErr},
	}
	d := New(m)
	err := d.WriteBlockReg(BlockUSB, USBSysctl, 0x09, 1)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want chain containing %v", err, wantErr)
	}
}

func TestXtalDefaultAndOverride(t *testing.T) {
	d := New(usb.NewMockTransport())
	if d.XtalHz() != DefaultXtalHz {
		t.Errorf("default xtal = %d, want %d", d.XtalHz(), DefaultXtalHz)
	}
	d.SetXtal(28_700_000)
	if d.XtalHz() != 28_700_000 {
		t.Errorf("overridden xtal = %d, want 28_700_000", d.XtalHz())
	}
}
