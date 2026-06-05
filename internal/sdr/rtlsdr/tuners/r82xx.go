package tuners

import (
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/usb"
)

// R82xx implements [Tuner] for the R820T / R820T2 / R828D chips, which
// share the same I2C register map and PLL synthesizer with a different
// I2C address for R828D and a slightly different chip-ID byte.
//
// Implementation is a straight port of osmocom librtlsdr's
// src/tuner_r82xx.c — register addresses, init flood, PLL math, and
// the mux frequency-range table are all kept byte-identical so that
// real-hardware captures from librtlsdr replay-validate against the
// mock USB transport in tests.
//
// The shadow-register cache makes read-modify-write operations free
// and skips redundant writes: the chip silently drops a write whose
// value matches the previous one anyway, so eliding it saves a
// USB roundtrip without changing observable behavior. The cache also
// papers over the R820T quirk that its writable registers can't be
// read back through the I2C bridge.
type R82xx struct {
	demod    *rtl2832u.Demod
	i2cAddr  uint8
	chipType Type
	xtalHz   uint32

	// regs[0x05..0x1F] is the shadow for writable registers.
	// regs[0x00..0x04] holds the most recently read read-only
	// status bytes (chip ID, lock bits, VCO fine-tune).
	regs [r82xxNumRegs]byte

	initDone bool
	manual   bool   // gain mode: true = manual, false = AGC
	bwHz     uint32 // last requested bandwidth
	freqHz   uint32 // last requested center frequency
	ppmCorr  int    // tuner-LO frequency correction, parts-per-million

	// blogV4 marks an RTL-SDR Blog V4 (R828D) dongle, which needs the
	// 28.8 MHz reference crystal (NOT the 16 MHz generic-R828D default)
	// and per-band switching of its HF/VHF/UHF input bank in SetFreq.
	// blogV4L is the two-band "Lite" variant (no separate VHF input).
	// v4Input caches the last-selected input bank so SetFreq only
	// rewrites the switch registers on a band change. See SetBlogV4.
	blogV4  bool
	blogV4L bool
	v4Input v4Band
}

// v4Band identifies the RTL-SDR Blog V4's switched input bank. The
// zero value (v4BandNone) means "no band selected yet", so the first
// SetFreq always writes the switch registers.
type v4Band uint8

const (
	v4BandNone v4Band = iota
	v4BandHF
	v4BandVHF
	v4BandUHF
)

// NewR82xx constructs a driver bound to the given RTL2832U demod and
// I2C address. Callers normally obtain the right address via the
// Detect helper. The reference-crystal frequency defaults are
// per-chip-type: R820T/R820T2 run from 28.8 MHz (the RTL2832U's
// reference crystal), R828D runs from 16 MHz (a separate on-board
// crystal). Boards with non-standard crystals can override via
// [R82xx.SetXtal] after construction.
func NewR82xx(d *rtl2832u.Demod, i2cAddr uint8, chip Type) *R82xx {
	xtal := r82xxXtalHz
	if chip == TypeR828D {
		xtal = r828dXtalHz
	}
	return &R82xx{
		demod:    d,
		i2cAddr:  i2cAddr,
		chipType: chip,
		xtalHz:   xtal,
	}
}

// SetXtal overrides the reference-crystal frequency. R820T chips
// derive every PLL division off the RTL2832U's crystal, so a
// non-default board crystal must be propagated here too. Per-chip
// defaults are set by [NewR82xx] (28.8 MHz for R820T/R820T2, 16 MHz
// for R828D); SetXtal exists for boards that deviate from those.
func (r *R82xx) SetXtal(hz uint32) { r.xtalHz = hz }

// SetBlogV4 marks this tuner as an RTL-SDR Blog V4 (lite=false) or V4
// Lite (lite=true) so SetFreq drives the V4's switched HF/VHF/UHF input
// bank. It also restores the 28.8 MHz reference crystal: the V4 runs
// its R828D from 28.8 MHz, but NewR82xx defaults every R828D to the
// 16 MHz generic crystal, which would mis-tune the V4 by ~28.8/16 =
// 1.8× (issue #264). Detection keys off the V4's USB iManufacturer/
// iProduct strings; mirrors the rtlsdr-blog fork's per-V4 handling
// (tuner_r82xx.c). Call once after detection, before the first SetFreq.
func (r *R82xx) SetBlogV4(lite bool) {
	r.blogV4 = true
	r.blogV4L = lite
	r.xtalHz = r82xxXtalHz // 28.8 MHz, overriding the R828D 16 MHz default
}

// SetFreqCorrection applies a parts-per-million correction to the
// tuner LO, mirroring the tuner half of librtlsdr's
// rtlsdr_set_freq_correction. The RTL2832U sample-clock half is
// applied separately via the demod's SetSampleFreqCorrection; without
// this method a configured ppm only retimed the resampler and never
// moved the carrier, so a real crystal offset stayed in the signal
// (issue #264: the reporter's `ppm: -4` was "not adopted"). A static
// carrier offset that survives here pushes the C4FM eye off the
// fixed-threshold slicer and breaks digital decode.
//
// The correction biases the PLL reference in setPLL; if a frequency
// has already been tuned it re-tunes so the change takes effect
// immediately, otherwise the next SetFreq picks it up. No-op when the
// value is unchanged.
func (r *R82xx) SetFreqCorrection(ppm int) error {
	if r.ppmCorr == ppm {
		return nil
	}
	r.ppmCorr = ppm
	if r.initDone && r.freqHz != 0 {
		return r.SetFreq(r.freqHz)
	}
	return nil
}

// effectiveXtalHz returns the reference-crystal frequency adjusted for
// the configured ppm correction, mirroring librtlsdr's APPLY_PPM_CORR
// (xtal · (1 + ppm·1e-6)). A positive ppm raises the effective
// reference so setPLL's registers target a slightly lower nominal LO,
// compensating a fast crystal. ppm == 0 returns the raw crystal
// unchanged, so the integer-divider math reproduces byte-for-byte.
func (r *R82xx) effectiveXtalHz() uint32 {
	if r.ppmCorr == 0 {
		return r.xtalHz
	}
	return uint32(int64(r.xtalHz) + int64(r.xtalHz)*int64(r.ppmCorr)/1_000_000)
}

// Type returns the detected chip family.
func (r *R82xx) Type() Type { return r.chipType }

// XtalHz returns the reference-crystal frequency currently in effect
// (before any ppm correction). It exists for boot-time diagnostics:
// 16 MHz on an R828D means the RTL-SDR Blog V4 path did NOT arm, so the
// LO mistunes by ~28.8/16 = 1.8× (issue #264); 28.8 MHz means SetBlogV4
// ran. See [R82xx.IsBlogV4].
func (r *R82xx) XtalHz() uint32 { return r.xtalHz }

// IsBlogV4 reports whether the RTL-SDR Blog V4 path is armed and, if so,
// whether it's the two-band "Lite" variant. Surfaced for the pool's
// per-device "sdr tuner detected" diagnostic line (issue #264).
func (r *R82xx) IsBlogV4() (enabled, lite bool) { return r.blogV4, r.blogV4L }

// IFFreqHz returns the 3.57 MHz intermediate frequency the R820T
// emits.
func (r *R82xx) IFFreqHz() uint32 { return r82xxIFFreqHz }

// Gains returns the supported manual-gain ladder in tenths of dB.
func (r *R82xx) Gains() []int {
	out := make([]int, len(r82xxGainsTenthDB))
	copy(out, r82xxGainsTenthDB)
	return out
}

// detectR82xx probes the two candidate I2C addresses for an R820T
// family chip and returns a ready (uninitialized) driver, or nil if
// no chip responded. Caller is responsible for the surrounding
// SetI2CRepeater(true)/(false) pair — the orchestrator in detect.go
// does this once across all candidate tuners.
func detectR82xx(d *rtl2832u.Demod) Tuner {
	for _, c := range []struct {
		addr uint8
		typ  Type
	}{
		{addr: r82xxI2CAddr, typ: TypeR820T2},
		{addr: r828dI2CAddr, typ: TypeR828D},
	} {
		out, err := d.I2CRead(c.addr, 1)
		if err != nil || len(out) == 0 {
			continue
		}
		id := r82xxBitReverse(out[0])
		// Chip ID byte: 0x69 for R820T family. Other tuners
		// respond with different patterns; we only claim a
		// match if the ID is plausible.
		if id == 0x69 || id == 0x96 { // includes some bit-reversed clones
			return NewR82xx(d, c.addr, c.typ)
		}
	}
	return nil
}

// PrepareDemod runs the four librtlsdr-parity demod-register writes
// that must happen BETWEEN Detect and Init for R820T-family tuners:
// disable Zero-IF mode, enable only the In-phase ADC input, program
// the 3.57 MHz IF frequency, and enable spectrum inversion. Mirrors
// the RTLSDR_TUNER_R820T / R828D switch arm in librtlsdr's
// rtlsdr_open. All four go through the RTL2832U demod-register path,
// not the I²C bridge — PrepareDemod intentionally does not touch
// the SetI2CRepeater state so the post-Detect off-state (from
// detect.go's defer) is preserved for Init's leading on-toggle
// (issue #248).
func (r *R82xx) PrepareDemod() error {
	if err := r.demod.WriteDemodReg(1, 0xB1, 0x1A, 1); err != nil {
		return fmt.Errorf("r82xx prep: disable Zero-IF: %w", err)
	}
	if err := r.demod.WriteDemodReg(0, 0x08, 0x4D, 1); err != nil {
		return fmt.Errorf("r82xx prep: In-phase ADC only: %w", err)
	}
	if err := r.demod.SetIFFreq(r82xxIFFreqHz); err != nil {
		return fmt.Errorf("r82xx prep: IF freq: %w", err)
	}
	if err := r.demod.WriteDemodReg(1, 0x15, 0x01, 1); err != nil {
		return fmt.Errorf("r82xx prep: spectrum inversion: %w", err)
	}
	return nil
}

// Init walks the librtlsdr power-on sequence: open the I²C repeater
// (a fresh wire write — load-bearing on NESDR v5 silicon, issue
// #248), wait briefly for the chip-settle window, write the 27-byte
// init flood to registers 0x05..0x1F via writeBurstRaw's halving
// fallback (16 → 8 → 4), close the repeater on return.
func (r *R82xx) Init() error {
	if r.initDone {
		return nil
	}
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer r.demod.SetI2CRepeater(false)
	// Brief chip-settle window between the repeater open and the
	// multi-byte burst — covers a timing gap librtlsdr gets
	// incidentally via function-call latency that our tight
	// PrepareDemod → Init back-to-back path doesn't. See
	// r82xxPostPrepDemodSettleMillis docstring (issue #248).
	time.Sleep(r82xxPostPrepDemodSettleMillis * time.Millisecond)
	// Prime the shadow with the init values; the burst write below
	// makes them real.
	for i, v := range r82xxInitArray {
		r.regs[r82xxShadowStart+i] = v
	}
	if err := r.writeBurstRaw(r82xxShadowStart, r82xxInitArray[:]); err != nil {
		return fmt.Errorf("r82xx init: burst write: %w", err)
	}
	r.initDone = true
	return nil
}

// Standby puts the chip in low-power mode. Reversible by another call
// to Init (which restores the full register state).
func (r *R82xx) Standby() error {
	if !r.initDone {
		return nil
	}
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer r.demod.SetI2CRepeater(false)
	// Sequence taken from osmocom r82xx_standby — power down LDO,
	// PLL, mixer, LNA, VGA, filter in one burst-style write set.
	standbyRegs := []struct {
		addr uint8
		val  byte
	}{
		{0x06, 0xB1}, // PLL pwd
		{0x05, 0xA0}, // LNA pwd
		{0x07, 0x3A}, // mixer pwd
		{0x08, 0x40}, // filter pwd
		{0x09, 0xC0}, // PGA pwd
		{0x0A, 0x36}, // PLL pwd
		{0x0C, 0x35}, // VCO pwd
		{0x0F, 0x68}, // Buffer + xtal pwd
		{0x11, 0x03}, // PWD
		{0x17, 0xF4}, // PWD
		{0x19, 0x0C}, // PWD
	}
	for _, s := range standbyRegs {
		if err := r.writeReg(s.addr, s.val); err != nil {
			return fmt.Errorf("r82xx standby: addr=0x%02x: %w", s.addr, err)
		}
	}
	r.initDone = false
	return nil
}

// Close is Standby + nothing else; the demod handle is owned by the
// caller and stays alive for any subsequent tuner re-init.
func (r *R82xx) Close() error { return r.Standby() }

// SetFreq tunes the LO. The R820T converts the input RF to a 3.57 MHz
// IF, so the actual PLL target is freq + IF.
func (r *R82xx) SetFreq(hz uint32) error {
	if !r.initDone {
		return errors.New("r82xx: Init not called")
	}
	// On the Blog V4, HF (≤ 28.8 MHz) is reached through the on-board
	// upconverter, so the R828D mixer/PLL actually sees hz + 28.8 MHz.
	// VHF/UHF tune directly (target == hz), so this is a no-op outside
	// HF and leaves every non-V4 path byte-for-byte unchanged.
	target := hz
	if r.blogV4 && hz <= r82xxV4HFCrossHz {
		target = hz + r82xxXtalHz
	}
	if target < 24_000_000 || target > 1_766_000_000 {
		return &ErrUnsupportedFreq{Hz: hz, MinHz: 24_000_000, MaxHz: 1_766_000_000, TunerStr: r.chipType.String()}
	}
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer r.demod.SetI2CRepeater(false)
	r.freqHz = hz
	if err := r.setMux(target); err != nil {
		return fmt.Errorf("r82xx SetFreq: setMux: %w", err)
	}
	// V4 front-end: select the HF/VHF/UHF input bank and notch for the
	// *requested* RF frequency (not the upconverted target). Must run
	// after setMux because the HF tracking-filter bypass overrides the
	// mux's 0x1A/0x1B writes.
	if r.blogV4 {
		if err := r.applyBlogV4Band(hz); err != nil {
			return fmt.Errorf("r82xx SetFreq: v4 band: %w", err)
		}
	}
	loHz := target + r82xxIFFreqHz
	if err := r.setPLL(loHz); err != nil {
		return fmt.Errorf("r82xx SetFreq: setPLL(%d): %w", loHz, err)
	}
	return nil
}

// v4BandFor maps a requested RF frequency to the RTL-SDR Blog V4 input
// bank: HF ≤ 28.8 MHz (upconverter), VHF (28.8, 250) MHz, UHF ≥ 250 MHz.
// The two-band V4 Lite has no separate VHF input — everything above HF
// is UHF. Thresholds match the rtlsdr-blog fork.
func v4BandFor(hz uint32, lite bool) v4Band {
	switch {
	case hz <= r82xxV4HFCrossHz:
		return v4BandHF
	case !lite && hz < r82xxV4UHFCrossHz:
		return v4BandVHF
	default:
		return v4BandUHF
	}
}

// applyBlogV4Band drives the RTL-SDR Blog V4's switched input bank,
// notch filter, and (for HF) tracking-filter bypass for the requested
// RF frequency. Ported verbatim from the rtlsdr-blog fork's
// r82xx_set_freq V4 block; the stock R828D init leaves every V4 input
// off, so without these writes the V4 routes no RF and receives only
// noise (issue #264). hz is the original requested frequency (the band
// decision is on the antenna-plane frequency, not the upconverted IF
// target). Caller holds the I2C repeater.
func (r *R82xx) applyBlogV4Band(hz uint32) error {
	// Notch ON (0x08) except when tuned inside one of the V4's notch
	// windows (≤2.2 MHz, 85–112 MHz, 172–242 MHz), where it's OFF.
	openD := byte(0x08)
	if hz <= 2_200_000 ||
		(hz >= 85_000_000 && hz <= 112_000_000) ||
		(hz >= 172_000_000 && hz <= 242_000_000) {
		openD = 0x00
	}
	if err := r.writeRegMask(0x17, openD, 0x08); err != nil {
		return err
	}

	// Band: HF ≤ 28.8 MHz; VHF (28.8, 250) MHz; UHF ≥ 250 MHz. The V4
	// Lite has no separate VHF input — everything above HF is UHF.
	band := v4BandFor(hz, r.blogV4L)

	// HF: bypass the tracking filter to cut upconverter insertion loss.
	// Re-applied every tune since setMux rewrites 0x1A/0x1B above.
	if band == v4BandHF {
		if err := r.writeRegMask(0x1A, 0x40, 0xC3); err != nil {
			return err
		}
		if err := r.writeReg(0x1B, 0x00); err != nil {
			return err
		}
	}

	// Only rewrite the input switches on a band change.
	if band == r.v4Input {
		return nil
	}
	r.v4Input = band

	// Cable 2 = HF input (reg 0x06 bit 3).
	cable2 := byte(0x00)
	if band == v4BandHF {
		cable2 = 0x08
	}
	if err := r.writeRegMask(0x06, cable2, 0x08); err != nil {
		return err
	}
	// Upconverter relay on RTL2832 GPIO5: driven high for every band
	// except HF (the fork's !cable_2_in).
	if err := r.demod.SetGPIOOutput(5); err != nil {
		return err
	}
	if err := r.demod.SetGPIOBit(5, cable2 == 0x00); err != nil {
		return err
	}
	// Cable 1 = VHF input (reg 0x05 bit 6).
	cable1 := byte(0x00)
	if band == v4BandVHF {
		cable1 = 0x40
	}
	if err := r.writeRegMask(0x05, cable1, 0x40); err != nil {
		return err
	}
	// Air-in = UHF input (reg 0x05 bit 5): on for HF/VHF, off for UHF.
	air := byte(0x20)
	if band == v4BandUHF {
		air = 0x00
	}
	return r.writeRegMask(0x05, air, 0x20)
}

// SetBandwidth picks the filter that matches the requested occupied
// bandwidth. Pass 0 to use the last-set sample rate (the driver layer
// passes the demod's current rate here).
func (r *R82xx) SetBandwidth(hz uint32) error {
	if !r.initDone {
		return errors.New("r82xx: Init not called")
	}
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer r.demod.SetI2CRepeater(false)
	if hz == 0 {
		hz = 6_000_000 // librtlsdr's default when nothing else is set
	}
	r.bwHz = hz
	// Walk the BW table (descending order) and keep the last entry
	// that still ≥ hz — that's the smallest filter wide enough not
	// to clip useful signal. When hz exceeds every entry, idx stays
	// at 0 (widest filter). When hz is below every entry, we update
	// idx until the loop ends, landing on the narrowest filter.
	idx := 0
	for i, bw := range r82xxFilterBWTable {
		if bw >= hz {
			idx = i
		} else {
			break
		}
	}
	// Register 0x0A low nibble = coarse BW index.
	if err := r.writeRegMask(0x0A, byte(idx&0x0F), 0x0F); err != nil {
		return fmt.Errorf("r82xx SetBandwidth: reg 0x0A: %w", err)
	}
	// Register 0x0B fine-tune defaults to 0x00 — librtlsdr does
	// the more careful selection inside r82xx_set_bandwidth's IF
	// filter calibration sweep, which depends on tracking
	// against a captured tone. We mirror librtlsdr's pre-cal
	// behavior and leave fine-tune at zero.
	if err := r.writeRegMask(0x0B, 0x00, 0xF0); err != nil {
		return fmt.Errorf("r82xx SetBandwidth: reg 0x0B: %w", err)
	}
	return nil
}

// SetGain selects the LNA + mixer index whose cumulative gain is
// closest to (without exceeding) the requested tenthDB value.
// Caller must have set manual mode via SetGainMode(true) first; this
// function returns silently when AGC is active.
func (r *R82xx) SetGain(tenthDB int) error {
	if !r.initDone {
		return errors.New("r82xx: Init not called")
	}
	if !r.manual {
		// Mirror librtlsdr: SetGain is a no-op in AGC mode.
		return nil
	}
	if tenthDB < 0 {
		// SetGain(-1) historically means "leave as is" — librtlsdr
		// callers use it as a sentinel.
		return nil
	}
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer r.demod.SetI2CRepeater(false)
	// Alternate LNA and mixer increments, pre-incrementing the index
	// each step. Matches librtlsdr's r82xx_set_gain — the published
	// gain ladder (r82xxGainsTenthDB) is the alternating sum, so this
	// is the only walk that lands on a balanced LNA+Mixer split for
	// each target. The LNA-first-then-mixer alternative produces the
	// same total at most ladder entries but with all gain concentrated
	// on LNA — wrong noise figure and front-end linearity.
	var lnaIdx, mixIdx int
	total := 0
	for i := 0; i < 15; i++ {
		if total >= tenthDB {
			break
		}
		if lnaIdx+1 < len(r82xxLNAGainSteps) {
			lnaIdx++
			total += r82xxLNAGainSteps[lnaIdx]
		}
		if total >= tenthDB {
			break
		}
		if mixIdx+1 < len(r82xxMixerGainSteps) {
			mixIdx++
			total += r82xxMixerGainSteps[mixIdx]
		}
	}
	// Register 0x05 low nibble = LNA gain index; bit 4 must be 0 for manual mode.
	if err := r.writeRegMask(0x05, byte(lnaIdx&0x0F), 0x0F); err != nil {
		return err
	}
	// Register 0x07 low nibble = mixer gain index.
	if err := r.writeRegMask(0x07, byte(mixIdx&0x0F), 0x0F); err != nil {
		return err
	}
	// Register 0x0C low nibble = VGA gain index; bit 4 controls
	// VGA fixed/manual. Use a middling fixed value (0x0B = +16.3 dB
	// per librtlsdr's default).
	if err := r.writeRegMask(0x0C, 0x0B, 0x9F); err != nil {
		return err
	}
	return nil
}

// SetGainMode flips between AGC (auto) and manual.
//
// The LNA and mixer AGC-enable bits have OPPOSITE polarity in
// librtlsdr's r82xx_set_gain, which is easy to get backwards:
//
//	             reg 0x05 bit4 (LNA)   reg 0x07 bit4 (mixer)
//	manual:      1 (auto off)          0 (auto off)
//	AGC:         0 (auto on)           1 (auto on)
//
// In AGC mode librtlsdr also pins the VGA at a fixed value (reg 0x0C =
// 0x0B). SetGain handles the VGA in manual mode, but it is a no-op in
// AGC mode, so the AGC branch must write it here — otherwise the VGA is
// left at the init default and the front end runs ~17 dB low, deafening
// marginal-signal dongles like the RTL-SDR Blog V4 (issue #264).
func (r *R82xx) SetGainMode(manual bool) error {
	if !r.initDone {
		return errors.New("r82xx: Init not called")
	}
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer r.demod.SetI2CRepeater(false)
	r.manual = manual
	// LNA gain mode: reg 0x05 bit 4 set = manual; clear = AGC.
	lnaBit := byte(0x00)
	if manual {
		lnaBit = 0x10
	}
	if err := r.writeRegMask(0x05, lnaBit, 0x10); err != nil {
		return err
	}
	// Mixer gain mode: reg 0x07 bit 4 — inverted from the LNA bit.
	// set = AGC; clear = manual.
	mixBit := byte(0x10)
	if manual {
		mixBit = 0x00
	}
	if err := r.writeRegMask(0x07, mixBit, 0x10); err != nil {
		return err
	}
	// AGC mode: pin the VGA at librtlsdr's fixed default (+16.3 dB).
	if !manual {
		if err := r.writeRegMask(0x0C, 0x0B, 0x9F); err != nil {
			return err
		}
	}
	return nil
}

// ----------------------------------------------------------------------
// PLL synthesis

// vcoPowerRef is the VCO fine-tune comparison threshold setPLL uses to
// nudge the mixer divider. osmocom librtlsdr uses r82xxVCOPowerRef (2)
// for every chip; the rtlsdr-blog fork's r82xx_set_pll lowers it to 1
// for the R828D (rafael_chip == CHIP_R828D, which includes the Blog V4).
// Without it the V4's LO mistunes and the front end receives only noise
// while an R820T2 on the same signal decodes cleanly. See issue #264.
func (r *R82xx) vcoPowerRef() byte {
	if r.chipType == TypeR828D {
		return 1
	}
	return r82xxVCOPowerRef
}

// setPLL programs the R820T's frequency synthesizer to land on the
// requested LO frequency. Faithful port of osmocom r82xx_set_pll —
// integer / sigma-delta fractional path, mixer divider sweep, VCO
// fine-tune compensation. Caller must have already pushed reg 0x10
// bit 4 to zero (no refdiv/2) for the math here to be correct.
//
// Returns an error if no mixer divider produces a VCO frequency inside
// [vcoMin, vcoMax]; that happens only at frequencies far outside the
// chip's documented 24 MHz .. 1.766 GHz tuning range.
func (r *R82xx) setPLL(freqHz uint32) error {
	if err := r.writeRegMask(0x10, 0x00, 0x10); err != nil { // refdiv2 = 0
		return err
	}
	if err := r.writeRegMask(0x1A, 0x00, 0x0C); err != nil {
		return err
	}
	if err := r.writeRegMask(0x12, 0x80, 0xE0); err != nil { // VCO current = 100
		return err
	}
	// Find mixer divider so freqHz*mixDiv falls inside [vcoMin, vcoMax).
	var mixDiv uint32 = 2
	var divNum uint8
	for mixDiv <= 64 {
		v := uint64(freqHz) * uint64(mixDiv)
		if v >= r82xxVCOMin && v < r82xxVCOMax {
			break
		}
		mixDiv <<= 1
	}
	if mixDiv > 64 {
		return fmt.Errorf("r82xx setPLL: no mixer divider for %d Hz", freqHz)
	}
	// divNum is log2(mixDiv) - 1: mixDiv=2→0, 4→1, 8→2, 16→3, 32→4, 64→5.
	d := mixDiv
	for d > 2 {
		d >>= 1
		divNum++
	}
	// Read VCO fine-tune from chip (reg 0x04 bits 5..4).
	rd, err := r.readRegRaw(0x00, 5)
	if err != nil {
		return fmt.Errorf("r82xx setPLL: read status: %w", err)
	}
	vcoFineTune := (rd[4] & 0x30) >> 4
	vcoPowerRef := r.vcoPowerRef()
	if vcoFineTune > vcoPowerRef && divNum > 0 {
		divNum--
	} else if vcoFineTune < vcoPowerRef {
		divNum++
	}
	if err := r.writeRegMask(0x10, divNum<<5, 0xE0); err != nil {
		return err
	}
	vcoFreq := uint64(freqHz) * uint64(mixDiv)
	effXtal := r.effectiveXtalHz()
	pllRef := uint64(effXtal)
	nint := uint32(vcoFreq / (2 * pllRef))
	vcoFra := uint32((vcoFreq - 2*pllRef*uint64(nint)) / 1000)
	if nint > r82xxMaxNint {
		return fmt.Errorf("r82xx setPLL: nint=%d overflows", nint)
	}
	ni := uint8((nint - 13) / 4)
	si := uint8(nint - 4*uint32(ni) - 13)
	if err := r.writeReg(0x14, ni+(si<<6)); err != nil {
		return err
	}
	// pw_sdm: bit 3 of reg 0x12. Set when fractional part is zero
	// (integer-only mode); clear when SDM is in use.
	pwSDM := byte(0x08)
	if vcoFra != 0 {
		pwSDM = 0x00
	}
	if err := r.writeRegMask(0x12, pwSDM, 0x08); err != nil {
		return err
	}
	// SDM calculator. Faithfully ports osmocom's loop; the loop
	// converges in ≤16 iterations because n_sdm doubles each step.
	var sdm uint16
	nSDM := uint32(2)
	pllRefkHz := effXtal / 1000
	for vcoFra > 1 {
		if vcoFra > (2 * pllRefkHz / nSDM) {
			sdm += 32768 / uint16(nSDM/2)
			vcoFra -= 2 * pllRefkHz / nSDM
			if nSDM >= 0x8000 {
				break
			}
		}
		nSDM <<= 1
		if nSDM > 0x10000 {
			break
		}
	}
	if err := r.writeReg(0x16, byte(sdm>>8)); err != nil {
		return err
	}
	if err := r.writeReg(0x15, byte(sdm&0xFF)); err != nil {
		return err
	}
	return nil
}

// setMux walks the frequency-range table and writes the matching
// RF-mux / tracking-filter values to registers 0x17, 0x1A, 0x1B, 0x10.
// Called once per SetFreq; the values are cached in shadow so
// redundant writes to neighboring rows skip the I2C burst.
func (r *R82xx) setMux(freqHz uint32) error {
	row := r82xxFreqRanges[len(r82xxFreqRanges)-1]
	for _, candidate := range r82xxFreqRanges {
		if freqHz <= candidate.freqHz {
			row = candidate
			break
		}
	}
	if err := r.writeRegMask(0x17, row.openD, 0x08); err != nil {
		return err
	}
	if err := r.writeRegMask(0x1A, row.rfMux, 0xC3); err != nil {
		return err
	}
	if err := r.writeReg(0x1B, row.tfC); err != nil {
		return err
	}
	if err := r.writeRegMask(0x10, row.xtalCap0p, 0x0B); err != nil {
		return err
	}
	if err := r.writeRegMask(0x08, 0x00, 0x3F); err != nil {
		return err
	}
	return r.writeRegMask(0x09, 0x00, 0x3F)
}

// ----------------------------------------------------------------------
// Shadow-register I/O

// writeReg writes one byte to the chip. The new value is cached in
// the shadow if the register is writable (>= 0x05); writes whose new
// value matches the existing shadow are skipped to save USB traffic.
func (r *R82xx) writeReg(addr uint8, val byte) error {
	if addr >= r82xxShadowStart {
		if r.regs[addr] == val {
			return nil
		}
		r.regs[addr] = val
	}
	return r.writeBurstRaw(addr, []byte{val})
}

// writeRegMask reads the shadow, applies (val & mask) over the masked
// bits, and writes only if the result differs.
func (r *R82xx) writeRegMask(addr uint8, val, mask byte) error {
	if addr < r82xxShadowStart {
		return fmt.Errorf("r82xx writeRegMask: addr=0x%02x is read-only", addr)
	}
	cur := r.regs[addr]
	next := (cur &^ mask) | (val & mask)
	if cur == next {
		return nil
	}
	r.regs[addr] = next
	return r.writeBurstRaw(addr, []byte{next})
}

// writeBurstRaw bypasses the shadow cache and emits one or more I2C
// burst writes (address byte followed by data bytes). The caller is
// responsible for SetI2CRepeater(true)/(false) bracketing — each
// public method (Init, SetFreq, ...) does this once around its whole
// body, matching librtlsdr's rtlsdr_set_tuner_* wrap pattern.
//
// Data is normally split into chunks of r82xxBurstMaxData bytes to
// mirror librtlsdr's r82xx_write (NMAX_WRITES = 16). If a chunk
// EPIPEs even after writeBurstChunk's per-chunk inner retry, the
// whole burst is replayed at half the previous chunk size, walking
// 16 → 8 → 4 until one size succeeds (issue #248: two NESDR SMArt
// v5 units rejected the 17-byte first chunk even after PR #263's
// inner retry + outer USBDEVFS_RESET hammer; the hypothesis being
// tested is that the chip's I²C-bridge FIFO depth on that firmware
// revision is below librtlsdr's 16-byte assumption).
//
// Idempotency note: when a later chunk EPIPEs and the burst restarts
// at a smaller size, chunks already written at the larger size get
// their wire bytes replayed against the same chip registers. Safe
// for R820T because every register write is idempotent and the
// shadow holds the same values across attempts. Future tuner ports
// that route non-idempotent writes through writeBurstRaw must
// revisit this contract.
func (r *R82xx) writeBurstRaw(addr uint8, data []byte) error {
	var lastErr error
	for chunkSize := r82xxBurstMaxData; chunkSize >= r82xxBurstMinData; chunkSize /= 2 {
		err := r.writeBurstAtSize(addr, data, chunkSize)
		if err == nil {
			return nil
		}
		if !isI2CBurstStall(err) {
			return err
		}
		lastErr = err
		if chunkSize > r82xxBurstMinData {
			// Let the chip's I²C bridge drain before the next pass.
			time.Sleep(r82xxBurstRetryDelayMillis * time.Millisecond)
		}
	}
	return fmt.Errorf("tried chunk sizes 16,8,4; all stalled: %w", lastErr)
}

// isI2CBurstStall reports whether err is the recoverable I²C-bridge
// stall the NESDR v5 cold-boot path retries (per-chunk retry + chunk-size
// halving). The same logical stall surfaces differently per OS: Linux's
// USBDEVFS returns a raw syscall.EPIPE, while Windows/WinUSB maps the
// equivalent ERROR_GEN_FAILURE to usb.ErrPipeStalled (see winErr in
// usb_windows.go). Both must drive the recovery — checking only EPIPE
// meant the entire NESDR v5 burst recovery (issue #248) silently never
// fired on Windows, so the first chunk failure propagated straight out
// as `tuner init: r82xx init: burst write: ... ERROR_GEN_FAILURE`.
// Timeouts / ErrDeviceGone / ErrClosed are deliberately excluded — the
// outer openDevice envelope owns full-reset recovery for those.
func isI2CBurstStall(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, usb.ErrPipeStalled)
}

// writeBurstAtSize emits the burst with a specific chunk size cap.
// Pulled out of writeBurstRaw so the halving-fallback loop has a
// single per-pass entry point. Each chunk is its own control-OUT;
// the register pointer advances by chunkSize per OUT, matching the
// chip's auto-increment.
func (r *R82xx) writeBurstAtSize(addr uint8, data []byte, chunkSize int) error {
	for pos := 0; pos < len(data); {
		size := len(data) - pos
		if size > chunkSize {
			size = chunkSize
		}
		if err := r.writeBurstChunk(addr+uint8(pos), data[pos:pos+size]); err != nil {
			return err
		}
		pos += size
	}
	return nil
}

// writeBurstChunk emits exactly one I²C-bridge OUT to the tuner with
// a 1-byte register pointer prefix and len(chunk) data bytes. Caller
// is responsible for chunking to r82xxBurstMaxData and for holding
// the SetI2CRepeater bracket open across the call — writeBurstChunk
// never touches the repeater (PR #262's wire-toggle contract).
//
// On a recoverable I²C-bridge stall (Linux EPIPE / Windows
// ErrPipeStalled, see isI2CBurstStall) the chip's USB firmware NACK'd
// this specific request; the post-PR-#262 trace on issue #248 confirms
// the EP0 endpoint stays healthy (subsequent control transfers succeed
// without USBDEVFS_CLEAR_HALT). After a short settle delay we retry the
// same wire bytes once. Other errors (timeout, ErrDeviceGone, ErrClosed)
// return immediately — the outer openDevice envelope owns reset
// recovery for those.
func (r *R82xx) writeBurstChunk(addr uint8, chunk []byte) error {
	buf := make([]byte, 1+len(chunk))
	buf[0] = addr
	copy(buf[1:], chunk)
	err := r.demod.I2CWrite(r.i2cAddr, buf)
	if err == nil {
		return nil
	}
	if !isI2CBurstStall(err) {
		return err
	}
	time.Sleep(r82xxBurstRetryDelayMillis * time.Millisecond)
	if retryErr := r.demod.I2CWrite(r.i2cAddr, buf); retryErr != nil {
		return fmt.Errorf("after 1 retry on stall: %w", retryErr)
	}
	return nil
}

// readRegRaw reads n bytes from the chip starting at addr 0. The
// chip auto-increments so a single read returns regs 0x00..0x00+n-1.
// Bytes are bit-reversed on the wire; we un-reverse before returning.
// The result is also stored into the shadow so callers querying via
// the cache see fresh values for the read-only block. Caller owns
// the SetI2CRepeater bracket.
func (r *R82xx) readRegRaw(addr uint8, n int) ([]byte, error) {
	// The R820T family auto-increments from register 0 on every
	// read; pointer-setting only matters when addr != 0. For PLL
	// status reads we always pass addr=0, so we skip the pointer
	// write in the common path.
	if addr != 0 {
		if err := r.demod.I2CWrite(r.i2cAddr, []byte{addr}); err != nil {
			return nil, err
		}
	}
	out, err := r.demod.I2CRead(r.i2cAddr, n)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i] = r82xxBitReverse(out[i])
	}
	for i, b := range out {
		off := int(addr) + i
		if off < r82xxNumRegs {
			r.regs[off] = b
		}
	}
	return out, nil
}

// SettleAfterRetune is a small spin librtlsdr inserts between SetFreq
// and the next sample-buffer reset so the PLL has time to lock. The
// driver layer (PR-06) calls this in its tuning path.
func (r *R82xx) SettleAfterRetune() {
	time.Sleep(2 * time.Millisecond)
}
