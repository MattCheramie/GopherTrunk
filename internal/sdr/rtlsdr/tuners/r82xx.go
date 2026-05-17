package tuners

import (
	"errors"
	"fmt"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
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
}

// NewR82xx constructs a driver bound to the given RTL2832U demod and
// I2C address. Callers normally obtain the right address via the
// Detect helper.
func NewR82xx(d *rtl2832u.Demod, i2cAddr uint8, chip Type) *R82xx {
	return &R82xx{
		demod:    d,
		i2cAddr:  i2cAddr,
		chipType: chip,
		xtalHz:   r82xxXtalHz,
	}
}

// SetXtal overrides the reference-crystal frequency. R820T chips
// derive every PLL division off the RTL2832U's crystal, so a
// non-default board crystal must be propagated here too.
func (r *R82xx) SetXtal(hz uint32) { r.xtalHz = hz }

// Type returns the detected chip family.
func (r *R82xx) Type() Type { return r.chipType }

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

// Init walks the librtlsdr power-on sequence: write the 27-byte init
// flood to registers 0x05..0x1F, then perform the standard tuner-side
// soft-reset by toggling the demod's I2C bridge.
func (r *R82xx) Init() error {
	if r.initDone {
		return nil
	}
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
	if hz < 24_000_000 || hz > 1_766_000_000 {
		return &ErrUnsupportedFreq{Hz: hz, MinHz: 24_000_000, MaxHz: 1_766_000_000, TunerStr: r.chipType.String()}
	}
	r.freqHz = hz
	if err := r.setMux(hz); err != nil {
		return fmt.Errorf("r82xx SetFreq: setMux: %w", err)
	}
	loHz := hz + r82xxIFFreqHz
	if err := r.setPLL(loHz); err != nil {
		return fmt.Errorf("r82xx SetFreq: setPLL(%d): %w", loHz, err)
	}
	return nil
}

// SetBandwidth picks the filter that matches the requested occupied
// bandwidth. Pass 0 to use the last-set sample rate (the driver layer
// passes the demod's current rate here).
func (r *R82xx) SetBandwidth(hz uint32) error {
	if !r.initDone {
		return errors.New("r82xx: Init not called")
	}
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
	// Walk the LNA + mixer LUTs in lockstep, accumulating gain.
	// Pick the first index where the next step would exceed tenthDB.
	var lnaIdx, mixIdx int
	var total int
	for i := 0; i < len(r82xxLNAGainSteps) && total+r82xxLNAGainSteps[i] <= tenthDB; i++ {
		total += r82xxLNAGainSteps[i]
		lnaIdx = i
	}
	for i := 0; i < len(r82xxMixerGainSteps) && total+r82xxMixerGainSteps[i] <= tenthDB; i++ {
		if r82xxMixerGainSteps[i] > 0 {
			total += r82xxMixerGainSteps[i]
		}
		mixIdx = i
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
// AGC = bit 4 of register 0x05 and 0x07 set; manual = both clear.
// LNA + Mixer in librtlsdr enable AGC by setting LNA_AGC_EN = 0
// and MIXER_AGC_EN = 0 — register-bit semantics are inverted from
// what you'd naively expect.
func (r *R82xx) SetGainMode(manual bool) error {
	if !r.initDone {
		return errors.New("r82xx: Init not called")
	}
	r.manual = manual
	// LNA gain mode: reg 0x05 bit 4 clear = AGC; set = manual.
	lnaBit := byte(0x00)
	if manual {
		lnaBit = 0x10
	}
	if err := r.writeRegMask(0x05, lnaBit, 0x10); err != nil {
		return err
	}
	// Mixer gain mode: reg 0x07 bit 4. Same polarity as LNA.
	mixBit := byte(0x00)
	if manual {
		mixBit = 0x10
	}
	if err := r.writeRegMask(0x07, mixBit, 0x10); err != nil {
		return err
	}
	return nil
}

// ----------------------------------------------------------------------
// PLL synthesis

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
	if vcoFineTune > r82xxVCOPowerRef && divNum > 0 {
		divNum--
	} else if vcoFineTune < r82xxVCOPowerRef {
		divNum++
	}
	if err := r.writeRegMask(0x10, divNum<<5, 0xE0); err != nil {
		return err
	}
	vcoFreq := uint64(freqHz) * uint64(mixDiv)
	pllRef := uint64(r.xtalHz)
	nint := uint32(vcoFreq / (2 * pllRef))
	vcoFra := uint32((vcoFreq - 2*pllRef*uint64(nint)) / 1000)
	if nint > 0x3F+13 {
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
	pllRefkHz := r.xtalHz / 1000
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
// burst writes (address byte followed by data bytes) wrapped in a
// single I2C-repeater on/off pair.
//
// Data is split into chunks of at most r82xxBurstMaxData bytes to
// mirror librtlsdr's r82xx_write (NMAX_WRITES = 16). Going beyond
// that limit stalls the very first multi-byte OUT on some NESDR v5
// dongles — observed as libusb EPIPE on the 27-byte init flood
// (issue #248). Each chunk is its own control-OUT under the same
// repeater session; the register pointer advances by the chunk
// length between chunks, which matches the chip's auto-increment.
func (r *R82xx) writeBurstRaw(addr uint8, data []byte) error {
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return err
	}
	defer r.demod.SetI2CRepeater(false)
	for pos := 0; pos < len(data); {
		size := len(data) - pos
		if size > r82xxBurstMaxData {
			size = r82xxBurstMaxData
		}
		buf := make([]byte, 1+size)
		buf[0] = addr + uint8(pos)
		copy(buf[1:], data[pos:pos+size])
		if err := r.demod.I2CWrite(r.i2cAddr, buf); err != nil {
			return err
		}
		pos += size
	}
	return nil
}

// readRegRaw reads n bytes from the chip starting at addr 0. The
// chip auto-increments so a single read returns regs 0x00..0x00+n-1.
// Bytes are bit-reversed on the wire; we un-reverse before returning.
// The result is also stored into the shadow so callers querying via
// the cache see fresh values for the read-only block.
func (r *R82xx) readRegRaw(addr uint8, n int) ([]byte, error) {
	if err := r.demod.SetI2CRepeater(true); err != nil {
		return nil, err
	}
	defer r.demod.SetI2CRepeater(false)
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
