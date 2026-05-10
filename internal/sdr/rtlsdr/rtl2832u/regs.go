// Package rtl2832u is the pure-Go register / I2C-bridge layer that sits
// between the platform USB transport (internal/sdr/rtlsdr/usb) and the
// per-tuner drivers. It mirrors osmocom librtlsdr's src/librtlsdr.c
// register and demodulator access primitives, with no behavior change
// on the wire — every control transfer that leaves a [Demod] matches
// what the C library would have sent under the same call.
//
// The package is consumed by:
//
//   - internal/sdr/rtlsdr/tuners (R820T/R820T2/R828D, E4000, FC0012,
//     FC0013, FC2580) — for I2C reads and writes against the tuner
//     IC behind the RTL2832U's I2C bridge.
//   - internal/sdr/rtlsdr/driver.go (PR-06) — for baseband init,
//     sample-rate / IF-frequency / PPM programming, FIR coefficients,
//     bias-tee GPIO, and reset-buffer bookkeeping.
//
// No production code outside the rtlsdr driver tree should depend on
// this package.
package rtl2832u

// USB-block IDs used by [Demod.ReadBlockReg] / [Demod.WriteBlockReg].
// The chip exposes seven memory-mapped "blocks"; we send the block
// number in the high byte of wIndex (low byte carries the read/write
// direction flag).
const (
	BlockDemod uint8 = 0 // page-addressed demod registers (use ReadDemodReg/WriteDemodReg)
	BlockUSB   uint8 = 1 // USB controller registers (FIFO, EP config)
	BlockSys   uint8 = 2 // system registers (DEMOD_CTL, GPO, GPI, GPOE, GPD)
	BlockTuner uint8 = 3 // tuner-direct access (legacy; we use the I2C bridge instead)
	BlockROM   uint8 = 4 // boot ROM
	BlockIR    uint8 = 5 // IR receiver block (unused for SDR)
	BlockIIC   uint8 = 6 // I2C bridge to the tuner IC
)

// USB-block register addresses.
const (
	USBSysctl    uint16 = 0x2000
	USBCtrl      uint16 = 0x2010
	USBStat      uint16 = 0x2014
	USBEpaCfg    uint16 = 0x2144
	USBEpaCtl    uint16 = 0x2148
	USBEpaMaxpkt uint16 = 0x2158
)

// System-block register addresses.
const (
	SysDemodCtl  uint16 = 0x3000
	SysGPO       uint16 = 0x3001
	SysGPI       uint16 = 0x3002
	SysGPOE      uint16 = 0x3003 // GPIO output enable
	SysGPD       uint16 = 0x3004 // GPIO direction
	SysDemodCtl1 uint16 = 0x300B
)

// DefaultXtalHz is the RTL2832U's default reference-crystal frequency
// (28.8 MHz). Some clones use 28.7715 MHz or other values; user-level
// PPM correction handles small offsets.
const DefaultXtalHz uint32 = 28_800_000

// CtrlTimeoutMs is the default per-transfer timeout for register
// reads/writes. Matches osmocom librtlsdr's CTRL_TIMEOUT.
const CtrlTimeoutMs = 300

// twoPow22 = 2^22; the RTL2832U's IF-frequency and sample-rate
// dividers use 28.4 fixed-point math seeded against this constant.
const twoPow22 = 1 << 22

// MinSampleRateHz / MaxSampleRateHz bound the range librtlsdr accepts.
// The hardware can theoretically run lower/higher but the resampler
// produces garbage outside these limits.
const (
	MinSampleRateHz uint32 = 225_001
	MaxSampleRateHz uint32 = 3_200_000

	// The RTL2832U has a forbidden sample-rate gap between 300 kS/s
	// and 900 kS/s where the resampler exhibits known artefacts;
	// librtlsdr rejects these values, so we do too.
	GapLowHz  uint32 = 300_000
	GapHighHz uint32 = 900_000
)

// IsValidSampleRate is the same predicate librtlsdr uses to gate
// rtlsdr_set_sample_rate inputs.
func IsValidSampleRate(hz uint32) bool {
	if hz <= MinSampleRateHz-1 || hz > MaxSampleRateHz {
		return false
	}
	if hz > GapLowHz && hz <= GapHighHz {
		return false
	}
	return true
}
