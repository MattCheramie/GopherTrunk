package tuners

import (
	"errors"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr/rtl2832u"
)

// Detect walks the list of supported tuner chips and returns a ready
// [Tuner] for the first one it finds. Probe order is R820T → R828D →
// E4000 → FC0013 → FC0012 → FC2580, matching librtlsdr's
// rtlsdr_open detect flow. Wraps the entire probe in a single
// SetI2CRepeater(true)/(false) pair so unnecessary repeater toggles
// don't appear on the bus between candidate chips.
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
	if t := detectE4000(d); t != nil {
		return t, nil
	}
	if t := detectFC0013(d); t != nil {
		return t, nil
	}
	if t := detectFC0012(d); t != nil {
		return t, nil
	}
	if t := detectFC2580(d); t != nil {
		return t, nil
	}
	return nil, ErrNoTunerDetected
}

// ErrNoTunerDetected is returned by [Detect] when none of the
// supported tuner chips responded on their candidate I2C addresses.
// Typically signals an unsupported clone — the user can still open
// the device but won't be able to tune.
var ErrNoTunerDetected = errors.New("tuners: no supported tuner detected (R820T/R820T2/R828D/E4000/FC0012/FC0013/FC2580)")
