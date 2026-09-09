package motorola

import "strings"

// BandPlan maps a SmartNet 10-bit channel number (an OSW.Command
// value that falls inside the plan) to its downlink frequency, and
// classifies which command values ARE channel numbers — the load-
// bearing discriminator of the whole OSW state machine, since a
// SmartNet grant is just "group address + channel number" with no
// explicit opcode.
//
// Formulas ported from trunk-recorder's SmartnetParser::get_freq /
// is_chan (which mirror the long-standing gr-smartnet / mottrunk.txt
// tables). All arithmetic is integer Hz.
type BandPlan interface {
	// Frequency returns the downlink frequency for a channel
	// number, or ok=false when the value is not a channel in this
	// plan.
	Frequency(ch uint16) (uint32, bool)
	// IsChannel reports whether the command value is a voice/control
	// channel number in this plan.
	IsChannel(ch uint16) bool
	// Name returns the config-facing plan name.
	Name() string
}

// ParseBandPlan maps the `motorola_band_plan` config string to a
// BandPlan. Empty selects the 800 MHz standard (domestic) plan —
// by far the most common SmartNet deployment. ok=false flags an
// unrecognised name (the caller warn-logs and falls back to the
// default).
func ParseBandPlan(s string) (BandPlan, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "800", "800_standard", "800_domestic":
		return plan800{name: "800_standard"}, true
	case "800_reband", "800_rebanded":
		return plan800{name: "800_rebanded", rebanded: true}, true
	case "800_splinter":
		return plan800{name: "800_splinter", splinter: true}, true
	case "900":
		return plan900{}, true
	default:
		return plan800{name: "800_standard"}, false
	}
}

// plan800 covers the domestic 800 MHz plans: standard, rebanded and
// splinter. The 866–869 MHz segments are shared by all three.
type plan800 struct {
	name     string
	rebanded bool
	splinter bool
}

func (p plan800) Name() string { return p.name }

func (p plan800) Frequency(ch uint16) (uint32, bool) {
	const spacing = 25_000
	// Shared upper segments (all 800 MHz variants).
	switch {
	case ch >= 0x2D0 && ch <= 0x2F7:
		return 866_000_000 + spacing*uint32(ch-0x2D0), true
	case ch >= 0x32F && ch <= 0x33F:
		return 867_000_000 + spacing*uint32(ch-0x32F), true
	case ch >= 0x3C1 && ch <= 0x3FE:
		return 867_425_000 + spacing*uint32(ch-0x3C1), true
	case ch == 0x3BE:
		return 868_975_000, true
	}
	switch {
	case p.rebanded:
		if ch <= 0x1B7 {
			return 851_012_500 + spacing*uint32(ch), true
		}
		if ch >= 0x1B8 && ch <= 0x22F {
			return 851_025_000 + spacing*uint32(ch-0x1B8), true
		}
	case p.splinter:
		if ch <= 0x257 {
			return 851_000_000 + spacing*uint32(ch), true
		}
		if ch >= 0x258 && ch <= 0x2CF {
			return 866_012_500 + spacing*uint32(ch-0x258), true
		}
	default:
		if ch <= 0x2CF {
			return 851_012_500 + spacing*uint32(ch), true
		}
	}
	return 0, false
}

func (p plan800) IsChannel(ch uint16) bool {
	_, ok := p.Frequency(ch)
	return ok
}

// plan900 is the 900 MHz plan: 935.0125 MHz + 12.5 kHz × channel.
type plan900 struct{}

func (plan900) Name() string { return "900" }

func (plan900) Frequency(ch uint16) (uint32, bool) {
	if ch <= 0x1DE {
		return 935_012_500 + 12_500*uint32(ch), true
	}
	return 0, false
}

func (p plan900) IsChannel(ch uint16) bool {
	_, ok := p.Frequency(ch)
	return ok
}
