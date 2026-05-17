package rtl2832u

import "fmt"

// firDefault are the 20-tap FIR coefficients librtlsdr ships for DAB/FM
// reception (the DVB-T variant is unused for SDR). First 8 entries are
// 8-bit signed; entries 8..15 are 12-bit signed and pack into 12 bytes
// on the wire (see SetFIR for the packing math).
var firDefault = [20]int16{
	-54, -36, -41, -40, -32, -14, 14, 53, // 8-bit signed taps
	101, 156, 220, 298, 401, 532, 686, 689, // 12-bit signed taps (12-bit limit)
	// the last 4 entries are unused (kept for table symmetry with the C library)
	0, 0, 0, 0,
}

// initBasebandStep is one transfer in the InitBaseband golden sequence.
// Captured verbatim from librtlsdr's rtlsdr_init_baseband — the
// constants live here so the unit tests can replay the exact same
// sequence against the mock transport.
type initBasebandStep struct {
	demod bool   // true → ReadDemodReg/WriteDemodReg, false → ReadBlockReg/WriteBlockReg
	block uint8  // when !demod
	page  uint8  // when demod
	addr  uint16 // demod uses 1-byte addresses (cast from uint8 in the C source)
	val   uint16
	n     int
}

// initBasebandSteps is the fixed sequence rtlsdr_init_baseband sends
// to the chip after open. Order is load-bearing: the FIR upload sits
// between the demod soft-reset and the SDR-mode enable.
var initBasebandSteps = []initBasebandStep{
	// USB init
	{block: BlockUSB, addr: USBSysctl, val: 0x09, n: 1},
	{block: BlockUSB, addr: USBEpaMaxpkt, val: 0x0002, n: 2},
	{block: BlockUSB, addr: USBEpaCtl, val: 0x1002, n: 2},
	// Power on demod
	{block: BlockSys, addr: SysDemodCtl1, val: 0x22, n: 1},
	{block: BlockSys, addr: SysDemodCtl, val: 0xE8, n: 1},
	// Demod soft-reset (bit 3) + release
	{demod: true, page: 1, addr: 0x01, val: 0x14, n: 1},
	{demod: true, page: 1, addr: 0x01, val: 0x10, n: 1},
	// Disable spectrum inversion / adjacent-channel rejection
	{demod: true, page: 1, addr: 0x15, val: 0x00, n: 1},
	{demod: true, page: 1, addr: 0x16, val: 0x0000, n: 2},
	// Clear DDC shift + IF frequency registers
	{demod: true, page: 1, addr: 0x16, val: 0x00, n: 1},
	{demod: true, page: 1, addr: 0x17, val: 0x00, n: 1},
	{demod: true, page: 1, addr: 0x18, val: 0x00, n: 1},
	{demod: true, page: 1, addr: 0x19, val: 0x00, n: 1},
	{demod: true, page: 1, addr: 0x1A, val: 0x00, n: 1},
	{demod: true, page: 1, addr: 0x1B, val: 0x00, n: 1},
	// FIR coefficients land at page 1, addr 0x1C..0x2F (20 single-byte writes).
	// SetFIRDefault expands the firDefault table into the corresponding writes
	// — they're not enumerated here so the test stays robust to FIR table
	// changes; SetFIR's own test pins the byte-packing math.
	// SDR mode, DAGC off (bit 5)
	{demod: true, page: 0, addr: 0x19, val: 0x05, n: 1},
	// FSM state-holding registers
	{demod: true, page: 1, addr: 0x93, val: 0xF0, n: 1},
	{demod: true, page: 1, addr: 0x94, val: 0x0F, n: 1},
	// Disable demod AGC (en_dagc bit 0)
	{demod: true, page: 1, addr: 0x11, val: 0x00, n: 1},
	// Disable RF + IF AGC loop
	{demod: true, page: 1, addr: 0x04, val: 0x00, n: 1},
	// Disable PID filter
	{demod: true, page: 0, addr: 0x61, val: 0x60, n: 1},
	// Default ADC I/Q datapath (opt_adc_iq = 0)
	{demod: true, page: 0, addr: 0x06, val: 0x80, n: 1},
	// Zero-IF mode + DC cancellation + IQ estimation/compensation
	{demod: true, page: 1, addr: 0xB1, val: 0x1B, n: 1},
	// Disable 4.096 MHz clock output on TP_CK0
	{demod: true, page: 0, addr: 0x0D, val: 0x83, n: 1},
}

// InitBaseband walks the librtlsdr initialization sequence: power on
// the demod, soft-reset, load FIR coefficients, enable SDR mode, and
// configure zero-IF + IQ compensation. Idempotent on re-open.
func (d *Demod) InitBaseband() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	// Steps before the FIR upload.
	for i := 0; i < 15; i++ {
		if err := d.runInitStep(initBasebandSteps[i]); err != nil {
			return fmt.Errorf("init baseband step %d: %w", i, err)
		}
	}
	// Default FIR (20 single-byte writes at page 1, addr 0x1C..0x2F).
	if err := d.setFIRLocked(firDefault); err != nil {
		return fmt.Errorf("init baseband: FIR upload: %w", err)
	}
	for i := 15; i < len(initBasebandSteps); i++ {
		if err := d.runInitStep(initBasebandSteps[i]); err != nil {
			return fmt.Errorf("init baseband step %d: %w", i, err)
		}
	}
	return nil
}

func (d *Demod) runInitStep(s initBasebandStep) error {
	if s.demod {
		return d.writeDemodRegLocked(s.page, s.addr, s.val, s.n)
	}
	return d.writeBlockRegLocked(s.block, s.addr, s.val, s.n)
}

// DeinitBaseband is the inverse of InitBaseband — power down the demod
// so the next consumer of the USB device sees a clean state. Today
// this is just clearing DEMOD_CTL; librtlsdr's rtlsdr_deinit_baseband
// is similarly minimal.
func (d *Demod) DeinitBaseband() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.writeBlockRegLocked(BlockSys, SysDemodCtl, 0x20, 1)
}

// WarmupUSBSysctl issues the single USB-block write that librtlsdr's
// rtlsdr_open uses as a dummy-write probe immediately after claiming
// the interface — if it returns EPIPE the caller is expected to
// libusb_reset_device and retry. The wire bytes match initBasebandSteps[0]
// on purpose: librtlsdr also repeats the write inside the full init
// flood, and the chip is happy to receive it twice. Kept separate so
// the open path can run it with reset-on-EPIPE recovery without
// dragging the whole baseband sequence into the retry.
func (d *Demod) WarmupUSBSysctl() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.writeBlockRegLocked(BlockUSB, USBSysctl, 0x09, 1)
}

// SetFIR uploads custom FIR filter coefficients. Layout matches
// rtlsdr_set_fir: first 8 entries are 8-bit signed (one byte each),
// entries 8..15 are 12-bit signed and pack three nibbles per pair —
// 24 bits per pair → 12 bytes total. Total wire footprint is 20
// single-byte demod writes at page 1, addr 0x1C..0x2F.
func (d *Demod) SetFIR(coeffs [20]int16) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.setFIRLocked(coeffs)
}

// SetFIRDefault reloads the librtlsdr default FIR (DAB/FM tuned).
func (d *Demod) SetFIRDefault() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.setFIRLocked(firDefault)
}

func (d *Demod) setFIRLocked(coeffs [20]int16) error {
	for i := 0; i < 8; i++ {
		if coeffs[i] < -128 || coeffs[i] > 127 {
			return fmt.Errorf("rtl2832u: FIR coeff[%d]=%d out of 8-bit range", i, coeffs[i])
		}
	}
	// Pack 8 × int16 into 8 wire bytes (8-bit signed).
	var fir [20]byte
	for i := 0; i < 8; i++ {
		fir[i] = byte(coeffs[i])
	}
	// Pack 4 pairs of 12-bit signed coeffs into 12 bytes (3 bytes per pair).
	for i := 0; i < 4; i++ {
		v0 := coeffs[8+i*2]
		v1 := coeffs[8+i*2+1]
		if v0 < -2048 || v0 > 2047 || v1 < -2048 || v1 > 2047 {
			return fmt.Errorf("rtl2832u: FIR coeff[%d..%d]=(%d,%d) out of 12-bit range", 8+i*2, 8+i*2+1, v0, v1)
		}
		fir[8+i*3] = byte(v0 >> 4)
		fir[8+i*3+1] = byte((v0 << 4) | ((v1 >> 8) & 0x0F))
		fir[8+i*3+2] = byte(v1)
	}
	for i := 0; i < 20; i++ {
		if err := d.writeDemodRegLocked(1, 0x1C+uint16(i), uint16(fir[i]), 1); err != nil {
			return fmt.Errorf("rtl2832u: FIR write %d: %w", i, err)
		}
	}
	return nil
}

// SetSampleRate programs the resampler divider for the requested rate.
// The chip can only land on rates that map exactly to its 28.4
// fixed-point divisor; the returned actualHz is the rate the resampler
// is actually running at (typically within 1 Hz of requested for the
// useful range, but the API exposes the truth so callers don't lie to
// their downstream DSP).
func (d *Demod) SetSampleRate(hz uint32) (actualHz uint32, err error) {
	if !IsValidSampleRate(hz) {
		return 0, fmt.Errorf("rtl2832u: sample rate %d Hz outside [%d, %d] (or in the [%d, %d] forbidden gap)", hz, MinSampleRateHz, MaxSampleRateHz, GapLowHz, GapHighHz)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.setSampleRateLocked(hz)
}

func (d *Demod) setSampleRateLocked(hz uint32) (uint32, error) {
	rsampRatio, realHz := computeResamplerDivisor(d.xtal, hz)
	// Top 16 bits of the divisor → page 1, addr 0x9F (2 bytes).
	if err := d.writeDemodRegLocked(1, 0x9F, uint16(rsampRatio>>16), 2); err != nil {
		return 0, err
	}
	// Bottom 16 bits → page 1, addr 0xA1 (2 bytes).
	if err := d.writeDemodRegLocked(1, 0xA1, uint16(rsampRatio&0xFFFF), 2); err != nil {
		return 0, err
	}
	d.rate = realHz
	// Apply the current PPM correction against the new rate.
	if err := d.setSampleFreqCorrectionLocked(d.ppm); err != nil {
		return 0, err
	}
	// Soft-reset the demod so the new divisor takes effect immediately.
	if err := d.writeDemodRegLocked(1, 0x01, 0x14, 1); err != nil {
		return 0, err
	}
	if err := d.writeDemodRegLocked(1, 0x01, 0x10, 1); err != nil {
		return 0, err
	}
	return realHz, nil
}

// computeResamplerDivisor is the 28.4 fixed-point math librtlsdr uses
// to translate a target sample rate into the chip's divisor register.
// Exposed for the golden-table test in this package; production code
// goes through SetSampleRate which calls this and writes the result.
func computeResamplerDivisor(xtalHz, sampHz uint32) (divisor, realHz uint32) {
	ratio := uint64(xtalHz) * twoPow22 / uint64(sampHz)
	ratio &= 0x0FFF_FFFC
	realRatio := ratio | ((ratio & 0x0800_0000) << 1)
	realHz = uint32(uint64(xtalHz) * twoPow22 / realRatio)
	return uint32(ratio), realHz
}

// SetSampleFreqCorrection applies a parts-per-million bias against the
// reference crystal. Negative ppm makes the chip run slow; positive
// makes it run fast. Math matches rtlsdr_set_sample_freq_correction.
func (d *Demod) SetSampleFreqCorrection(ppm int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.setSampleFreqCorrectionLocked(ppm)
}

func (d *Demod) setSampleFreqCorrectionLocked(ppm int) error {
	d.ppm = ppm
	offs := int16(int64(ppm) * -1 * (1 << 24) / 1_000_000)
	if err := d.writeDemodRegLocked(1, 0x3F, uint16(offs&0xFF), 1); err != nil {
		return err
	}
	return d.writeDemodRegLocked(1, 0x3E, uint16((offs>>8)&0x3F), 1)
}

// SetIFFreq programs the intermediate-frequency offset (Hz). The
// effective value is `-freq * 2^22 / xtal`, split into 22 bits across
// three demod registers (page 1, addr 0x19..0x1B).
func (d *Demod) SetIFFreq(freqHz uint32) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.setIFFreqLocked(freqHz)
}

func (d *Demod) setIFFreqLocked(freqHz uint32) error {
	ifFreq := -int32(uint64(freqHz) * twoPow22 / uint64(d.xtal))
	d.ifHz = ifFreq
	if err := d.writeDemodRegLocked(1, 0x19, uint16((ifFreq>>16)&0x3F), 1); err != nil {
		return err
	}
	if err := d.writeDemodRegLocked(1, 0x1A, uint16((ifFreq>>8)&0xFF), 1); err != nil {
		return err
	}
	return d.writeDemodRegLocked(1, 0x1B, uint16(ifFreq&0xFF), 1)
}

// ResetBuffer clears the USB FIFO so the next bulk-IN stream starts at
// a fresh buffer boundary. Always called immediately before
// StartBulkIn — matches rtlsdr_reset_buffer's two-write sequence.
func (d *Demod) ResetBuffer() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.writeBlockRegLocked(BlockUSB, USBEpaCtl, 0x1002, 2); err != nil {
		return err
	}
	return d.writeBlockRegLocked(BlockUSB, USBEpaCtl, 0x0000, 2)
}
