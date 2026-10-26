// Package sdr defines the abstract Device interface for IQ sources and the
// pool that supervises a fleet of dongles. Concrete drivers (RTL-SDR, mock,
// future HackRF/Airspy) live in subpackages and register themselves here.
package sdr

import (
	"context"
	"fmt"
)

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
	// RoleDedicated marks a dongle a single-channel decoder section
	// (aprs / ais / dsc / adsb / paging / m17 / mdc1200 / fleetsync /
	// lora) pins to one frequency for the daemon's lifetime. The pool
	// never hands it out by role: FirstByRole / AllByRole for control or
	// voice skip it, so the trunking voice pool, the conventional
	// scanner's spare-SDR heuristics, a live hunt and rigctld cannot
	// retune it (#1316). Assigned automatically from Hint.Dedicated;
	// `role: dedicated` may also be written explicitly.
	RoleDedicated
)

func (r Role) String() string {
	switch r {
	case RoleControl:
		return "control"
	case RoleVoice:
		return "voice"
	case RoleWideband:
		return "wideband"
	case RoleDedicated:
		return "dedicated"
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
	case "dedicated":
		return RoleDedicated
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

// FPGADCBlocker is an optional Device extension implemented by the
// HackRF Pro, whose gateware can strip the zero-IF DC-offset spike in
// the FPGA before the samples leave the device. Callers type-assert for
// it (like NarrowbandFilterer). Enabling it is an alternative to the
// software DC-block on the decode path — cheaper, and it also cleans the
// control channel, which the software block doesn't touch.
type FPGADCBlocker interface {
	SetFPGADCBlock(enable bool) error
}

// RFAmper is an optional Device extension implemented by backends with a
// switchable front-end RF amplifier (the HackRF). Callers type-assert for
// it (like FPGADCBlocker) so backends without an amp need not implement it.
// Enabling it lowers the noise figure for weak signals at the cost of ~14 dB
// of gain ahead of the front end, so it is opt-in per device.
type RFAmper interface {
	SetRFAmp(enable bool) error
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

// StreamEnder is an optional Device extension that reports why the most
// recent StreamIQ channel closed on its own: the remote end hung up, went
// silent, or a read failed. It returns nil while the stream runs and after
// the caller cancelled its context. A closed channel alone cannot tell a
// network source's lost session from a normal end, so callers that run until
// the stream stops use StreamErr to say what happened.
type StreamEnder interface {
	StreamErr() error
}

// StreamDeadCauser is an optional Device extension a USB driver implements to
// report the error that killed its most recent bulk-IN stream (nil when the
// stream ended cleanly or is still running).
type StreamDeadCauser interface {
	StreamDeadCause() error
}

// StreamErr says why dev's most recent StreamIQ channel closed on its own:
// its StreamErr when it implements StreamEnder (network sources), else the
// USB stream-death cause when it implements StreamDeadCauser (local
// dongles; issue #1256, where a phone's dying dongle printed only "IQ stream
// ended"), else nil.
func StreamErr(dev Device) error {
	if se, ok := dev.(StreamEnder); ok {
		return se.StreamErr()
	}
	if dc, ok := dev.(StreamDeadCauser); ok {
		if err := dc.StreamDeadCause(); err != nil {
			return fmt.Errorf("USB stream died: %w", err)
		}
	}
	return nil
}

// Driver is the factory each backend exposes.
type Driver interface {
	Name() string
	Enumerate() ([]Info, error)
	Open(idx int) (Device, error)
}

// ProbeOpener is an optional Driver extension offering a fast,
// read-only device open for `sdr list --probe`. Callers type-assert for
// it (like FreqRanger / TunerDiagnoser) and fall back to Open when a
// driver doesn't implement it.
//
// OpenProbe must bring the device up cheaply and touch ONLY that device
// — in particular it must not run any multi-pass device-reset recovery,
// because on macOS a reset re-enumerates the USB device and probing one
// dongle would then perturb a sibling on the same host controller (issue
// #1135). A device that doesn't come up on the first pass simply yields
// the empty Info fields from Enumerate; the daemon Open path owns the
// reset+retry recovery that actually getting a dongle streaming needs.
type ProbeOpener interface {
	OpenProbe(idx int) (Device, error)
}
