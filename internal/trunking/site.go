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
	ProtocolYSF                // System Fusion (C4FM, amateur trunked variant — config "ysf")
	ProtocolDStar              // D-STAR (GMSK 4800 bps, amateur — header-only repeater protocol; config "dstar")
	ProtocolDMRTier2           // DMR Tier II conventional (per-repeater; config "dmr-tier2")
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
	case ProtocolYSF:
		return "ysf"
	case ProtocolDStar:
		return "dstar"
	case ProtocolDMRTier2:
		return "dmr-tier2"
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
	case "ysf":
		return ProtocolYSF, nil
	case "dstar", "d-star", "d_star":
		return ProtocolDStar, nil
	case "dmr-tier2", "dmr_tier2", "dmr-t2", "dmrtier2":
		return ProtocolDMRTier2, nil
	default:
		return ProtocolUnknown, fmt.Errorf("trunking: unknown protocol %q "+
			"(want p25|p25-phase2|dmr|dmr-tier2|nxdn|dpmr|edacs|motorola|ltr|mpt1327|tetra|ysf|dstar)", s)
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
	// Zero is valid only for BSCH (§8.2.5.2). For all other channel
	// types the colour code is the per-cell secret the descrambler
	// needs to recover the type-3 stream — leaving it at zero with
	// channel coding on produces garbage. Bits 30..31 are silently
	// ignored downstream.
	TETRAColourCode uint32
	// TETRAChannel selects which TETRA logical channel lives in each
	// burst window under ChannelCodingOn. Recognised values:
	// "sch/hd" | "sch/f" | "sch/hu" | "bsch" | "aach" (case-insensitive,
	// "/" optional). Empty defaults to "sch/hd" — the most common
	// signaling carrier for cc.locked / Grant events. Forwarded into
	// tetra.ControlChannel.SetExpectedChannel by the ccdecoder
	// connector after parsing via tetra.ParseChannelType.
	TETRAChannel string
	// TETRAChannelCoding gates the full ETSI EN 300 392-2 §8.3.1
	// channel-coding chain (descramble + deinterleave + depuncture +
	// Viterbi + CRC-16 verify + tail strip). Recognised values
	// (case-insensitive): "" / "on" / "true" / "1" → ChannelCodingOn
	// (the new default; required for live on-air captures); "off" /
	// "false" / "0" → ChannelCodingOff (legacy raw-dibit path, opt-out
	// for operators feeding pre-stripped DSD-FME / OP25 fixtures).
	// Forwarded into tetra.ControlChannel.SetChannelCoding by the
	// ccdecoder connector after parsing via tetra.ParseChannelCoding.
	TETRAChannelCoding string

	// LTRFCSMode enables CRC-7 FCS verification on the LTR Status
	// Ingest path (per DSheirer/sdrtrunk's CRCLTR.java layout).
	// Recognised values (case-insensitive): "" / "on" / "true" / "1" →
	// FCSOn (the new default; drop Status words whose 7-bit FCS
	// trailer doesn't match the CRC over the 24-bit message vector);
	// "off" / "false" / "0" → FCSOff (no verification — opt-out for
	// pre-stripped fixtures). Forwarded into
	// ltr.ControlChannel.SetFCSMode by the ccdecoder connector after
	// parsing via ltr.ParseFCSMode.
	LTRFCSMode string
	// LTRManchesterMode controls Manchester decoding of the LTR
	// sub-audible bit stream. Recognised values (case-insensitive):
	// "" / "on" / "soft" → ManchesterSoft (the new default —
	// majority-decode each pair; matches the dominant on-air
	// encoding for sub-audible LTR signaling); "strict" —
	// require a mid-bit transition per pair, drop transition-less
	// pairs; "off" / "nrz" → ManchesterOff (raw NRZ — opt-out for
	// synthesized NRZ fixtures). Forwarded into
	// ltr.ControlChannel.SetManchesterMode by the ccdecoder
	// connector after parsing via ltr.ParseManchesterMode.
	LTRManchesterMode string

	// P25Phase1DemodMode selects the symbol-recovery path for the
	// P25 Phase 1 receiver. Recognised values (case-insensitive):
	// "" / "c4fm" / "fm" → DemodC4FM (the default — FM
	// discriminator + 4-level slicer; matches every previously
	// shipping config and works on conventional non-simulcast P25
	// transmitters); "cqpsk" / "lsm" / "linear" → DemodCQPSK (the
	// linear / LSM path — complex RRC + Gardner + differential
	// QPSK; required for simulcast P25 deployments whose control
	// channel transmits Linear Simulcast Modulation rather than
	// straight C4FM, see issue #275 and TIA-102.BAAA). Forwarded
	// into p25phase1rx.Options.DemodMode by the ccdecoder connector
	// after parsing via p25phase1rx.ParseDemodMode.
	P25Phase1DemodMode string

	// P25Phase2TrellisMode enables the 4-state ½-rate trellis FEC
	// decoder on the P25 Phase 2 MAC PDU window. Recognised values
	// (case-insensitive): "" / "on" / "true" / "1" → TrellisOn (the
	// new default — 146 channel dibits via the TIA-102.AABF trellis
	// decoder); "off" / "false" / "0" → TrellisOff (legacy 72-dibit
	// raw-MAC-PDU path, opt-out for pre-stripped fixtures). Forwarded
	// into p25phase2.ControlChannel.SetTrellisMode by the ccdecoder
	// connector after parsing via p25phase2.ParseTrellisMode.
	P25Phase2TrellisMode string
	// P25Phase2RSMode enables the outer Reed-Solomon RS(24, 16, 9)
	// verification layer on top of the trellis-decoded MAC PDU.
	// Recognised values (case-insensitive): "" / "off" / "false" /
	// "0" → RSOff (the default — no outer RS verification; matches
	// historical decoder behaviour); "on" / "true" / "1" → RSOn
	// (verify RS syndromes per TIA-102.BAAA-A §5.9; drop MAC PDUs
	// whose syndromes are non-zero before parsing). Forwarded into
	// p25phase2.ControlChannel.SetRSMode by the ccdecoder connector
	// after parsing via p25phase2.ParseRSMode.
	P25Phase2RSMode string
	// P25Phase2InterleaveMode enables the TIA-102.BBAC per-burst block
	// deinterleaver applied to the MAC-burst dibits before trellis
	// decoding. Recognised values (case-insensitive): "" / "off" /
	// "false" / "0" → InterleaveOff (the default); "on" / "true" / "1"
	// → InterleaveOn. Forwarded into p25phase2.ControlChannel.
	// SetInterleaveMode by the ccdecoder connector after parsing via
	// p25phase2.ParseInterleaveMode.
	P25Phase2InterleaveMode string
	// P25Phase2ScramblerMode enables the PN44 descrambler per
	// TIA-102.BBAC-1 §7.2.5 on the trellis-decoded MAC PDU bits.
	// Recognised values (case-insensitive): "" / "off" / "false" /
	// "0" → ScramblerOff (the default — no PN44 descrambling);
	// "on" / "true" / "1" → ScramblerOn. The seed is computed from
	// (WACN, SystemID, low 12 bits of Site as the spec's Color
	// Code = NAC) per spec equation (5). Forwarded into
	// p25phase2.ControlChannel.SetScramblerMode +
	// SetScramblerSeed by the ccdecoder connector.
	P25Phase2ScramblerMode string
	// P25Phase2ClockMode selects the symbol-timing-recovery strategy
	// for the P25 Phase 2 receiver. Recognised values (case-
	// insensitive): "" / "gardner" / "on" → ClockGardner (the new
	// default — non-data-aided Gardner loop; recommended for live
	// SDR captures); "naive" / "off" → ClockNaive (decimate every
	// sps-th sample; works on sample-aligned synthesized IQ).
	// Forwarded into p25phase2rx.Options.ClockMode by the ccdecoder
	// connector after parsing via p25phase2rx.ParseClockMode.
	P25Phase2ClockMode string
	// TETRAClockMode mirrors P25Phase2ClockMode for the TETRA
	// receiver. Same recognised values + parser semantics; the
	// underlying ClockMode enums in the two receivers share the
	// same name + values but are independent types.
	TETRAClockMode string
	// NXDNViterbiMode enables the K=5 ½-rate Viterbi FEC decoder
	// on the NXDN CAC region. Recognised values (case-insensitive):
	// "" / "spec" → ViterbiSpec (the new default — full NXDN-TS-1-A
	// §4.5.1.1 outbound CAC chain); "on" / "true" / "1" → ViterbiOn
	// (intermediate 92-dibit K=5 Viterbi path for older
	// MMDVMHost / DSDcc fixtures); "off" / "false" / "0" → ViterbiOff
	// (legacy 44-dibit raw-CAC path, opt-out for pre-stripped
	// fixtures). Forwarded into nxdn.ControlChannel.SetViterbiMode
	// by the ccdecoder connector after parsing via
	// nxdn.ParseViterbiMode.
	NXDNViterbiMode string
	// NXDNDeviationHz overrides the peak frequency deviation (Hz)
	// the NXDN receiver's slicer is calibrated against. Spec value
	// is 1800 Hz (matches the FM-discriminator output level so live
	// captures slice correctly out of the box). Operators with
	// on-air captures whose dibit distribution is bimodal (outer
	// ±3 dominate, inner ±1 underrepresented) can override here —
	// see samples/nxdn/README.md for the calibration recipe.
	// Forwarded into nxdnrx.Options.DeviationHz by the ccdecoder
	// connector; values <= 0 fall back to the spec default.
	NXDNDeviationHz float64
	// EDACSBCHMode enables the BCH(40, 28, 2) FEC layer on the
	// EDACS CCW. Recognised values (case-insensitive): "" / "on" /
	// "true" / "1" → BCHOn (the new default — 40-bit on-wire
	// BCH(40, 28, 2) decode + single/double-bit correction); "off" /
	// "false" / "0" → BCHOff (legacy pre-stripped 40-bit CCW path,
	// opt-out for pre-stripped fixtures). Forwarded into
	// edacs.ControlChannel.SetBCHMode by the ccdecoder connector
	// after parsing via edacs.ParseBCHMode.
	EDACSBCHMode string
	// MPT1327BCHMode enables the BCH(63, 38) FEC layer on the MPT
	// 1327 codeword. Recognised values (case-insensitive): "" /
	// "on" / "true" / "1" → BCHOn (the new default — 64-bit on-wire
	// BCH(63, 38) decode); "off" / "false" / "0" → BCHOff (legacy
	// 38-bit pre-stripped codeword path, opt-out for pre-stripped
	// fixtures). Forwarded into mpt1327.ControlChannel.SetBCHMode
	// by the ccdecoder connector after parsing via
	// mpt1327.ParseBCHMode.
	MPT1327BCHMode string
	// MPT1327CWSCTolerance sets the Hamming-distance threshold the
	// MPT 1327 Process adapter uses when matching the 16-bit
	// Codeword Synchronisation Code. Recognised values
	// (case-insensitive): "" → 2-bit tolerance (the new default,
	// matches commercial MPT 1327 receivers on noisy on-air
	// captures); "0" / "exact" / "off" → exact match (for
	// pre-stripped synthesized fixtures); a decimal integer in
	// [0, 15]. Forwarded into mpt1327.ControlChannel.SetCWSCTolerance
	// by the ccdecoder connector after parsing via
	// mpt1327.ParseCWSCTolerance.
	MPT1327CWSCTolerance string
	// MotorolaBCHMode enables the BCH(64, 16, 11) FEC layer on the
	// Motorola Type II OSW. Recognised values (case-insensitive):
	// "" / "on" / "true" / "1" → BCHOn (the new default — two
	// 64-bit BCH(64, 16, 11) codewords reassembled into the 32-bit
	// OSW, with up to 11 bit errors corrected per codeword); "off" /
	// "false" / "0" → BCHOff (legacy 32-bit raw-OSW path, opt-out
	// for pre-stripped fixtures). Forwarded into
	// motorola.ControlChannel.SetBCHMode by the ccdecoder
	// connector after parsing via motorola.ParseBCHMode.
	MotorolaBCHMode string
	// DStarFECMode enables the JARL DV-mode header FEC chain on the
	// D-STAR Process adapter. Recognised values (case-insensitive):
	// "" / "off" / "false" / "0" → FECOff (the default — reads 328
	// info bits straight off the wire, matches synthesized fixtures
	// + pre-FEC-stripped inputs); "on" / "true" / "1" → FECOn (660
	// on-wire bits → deinterleave 22×30 → PN15 descramble →
	// depuncture → K=5 R=1/2 Viterbi → 328 info bits → ParseHeader).
	// Forwarded into dstar.ControlChannel.SetFECMode by the
	// ccdecoder connector after parsing via dstar.ParseFECMode.
	DStarFECMode string
}

// Validate returns an error if the System lacks required fields.
func (s System) Validate() error {
	if s.Name == "" {
		return errors.New("trunking: system name is required")
	}
	if s.Protocol == ProtocolUnknown {
		return errors.New("trunking: protocol must be p25|p25-phase2|dmr|dmr-tier2|nxdn|dpmr|edacs|motorola|ltr|mpt1327|tetra|ysf|dstar")
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
