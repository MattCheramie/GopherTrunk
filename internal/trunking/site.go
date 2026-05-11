// Package trunking holds the cross-protocol orchestration: System
// definitions, control-channel hunting, talkgroup priority, voice grant
// following, and (later) multi-site neighbor tracking.
package trunking

import (
	"errors"
	"fmt"
	"strings"
)

// Protocol is the trunking protocol family in use on a System.
type Protocol uint8

const (
	ProtocolUnknown  Protocol = iota
	ProtocolP25               // P25 Phase 1 / Phase 2
	ProtocolDMR               // DMR Tier II / III
	ProtocolNXDN              // NXDN
	ProtocolDPMR              // dPMR Mode 3 (digital PMR446 trunking)
	ProtocolEDACS             // EDACS / GE-Marc
	ProtocolMotorola          // Motorola Type II / SmartZone
	ProtocolLTR               // Logic Trunked Radio (LTR / LTR-Net)
)

func (p Protocol) String() string {
	switch p {
	case ProtocolP25:
		return "p25"
	case ProtocolDMR:
		return "dmr"
	case ProtocolNXDN:
		return "nxdn"
	case ProtocolDPMR:
		return "dpmr"
	case ProtocolEDACS:
		return "edacs"
	case ProtocolMotorola:
		return "motorola"
	case ProtocolLTR:
		return "ltr"
	default:
		return "unknown"
	}
}

// ParseProtocol maps a string ("p25", "dmr", "nxdn", "dpmr",
// "edacs", "motorola", "ltr") to a Protocol value.
func ParseProtocol(s string) (Protocol, error) {
	switch strings.ToLower(s) {
	case "p25":
		return ProtocolP25, nil
	case "dmr":
		return ProtocolDMR, nil
	case "nxdn":
		return ProtocolNXDN, nil
	case "dpmr":
		return ProtocolDPMR, nil
	case "edacs":
		return ProtocolEDACS, nil
	case "motorola":
		return ProtocolMotorola, nil
	case "ltr":
		return ProtocolLTR, nil
	default:
		return ProtocolUnknown, fmt.Errorf("trunking: unknown protocol %q", s)
	}
}

// System describes one trunked radio system the engine should track.
type System struct {
	Name            string
	Protocol        Protocol
	ControlChannels []uint32 // candidate CC frequencies in Hz
	WACN            uint32   // 20-bit Wide-Area Communication Network ID (P25)
	SystemID        uint16   // 12-bit system identifier (P25 SYSID)
	RFSS            uint8    // RF SubSystem ID (P25)
	Site            uint8    // Site ID
}

// Validate returns an error if the System lacks required fields.
func (s System) Validate() error {
	if s.Name == "" {
		return errors.New("trunking: system name is required")
	}
	if s.Protocol == ProtocolUnknown {
		return errors.New("trunking: protocol must be p25|dmr|nxdn|dpmr|edacs|motorola|ltr")
	}
	if len(s.ControlChannels) == 0 {
		return errors.New("trunking: at least one control_channel frequency is required")
	}
	for i, f := range s.ControlChannels {
		if f < 25_000_000 || f > 1_300_000_000 {
			return fmt.Errorf("trunking: control_channels[%d]=%d Hz outside 25-1300 MHz", i, f)
		}
	}
	return nil
}

// HuntOrder returns the candidate frequency list with `lastKnown` (if non-zero
// and present in ControlChannels) moved to the front. This biases the hunter
// toward the most-recently-locked CC, falling back to the configured order.
func (s System) HuntOrder(lastKnown uint32) []uint32 {
	if lastKnown == 0 {
		out := make([]uint32, len(s.ControlChannels))
		copy(out, s.ControlChannels)
		return out
	}
	out := make([]uint32, 0, len(s.ControlChannels))
	out = append(out, lastKnown)
	for _, f := range s.ControlChannels {
		if f != lastKnown {
			out = append(out, f)
		}
	}
	// If lastKnown wasn't actually in the list, drop it.
	for _, f := range s.ControlChannels {
		if f == lastKnown {
			return out
		}
	}
	return out[1:]
}
