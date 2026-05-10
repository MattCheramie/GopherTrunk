package rtl2832u

import (
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func TestSetGPIOOutput_RMW(t *testing.T) {
	// SetGPIOOutput does a read-modify-write on both GPD and GPOE,
	// touching only the bit for the requested pin.
	m := usb.NewMockTransport()
	// Existing GPD = 0xFF (all inputs); we clear bit 0.
	m.Script = []usb.CtrlExchange{
		{In: true, BRequest: 0, WValue: SysGPD, WIndex: uint16(BlockSys) << 8, N: 1, Reply: []byte{0xFF}},
		expectBlockWrite(BlockSys, SysGPD, 0xFE, 1),
		{In: true, BRequest: 0, WValue: SysGPOE, WIndex: uint16(BlockSys) << 8, N: 1, Reply: []byte{0x00}},
		expectBlockWrite(BlockSys, SysGPOE, 0x01, 1),
	}
	d := New(m)
	if err := d.SetGPIOOutput(0); err != nil {
		t.Fatalf("SetGPIOOutput(0): %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestSetGPIOBit_HighThenLow(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		// High: read GPO=0x00, write 0x01 (bit 0 set).
		{In: true, BRequest: 0, WValue: SysGPO, WIndex: uint16(BlockSys) << 8, N: 1, Reply: []byte{0x00}},
		expectBlockWrite(BlockSys, SysGPO, 0x01, 1),
		// Low: read GPO=0x01 (the previous write), write 0x00.
		{In: true, BRequest: 0, WValue: SysGPO, WIndex: uint16(BlockSys) << 8, N: 1, Reply: []byte{0x01}},
		expectBlockWrite(BlockSys, SysGPO, 0x00, 1),
	}
	d := New(m)
	if err := d.SetGPIOBit(0, true); err != nil {
		t.Fatalf("SetGPIOBit(0,true): %v", err)
	}
	if err := d.SetGPIOBit(0, false); err != nil {
		t.Fatalf("SetGPIOBit(0,false): %v", err)
	}
}

func TestSetGPIO_PinOutOfRange(t *testing.T) {
	d := New(usb.NewMockTransport())
	if err := d.SetGPIOOutput(8); err == nil {
		t.Error("SetGPIOOutput(8) should fail")
	}
	if err := d.SetGPIOBit(255, true); err == nil {
		t.Error("SetGPIOBit(255) should fail")
	}
}

func TestSetBiasTee_FullSequence(t *testing.T) {
	// Combines SetGPIOOutput + SetGPIOBit. Pin 0, on.
	m := usb.NewMockTransport()
	m.Script = []usb.CtrlExchange{
		// SetGPIOOutput(0) — GPD r/m/w then GPOE r/m/w.
		{In: true, BRequest: 0, WValue: SysGPD, WIndex: uint16(BlockSys) << 8, N: 1, Reply: []byte{0xFF}},
		expectBlockWrite(BlockSys, SysGPD, 0xFE, 1),
		{In: true, BRequest: 0, WValue: SysGPOE, WIndex: uint16(BlockSys) << 8, N: 1, Reply: []byte{0x00}},
		expectBlockWrite(BlockSys, SysGPOE, 0x01, 1),
		// SetGPIOBit(0, true) — GPO r/m/w.
		{In: true, BRequest: 0, WValue: SysGPO, WIndex: uint16(BlockSys) << 8, N: 1, Reply: []byte{0x00}},
		expectBlockWrite(BlockSys, SysGPO, 0x01, 1),
	}
	d := New(m)
	if err := d.SetBiasTee(0, true); err != nil {
		t.Fatalf("SetBiasTee: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}
