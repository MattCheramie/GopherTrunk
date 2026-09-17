package tuners

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
)

// Elonics E4000 — older but capable zero-IF tuner present on some
// pre-2014 generic dongles and on Nooelec's "XTR" range. Faithful
// port of osmocom / librtlsdr's src/tuner_e4k.c, cross-checked
// register-for-register against that source (the wire-level writes,
// the PLL Σ-Δ math, the per-band RF tracking filter, the IF gain /
// filter stages and the LNA + mixer gain distribution from
// librtlsdr.c's e4000_set_gain). The chip's IMR (image-rejection)
// and DC-offset calibration sweeps are hardware-dependent and left
// at the factory defaults — as librtlsdr itself ships them (#if 0).
//
// E4000 is a zero-IF tuner: IFFreqHz() returns 0 so the demod runs
// in zero-IF mode and the chip emits I/Q baseband directly.

const (
	e4kI2CAddr   uint8  = 0xC8
	e4kCheckAddr uint8  = 0x02
	e4kCheckVal  uint8  = 0x40
	e4kIFFreqHz  uint32 = 0
	e4kXtalHz    uint32 = 28_800_000
)

// E4K register addresses (subset actually written; see librtlsdr's
// tuner_e4k.h for the full map). Values verbatim from that header.
const (
	e4kRegMaster1    uint8 = 0x00
	e4kRegClkInp     uint8 = 0x05
	e4kRegRefClk     uint8 = 0x06
	e4kRegSynth1     uint8 = 0x07
	e4kRegSynth3     uint8 = 0x09
	e4kRegSynth4     uint8 = 0x0A
	e4kRegSynth5     uint8 = 0x0B
	e4kRegSynth7     uint8 = 0x0D
	e4kRegFilt1      uint8 = 0x10
	e4kRegFilt2      uint8 = 0x11
	e4kRegFilt3      uint8 = 0x12
	e4kRegGain1      uint8 = 0x14
	e4kRegGain2      uint8 = 0x15
	e4kRegGain3      uint8 = 0x16
	e4kRegGain4      uint8 = 0x17
	e4kRegAGC1       uint8 = 0x1A
	e4kRegAGC4       uint8 = 0x1D
	e4kRegAGC5       uint8 = 0x1E
	e4kRegAGC6       uint8 = 0x1F
	e4kRegAGC7       uint8 = 0x20
	e4kRegAGC11      uint8 = 0x24
	e4kRegDC5        uint8 = 0x2D
	e4kRegDCTime1    uint8 = 0x70
	e4kRegDCTime2    uint8 = 0x71
	e4kRegBias       uint8 = 0x78
	e4kRegClkoutPwdn uint8 = 0x7A
)

// MASTER1 bit fields (tuner_e4k.h). The power-on write asserts a soft
// reset, keeps the chip out of standby, and clears the POR indicator.
const (
	e4kMaster1Reset   byte = 1 << 0
	e4kMaster1NormStby byte = 1 << 1
	e4kMaster1PorDet  byte = 1 << 2
	e4kMaster1Init    byte = e4kMaster1Reset | e4kMaster1NormStby | e4kMaster1PorDet // 0x07
)

// AGC mode nibbles (E4K_AGC_MOD_*) and control-bit masks.
const (
	e4kAGC1ModMask         byte = 0x0F
	e4kAGCModSerial        byte = 0x0 // manual LNA (serial-programmed) gain
	e4kAGCModIFSerialLNAAuto byte = 0x9 // auto LNA (E4K_AGC_MOD_IF_SERIAL_LNA_AUTON)
	e4kAGC7MixGainAuto     byte = 1 << 0
	e4kFilt3Disable        byte = 1 << 5
	e4kClkoutPwdnDisable   byte = 0x96 // value librtlsdr writes to power down clock-out
)

// width2mask[w] is a w-bit-wide mask, verbatim from tuner_e4k.c. Used
// to build the read-modify-write masks for the sub-byte register fields.
var e4kWidth2Mask = []byte{0, 1, 3, 7, 0xF, 0x1F, 0x3F, 0x7F, 0xFF}

// e4kRegField names a sub-byte field: register, low bit, and width.
// Mirrors librtlsdr's struct reg_field for the IF gain / filter stages.
type e4kRegField struct {
	reg   uint8
	shift uint8
	width uint8
}

// e4kUserGains is the combined gain ladder librtlsdr exposes to callers
// (rtlsdr.c e4k_gains[]), in tenths of dB. Each value is realised by
// e4000_set_gain's LNA+mixer split (see SetGain). This is what Gains()
// returns and what the managed-gain controller quantizes against.
var e4kUserGains = []int{-10, 15, 40, 65, 90, 115, 140, 165, 190, 215, 240, 290, 340, 420}

// e4kLNAGains maps an LNA gain (tenths of dB) to the GAIN1 low-nibble
// register value. Verbatim from librtlsdr's tuner_e4k.c lnagain[] — note
// the register codes are NOT contiguous (they skip 2 and 3), which is
// exactly why a linear 0x00..0x0F table (the pre-2026 port) programmed
// the wrong physical gain for most requests.
var e4kLNAGains = []struct {
	tenthDB int
	reg     byte
}{
	{-50, 0}, {-25, 1}, {0, 4}, {25, 5}, {50, 6}, {75, 7}, {100, 8},
	{125, 9}, {150, 10}, {175, 11}, {200, 12}, {250, 13}, {300, 14},
}

// IF VGA gain stages 1..6 (index 0 unused). Values are gains in dB;
// the slice index is what gets written into the stage's register field.
// Tables + field placements verbatim from tuner_e4k.c.
var e4kIFStageGains = [][]int{
	nil,
	{-3, 6},                       // stage 1
	{0, 3, 6, 9},                  // stage 2
	{0, 3, 6, 9},                  // stage 3
	{0, 1, 2, 2},                  // stage 4
	{3, 6, 9, 12, 15, 15, 15, 15}, // stage 5
	{3, 6, 9, 12, 15, 15, 15, 15}, // stage 6
}

var e4kIFStageFields = []e4kRegField{
	{0, 0, 0},
	{e4kRegGain3, 0, 1},
	{e4kRegGain3, 1, 2},
	{e4kRegGain3, 3, 2},
	{e4kRegGain3, 5, 2},
	{e4kRegGain4, 0, 3},
	{e4kRegGain4, 3, 3},
}

// IF filter families and their bandwidth tables (Hz) + register fields,
// verbatim from tuner_e4k.c (mix_filter_bw / ifch_filter_bw / ifrc_filter_bw).
const (
	e4kIFFilterMix  = 0
	e4kIFFilterChan = 1
	e4kIFFilterRC   = 2
)

var e4kMixFilterBw = []uint32{
	27_000_000, 27_000_000, 27_000_000, 27_000_000,
	27_000_000, 27_000_000, 27_000_000, 27_000_000,
	4_600_000, 4_200_000, 3_800_000, 3_400_000,
	3_300_000, 2_700_000, 2_300_000, 1_900_000,
}

var e4kIFRCFilterBw = []uint32{
	21_400_000, 21_000_000, 17_600_000, 14_700_000,
	12_400_000, 10_600_000, 9_000_000, 7_700_000,
	6_400_000, 5_300_000, 4_400_000, 3_400_000,
	2_600_000, 1_800_000, 1_200_000, 1_000_000,
}

var e4kIFChanFilterBw = []uint32{
	5_500_000, 5_300_000, 5_000_000, 4_800_000,
	4_600_000, 4_400_000, 4_300_000, 4_100_000,
	3_900_000, 3_800_000, 3_700_000, 3_600_000,
	3_400_000, 3_300_000, 3_200_000, 3_100_000,
	3_000_000, 2_950_000, 2_900_000, 2_800_000,
	2_750_000, 2_700_000, 2_600_000, 2_550_000,
	2_500_000, 2_450_000, 2_400_000, 2_300_000,
	2_280_000, 2_240_000, 2_200_000, 2_150_000,
}

var e4kIFFilterTables = [][]uint32{e4kMixFilterBw, e4kIFChanFilterBw, e4kIFRCFilterBw}

var e4kIFFilterFields = []e4kRegField{
	{e4kRegFilt2, 4, 4}, // MIX
	{e4kRegFilt3, 0, 5}, // CHAN
	{e4kRegFilt2, 0, 4}, // RC
}

// RF tracking-filter centre frequencies (Hz) for the UHF and L bands,
// verbatim from tuner_e4k.c. VHF bands use filter index 0; the closest
// centre to the LO selects the filter for UHF/L.
var e4kRFFiltCenterUHF = []uint32{
	360_000_000, 380_000_000, 405_000_000, 425_000_000,
	450_000_000, 475_000_000, 505_000_000, 540_000_000,
	575_000_000, 615_000_000, 670_000_000, 720_000_000,
	760_000_000, 840_000_000, 890_000_000, 970_000_000,
}

var e4kRFFiltCenterL = []uint32{
	1_300_000_000, 1_320_000_000, 1_360_000_000, 1_410_000_000,
	1_445_000_000, 1_460_000_000, 1_490_000_000, 1_530_000_000,
	1_560_000_000, 1_590_000_000, 1_640_000_000, 1_660_000_000,
	1_680_000_000, 1_700_000_000, 1_720_000_000, 1_750_000_000,
}

// e4kBand is the front-end band select (E4K_BAND_*).
type e4kBand byte

const (
	e4kBandVHF2 e4kBand = 0
	e4kBandVHF3 e4kBand = 1
	e4kBandUHF  e4kBand = 2
	e4kBandL    e4kBand = 3
)

// e4kPLLRange picks the synthesizer's divider for a given LO target.
// Table ported from osmocom librtlsdr's tuner_e4k.c `pll_vars[]`: each
// entry is (upper LO bound, VCO/LO divider `mul`, and the ready-to-write
// SYNTH7 byte). `bandSel` is written VERBATIM to reg 0x0D (SYNTH7) — it
// encodes the divider selection in the low nibble plus the 3-phase-mixing
// bit (0x08), and is NOT a linear function of the divider.
type e4kPLLRange struct {
	freqMax uint32 // upper LO bound (inclusive) for this band
	divLow  uint32 // VCO/LO divider (osmocom `mul`): fvco = flo * divLow
	bandSel byte   // osmocom reg_synth7, written verbatim to reg 0x0D (SYNTH7)
}

var e4kPLLRanges = []e4kPLLRange{
	{freqMax: 72_400_000, divLow: 48, bandSel: 0x0F},
	{freqMax: 81_200_000, divLow: 40, bandSel: 0x0E},
	{freqMax: 108_300_000, divLow: 32, bandSel: 0x0D},
	{freqMax: 162_500_000, divLow: 24, bandSel: 0x0C},
	{freqMax: 216_600_000, divLow: 16, bandSel: 0x0B},
	{freqMax: 325_000_000, divLow: 12, bandSel: 0x0A},
	{freqMax: 350_000_000, divLow: 8, bandSel: 0x09}, // mul 8, 3-phase mixing
	{freqMax: 432_000_000, divLow: 8, bandSel: 0x03}, // mul 8, 2-phase mixing
	{freqMax: 667_000_000, divLow: 6, bandSel: 0x02},
	{freqMax: 1_200_000_000, divLow: 4, bandSel: 0x01},
	// >1.2 GHz: osmocom's loop falls through to mul 2 / SYNTH7 0x00.
	{freqMax: ^uint32(0), divLow: 2, bandSel: 0x00},
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

// e4kMinFreqHz / e4kMaxFreqHz bound the E4000's 50 MHz .. 2.2 GHz tuning
// range — the single source for FreqRange and the SetFreq guard.
const (
	e4kMinFreqHz uint32 = 50_000_000
	e4kMaxFreqHz uint32 = 2_200_000_000
)

// FreqRange returns the E4000's inclusive tuning range in Hz.
func (e *E4000) FreqRange() (minHz, maxHz uint32) { return e4kMinFreqHz, e4kMaxFreqHz }

// Gains returns the combined gain ladder (tenths of dB) — the same list
// librtlsdr exposes for the E4000.
func (e *E4000) Gains() []int {
	out := make([]int, len(e4kUserGains))
	copy(out, e4kUserGains)
	return out
}

// Init walks the chip's power-on sequence faithfully after RTL2832U
// baseband init: dummy read, master reset, clock config (incl. the
// clock-output power-down), the "magic init" flood, AGC thresholds,
// auto-gain default, the moderate IF-gain stages, the narrow IF
// filters, and the DC-correction disable — matching librtlsdr's
// e4k_init() write-for-write. IMR / DC-offset calibration sweeps are
// left at defaults (as librtlsdr ships them).
func (e *E4000) Init() error {
	if e.initDone {
		return nil
	}
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer e.demod.SetI2CRepeater(false)

	// Dummy read — librtlsdr does this to wake the I2C engine; the
	// first transaction is expected to NAK. We swallow the error.
	_, _ = e.readReg(0)

	// Reset everything, keep out of standby, clear POR (MASTER1 = 0x07).
	if err := e.writeReg(e4kRegMaster1, e4kMaster1Init); err != nil {
		return fmt.Errorf("e4k init master1: %w", err)
	}
	// Configure clock input, disable clock output.
	if err := e.writeReg(e4kRegClkInp, 0x00); err != nil {
		return fmt.Errorf("e4k init clk_inp: %w", err)
	}
	if err := e.writeReg(e4kRegRefClk, 0x00); err != nil {
		return fmt.Errorf("e4k init ref_clk: %w", err)
	}
	if err := e.writeReg(e4kRegClkoutPwdn, e4kClkoutPwdnDisable); err != nil {
		return fmt.Errorf("e4k init clkout_pwdn: %w", err)
	}

	// "Magic init" — librtlsdr's factory-prescribed writes. Verbatim
	// from tuner_e4k.c magic_init(); the order/values are load-bearing.
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

	// AGC thresholds + LNA calibration/loop rate.
	if err := e.writeReg(e4kRegAGC4, 0x10); err != nil { // high threshold
		return fmt.Errorf("e4k init agc4: %w", err)
	}
	if err := e.writeReg(e4kRegAGC5, 0x04); err != nil { // low threshold
		return fmt.Errorf("e4k init agc5: %w", err)
	}
	if err := e.writeReg(e4kRegAGC6, 0x1A); err != nil { // LNA calib + loop rate
		return fmt.Errorf("e4k init agc6: %w", err)
	}

	// Default to auto gain (LNA + mixer), matching e4k_init().
	if err := e.enableManualGain(false); err != nil {
		return fmt.Errorf("e4k init gain mode: %w", err)
	}

	// Moderate IF VGA gain baseline: stages 1..6 = 6,0,0,0,9,9.
	for _, s := range []struct {
		stage int
		value int
	}{{1, 6}, {2, 0}, {3, 0}, {4, 0}, {5, 9}, {6, 9}} {
		if err := e.ifGainSet(s.stage, s.value); err != nil {
			return fmt.Errorf("e4k init if_gain %d: %w", s.stage, err)
		}
	}

	// Narrow IF filters (mix/rc/chan) + enable the channel filter.
	if err := e.ifFilterBwSet(e4kIFFilterMix, 1_900_000); err != nil {
		return fmt.Errorf("e4k init if_filter mix: %w", err)
	}
	if err := e.ifFilterBwSet(e4kIFFilterRC, 1_000_000); err != nil {
		return fmt.Errorf("e4k init if_filter rc: %w", err)
	}
	if err := e.ifFilterBwSet(e4kIFFilterChan, 2_150_000); err != nil {
		return fmt.Errorf("e4k init if_filter chan: %w", err)
	}
	if err := e.ifFilterChanEnable(true); err != nil {
		return fmt.Errorf("e4k init if_filter chan enable: %w", err)
	}

	// Disable time-variant DC correction + LUT.
	if err := e.regSetMask(e4kRegDC5, 0x03, 0); err != nil {
		return fmt.Errorf("e4k init dc5: %w", err)
	}
	if err := e.regSetMask(e4kRegDCTime1, 0x03, 0); err != nil {
		return fmt.Errorf("e4k init dctime1: %w", err)
	}
	if err := e.regSetMask(e4kRegDCTime2, 0x03, 0); err != nil {
		return fmt.Errorf("e4k init dctime2: %w", err)
	}

	e.initDone = true
	return nil
}

func (e *E4000) Standby() error {
	if !e.initDone {
		return nil
	}
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer e.demod.SetI2CRepeater(false)
	// Clear the master-enable bits — the chip drops into power-down.
	if err := e.writeReg(e4kRegMaster1, 0x00); err != nil {
		return fmt.Errorf("e4k standby: %w", err)
	}
	e.initDone = false
	return nil
}

func (e *E4000) Close() error { return e.Standby() }

// SetFreq programs the synthesizer to land on freqHz, then selects the
// front-end band bias and RF tracking filter for the resulting LO —
// mirroring librtlsdr's e4k_tune_freq → e4k_tune_params (synth regs,
// e4k_band_set, e4k_rf_filter_set).
func (e *E4000) SetFreq(hz uint32) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	if minHz, maxHz := e.FreqRange(); hz < minHz || hz > maxHz {
		return &ErrUnsupportedFreq{Hz: hz, MinHz: minHz, MaxHz: maxHz, TunerStr: "E4000"}
	}
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer e.demod.SetI2CRepeater(false)
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
	// X is a 16-bit fractional: (remainder / fosc) * 65536, truncated.
	x := uint32((remainder * 65536) / fosc)

	// flo is the actual LO the (z, x, div) triple produces. librtlsdr
	// selects the band + RF filter from this, not the intended freq;
	// they differ by at most a few Hz so a band boundary is never
	// straddled, but we compute it faithfully.
	flo := uint32((fosc * (uint64(z)*65536 + uint64(x))) / (uint64(rng.divLow) * 65536))

	// e4k_tune_params write order: SYNTH7 (R + phase), then Z, then X.
	if err := e.writeReg(e4kRegSynth7, rng.bandSel); err != nil {
		return err
	}
	if err := e.writeReg(e4kRegSynth3, byte(z&0xFF)); err != nil {
		return err
	}
	if err := e.writeReg(e4kRegSynth4, byte(x&0xFF)); err != nil {
		return err
	}
	if err := e.writeReg(e4kRegSynth5, byte((x>>8)&0xFF)); err != nil {
		return err
	}

	// Band bias + the 325-350 MHz gap workaround, then the RF filter.
	band := e4kBandFor(flo)
	if err := e.bandSet(band); err != nil {
		return err
	}
	return e.regSetMask(e4kRegFilt1, 0x0F, e4kChooseRFFilter(band, flo))
}

// bandSet programs the front-end bias for the band and the SYNTH1 band
// bits, including librtlsdr's reset-before-write workaround for the
// 325-350 MHz gap.
func (e *E4000) bandSet(band e4kBand) error {
	bias := byte(3)
	if band == e4kBandL {
		bias = 0
	}
	if err := e.writeReg(e4kRegBias, bias); err != nil {
		return err
	}
	if err := e.regSetMask(e4kRegSynth1, 0x06, 0); err != nil {
		return err
	}
	return e.regSetMask(e4kRegSynth1, 0x06, byte(band)<<1)
}

// SetBandwidth configures the three IF filters (mix / RC / channel) to
// the requested occupied bandwidth, matching librtlsdr's e4000_set_bw.
// Pass 0 to select the narrowest available filters.
func (e *E4000) SetBandwidth(hz uint32) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer e.demod.SetI2CRepeater(false)
	e.bwHz = hz
	if err := e.ifFilterBwSet(e4kIFFilterMix, hz); err != nil {
		return err
	}
	if err := e.ifFilterBwSet(e4kIFFilterRC, hz); err != nil {
		return err
	}
	return e.ifFilterBwSet(e4kIFFilterChan, hz)
}

// SetGain sets the combined manual gain (tenths of dB), distributing it
// across the LNA and mixer stages exactly as librtlsdr's e4000_set_gain:
// mixer = 4 dB (or 12 dB above 34 dB total), LNA = the remainder capped
// at 30 dB. The request is quantized to the nearest ladder value first.
func (e *E4000) SetGain(tenthDB int) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	if !e.manual || tenthDB < 0 {
		return nil
	}
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer e.demod.SetI2CRepeater(false)

	gain := e4kUserGains[nearestGainIndex(e4kUserGains, tenthDB)]
	mixerDB := 4
	if gain > 340 {
		mixerDB = 12
	}
	lna := gain - mixerDB*10
	if lna > 300 {
		lna = 300
	}
	if err := e.regSetMask(e4kRegGain1, 0x0F, e4kLNAGainReg(lna)); err != nil {
		return err
	}
	return e.regSetMask(e4kRegGain2, 0x01, e4kMixerGainBit(mixerDB))
}

// SetGainMode flips between manual (true) and automatic (false) LNA +
// mixer gain control, matching librtlsdr's e4k_enable_manual_gain.
func (e *E4000) SetGainMode(manual bool) error {
	if !e.initDone {
		return errors.New("e4k: Init not called")
	}
	if err := e.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer e.demod.SetI2CRepeater(false)
	e.manual = manual
	return e.enableManualGain(manual)
}

// ----------------------------------------------------------------------
// Internals

// enableManualGain mirrors librtlsdr's e4k_enable_manual_gain: manual
// selects serial-programmed LNA gain and manual mixer gain; auto selects
// the IF-serial/LNA-auto AGC mode and auto mixer gain (and clears the
// gain-enhancement bits).
func (e *E4000) enableManualGain(manual bool) error {
	if manual {
		if err := e.regSetMask(e4kRegAGC1, e4kAGC1ModMask, e4kAGCModSerial); err != nil {
			return err
		}
		return e.regSetMask(e4kRegAGC7, e4kAGC7MixGainAuto, 0)
	}
	if err := e.regSetMask(e4kRegAGC1, e4kAGC1ModMask, e4kAGCModIFSerialLNAAuto); err != nil {
		return err
	}
	if err := e.regSetMask(e4kRegAGC7, e4kAGC7MixGainAuto, e4kAGC7MixGainAuto); err != nil {
		return err
	}
	return e.regSetMask(e4kRegAGC11, 0x07, 0)
}

// ifGainSet writes the closest entry of an IF VGA stage's gain ladder
// into that stage's register field (librtlsdr e4k_if_gain_set).
func (e *E4000) ifGainSet(stage, value int) error {
	if stage < 1 || stage >= len(e4kIFStageGains) {
		return fmt.Errorf("e4k: if gain stage %d out of range", stage)
	}
	idx := nearestGainIndex(e4kIFStageGains[stage], value)
	return e.fieldWrite(e4kIFStageFields[stage], byte(idx))
}

// ifFilterBwSet selects the closest bandwidth entry for the given IF
// filter family and writes its index into the filter's field
// (librtlsdr e4k_if_filter_bw_set).
func (e *E4000) ifFilterBwSet(filter int, bwHz uint32) error {
	if filter < 0 || filter >= len(e4kIFFilterTables) {
		return fmt.Errorf("e4k: if filter %d out of range", filter)
	}
	idx := e4kClosestIdx(e4kIFFilterTables[filter], bwHz)
	return e.fieldWrite(e4kIFFilterFields[filter], idx)
}

// ifFilterChanEnable toggles the IF channel filter (librtlsdr
// e4k_if_filter_chan_enable): FILT3 disable bit is 0 = enabled.
func (e *E4000) ifFilterChanEnable(on bool) error {
	val := e4kFilt3Disable
	if on {
		val = 0
	}
	return e.regSetMask(e4kRegFilt3, e4kFilt3Disable, val)
}

// regSetMask is a read-modify-write of the masked bits, mirroring
// librtlsdr's e4k_reg_set_mask (including its skip-if-unchanged).
func (e *E4000) regSetMask(addr, mask, val byte) error {
	cur, err := e.readReg(addr)
	if err != nil {
		return err
	}
	if (cur & mask) == (val & mask) {
		return nil
	}
	return e.writeReg(addr, (cur&^mask)|(val&mask))
}

// fieldWrite writes val into a sub-byte register field (e4k_field_write).
func (e *E4000) fieldWrite(f e4kRegField, val byte) error {
	mask := e4kWidth2Mask[f.width] << f.shift
	return e.regSetMask(f.reg, mask, val<<f.shift)
}

// e4kBandFor maps an LO frequency to the front-end band, matching the
// boundaries in librtlsdr's e4k_tune_params.
func e4kBandFor(floHz uint32) e4kBand {
	switch {
	case floHz < 140_000_000:
		return e4kBandVHF2
	case floHz < 350_000_000:
		return e4kBandVHF3
	case floHz < 1_135_000_000:
		return e4kBandUHF
	default:
		return e4kBandL
	}
}

// e4kChooseRFFilter returns the FILT1 RF-tracking-filter index for the
// band + LO, mirroring librtlsdr's choose_rf_filter. VHF bands use index 0.
func e4kChooseRFFilter(band e4kBand, floHz uint32) byte {
	switch band {
	case e4kBandUHF:
		return e4kClosestIdx(e4kRFFiltCenterUHF, floHz)
	case e4kBandL:
		return e4kClosestIdx(e4kRFFiltCenterL, floHz)
	default:
		return 0
	}
}

// e4kClosestIdx returns the index of the table entry nearest to v
// (librtlsdr closest_arr_idx).
func e4kClosestIdx(arr []uint32, v uint32) byte {
	best, bestDelta := 0, ^uint32(0)
	for i, c := range arr {
		var d uint32
		if v > c {
			d = v - c
		} else {
			d = c - v
		}
		if d < bestDelta {
			bestDelta, best = d, i
		}
	}
	return byte(best)
}

// e4kLNAGainReg returns the GAIN1 low-nibble register code for the LNA
// gain nearest tenthDB (librtlsdr lnagain[] mapping).
func e4kLNAGainReg(tenthDB int) byte {
	best, bestDelta := 0, 1<<30
	for i, g := range e4kLNAGains {
		d := tenthDB - g.tenthDB
		if d < 0 {
			d = -d
		}
		if d < bestDelta {
			bestDelta, best = d, i
		}
	}
	return e4kLNAGains[best].reg
}

// e4kMixerGainBit returns the GAIN2 bit for a 4 dB (0) or 12 dB (1)
// mixer gain (librtlsdr e4k_mixer_gain_set).
func e4kMixerGainBit(dB int) byte {
	if dB >= 12 {
		return 1
	}
	return 0
}

// writeReg / readReg are private I2C plumbing. Callers (public
// methods) own the SetI2CRepeater bracket — librtlsdr's pattern.
func (e *E4000) writeReg(addr, val byte) error {
	return e.demod.I2CWriteReg(e4kI2CAddr, addr, val)
}

func (e *E4000) readReg(addr byte) (byte, error) {
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
