package tuners

import (
	"errors"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// expectI2CReadRegRaw is the wire sequence for one I2CReadReg call —
// a 1-byte register-pointer write followed by a 1-byte read, no
// surrounding repeater toggles. Use when the caller has already
// established the repeater state (Detect's outer bracket).
func expectI2CReadRegRaw(addr, reg, replyByte byte) []usb.CtrlExchange {
	return []usb.CtrlExchange{
		{In: false, BRequest: 0, WValue: uint16(addr), WIndex: uint16(rtl2832u.BlockIIC)<<8 | 0x10, Data: []byte{reg}},
		{In: true, BRequest: 0, WValue: uint16(addr), WIndex: uint16(rtl2832u.BlockIIC) << 8, N: 1, Reply: []byte{replyByte}},
	}
}

// expectSetGPIOOutput is the read-modify-write sequence Demod.SetGPIOOutput
// emits to enable the given pin as a system-block output. Reads GPD and
// GPOE (returning the supplied starting bytes), then writes back the
// pin-cleared / pin-set values. Matches gpio.go:setGPIOOutputLocked.
func expectSetGPIOOutput(pin, gpdStart, gpoeStart byte) []usb.CtrlExchange {
	const (
		sysGPOE uint16 = 0x3003
		sysGPD  uint16 = 0x3004
	)
	gpdNew := gpdStart &^ (1 << pin)
	gpoeNew := gpoeStart | (1 << pin)
	return []usb.CtrlExchange{
		{In: true, BRequest: 0, WValue: sysGPD, WIndex: 0x0200, N: 1, Reply: []byte{gpdStart}},
		{In: false, BRequest: 0, WValue: sysGPD, WIndex: 0x0210, Data: []byte{gpdNew}},
		{In: true, BRequest: 0, WValue: sysGPOE, WIndex: 0x0200, N: 1, Reply: []byte{gpoeStart}},
		{In: false, BRequest: 0, WValue: sysGPOE, WIndex: 0x0210, Data: []byte{gpoeNew}},
	}
}

// expectSetGPIOBit is the read-modify-write Demod.SetGPIOBit emits.
// Reads GPO (returning gpoStart), writes back with the pin toggled to
// `high`. Matches gpio.go:setGPIOBitLocked.
func expectSetGPIOBit(pin byte, high bool, gpoStart byte) []usb.CtrlExchange {
	const sysGPO uint16 = 0x3001
	var v byte = gpoStart
	if high {
		v |= 1 << pin
	} else {
		v &^= 1 << pin
	}
	return []usb.CtrlExchange{
		{In: true, BRequest: 0, WValue: sysGPO, WIndex: 0x0200, N: 1, Reply: []byte{gpoStart}},
		{In: false, BRequest: 0, WValue: sysGPO, WIndex: 0x0210, Data: []byte{v}},
	}
}

// expectGPIOPulse is the full SetGPIOOutput + per-level SetGPIOBit
// sequence Detect's pulseGPIO helper emits. All starting register
// reads return 0x00 (mock has no state — production reads what we
// say it reads).
func expectGPIOPulse(pin byte, levels ...bool) []usb.CtrlExchange {
	out := append([]usb.CtrlExchange{}, expectSetGPIOOutput(pin, 0x00, 0x00)...)
	for _, lvl := range levels {
		out = append(out, expectSetGPIOBit(pin, lvl, 0x00)...)
	}
	return out
}

func TestDetect_R820T2MatchesAt0x34(t *testing.T) {
	// Detect enables the I2C repeater, probes 0x34 (returns
	// bit-reversed 0x69 = 0x96), then toggles the repeater OFF on
	// return. The OFF-on-return contract is load-bearing for issue
	// #248: it leaves R82xx.Init's leading SetI2CRepeater(true) as
	// a real wire write rather than a cache-skip, which the chip's
	// I²C bridge needs to arm the next multi-byte OUT.
	m := usb.NewMockTransport()
	m.Script = append(m.Script, expectRepeaterToggle(true)...)
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
	if m.Remaining() != 0 {
		t.Errorf("script remaining = %d, want 0 (Detect must emit a trailing repeater-off on success — see issue #248)", m.Remaining())
	}
}

func TestDetect_FallsThroughToE4000(t *testing.T) {
	// Pins librtlsdr's rtlsdr_open probe order (see detect.go Detect):
	//   1. SetI2CRepeater(true)
	//   2. R820T@0x34 chip-ID read → miss
	//   3. R828D@0x74 chip-ID read → miss
	//   4. GPIO5 reset pulse: out + high + low
	//   5. FC2580@0xAC reg 0x01 read → miss
	//   6. GPIO4 output enable (no pulse — librtlsdr just enables)
	//   7. FC0013@0xC6 chip-ID read → miss
	//   8. E4000@0xC8 dummy chip-ID read (mock returns 0x00)
	//   9. E4000@0xC8 reg 0x02 read → 0x40 → match
	//  10. SetI2CRepeater(false)
	m := usb.NewMockTransport()
	m.Script = append(m.Script, expectRepeaterToggle(true)...)
	// R820T / R828D both miss.
	m.Script = append(m.Script, expectI2CReadRaw(0x34, 1, []byte{0x00}))
	m.Script = append(m.Script, expectI2CReadRaw(0x74, 1, []byte{0x00}))
	// GPIO5 pulse: high → low.
	m.Script = append(m.Script, expectGPIOPulse(5, true, false)...)
	// FC2580 miss — full read-reg sequence.
	m.Script = append(m.Script, expectI2CReadRegRaw(0xAC, 0x01, 0x00)...)
	// GPIO4 output enable (no level writes — librtlsdr only flips
	// the direction, leaves the value undriven).
	m.Script = append(m.Script, expectSetGPIOOutput(4, 0x00, 0x00)...)
	// FC0013 miss.
	m.Script = append(m.Script, expectI2CReadRaw(0xC6, 1, []byte{0x00}))
	// E4000 dummy read (NAK wakeup) — mock returns 0x00.
	m.Script = append(m.Script, expectI2CReadRaw(0xC8, 1, []byte{0x00}))
	// E4000 chip-ID match.
	m.Script = append(m.Script, expectI2CReadRegRaw(0xC8, 0x02, 0x40)...)
	m.Script = append(m.Script, expectRepeaterToggle(false)...)

	demod := rtl2832u.New(m)
	tuner, err := Detect(demod)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if tuner.Type() != TypeE4000 {
		t.Errorf("Type() = %v, want E4000", tuner.Type())
	}
	if m.Remaining() != 0 {
		t.Errorf("script remaining = %d, want 0 (Detect must emit a trailing repeater-off on success — see issue #248)", m.Remaining())
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
func (p *permissiveMockTransport) ClaimInterface(int) error                               { return nil }
func (p *permissiveMockTransport) ReleaseInterface(int) error                             { return nil }
func (p *permissiveMockTransport) StartBulkIn(byte, int, int, func([]byte)) error         { return nil }
func (p *permissiveMockTransport) StopBulkIn() error                                      { return nil }
func (p *permissiveMockTransport) Reset() error                                           { return nil }
func (p *permissiveMockTransport) Close() error                                           { return nil }
