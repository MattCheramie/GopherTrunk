// Package tuners houses the per-chip tuner drivers that sit between the
// RTL2832U register layer (internal/sdr/rtlsdr/rtl2832u) and the
// top-level [sdr.Device]. Each supported tuner IC — R820T family, E4000,
// FC0012/13, FC2580 — implements the same [Tuner] interface so the
// driver layer (PR-06) can detect-and-dispatch without per-tuner
// special cases bleeding upwards.
//
// All I2C traffic goes through the RTL2832U's I2C bridge; the tuner
// drivers never touch USB directly. This is identical to how
// osmocom librtlsdr structures its tuner_*.c sources.
package tuners

import "fmt"

// Type enumerates the tuner ICs the rewrite plan covers. Values
// match the librtlsdr enum positions to ease cross-referencing
// against the C source.
type Type int

const (
	TypeUnknown Type = iota
	TypeR820T
	TypeR820T2
	TypeR828D
	TypeE4000  // landed in PR-07
	TypeFC0012 // landed in PR-07
	TypeFC0013 // landed in PR-07
	TypeFC2580 // landed in PR-07
)

// String returns the marketing name of the tuner, matching the
// strings librtlsdr's rtlsdr_get_tuner_type returns. The current
// driver surfaces this via sdr.Info.TunerName.
func (t Type) String() string {
	switch t {
	case TypeR820T:
		return "R820T"
	case TypeR820T2:
		return "R820T2"
	case TypeR828D:
		return "R828D"
	case TypeE4000:
		return "E4000"
	case TypeFC0012:
		return "FC0012"
	case TypeFC0013:
		return "FC0013"
	case TypeFC2580:
		return "FC2580"
	default:
		return "unknown"
	}
}

// Tuner is the contract every per-chip driver implements. The
// top-level [sdr.Device] composes one [Tuner] with the rtl2832u.Demod
// and dispatches public API calls accordingly.
//
// Implementations are not required to be goroutine-safe — the driver
// serializes calls via its own mutex.
type Tuner interface {
	// Type returns the detected tuner chip family.
	Type() Type
	// IFFreqHz is the intermediate frequency the demod should be
	// programmed to (via rtl2832u.Demod.SetIFFreq) for this tuner.
	// R820T family uses 3.57 MHz; E4000 uses 0 (zero-IF); the
	// FC-series each have their own value.
	IFFreqHz() uint32
	// Init brings the tuner up after RTL2832U baseband init. Writes
	// the chip's power-on register flood and configures default
	// gain / bandwidth.
	Init() error
	// Standby puts the tuner in low-power mode; reversible by Init.
	Standby() error
	// Close releases any tuner-private state. Implementations
	// should call Standby first as a courtesy.
	Close() error
	// SetFreq tunes the LO so the demod sees an IF-offset version
	// of the requested center frequency. Returns an error if the
	// PLL can't lock or if hz is out of the chip's supported range.
	SetFreq(hz uint32) error
	// SetBandwidth picks the IF filter that matches the requested
	// occupied bandwidth (typically the same as the sample rate).
	// Pass 0 to let the driver pick a sensible default.
	SetBandwidth(hz uint32) error
	// SetGain sets the manual-mode gain in tenths of dB. The driver
	// quantizes to the nearest value on the chip's gain ladder
	// (returned by Gains). Pass -1 to leave the current value alone
	// — useful when only SetGainMode is changing.
	SetGain(tenthDB int) error
	// SetGainMode flips between automatic (true) and manual (false)
	// gain control.
	SetGainMode(manual bool) error
	// Gains returns the discrete gain ladder, in tenths of dB,
	// supported by this chip. The slice is sorted ascending.
	Gains() []int
}

// ErrUnsupportedFreq is returned by [Tuner.SetFreq] when the requested
// frequency is outside the chip's PLL range.
type ErrUnsupportedFreq struct {
	Hz       uint32
	MinHz    uint32
	MaxHz    uint32
	TunerStr string
}

func (e *ErrUnsupportedFreq) Error() string {
	return fmt.Sprintf("tuners: %s cannot tune %d Hz (range %d..%d)", e.TunerStr, e.Hz, e.MinHz, e.MaxHz)
}
