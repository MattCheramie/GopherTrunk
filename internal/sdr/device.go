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
)

func (r Role) String() string {
	switch r {
	case RoleControl:
		return "control"
	case RoleVoice:
		return "voice"
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

// Driver is the factory each backend exposes.
type Driver interface {
	Name() string
	Enumerate() ([]Info, error)
	Open(idx int) (Device, error)
}
