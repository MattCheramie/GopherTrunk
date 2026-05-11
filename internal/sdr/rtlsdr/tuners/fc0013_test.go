package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

func TestFC0013_TypeAndIF(t *testing.T) {
	f := NewFC0013(rtl2832u.New(usb.NewMockTransport()))
	if f.Type() != TypeFC0013 {
		t.Errorf("Type() = %v, want FC0013", f.Type())
	}
	if f.IFFreqHz() != 6_000_000 {
		t.Errorf("IFFreqHz() = %d, want 6_000_000", f.IFFreqHz())
	}
}

func TestFC0013_GainsLadder26Steps(t *testing.T) {
	f := NewFC0013(rtl2832u.New(usb.NewMockTransport()))
	g := f.Gains()
	if len(g) != 26 {
		t.Errorf("Gains() returned %d entries, want 26 (librtlsdr fc0013_lna_gains)", len(g))
	}
	if g[0] != -99 || g[len(g)-1] != 118 {
		t.Errorf("Gains() endpoints = (%d, %d), want (-99, 118)", g[0], g[len(g)-1])
	}
}

func TestFC0013_InitWritesSoftResetThenFlood(t *testing.T) {
	m := usb.NewMockTransport()
	m.Script = append(m.Script, expectI2CWriteReg(fc0013I2CAddr, 0x0C, 0x05)...)
	m.Script = append(m.Script, expectI2CWriteReg(fc0013I2CAddr, 0x0C, 0x00)...)
	for i, v := range fc0013InitArray {
		m.Script = append(m.Script, expectI2CWriteReg(fc0013I2CAddr, byte(i+1), v)...)
	}
	f := NewFC0013(rtl2832u.New(m))
	if err := f.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if m.Err != nil || m.Remaining() != 0 {
		t.Errorf("mock state: err=%v remaining=%d", m.Err, m.Remaining())
	}
}

func TestFC0013_SetFreqRangeGuard(t *testing.T) {
	f := NewFC0013(rtl2832u.New(usb.NewMockTransport()))
	f.initDone = true
	var rangeErr *ErrUnsupportedFreq
	if err := f.SetFreq(10_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("below-floor err = %v, want *ErrUnsupportedFreq", err)
	}
	if err := f.SetFreq(2_000_000_000); !errors.As(err, &rangeErr) {
		t.Errorf("above-ceiling err = %v, want *ErrUnsupportedFreq", err)
	}
}
