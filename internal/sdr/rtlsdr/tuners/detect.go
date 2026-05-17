package tuners

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
)

// Detect walks the list of supported tuner chips and returns a ready
// [Tuner] for the first one it finds.
//
// Probe order matches librtlsdr's rtlsdr_open exactly:
//
//  1. R820T  (0x34) — chip-ID byte 0x69
//  2. R828D  (0x74) — chip-ID byte 0x69
//  3. GPIO5 setup + pulse high→low — resets the non-R820T tuners
//  4. FC2580 (0xAC) — chip-ID byte 0x56 (masked 0x7F)
//  5. GPIO4 output enable — powers FC0013/E4000
//  6. FC0013 (0xC6) — chip-ID byte 0xA3
//  7. E4000  (0xC8) — chip-ID byte 0x40
//  8. FC0012 (0xC6) — chip-ID byte 0xA1; if found, GPIO6 pulse high→low
//
// All I2C probes happen under a single SetI2CRepeater(true)/(false)
// bracket so the bridge isn't flapping between candidates. The GPIO
// writes are demod-block traffic and don't disturb the repeater state.
//
// On success the repeater is toggled OFF before return. The
// subsequent tuner.Init burst write re-asserts it via
// R82xx.writeBurstRaw's own SetI2CRepeater(true) — that fresh
// wire write is load-bearing on NESDR v5 silicon (issue #248):
// the chip needs the explicit "kick" to arm the I²C bridge for
// the next multi-byte OUT, even though the demod register already
// holds the on-value. A previous attempt to leave the repeater on
// across detect → tuner-init (PR #260) suppressed that toggle via
// the SetI2CRepeater cache and reproduced the EPIPE this code is
// meant to prevent.
//
// Returns ErrNoTunerDetected when no chip matches.
func Detect(d *rtl2832u.Demod) (Tuner, error) {
	if err := d.SetI2CRepeater(true); err != nil {
		return nil, fmt.Errorf("tuners: I2C repeater on: %w", err)
	}
	defer d.SetI2CRepeater(false)

	if t := detectR82xx(d); t != nil {
		return t, nil
	}

	// Reset the non-R820T tuners via GPIO5 before probing them.
	// librtlsdr does this as a chip-enable pulse — without it, some
	// dongles leave FC2580/FC0013 in an indeterminate state from the
	// previous session.
	if err := pulseGPIO(d, 5, true, false); err != nil {
		return nil, fmt.Errorf("tuners: GPIO5 reset pulse: %w", err)
	}

	if t := detectFC2580(d); t != nil {
		return t, nil
	}

	// FC0013 and E4000 share GPIO4 as their power-up enable in
	// librtlsdr's bring-up.
	if err := d.SetGPIOOutput(4); err != nil {
		return nil, fmt.Errorf("tuners: GPIO4 output: %w", err)
	}

	if t := detectFC0013(d); t != nil {
		return t, nil
	}
	if t := detectE4000(d); t != nil {
		return t, nil
	}

	if t := detectFC0012(d); t != nil {
		// FC0012 needs an additional GPIO6 reset pulse after I2C
		// detection (per librtlsdr rtlsdr_open).
		if err := pulseGPIO(d, 6, true, false); err != nil {
			return nil, fmt.Errorf("tuners: GPIO6 FC0012 reset: %w", err)
		}
		return t, nil
	}
	return nil, ErrNoTunerDetected
}

// pulseGPIO sets the given pin to output mode and drives it through
// the requested level sequence. Matches librtlsdr's rtlsdr_open dance
// (typically "high then low" to assert + release a reset line). The
// I2C repeater state is not affected — GPIO writes target the system
// block, not the demod's bridge register.
func pulseGPIO(d *rtl2832u.Demod, pin uint8, levels ...bool) error {
	if err := d.SetGPIOOutput(pin); err != nil {
		return err
	}
	for _, lvl := range levels {
		if err := d.SetGPIOBit(pin, lvl); err != nil {
			return err
		}
	}
	return nil
}

// ErrNoTunerDetected is returned by [Detect] when none of the
// supported tuner chips responded on their candidate I2C addresses.
// Typically signals an unsupported clone — the user can still open
// the device but won't be able to tune.
var ErrNoTunerDetected = errors.New("tuners: no supported tuner detected (R820T/R820T2/R828D/E4000/FC0012/FC0013/FC2580)")
