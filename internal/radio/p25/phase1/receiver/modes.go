package receiver

import "strings"

// DemodMode selects how the receiver recovers symbols from the
// complex IQ stream.
//
//   - DemodC4FM (default) is the pre-existing path: FM
//     discriminator → real-valued RRC matched filter → 4-level
//     slicer. Works on conventional non-simulcast P25 Phase 1
//     transmitters that use straight C4FM on the wire.
//
//   - DemodCQPSK is the linear-modulation path: complex RRC matched
//     filter → symbol-time decimation → differential QPSK decode.
//     This is the path operators on simulcast P25 sites need:
//     simulcast deployments commonly transmit Linear Simulcast
//     Modulation (LSM, TIA-102.BAAA), a CQPSK-shaped variant
//     designed to survive multi-transmitter overlap. LSM pushed
//     through a quadrature FM discriminator produces near-random
//     dibits and the FSW never matches — the failure mode reported
//     against issue #275 on the MMR system.
//
// The dibit value emitted by DemodCQPSK after the LSM remap matches
// the canonical TIA-102.BAAA convention SymbolToDibit produces from
// C4FM, so downstream code (FSW detect, NID parse, TSBK trellis) is
// demod-agnostic.
type DemodMode uint8

const (
	DemodC4FM DemodMode = iota
	DemodCQPSK
)

// ParseDemodMode maps a config / user-facing string into a DemodMode.
// Recognised values (case-insensitive): "" / "c4fm" / "fm" → DemodC4FM
// (the default — matches every previously-shipping config and the
// receiver's pre-CQPSK behaviour); "cqpsk" / "lsm" / "linear" →
// DemodCQPSK (the simulcast / LSM path). Unknown strings return
// DemodC4FM with ok=false so the caller can warn-log and fall back.
func ParseDemodMode(s string) (DemodMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "c4fm", "fm":
		return DemodC4FM, true
	case "cqpsk", "lsm", "linear":
		return DemodCQPSK, true
	default:
		return DemodC4FM, false
	}
}

// Clock-mode selection is intentionally not exposed on this
// receiver. The C4FM path uses Mueller-Müller (proven on Phase 1
// for years; well-suited to the real-valued FM-discriminator
// output) and the CQPSK path uses Gardner (mandatory — the LSM
// demod operates on complex IQ at the sample rate where naive
// every-sps-th decimation produces meaningless symbols at any
// non-trivial timing offset).
