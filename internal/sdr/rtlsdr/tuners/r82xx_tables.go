package tuners

// R820T-family constants and lookup tables. All values come from
// osmocom librtlsdr's src/tuner_r82xx.c — the C source is the canonical
// reference for what each magic number means. Comments here highlight
// only the bits that matter for porting; consult the C source for
// chip-internal details.

const (
	// r82xxI2CAddr is the bridged I2C address librtlsdr always uses
	// for R820T / R820T2. The R828D variant lives at 0x74 instead;
	// detection in r82xx.go picks the right address.
	r82xxI2CAddr uint8 = 0x34
	r828dI2CAddr uint8 = 0x74

	// r82xxIFFreqHz is the intermediate frequency the demod should
	// be programmed to when this tuner is bound.
	r82xxIFFreqHz uint32 = 3_570_000

	// r82xxNumRegs is the number of registers; addresses 0x00..0x04
	// are read-only chip ID + lock status; 0x05..0x1F are writable
	// and tracked by the shadow-register cache.
	r82xxNumRegs     = 32
	r82xxShadowStart = 0x05

	// r82xxXtalHz matches the RTL2832U reference crystal librtlsdr
	// drives the R820T from (28.8 MHz). Boards with non-standard
	// crystals can override via [R82xx.SetXtal].
	r82xxXtalHz uint32 = 28_800_000

	// vcoMin / vcoMax bound the R820T VCO range. The PLL math
	// picks a mixer divider (2/4/8/16/32/64) so freq*div lands
	// inside this window.
	r82xxVCOMin uint64 = 1_770_000_000
	r82xxVCOMax uint64 = 3_900_000_000

	// vcoPowerRef is the comparison threshold for fine-tuning
	// divNum based on the chip's VCO fine-tune status bits.
	r82xxVCOPowerRef = 2

	// r82xxBurstMaxData caps the data-byte count per I2C-bridge OUT
	// to the tuner. librtlsdr's r82xx_write uses NMAX_WRITES = 16 for
	// the same reason: some R820T2 dongles stall the very first
	// multi-byte burst when it exceeds 16 data bytes (issue #248).
	// The 1-byte register pointer prepended to each chunk is not
	// counted toward this limit, matching librtlsdr's behavior.
	r82xxBurstMaxData = 16

	// r82xxBurstRetryDelayMillis is the per-chunk EPIPE-recovery
	// settle delay inside writeBurstChunk. The post-PR-#262 trace on
	// NESDR v5 silicon (issue #248) shows the chip's USB firmware
	// NACK'ing the very first 17-byte I²C-bridge OUT while the
	// surrounding control transfers stay healthy — the endpoint is
	// not halted in the libusb sense, the I²C bridge inside the chip
	// just rejected the burst. 8 ms is long enough for the I²C bus
	// to drain prior PrepareDemod traffic and short enough to be
	// invisible in the happy path (retry never fires).
	r82xxBurstRetryDelayMillis = 8

	// r82xxBurstMinData is the smallest chunk size writeBurstRaw will
	// try before giving up its halving-on-EPIPE retry walk
	// (16 → 8 → 4). Going below 4 burns wire traffic without
	// distinguishing a true multi-byte FIFO threshold from any other
	// behavior — single-byte demod writes already succeed against the
	// post-EPIPE chip on issue #248's reproduction, so finding a "size
	// 1 works" branch doesn't tell us anything new. See
	// writeBurstRaw's halving loop for details.
	r82xxBurstMinData = 4

	// r82xxPostPrepDemodSettleMillis is a brief chip-settle window
	// after the I²C repeater opens at the top of R82xx.Init and
	// before the multi-byte burst goes out. PR #263's trace data
	// (issue #248) showed the burst EPIPEs on NESDR v5 silicon even
	// after the load-bearing SetI2CRepeater(true) wire write fired
	// and after a USBDEVFS_RESET. librtlsdr's rtlsdr_open has
	// natural latency (function-call boundary + libusb queue
	// management) between PrepareDemod's last demod-side write and
	// r82xx_init's first I²C-bridge OUT; gophertrunk's Go code runs
	// them back-to-back. 5 ms covers that gap.
	r82xxPostPrepDemodSettleMillis = 5
)

// r82xxInitArray is the 27-byte register flood that lands at addr
// 0x05..0x1F at the end of Init. Verbatim from osmocom's
// r82xx_init_array in tuner_r82xx.c.
var r82xxInitArray = [r82xxNumRegs - r82xxShadowStart]byte{
	0x83, 0x32, 0x75, // 0x05 .. 0x07
	0xc0, 0x40, 0xd6, 0x6c, // 0x08 .. 0x0b
	0xf5, 0x63, 0x75, 0x68, // 0x0c .. 0x0f
	0x6c, 0x83, 0x80, 0x00, // 0x10 .. 0x13
	0x0f, 0x00, 0xc0, 0x30, // 0x14 .. 0x17
	0x48, 0xcc, 0x60, 0x00, // 0x18 .. 0x1b
	0x54, 0xae, 0x4a, 0xc0, // 0x1c .. 0x1f
}

// r82xxBitRevTable is the byte-bit-reversal table the R820T uses to
// emit register reads. After issuing a read, every received byte must
// be passed through this table before consuming. Identical to
// osmocom's bit_reverse implementation (precomputed for speed).
var r82xxBitRevTable = [16]byte{
	0x0, 0x8, 0x4, 0xC, 0x2, 0xA, 0x6, 0xE,
	0x1, 0x9, 0x5, 0xD, 0x3, 0xB, 0x7, 0xF,
}

func r82xxBitReverse(b byte) byte {
	return (r82xxBitRevTable[b&0x0F] << 4) | r82xxBitRevTable[b>>4]
}

// r82xxGainsTenthDB is the discrete gain ladder, in tenths of a dB,
// the R820T family supports. Identical to librtlsdr's r820t_gains[].
// Surfaced through [Tuner.Gains].
var r82xxGainsTenthDB = []int{
	0, 9, 14, 27, 37, 77, 87, 125, 144, 157,
	166, 197, 207, 229, 254, 280, 297, 328, 338, 364,
	372, 386, 402, 421, 434, 439, 445, 480, 496,
}

// r82xxLNAGainSteps and r82xxMixerGainSteps are the per-stage gain
// LUTs. The total tuner gain is the sum of LNA + mixer + VGA stages;
// librtlsdr's r82xx_set_gain walks the LNA + mixer arrays in lockstep
// and picks the index whose cumulative gain best matches the target.
var (
	r82xxLNAGainSteps = []int{
		0, 9, 13, 40, 38, 13, 31, 22,
		26, 31, 26, 14, 19, 5, 35, 13,
	}
	r82xxMixerGainSteps = []int{
		0, 5, 10, 10, 19, 9, 10, 25,
		17, 10, 8, 16, 13, 6, 3, -8,
	}
)

// r82xxFilterBWTable maps register 0x0A (low nibble) → IF filter
// bandwidth in Hz. The driver picks the entry with bandwidth closest
// to (but not less than) the requested rate.
//
// Values follow librtlsdr's r82xx_set_bandwidth behavior: bits in
// register 0x0A select coarse bandwidth, bits in register 0x0B select
// fine-tune. We expose only the coarse table here; SetBandwidth's
// implementation handles the fine-tune nibble.
var r82xxFilterBWTable = []uint32{
	2_400_000, // coarse 0
	2_300_000, // coarse 1
	2_200_000, // coarse 2
	2_100_000, // coarse 3
	2_000_000, // coarse 4
	1_900_000, // coarse 5
	1_800_000, // coarse 6
	1_700_000, // coarse 7
	1_600_000, // coarse 8
	1_500_000, // coarse 9
	1_450_000, // coarse 10
	1_400_000, // coarse 11
	1_350_000, // coarse 12
	1_300_000, // coarse 13
	1_250_000, // coarse 14
	1_200_000, // coarse 15
}

// r82xxFreqRange is one entry in librtlsdr's "freq_ranges" table that
// chooses RF input + tracking-filter for a given center frequency.
// SetMux walks the table and applies the first row whose
// frequency-range contains the target hz.
type r82xxFreqRange struct {
	freqHz     uint32 // upper bound (inclusive) for which this row applies
	openD      byte   // reg 0x17 (low nibble) — input switch / open drain
	rfMux      byte   // reg 0x1A — RF mux configuration
	tfC        byte   // reg 0x1B — tracking-filter cap
	xtalCap20p byte   // reg 0x10 — xtal-cap selection bits
	xtalCap10p byte
	xtalCap0p  byte
}

// r82xxFreqRanges mirrors osmocom's table, lifted verbatim. Entries
// are scanned in order; the first whose freqHz ≥ target wins. The
// final entry's freqHz is INT_MAX so it always matches.
var r82xxFreqRanges = []r82xxFreqRange{
	{freqHz: 50_000_000, openD: 0x08, rfMux: 0x02, tfC: 0xDF, xtalCap20p: 0x02, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 55_000_000, openD: 0x08, rfMux: 0x02, tfC: 0xBE, xtalCap20p: 0x02, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 60_000_000, openD: 0x08, rfMux: 0x02, tfC: 0x8B, xtalCap20p: 0x02, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 65_000_000, openD: 0x08, rfMux: 0x02, tfC: 0x7B, xtalCap20p: 0x02, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 70_000_000, openD: 0x08, rfMux: 0x02, tfC: 0x69, xtalCap20p: 0x02, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 75_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x58, xtalCap20p: 0x02, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 80_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x44, xtalCap20p: 0x02, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 90_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x34, xtalCap20p: 0x01, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 100_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x24, xtalCap20p: 0x01, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 110_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x24, xtalCap20p: 0x01, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 120_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x14, xtalCap20p: 0x01, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 140_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x13, xtalCap20p: 0x01, xtalCap10p: 0x01, xtalCap0p: 0x00},
	{freqHz: 180_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x11, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	{freqHz: 220_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	{freqHz: 250_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	{freqHz: 280_000_000, openD: 0x00, rfMux: 0x02, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	{freqHz: 310_000_000, openD: 0x00, rfMux: 0x41, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	{freqHz: 450_000_000, openD: 0x00, rfMux: 0x41, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	{freqHz: 588_000_000, openD: 0x00, rfMux: 0x40, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	{freqHz: 650_000_000, openD: 0x00, rfMux: 0x40, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
	// Final fallback catches every higher frequency.
	{freqHz: ^uint32(0), openD: 0x00, rfMux: 0x40, tfC: 0x00, xtalCap20p: 0x00, xtalCap10p: 0x00, xtalCap0p: 0x00},
}
