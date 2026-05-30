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

// SampleRateGetter is an optional interface a Device may implement to
// surface the rate the underlying hardware actually settled on after
// SetSampleRate. The Device contract's SetSampleRate returns only an
// error; backends like rtlsdr's RTL2832U resampler quantize to their
// fixed-point divisor, and the R820T2 silently caps requests over
// ~2.4 MHz, so the rate the driver acknowledged and the rate the chip
// is streaming at can differ. Downstream DSP (the conventional FM
// demod, the wideband channelizer, the CC decoder) must work against
// the truth — otherwise resampler ratios are off by a small constant
// and audio plays back at the wrong speed.
//
// Implementations should return 0 to signal "I don't know" (the
// helper [ActualSampleRate] then falls back to the operator-supplied
// requested rate).
type SampleRateGetter interface {
	SampleRate() uint32
}

// ActualSampleRate returns the device's reported sample rate when the
// device implements [SampleRateGetter] and reports a non-zero value;
// otherwise it falls back to requested. Use this everywhere the
// daemon plumbs cfg.SDR.SampleRate into DSP options — the demod
// chain, the channelizer, the CC decoder — so chains stay anchored
// to the chip's truth rather than the operator's wish.
func ActualSampleRate(dev Device, requested uint32) uint32 {
	if rg, ok := dev.(SampleRateGetter); ok {
		if got := rg.SampleRate(); got > 0 {
			return got
		}
	}
	return requested
}

// Driver is the factory each backend exposes.
type Driver interface {
	Name() string
	Enumerate() ([]Info, error)
	Open(idx int) (Device, error)
}
