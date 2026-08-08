// Package sdr defines the abstract Device interface for IQ sources and the
// pool that supervises a fleet of dongles. Concrete drivers (RTL-SDR, mock,
// future HackRF/Airspy) live in subpackages and register themselves here.
package sdr

import "context"

type Role int

const (
	RoleAuto Role = iota
	RoleControl
	RoleVoice
	// RoleWideband pins a dongle to a single configured centre
	// frequency. Several decoders share the IQ stream — each one is
	// tapped to a different repeater frequency inside the dongle's
	// IQ bandwidth via the internal/dsp/tuner package. Used to cover
	// a cluster of co-band conventional repeaters (e.g. several DMR
	// Tier II carriers around 453 MHz) with a single SDR.
	RoleWideband
)

func (r Role) String() string {
	switch r {
	case RoleControl:
		return "control"
	case RoleVoice:
		return "voice"
	case RoleWideband:
		return "wideband"
	default:
		return "auto"
	}
}

func ParseRole(s string) Role {
	switch s {
	case "control":
		return RoleControl
	case "voice":
		return RoleVoice
	case "wideband":
		return RoleWideband
	default:
		return RoleAuto
	}
}

// Info describes a discovered device, returned by drivers' enumeration.
type Info struct {
	Driver       string
	Index        int
	Serial       string
	Manufacturer string
	Product      string
	TunerName    string
	Gains        []int
}

// Device is the per-dongle handle. Implementations must be safe for the
// goroutines that call StreamIQ; concurrent SetCenterFreq during streaming
// is allowed (the underlying USB transport handles it).
type Device interface {
	Info() Info
	SetCenterFreq(hz uint32) error
	SetSampleRate(hz uint32) error
	SetGain(tenthDB int) error // -1 selects automatic gain control
	SetPPM(ppm int) error
	// SetBiasTee toggles the dongle's 5V bias-tee output (used to
	// power external LNAs through the antenna SMA). Devices without
	// the circuit silently no-op. Implementations should return nil
	// if the underlying driver doesn't model bias-tee at all.
	SetBiasTee(enable bool) error
	StreamIQ(ctx context.Context) (<-chan []complex64, error)
	Close() error
}

// NarrowbandFilterer is an optional Device extension implemented by the
// HackRF Pro, whose RF front-end carries a switchable narrowband
// anti-alias filter. Callers type-assert for it (like TunerDiagnoser)
// so backends without the filter need not implement it. Enabling it
// tightens adjacent-channel rejection for narrowband voice channels at
// the cost of usable bandwidth.
type NarrowbandFilterer interface {
	SetNarrowbandFilter(enable bool) error
}

// TunerDiagnoser is an optional Device extension that surfaces tuner
// detection state for boot-time diagnostics. Callers type-assert for it
// (like ActualSampleRate / SettleAfterRetune) so backends that don't
// model a tuner crystal need not implement it. xtalHz of 16 MHz on an
// R828D is the signature that RTL-SDR Blog V4 auto-detection missed and
// the LO is mistuned by ~1.8× (issue #264).
type TunerDiagnoser interface {
	TunerDiag() (tunerName string, blogV4, blogV4Lite bool, xtalHz uint32)
}

// BlogV4Forcer is an optional Device extension that forces RTL-SDR Blog
// V4 mode (28.8 MHz reference crystal + switched HF/VHF/UHF input bank)
// regardless of USB-string auto-detection. Callers type-assert for it.
// Used when a V4's EEPROM strings are blank/non-standard so the R828D
// otherwise stays on the 16 MHz crystal and mistunes (issue #264).
type BlogV4Forcer interface {
	SetBlogV4(lite bool) error
}

// FreqRanger is an optional Device extension that reports the dongle's
// inclusive tuning range in Hz — the lowest and highest center frequency
// the front-end can reach. Callers type-assert for it (like
// TunerDiagnoser) so a backend that cannot model a hard range need not
// implement it. The whole-device hunt sweep derives its band from here:
// a caller with no explicit -band asks the open device for [min,max] and
// sweeps that span end-to-end. Bounds are inclusive and minHz <= maxHz.
type FreqRanger interface {
	FreqRange() (minHz, maxHz uint32)
}

// Driver is the factory each backend exposes.
type Driver interface {
	Name() string
	Enumerate() ([]Info, error)
	Open(idx int) (Device, error)
}
