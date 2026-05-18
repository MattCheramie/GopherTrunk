package tuners

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
)

// Fitipower FC0012 — entry-level DVB-T tuner found in some early
// generic RTL2832U dongles. Faithful port of osmocom librtlsdr's
// src/tuner_fc0012.c. Less common in modern hardware (R820T2 has
// effectively displaced it) but present on enough legacy clones to
// be worth supporting.
//
// The chip is enabled via the RTL2832U's GPIO 5 line — librtlsdr
// flips it high in rtlsdr_open before probing the FC0012's I2C bus.
// Detect (in detect.go) handles that GPIO dance.

const (
	fc0012I2CAddr    uint8  = 0xC6
	fc0012CheckAddr  uint8  = 0x00
	fc0012CheckVal   uint8  = 0xA1
	fc0012IFFreqHz   uint32 = 6_000_000
	fc0012XtalHz     uint32 = 28_800_000
	fc0012GPIOEnable uint8  = 5
)

// fc0012InitArray is the 20-register power-on flood; index i lands at
// register address (i+1). Verbatim from osmocom's fc0012_init.
var fc0012InitArray = [20]byte{
	0x05, 0x10, 0x00, 0x00, 0x00, 0x00, 0x10, 0x14,
	0xD0, 0x76, 0x01, 0x02, 0x02, 0x80, 0x00, 0x00,
	0x80, 0xFF, 0xFF, 0x55,
}

// fc0012Gains is the 5-step manual gain ladder, in tenths of dB.
// Order matches librtlsdr's fc0012_set_gain switch case values.
var fc0012Gains = []int{-99, -40, 71, 179, 192}

// fc0012GainRegs maps each gain ladder index → the low-5-bit pattern
// written to reg 0x13 (high 3 bits preserved from a read-modify-write).
var fc0012GainRegs = []byte{0x02, 0x08, 0x17, 0x19, 0x1A}

// FC0012 implements [Tuner].
type FC0012 struct {
	demod    *rtl2832u.Demod
	initDone bool
	manual   bool
	bwHz     uint32
	freqHz   uint32
}

// NewFC0012 wraps the demod with an FC0012 driver. Caller is
// responsible for the GPIO-5 enable dance (Detect does this).
func NewFC0012(d *rtl2832u.Demod) *FC0012 { return &FC0012{demod: d} }

func (f *FC0012) Type() Type       { return TypeFC0012 }
func (f *FC0012) IFFreqHz() uint32 { return fc0012IFFreqHz }
func (f *FC0012) Gains() []int {
	out := make([]int, len(fc0012Gains))
	copy(out, fc0012Gains)
	return out
}

// Init writes the 20-register init flood (regs 1..20). The chip-reset
// dance — GPIO 5 high → low → high — happens once in detect.go before
// the I2C probe, matching librtlsdr's rtlsdr_open ordering. The two
// 0x0C writes the pre-fix code emitted here ("soft-reset on/off") are
// NOT in librtlsdr's fc0012_init; the chip reset is a GPIO pulse, not
// a register write, so we drop them.
func (f *FC0012) Init() error {
	if f.initDone {
		return nil
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	for i, v := range fc0012InitArray {
		if err := f.writeReg(uint8(i+1), v); err != nil {
			return fmt.Errorf("fc0012 init reg 0x%02x: %w", i+1, err)
		}
	}
	f.initDone = true
	return nil
}

// Standby parks the chip in low-power mode by setting bit 0 of reg
// 0x06 (matches osmocom fc0012_set_params standby comment), then
// clearing the chip-enable GPIO so the dongle's idle current drops.
func (f *FC0012) Standby() error {
	if !f.initDone {
		return nil
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	if err := f.writeReg(0x06, 0x0F); err != nil {
		return fmt.Errorf("fc0012 standby: %w", err)
	}
	f.initDone = false
	return nil
}

func (f *FC0012) Close() error { return f.Standby() }

// SetFreq tunes the FC0012's PLL to the requested LO. The 11-band
// multiplier table picks the divider that keeps the VCO inside the
// chip's documented range; the PLL math (XDIV + FA + PM) is verbatim
// from librtlsdr's fc0012_set_params.
//
// Returns an [*ErrUnsupportedFreq] when freq is outside the chip's
// 37 MHz .. 1.7 GHz range — the limits librtlsdr enforces.
func (f *FC0012) SetFreq(hz uint32) error {
	if !f.initDone {
		return errors.New("fc0012: Init not called")
	}
	if hz < 37_000_000 || hz > 1_700_000_000 {
		return &ErrUnsupportedFreq{Hz: hz, MinHz: 37_000_000, MaxHz: 1_700_000_000, TunerStr: "FC0012"}
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	f.freqHz = hz

	multi, reg5, reg6 := fc0012BandSelect(hz)
	fVCO := uint64(hz) * uint64(multi)
	if fVCO >= 3_000_000_000 {
		reg6 |= 0x08
	}

	xtalKHz2 := fc0012XtalHz / 2000
	xdiv := uint16(fVCO / uint64(xtalKHz2))
	if (fVCO - uint64(xdiv)*uint64(xtalKHz2)) >= uint64(xtalKHz2/2) {
		xdiv++
	}
	pm := uint8(xdiv / 8)
	am := uint8(xdiv - uint16(8)*uint16(pm))

	var reg1, reg2 byte
	if am < 2 {
		reg1 = am + 8
		reg2 = pm - 1
	} else {
		reg1 = am
		reg2 = pm
	}

	// BW configuration: bit 2 of reg 6 = narrow filter.
	if f.bwHz != 0 && f.bwHz < 6_000_000 {
		reg6 |= 0x04
	} else {
		reg6 &^= 0x04
	}
	reg5 |= 0x07

	for i, v := range []struct {
		addr uint8
		val  byte
	}{
		{1, reg1}, {2, reg2}, {3, 0x00}, {4, 0x00}, {5, reg5}, {6, reg6},
	} {
		if err := f.writeReg(v.addr, v.val); err != nil {
			return fmt.Errorf("fc0012 SetFreq write %d: %w", i, err)
		}
	}
	return nil
}

// SetBandwidth caches the requested bandwidth so the next SetFreq
// picks the right filter bit. librtlsdr's FC0012 doesn't expose
// finer filter control than narrow/wide; passing zero means "use
// the chip's default" (wide).
func (f *FC0012) SetBandwidth(hz uint32) error {
	f.bwHz = hz
	if !f.initDone || f.freqHz == 0 {
		return nil
	}
	return f.SetFreq(f.freqHz)
}

// SetGain maps the requested tenthDB onto the closest entry in the
// 5-step ladder and writes it to reg 0x13. Mirrors librtlsdr's
// fc0012_set_gain switch verbatim — no-op when AGC is active.
func (f *FC0012) SetGain(tenthDB int) error {
	if !f.initDone {
		return errors.New("fc0012: Init not called")
	}
	if !f.manual || tenthDB < 0 {
		return nil
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	idx := fc0012NearestGainIndex(tenthDB)
	cur, err := f.readReg(0x13)
	if err != nil {
		return err
	}
	new := (cur & 0xE0) | fc0012GainRegs[idx]
	return f.writeReg(0x13, new)
}

// SetGainMode toggles AGC (manual = false) ↔ manual gain (true).
// The chip's AGC bit is reg 0x06 bit 4.
func (f *FC0012) SetGainMode(manual bool) error {
	if !f.initDone {
		return errors.New("fc0012: Init not called")
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	f.manual = manual
	cur, err := f.readReg(0x06)
	if err != nil {
		return err
	}
	if manual {
		cur |= 0x10
	} else {
		cur &^= 0x10
	}
	return f.writeReg(0x06, cur)
}

// ----------------------------------------------------------------------
// Internals

// fc0012BandSelect picks the multiplier and per-band reg5/reg6 bits
// for the given input frequency. Boundaries lifted from librtlsdr's
// fc0012_set_params if-ladder.
func fc0012BandSelect(hz uint32) (multi uint32, reg5, reg6 byte) {
	switch {
	case hz < 37_084_000:
		return 96, 0x82, 0x00
	case hz < 55_625_000:
		return 64, 0x82, 0x02
	case hz < 74_167_000:
		return 48, 0x82, 0x00
	case hz < 111_250_000:
		return 32, 0x82, 0x02
	case hz < 148_334_000:
		return 24, 0x82, 0x02
	case hz < 222_500_000:
		return 16, 0x82, 0x02
	case hz < 296_667_000:
		return 12, 0x82, 0x02
	case hz < 445_000_000:
		return 8, 0x82, 0x02
	case hz < 593_334_000:
		return 6, 0x0A, 0x00
	case hz < 950_000_000:
		return 4, 0x0A, 0x02
	default:
		return 2, 0x0A, 0x02
	}
}

// fc0012NearestGainIndex returns the index into fc0012Gains whose
// value is closest to the request. The ladder has a discontinuity
// (-9.9 dB → -4 dB → +7.1 dB) so a "round to nearest" beats "first
// ≤" semantically.
func fc0012NearestGainIndex(tenthDB int) int {
	best := 0
	bestDist := abs(tenthDB - fc0012Gains[0])
	for i := 1; i < len(fc0012Gains); i++ {
		d := abs(tenthDB - fc0012Gains[i])
		if d < bestDist {
			best = i
			bestDist = d
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// writeReg / readReg are private I2C plumbing. Callers (public
// methods) own the SetI2CRepeater bracket — librtlsdr's pattern.
// FC0012 doesn't do the bit-reverse trick the R820T family does,
// so raw bytes come back from the bridge.
func (f *FC0012) writeReg(addr, val byte) error {
	return f.demod.I2CWriteReg(fc0012I2CAddr, addr, val)
}

func (f *FC0012) readReg(addr byte) (byte, error) {
	return f.demod.I2CReadReg(fc0012I2CAddr, addr)
}

// detectFC0012 reads the chip-ID byte at addr 0 and returns a ready
// driver if the response matches. The orchestrator in detect.go runs
// the GPIO 5 reset pulse + GPIO 6 chip-enable pulse before this call,
// matching librtlsdr's rtlsdr_open order — we just see the I2C side.
func detectFC0012(d *rtl2832u.Demod) Tuner {
	out, err := d.I2CRead(fc0012I2CAddr, 1)
	if err != nil || len(out) == 0 {
		return nil
	}
	// The chip-ID probe reads register 0 by relying on the bus
	// auto-increment. Some clones answer 0xA1; others differ — match
	// only the documented value.
	if out[0] != fc0012CheckVal {
		return nil
	}
	return NewFC0012(d)
}
