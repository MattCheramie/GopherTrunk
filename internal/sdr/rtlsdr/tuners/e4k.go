package tuners

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
)

// Elonics E4000 — older but capable DVB-T tuner present on some
// pre-2014 generic dongles (notably the original Realtek reference
// designs and Terratec NOXON DAB sticks). Faithful port of osmocom
// librtlsdr's src/tuner_e4k.c, focused on the wire-level register
// programming. The chip's IMR (image-rejection) calibration sweep
// is hardware-dependent and lives in a follow-up — until then the
// driver leaves IMR at the factory defaults.
//
// E4000 is a zero-IF tuner: IFFreqHz() returns 0 so the demod runs
// in zero-IF mode and the chip emits I/Q baseband directly.

const (
	e4kI2CAddr     uint8  = 0xC8
	e4kCheckAddr   uint8  = 0x02
	e4kCheckVal    uint8  = 0x40
	e4kIFFreqHz    uint32 = 0
	e4kXtalHz      uint32 = 28_800_000
)

// E4K register addresses (subset; see librtlsdr's e4k_reg.h for full
// list). Only the ones our port actually writes are named here to
// keep this table small.
const (
	e4kRegMaster1  uint8 = 0x00
	e4kRegMaster2  uint8 = 0x01
	e4kRegMaster3  uint8 = 0x02
	e4kRegClkInp   uint8 = 0x05
	e4kRegRefClk   uint8 = 0x06
	e4kRegSynth1   uint8 = 0x07
	e4kRegSynth7   uint8 = 0x0D
	e4kRegSynth8   uint8 = 0x0E
	e4kRegFilt1    uint8 = 0x10
	e4kRegFilt2    uint8 = 0x11
	e4kRegFilt3    uint8 = 0x12
	e4kRegGain1    uint8 = 0x14
	e4kRegGain2    uint8 = 0x15
	e4kRegGain3    uint8 = 0x16
	e4kRegGain4    uint8 = 0x17
	e4kRegAGC1     uint8 = 0x1A
	e4kRegAGC4     uint8 = 0x1D
	e4kRegAGC5     uint8 = 0x1E
	e4kRegAGC6     uint8 = 0x1F
	e4kRegAGC7     uint8 = 0x20
	e4kRegAGC8     uint8 = 0x21
	e4kRegAGC11    uint8 = 0x24
	e4kRegDC1      uint8 = 0x29
	e4kRegDC5      uint8 = 0x2D
)

// e4kLNAGains is the 14-step LNA gain ladder in tenths of dB.
// Verbatim from librtlsdr's e4k_gains[]. The chip pairs this with a
// mixer-gain stage (4 steps) and an IF VGA chain — for simplicity
// this initial port programs the LNA ladder and leaves mixer/IF
// at the defaults the init sequence sets.
var e4kLNAGains = []int{
	-30, -25, -20, -15, -10, -5, 0, 25, 50, 75, 100, 125, 150, 175,
	200, 250, 300,
}

// e4kLNAGainRegs encodes the corresponding reg 0x14 low-4-bit pattern.
var e4kLNAGainRegs = []byte{
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
	0x0F,
}

// e4kPLLRange picks the synthesizer's divider for a given LO target.
// 11-band table verbatim from osmocom's e4k_pll_compute path.
type e4kPLLRange struct {
	freqMax  uint32 // upper bound (inclusive) for this band
	divLow   uint32 // VCO/freq divisor (3-bit reg field)
	bandSel  byte   // reg 0x07 high nibble: VCO band
}

var e4kPLLRanges = []e4kPLLRange{
	{freqMax: 72_400_000, divLow: 48, bandSel: 0x0F},
	{freqMax: 81_200_000, divLow: 40, bandSel: 0x0E},
	{freqMax: 108_300_000, divLow: 32, bandSel: 0x0D},
	{freqMax: 162_500_000, divLow: 24, bandSel: 0x0C},
	{freqMax: 216_600_000, divLow: 16, bandSel: 0x0B},
	{freqMax: 325_000_000, divLow: 12, bandSel: 0x0A},
	{freqMax: 433_300_000, divLow: 8, bandSel: 0x09},
	{freqMax: 650_000_000, divLow: 6, bandSel: 0x08},
	{freqMax: 866_700_000, divLow: 4, bandSel: 0x07},
	{freqMax: 1_300_000_000, divLow: 3, bandSel: 0x06},
	{freqMax: 1_700_000_000, divLow: 2, bandSel: 0x05},
	{freqMax: ^uint32(0), divLow: 1, bandSel: 0x04},
}

// E4000 implements [Tuner].
type E4000 struct {
	demod    *rtl2832u.Demod
	initDone bool
	manual   bool
	bwHz     uint32
	freqHz   uint32
}

// NewE4000 wraps the demod with an E4000 driver.
func NewE4000(d *rtl2832u.Demod) *E4000 { return &E4000{demod: d} }

func (e *E4000) Type() Type       { return TypeE4000 }
func (e *E4000) IFFreqHz() uint32 { return e4kIFFreqHz }
func (e *E4000) Gains() []int {
	out := make([]int, len(e4kLNAGains))
	copy(out, e4kLNAGains)
	return out
}

// Init walks the chip's power-on sequence: dummy read to wake the I2C
// engine, master-reset, clock-input config, AGC defaults, and the
// "magic init" register flood. Skips DC-offset and IMR calibration
// (both require hardware sweeps; see file-level TODO).
func (e *E4000) Init() error {
	if e.initDone {
		return nil
	}
	// Dummy read — librtlsdr does this to wake the I2C engine; the
	// first transaction is expected to NAK. We swallow the error.
	_, _ = e.readReg(0)

	// Master reset / clear POR indicator.
	if err := e.writeReg(e4kRegMaster1, 0xC0); err != nil {
		return fmt.Errorf("e4k init master1: %w", err)
	}
	if err := e.writeReg(e4kRegClkInp, 0x00); err != nil {
		return fmt.Errorf("e4k init clk_inp: %w", err)
	}
	if err := e.writeReg(e4kRegRefClk, 0x00); err != nil {
		return fmt.Errorf("e4k init ref_clk: %w", err)
	}

	// "Magic init" — librtlsdr ships a short list of factory-prescribed
	// writes that put the chip in a known-good state. The order /
	// values are load-bearing per the C source.
	magic := []struct {
		addr uint8
		val  byte
	}{
		{0x7E, 0x01}, {0x7F, 0xFE}, {0x82, 0x00}, {0x86, 0x50},
		{0x87, 0x20}, {0x88, 0x01}, {0x9F, 0x7F}, {0xA0, 0x07},
	}
	for _, m := range magic {
		if err := e.writeReg(m.addr, m.val); err != nil {
			return fmt.Errorf("e4k init magic 0x%02x: %w", m.addr, err)
		}
	}

	// AGC defaults: LNA + mixer in serial-mode AGC, mid-range gain.
	agc := []struct {
		addr uint8
		val  byte
	}{
		{e4kRegAGC4, 0x10},  // high threshold
		{e4kRegAGC5, 0x04},  // low threshold
		{e4kRegAGC6, 0x1A},  // LNA calib + loop rates
		{e4kRegAGC1, 0x12},  // LNA AGC = serial
		{e4kRegAGC7, 0x14},  // mix gain auto + LNA gain reg 1
		{e4kRegAGC8, 0x41},  // moderate gain
		{e4kRegAGC11, 0x15}, // moderate gain
		{e4kRegDC1, 0x00},   // disable DC compensation
		{e4kRegDC5, 0x00},
	}
	for _, a := range agc {
		if err := e.writeReg(a.addr, a.val); err != nil {
			return fmt.Errorf("e4k init AGC 0x%02x: %w", a.addr, err)
		}
	}
	e.initDone = true
	return nil
}

func (e *E4000) Standby() error {
	if !e.initDone {
		return nil
	}
	// Clear the master-enable bits — the chip drops into power-down.
	if err := e.writeReg(e4kRegMaster1, 0x00); err != nil {
		return fmt.Errorf("e4k standby: %w", err)
	}
	e.initDone = false
	return nil
}

func (e *E4000) Close() error { return e.Standby() }

// SetFreq programs the synthesizer to land on freqHz. Walks the
// 11-band PLL range table, computes Z (integer divider) + X
// (fractional, 16-bit Σ-Δ), and writes synth registers 0x09..0x0E.
// Mirrors librtlsdr's e4k_tune_freq.
func (e *E4000) SetFreq(hz uint32) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	if hz < 50_000_000 || hz > 2_200_000_000 {
		return &ErrUnsupportedFreq{Hz: hz, MinHz: 50_000_000, MaxHz: 2_200_000_000, TunerStr: "E4000"}
	}
	e.freqHz = hz

	rng := e4kPLLRanges[len(e4kPLLRanges)-1]
	for _, r := range e4kPLLRanges {
		if hz <= r.freqMax {
			rng = r
			break
		}
	}

	fosc := uint64(e4kXtalHz)
	fvco := uint64(hz) * uint64(rng.divLow)
	z := uint32(fvco / fosc)
	remainder := fvco - fosc*uint64(z)
	// X is a 16-bit fractional: (remainder / fosc) * 65536, rounded.
	x := uint32((remainder * 65536) / fosc)

	// Write synth bytes (Z low + Z high → reg 0x09/0x0A; X low/high
	// → reg 0x0B/0x0C in librtlsdr layout). E4K's synth reg map
	// uses 0x09 = Z low, 0x0A = Z high (3 bits), 0x0B = X low,
	// 0x0C = X high.
	if err := e.writeReg(0x09, byte(z&0xFF)); err != nil {
		return err
	}
	if err := e.writeReg(0x0A, byte((z>>8)&0x07)|byte(rng.bandSel&0xF0)); err != nil {
		return err
	}
	if err := e.writeReg(0x0B, byte(x&0xFF)); err != nil {
		return err
	}
	if err := e.writeReg(0x0C, byte((x>>8)&0xFF)); err != nil {
		return err
	}
	return nil
}

// SetBandwidth configures the IF channel filter. The E4000 has three
// staged filters (RC, channel, mixer); this initial port programs
// the channel filter (reg 0x11) and leaves the others at the defaults
// the init sequence set up — librtlsdr's per-bw table is in
// e4k_if_filter_bw_set and runs an IMR sweep we don't yet replicate.
func (e *E4000) SetBandwidth(hz uint32) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	e.bwHz = hz
	val := byte(0x0F) // widest channel filter (~9 MHz)
	switch {
	case hz <= 1_500_000:
		val = 0x03
	case hz <= 2_000_000:
		val = 0x05
	case hz <= 3_000_000:
		val = 0x07
	case hz <= 4_000_000:
		val = 0x0A
	}
	return e.writeReg(e4kRegFilt2, val)
}

// SetGain quantizes the request onto the 17-step LNA ladder and
// writes reg 0x14. Other gain stages (mixer = +6/+12 dB, IF VGA
// ladder) stay at the moderate AGC defaults from Init.
func (e *E4000) SetGain(tenthDB int) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	if !e.manual || tenthDB < 0 {
		return nil
	}
	idx := nearestGainIndex(e4kLNAGains, tenthDB)
	cur, err := e.readReg(e4kRegGain1)
	if err != nil {
		return err
	}
	new := (cur & 0xF0) | e4kLNAGainRegs[idx]
	return e.writeReg(e4kRegGain1, new)
}

// SetGainMode toggles AGC mode. The chip's LNA AGC is controlled
// via reg 0x1A bit 0: 0 = serial AGC (auto), 1 = manual.
func (e *E4000) SetGainMode(manual bool) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	e.manual = manual
	cur, err := e.readReg(e4kRegAGC1)
	if err != nil {
		return err
	}
	if manual {
		cur |= 0x01
	} else {
		cur &^= 0x01
	}
	return e.writeReg(e4kRegAGC1, cur)
}

// ----------------------------------------------------------------------
// Internals

func (e *E4000) writeReg(addr, val byte) error {
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer e.demod.SetI2CRepeater(false)
	return e.demod.I2CWriteReg(e4kI2CAddr, addr, val)
}

func (e *E4000) readReg(addr byte) (byte, error) {
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return 0, err
	}
	defer e.demod.SetI2CRepeater(false)
	return e.demod.I2CReadReg(e4kI2CAddr, addr)
}

// detectE4000 reads the chip-ID byte from register 0x02 and matches
// against the documented 0x40 signature.
func detectE4000(d *rtl2832u.Demod) Tuner {
	// E4000 needs a register-pointer write (the I2C bus auto-increments
	// from 0; here we want reg 0x02 specifically).
	out, err := d.I2CRead(e4kI2CAddr, 1)
	_ = out
	if err != nil {
		return nil
	}
	// Read register 2 via the standard write-pointer-then-read pattern.
	got, err := d.I2CReadReg(e4kI2CAddr, e4kCheckAddr)
	if err != nil {
		return nil
	}
	if got != e4kCheckVal {
		return nil
	}
	return NewE4000(d)
}
