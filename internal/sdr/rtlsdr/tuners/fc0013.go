package tuners

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
)

// Fitipower FC0013 — successor to FC0012, found on some Realtek
// reference designs and rebrands (Logilink VG0002A, Sweex). Port of
// osmocom librtlsdr's src/tuner_fc0013.c. Shares the I2C address
// (0xC6) and a similar PLL architecture with FC0012 but has a
// different chip ID and a 26-step LNA gain ladder.

const (
	fc0013I2CAddr   uint8  = 0xC6
	fc0013CheckAddr uint8  = 0x00
	fc0013CheckVal  uint8  = 0xA3
	fc0013IFFreqHz  uint32 = 6_000_000
	fc0013XtalHz    uint32 = 28_800_000
)

// fc0013InitArray is the 21-register power-on flood; index i lands
// at register address (i+1). Verbatim from osmocom's fc0013_init.
var fc0013InitArray = [21]byte{
	0x09, 0x16, 0x00, 0x00, 0x17, 0x02, 0x0A, 0xFF,
	0x6F, 0xB8, 0x82, 0xFC, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x50, 0x01,
}

// fc0013LNAGains is the 26-step manual gain ladder in tenths of dB.
// Order matches librtlsdr's fc0013_lna_gains.
var fc0013LNAGains = []int{
	-99, -73, -65, -63, -59, -50, -25, -20, -8, -1,
	7, 13, 14, 21, 22, 30, 38, 46, 54, 62, 70, 80, 90, 99,
	108, 118,
}

// fc0013GainRegs encodes the corresponding low-5-bit pattern for
// register 0x14 (LNA gain). Indices align with fc0013LNAGains.
var fc0013GainRegs = []byte{
	0x02, 0x03, 0x05, 0x07, 0x06, 0x0C, 0x0D, 0x0E,
	0x0F, 0x16, 0x17, 0x14, 0x15, 0x1C, 0x1D, 0x1E,
	0x1F, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D,
	0x3E, 0x3F,
}

// FC0013 implements [Tuner].
type FC0013 struct {
	demod    *rtl2832u.Demod
	initDone bool
	manual   bool
	bwHz     uint32
	freqHz   uint32
}

// NewFC0013 wraps the demod with an FC0013 driver.
func NewFC0013(d *rtl2832u.Demod) *FC0013 { return &FC0013{demod: d} }

func (f *FC0013) Type() Type       { return TypeFC0013 }
func (f *FC0013) IFFreqHz() uint32 { return fc0013IFFreqHz }
func (f *FC0013) Gains() []int {
	out := make([]int, len(fc0013LNAGains))
	copy(out, fc0013LNAGains)
	return out
}

func (f *FC0013) Init() error {
	if f.initDone {
		return nil
	}
	// Soft reset: bit 3 of reg 0x0C set then clear.
	if err := f.writeReg(0x0C, 0x05); err != nil {
		return fmt.Errorf("fc0013 init soft-reset on: %w", err)
	}
	if err := f.writeReg(0x0C, 0x00); err != nil {
		return fmt.Errorf("fc0013 init soft-reset off: %w", err)
	}
	for i, v := range fc0013InitArray {
		if err := f.writeReg(uint8(i+1), v); err != nil {
			return fmt.Errorf("fc0013 init reg 0x%02x: %w", i+1, err)
		}
	}
	f.initDone = true
	return nil
}

func (f *FC0013) Standby() error {
	if !f.initDone {
		return nil
	}
	if err := f.writeReg(0x06, 0x0F); err != nil {
		return fmt.Errorf("fc0013 standby: %w", err)
	}
	f.initDone = false
	return nil
}

func (f *FC0013) Close() error { return f.Standby() }

// SetFreq tunes the PLL. Shares the FC0012 band-table structure but
// with FC0013-specific reg6 bit patterns (full-bands include UHF
// L-band coverage up to 1.7 GHz). Verbatim port of
// fc0013_set_params.
func (f *FC0013) SetFreq(hz uint32) error {
	if !f.initDone {
		return errors.New("fc0013: Init not called")
	}
	if hz < 22_000_000 || hz > 1_700_000_000 {
		return &ErrUnsupportedFreq{Hz: hz, MinHz: 22_000_000, MaxHz: 1_700_000_000, TunerStr: "FC0013"}
	}
	f.freqHz = hz

	multi, reg5, reg6 := fc0013BandSelect(hz)
	fVCO := uint64(hz) * uint64(multi)
	if fVCO >= 3_000_000_000 {
		reg6 |= 0x08
	}

	xtalKHz2 := fc0013XtalHz / 2000
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
			return fmt.Errorf("fc0013 SetFreq write %d: %w", i, err)
		}
	}
	return nil
}

func (f *FC0013) SetBandwidth(hz uint32) error {
	f.bwHz = hz
	if !f.initDone || f.freqHz == 0 {
		return nil
	}
	return f.SetFreq(f.freqHz)
}

// SetGain quantizes onto the 26-step ladder and writes reg 0x14
// (low 6 bits = LNA gain, top 2 preserved from a read-modify-write).
func (f *FC0013) SetGain(tenthDB int) error {
	if !f.initDone {
		return errors.New("fc0013: Init not called")
	}
	if !f.manual || tenthDB < 0 {
		return nil
	}
	idx := nearestGainIndex(fc0013LNAGains, tenthDB)
	cur, err := f.readReg(0x14)
	if err != nil {
		return err
	}
	new := (cur & 0xC0) | fc0013GainRegs[idx]
	return f.writeReg(0x14, new)
}

// SetGainMode toggles AGC (reg 0x0D bit 3): manual=true sets the
// LNA gain mode to manual; manual=false enables the chip's AGC loop.
func (f *FC0013) SetGainMode(manual bool) error {
	if !f.initDone {
		return errors.New("fc0013: Init not called")
	}
	f.manual = manual
	cur, err := f.readReg(0x0D)
	if err != nil {
		return err
	}
	if manual {
		cur |= 0x08
	} else {
		cur &^= 0x08
	}
	return f.writeReg(0x0D, cur)
}

// ----------------------------------------------------------------------
// Internals

func fc0013BandSelect(hz uint32) (multi uint32, reg5, reg6 byte) {
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

func (f *FC0013) writeReg(addr, val byte) error {
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	return f.demod.I2CWriteReg(fc0013I2CAddr, addr, val)
}

func (f *FC0013) readReg(addr byte) (byte, error) {
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return 0, err
	}
	defer f.demod.SetI2CRepeater(false)
	return f.demod.I2CReadReg(fc0013I2CAddr, addr)
}

// detectFC0013 probes the FC0013's chip-ID byte. Shares the 0xC6
// address with FC0012; the ID byte (0xA3 vs 0xA1) is the disambiguator.
// FC0013 isn't gated by GPIO 5 the way FC0012 is, so this probe runs
// before the FC0012 one in the orchestrator.
func detectFC0013(d *rtl2832u.Demod) Tuner {
	out, err := d.I2CRead(fc0013I2CAddr, 1)
	if err != nil || len(out) == 0 {
		return nil
	}
	if out[0] != fc0013CheckVal {
		return nil
	}
	return NewFC0013(d)
}

// nearestGainIndex is shared between FC0013 / E4000 / FC2580 — picks
// the ladder entry closest in value to the target. FC0012 has its
// own copy with five very different anchor points, kept separate to
// document the "ladder has a discontinuity" quirk explicitly.
func nearestGainIndex(ladder []int, tenthDB int) int {
	best := 0
	bestDist := abs(tenthDB - ladder[0])
	for i := 1; i < len(ladder); i++ {
		d := abs(tenthDB - ladder[i])
		if d < bestDist {
			best = i
			bestDist = d
		}
	}
	return best
}
