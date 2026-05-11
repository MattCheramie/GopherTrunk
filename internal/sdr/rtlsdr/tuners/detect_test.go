package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// I2CRead at addr A returns 1 byte: 3 USB transfers (no pointer write,
// auto-increment from 0).
func expectI2CReadAddr0(addr uint8, replyByte byte) []usb.CtrlExchange {
	out := append([]usb.CtrlExchange{}, expectRepeaterToggle(true)...)
	out = append(out, usb.CtrlExchange{
		In: true, BRequest: 0, WValue: uint16(addr), WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{replyByte},
	})
	out = append(out, expectRepeaterToggle(false)...)
	return out
}

// I2CReadReg(addr, reg) → I2C-bridge write of reg-pointer + I2C read of 1 byte.
func expectI2CReadReg(addr, reg, replyByte byte) []usb.CtrlExchange {
	out := append([]usb.CtrlExchange{}, expectRepeaterToggle(true)...)
	out = append(out, usb.CtrlExchange{
		In: false, BRequest: 0, WValue: uint16(addr), WIndex: uint16(rtl2832u.BlockIIC)<<8 | 0x10, Data: []byte{reg},
	})
	out = append(out, expectRepeaterToggle(false)...)
	out = append(out, expectRepeaterToggle(true)...)
	out = append(out, usb.CtrlExchange{
		In: true, BRequest: 0, WValue: uint16(addr), WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{replyByte},
	})
	out = append(out, expectRepeaterToggle(false)...)
	return out
}

func TestDetect_R820T2MatchesAt0x34(t *testing.T) {
	// Detect calls SetI2CRepeater(true) once at the top (already-true
	// from the prior repeater state… wait, initial repON=false, so
	// the toggle emits). Then probes 0x34 which returns bit-reversed
	// 0x69 = 0x96. Then closes the repeater.
	m := usb.NewMockTransport()
	m.Script = append(m.Script, expectRepeaterToggle(true)...)
	// Probe R820T at 0x34. Detect's detectR82xx walks both
	// candidate addresses and the first match wins.
	m.Script = append(m.Script, usb.CtrlExchange{
		In: true, BRequest: 0, WValue: 0x0034, WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{0x96},
	})
	m.Script = append(m.Script, expectRepeaterToggle(false)...)

	demod := rtl2832u.New(m)
	tuner, err := Detect(demod)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if tuner.Type() != TypeR820T2 {
		t.Errorf("Type() = %v, want R820T2", tuner.Type())
	}
}

func TestDetect_FallsThroughToE4000(t *testing.T) {
	// Detect wraps the whole probe sweep in one SetI2CRepeater on/off
	// pair. Inside, each detect helper just emits raw I2C transfers
	// (no per-helper repeater toggles). The detection order is:
	// R820T@0x34 → R820T@0x74 → E4000@0xC8 dummy → E4000 reg 0x02.
	// First miss / miss / miss / match.
	m := usb.NewMockTransport()
	m.Script = append(m.Script, expectRepeaterToggle(true)...)
	// R820T probes: both fail (return 0x00).
	m.Script = append(m.Script, usb.CtrlExchange{In: true, BRequest: 0, WValue: 0x0034, WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{0x00}})
	m.Script = append(m.Script, usb.CtrlExchange{In: true, BRequest: 0, WValue: 0x0074, WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{0x00}})
	// E4000 dummy I2CRead (wakes the engine; first transaction NAKs
	// on some chips but our mock just returns 0x00).
	m.Script = append(m.Script, usb.CtrlExchange{In: true, BRequest: 0, WValue: 0x00C8, WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{0x00}})
	// I2CReadReg(0xC8, 0x02): write reg pointer 0x02, then read 1 byte.
	m.Script = append(m.Script, usb.CtrlExchange{In: false, BRequest: 0, WValue: 0x00C8, WIndex: uint16(rtl2832u.BlockIIC)<<8 | 0x10, Data: []byte{0x02}})
	m.Script = append(m.Script, usb.CtrlExchange{In: true, BRequest: 0, WValue: 0x00C8, WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{0x40}})
	// Final defer SetI2CRepeater(false) from Detect.
	m.Script = append(m.Script, expectRepeaterToggle(false)...)

	demod := rtl2832u.New(m)
	tuner, err := Detect(demod)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if tuner.Type() != TypeE4000 {
		t.Errorf("Type() = %v, want E4000", tuner.Type())
	}
}

func TestDetect_NoChipReturnsError(t *testing.T) {
	// All probes return 0 → orchestrator returns ErrNoTunerDetected.
	// This test only constructs enough script to drive Detect through
	// all five probe stages, so we use a permissive setup that
	// returns 0 for every I2C read.
	m := &permissiveMockTransport{returnByte: 0x00}
	demod := rtl2832u.New(m)
	_, err := Detect(demod)
	if !errors.Is(err, ErrNoTunerDetected) {
		t.Errorf("got %v, want ErrNoTunerDetected", err)
	}
}

// permissiveMockTransport ignores script matching; control reads return
// returnByte, writes succeed silently. Used by detection tests that
// only care about the "none-of-them-match" path.
type permissiveMockTransport struct {
	returnByte byte
}

func (p *permissiveMockTransport) ControlIn(_ uint8, _, _ uint16, n int, _ int) ([]byte, error) {
	out := make([]byte, n)
	for i := range out {
		out[i] = p.returnByte
	}
	return out, nil
}
func (p *permissiveMockTransport) ControlOut(_ uint8, _, _ uint16, _ []byte, _ int) error { return nil }
func (p *permissiveMockTransport) ClaimInterface(int) error                                 { return nil }
func (p *permissiveMockTransport) ReleaseInterface(int) error                               { return nil }
func (p *permissiveMockTransport) StartBulkIn(byte, int, int, func([]byte)) error           { return nil }
func (p *permissiveMockTransport) StopBulkIn() error                                        { return nil }
func (p *permissiveMockTransport) Reset() error                                             { return nil }
func (p *permissiveMockTransport) Close() error                                             { return nil }
