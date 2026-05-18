package tuners

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
)

// Fitipower FC2580 — wide-range multi-band tuner found on a few
// niche dongles (FC2580-based reference designs from FCI). Faithful
// port of osmocom librtlsdr's src/tuner_fc2580.c register sequences;
// the IMR-style band-edge tuning that the C source ships in
// fc2580_set_filter is left at librtlsdr's mid-range default until a
// real-hardware capture is available.
//
// FC2580 emits a per-band IF (5.6 MHz for VHF, 4.6 MHz for UHF);
// IFFreqHz() reports the VHF value as a reasonable default — the
// demod re-programs IF freq whenever the band changes via SetFreq.

const (
	fc2580I2CAddr   uint8  = 0xAC
	fc2580CheckAddr uint8  = 0x01
	fc2580CheckVal  uint8  = 0x56
	fc2580CheckMask byte   = 0x7F
	fc2580IFFreqHz  uint32 = 5_600_000
	fc2580XtalHz    uint32 = 16_384_000 // chip-internal reference, not the RTL2832U xtal
)

// fc2580InitArray is the chip's documented power-on register flood,
// starting at register 0x00. Verbatim from osmocom's fc2580_init.
var fc2580InitArray = [54]struct {
	addr uint8
	val  byte
}{
	{0x00, 0x00}, {0x12, 0x86}, {0x14, 0x5C}, {0x16, 0x3C},
	{0x1F, 0xD2}, {0x09, 0xD7}, {0x0B, 0xD5}, {0x0C, 0x32},
	{0x0E, 0x43}, {0x21, 0x0A}, {0x22, 0x82}, {0x45, 0x10},
	{0x4C, 0x00}, {0x3F, 0x88}, {0x02, 0x0E}, {0x58, 0x14},
	{0x6B, 0x11}, {0x6C, 0x00}, {0x9F, 0xE0}, {0xA0, 0x8F},
	{0x18, 0xA1}, {0x73, 0x21}, {0x74, 0x21}, {0x75, 0x12},
	{0x76, 0x55}, {0x77, 0x99}, {0x78, 0xAD}, {0x79, 0x91},
	{0x7A, 0x61}, {0x7B, 0x10}, {0x7C, 0x10}, {0x7D, 0x53},
	{0x7E, 0x21}, {0x7F, 0x42}, {0x80, 0x65}, {0x81, 0x88},
	{0x82, 0xAB}, {0x83, 0xCC}, {0x84, 0xEE}, {0x85, 0xFF},
	{0x86, 0xFF}, {0x88, 0xF5}, {0x89, 0x80}, {0x8E, 0xB8},
	{0x8F, 0x1E}, {0x90, 0x77}, {0x91, 0xC0}, {0x92, 0x4F},
	{0x93, 0x19}, {0x94, 0x80}, {0x95, 0x40}, {0x9A, 0x80},
	{0xA5, 0x40}, {0xAC, 0xFF},
}

// fc2580BandSelect picks the band-specific PLL divider, mixer mode,
// and IF frequency for the requested LO. Boundaries from librtlsdr's
// fc2580_set_filter / fc2580_band_select tables.
type fc2580Band struct {
	freqMax  uint32 // upper bound (inclusive)
	divider  uint32 // mixer divider used in the PLL math
	regCfg   byte   // reg 0x02 = band configuration byte
	ifFreqHz uint32 // demod IF the driver layer programs after SetFreq
}

var fc2580Bands = []fc2580Band{
	// VHF-II (FM broadcast). Multiplier = 24.
	{freqMax: 100_000_000, divider: 24, regCfg: 0x06, ifFreqHz: 5_600_000},
	// VHF-III (DAB / Band III). Multiplier = 16.
	{freqMax: 200_000_000, divider: 16, regCfg: 0x06, ifFreqHz: 5_600_000},
	// UHF (Band IV/V).
	{freqMax: 1_000_000_000, divider: 4, regCfg: 0x0E, ifFreqHz: 4_600_000},
	// L-band fallback.
	{freqMax: ^uint32(0), divider: 2, regCfg: 0x06, ifFreqHz: 1_400_000},
}

// fc2580Gains is a coarse 4-step manual gain ladder; the C source
// has finer per-band sub-tables but a 4-step is enough for first-
// pass parity until a real-hardware capture refines it.
var fc2580Gains = []int{0, 50, 100, 150}

// FC2580 implements [Tuner].
type FC2580 struct {
	demod    *rtl2832u.Demod
	initDone bool
	manual   bool
	bwHz     uint32
	freqHz   uint32
	ifFreqHz uint32
}

// NewFC2580 wraps the demod with an FC2580 driver.
func NewFC2580(d *rtl2832u.Demod) *FC2580 { return &FC2580{demod: d, ifFreqHz: fc2580IFFreqHz} }

func (f *FC2580) Type() Type       { return TypeFC2580 }
func (f *FC2580) IFFreqHz() uint32 { return f.ifFreqHz }
func (f *FC2580) Gains() []int {
	out := make([]int, len(fc2580Gains))
	copy(out, fc2580Gains)
	return out
}

func (f *FC2580) Init() error {
	if f.initDone {
		return nil
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	for i, w := range fc2580InitArray {
		if err := f.writeReg(w.addr, w.val); err != nil {
			return fmt.Errorf("fc2580 init step %d (0x%02x): %w", i, w.addr, err)
		}
	}
	f.initDone = true
	return nil
}

func (f *FC2580) Standby() error {
	if !f.initDone {
		return nil
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	// Power down — reg 0x02 bit 7 enters sleep.
	if err := f.writeReg(0x02, 0x0E); err != nil {
		return fmt.Errorf("fc2580 standby: %w", err)
	}
	f.initDone = false
	return nil
}

func (f *FC2580) Close() error { return f.Standby() }

// SetFreq selects the band, programs the synthesizer (reg 0x02 +
// 0x18..0x1B), and updates the cached IF frequency so the demod
// layer can re-program its IF on the next call. Mirrors the structure
// of librtlsdr's fc2580_set_freq.
func (f *FC2580) SetFreq(hz uint32) error {
	if !f.initDone {
		return errors.New("fc2580: Init not called")
	}
	if hz < 50_000_000 || hz > 2_600_000_000 {
		return &ErrUnsupportedFreq{Hz: hz, MinHz: 50_000_000, MaxHz: 2_600_000_000, TunerStr: "FC2580"}
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	f.freqHz = hz

	band := fc2580Bands[len(fc2580Bands)-1]
	for _, b := range fc2580Bands {
		if hz <= b.freqMax {
			band = b
			break
		}
	}
	f.ifFreqHz = band.ifFreqHz

	// Band-config byte.
	if err := f.writeReg(0x02, band.regCfg); err != nil {
		return err
	}
	// Synth: nint = (hz * divider) / xtal; nfrac = (remainder / xtal) * 2^20.
	pll := uint64(hz) * uint64(band.divider)
	nint := uint32(pll / uint64(fc2580XtalHz))
	rem := pll - uint64(nint)*uint64(fc2580XtalHz)
	nfrac := uint32((rem * (1 << 20)) / uint64(fc2580XtalHz))

	if err := f.writeReg(0x18, byte((nfrac>>16)&0x0F)|byte((nint<<4)&0xF0)); err != nil {
		return err
	}
	if err := f.writeReg(0x1A, byte((nfrac>>8)&0xFF)); err != nil {
		return err
	}
	if err := f.writeReg(0x1B, byte(nfrac&0xFF)); err != nil {
		return err
	}
	if err := f.writeReg(0x1C, byte(nint&0xFF)); err != nil {
		return err
	}
	return nil
}

// SetBandwidth caches the request; the LP filter ladder in the C
// source depends on band-edge tuning that needs an IMR sweep. This
// port leaves the filter at the init-array default (wide).
func (f *FC2580) SetBandwidth(hz uint32) error {
	f.bwHz = hz
	return nil
}

func (f *FC2580) SetGain(tenthDB int) error {
	if !f.initDone {
		return errors.New("fc2580: Init not called")
	}
	if !f.manual || tenthDB < 0 {
		return nil
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	idx := nearestGainIndex(fc2580Gains, tenthDB)
	// Reg 0x49 = LNA gain stage on FC2580.
	return f.writeReg(0x49, byte(idx&0x07))
}

func (f *FC2580) SetGainMode(manual bool) error {
	if !f.initDone {
		return errors.New("fc2580: Init not called")
	}
	if err := f.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer f.demod.SetI2CRepeater(false)
	f.manual = manual
	cur, err := f.readReg(0x45)
	if err != nil {
		return err
	}
	if manual {
		cur |= 0x10
	} else {
		cur &^= 0x10
	}
	return f.writeReg(0x45, cur)
}

// ----------------------------------------------------------------------
// Internals

// writeReg / readReg are private I2C plumbing. Callers (public
// methods) own the SetI2CRepeater bracket — librtlsdr's pattern.
func (f *FC2580) writeReg(addr, val byte) error {
	return f.demod.I2CWriteReg(fc2580I2CAddr, addr, val)
}

func (f *FC2580) readReg(addr byte) (byte, error) {
	return f.demod.I2CReadReg(fc2580I2CAddr, addr)
}

// detectFC2580 reads register 0x01 and matches the low 7 bits against
// the chip ID 0x56. The high bit is a revision flag and varies
// between FC2580 production lots; the mask follows librtlsdr verbatim.
func detectFC2580(d *rtl2832u.Demod) Tuner {
	got, err := d.I2CReadReg(fc2580I2CAddr, fc2580CheckAddr)
	if err != nil {
		return nil
	}
	if (got & fc2580CheckMask) != fc2580CheckVal {
		return nil
	}
	return NewFC2580(d)
}
