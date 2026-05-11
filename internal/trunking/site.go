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
	ProtocolUnknown   Protocol = iota
	ProtocolP25                // P25 Phase 1 (config "p25" — Phase 2 uses ProtocolP25Phase2)
	ProtocolDMR                // DMR Tier II / III
	ProtocolNXDN               // NXDN
	ProtocolDPMR               // dPMR Mode 3 (digital PMR446 trunking)
	ProtocolEDACS              // EDACS / GE-Marc
	ProtocolMotorola           // Motorola Type II / SmartZone
	ProtocolLTR                // Logic Trunked Radio (LTR / LTR-Net)
	ProtocolMPT1327            // MPT 1327 (UK / Commonwealth utility trunking)
	ProtocolP25Phase2          // P25 Phase 2 (H-DQPSK TDMA, config "p25-phase2")
	ProtocolTETRA              // TETRA TMO (π/4-DQPSK, ETSI EN 300 392-2)
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
	case ProtocolMPT1327:
		return "mpt1327"
	case ProtocolP25Phase2:
		return "p25-phase2"
	case ProtocolTETRA:
		return "tetra"
	default:
		return "unknown"
	}
}

// ParseProtocol maps a string ("p25", "dmr", "nxdn", "dpmr",
// "edacs", "motorola", "ltr", "mpt1327", "p25-phase2", "tetra") to
// a Protocol value.
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
	case "mpt1327":
		return ProtocolMPT1327, nil
	case "p25-phase2", "p25_phase2", "p25p2":
		return ProtocolP25Phase2, nil
	case "tetra":
		return ProtocolTETRA, nil
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

	// TETRAColourCode is the low 30 bits of the extended colour code
	// the TETRA scrambler uses to seed its LFSR per ETSI EN 300 392-2
	// §8.2.5 ("ec" in the spec). The ccdecoder connector forwards this
	// into tetra.ControlChannel.SetColourCode under ChannelCodingOn.
	// Zero is valid (BSCH always uses 0 per §8.2.5.2) but disables
	// channel coding because the colour code is the per-cell secret
	// the descrambler needs to recover the type-3 stream. Bits 30..31
	// are silently ignored downstream.
	TETRAColourCode uint32
	// TETRAChannel selects which TETRA logical channel lives in each
	// burst window under ChannelCodingOn. Recognised values:
	// "sch/hd" | "sch/f" | "sch/hu" | "bsch" | "aach" (case-insensitive,
	// "/" optional). Empty defaults to "sch/hd" — the most common
	// signaling carrier for cc.locked / Grant events. Forwarded into
	// tetra.ControlChannel.SetExpectedChannel by the ccdecoder
	// connector after parsing via tetra.ParseChannelType.
	TETRAChannel string

	// LTRFCSMode enables CRC-7 FCS verification on the LTR Status
	// Ingest path (per DSheirer/sdrtrunk's CRCLTR.java layout).
	// Recognised values (case-insensitive): "" / "off" — no
	// verification (default, pre-PR #40 behaviour); "on" / "true" —
	// drop Status words whose 7-bit FCS trailer doesn't match the
	// CRC over the 24-bit message vector. Forwarded into
	// ltr.ControlChannel.SetFCSMode by the ccdecoder connector
	// after parsing via ltr.ParseFCSMode.
	LTRFCSMode string
	// LTRManchesterMode controls Manchester decoding of the LTR
	// sub-audible bit stream. Recognised values (case-insensitive):
	// "" / "off" / "nrz" — raw NRZ (default, in-package tests);
	// "strict" — require a mid-bit transition per pair, drop
	// transition-less pairs; "soft" / "on" — majority-decode each
	// pair and tolerate noise bursts. Live captures of sub-audible
	// LTR signaling should use "soft" (the dominant on-air encoding).
	// Forwarded into ltr.ControlChannel.SetManchesterMode by the
	// ccdecoder connector after parsing via ltr.ParseManchesterMode.
	LTRManchesterMode string

	// P25Phase2TrellisMode enables the 4-state ½-rate trellis FEC
	// decoder on the P25 Phase 2 MAC PDU window. Recognised values
	// (case-insensitive): "" / "off" → TrellisOff (legacy 72-dibit
	// raw-MAC-PDU path); "on" → TrellisOn (146 channel dibits via
	// the TIA-102.AABF trellis decoder). Forwarded into
	// p25phase2.ControlChannel.SetTrellisMode by the ccdecoder
	// connector after parsing via p25phase2.ParseTrellisMode.
	P25Phase2TrellisMode string
	// NXDNViterbiMode enables the K=5 ½-rate Viterbi FEC decoder
	// on the NXDN CAC region. Recognised values (case-insensitive):
	// "" / "off" → ViterbiOff (legacy 44-dibit raw-CAC path);
	// "on" → ViterbiOn (92 dibits via the K=5 Viterbi decoder).
	// Forwarded into nxdn.ControlChannel.SetViterbiMode by the
	// ccdecoder connector after parsing via nxdn.ParseViterbiMode.
	NXDNViterbiMode string
	// EDACSBCHMode enables the BCH(40, 28, 2) FEC layer on the
	// EDACS CCW. Recognised values (case-insensitive): "" / "off"
	// → BCHOff (legacy pre-stripped 40-bit CCW path); "on" → BCHOn
	// (40-bit on-wire BCH(40, 28, 2) decode + single/double-bit
	// correction). Forwarded into edacs.ControlChannel.SetBCHMode
	// by the ccdecoder connector after parsing via
	// edacs.ParseBCHMode.
	EDACSBCHMode string
	// MPT1327BCHMode enables the BCH(63, 38) FEC layer on the MPT
	// 1327 codeword. Recognised values (case-insensitive): "" /
	// "off" → BCHOff (legacy 38-bit pre-stripped codeword path);
	// "on" → BCHOn (64-bit on-wire BCH(63, 38) decode). Forwarded
	// into mpt1327.ControlChannel.SetBCHMode by the ccdecoder
	// connector after parsing via mpt1327.ParseBCHMode.
	MPT1327BCHMode string
}

// Validate returns an error if the System lacks required fields.
func (s System) Validate() error {
	if s.Name == "" {
		return errors.New("trunking: system name is required")
	}
	if s.Protocol == ProtocolUnknown {
		return errors.New("trunking: protocol must be p25|p25-phase2|dmr|nxdn|dpmr|edacs|motorola|ltr|mpt1327|tetra")
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
