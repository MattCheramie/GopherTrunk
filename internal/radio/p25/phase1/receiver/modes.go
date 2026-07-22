package receiver

import "strings"

// DemodMode selects how the receiver recovers symbols from the
// complex IQ stream.
//
//   - DemodC4FM (default) is the pre-existing path: FM
//     discriminator → real-valued RRC matched filter → 4-level
//     slicer. C4FM (4-level FSK) is the standard P25 Phase 1
//     baseband modulation and the correct choice for the large
//     majority of systems — including most simulcast systems, which
//     transmit C4FM across their multiple transmitters.
//
//   - DemodCQPSK is the linear-modulation path: complex RRC matched
//     filter → symbol-time decimation → differential QPSK decode.
//     It is for the minority of P25 Phase 1 systems that transmit a
//     *linear* modulation on the wire — Linear Simulcast Modulation
//     (LSM, TIA-102.BAAA), a π/4-DQPSK-family scheme, being the
//     common one. A linear signal pushed through a quadrature FM
//     discriminator produces near-random dibits and the FSW never
//     matches (issue #275).
//
// IMPORTANT — modulation is not implied by simulcast. LSM is a linear
// modulation that *some* operators choose (often for simulcast, because
// a linear waveform survives multi-transmitter overlap better), but
// simulcast does not imply LSM: many simulcast systems transmit plain
// C4FM. The modulation also cannot be read off licensing / emission-
// designator data (e.g. ACMA "10K1D7W" covers both C4FM and CQPSK).
// Determine it empirically: C4FM is the default, and DemodCQPSK is
// warranted only when a strong, clean signal will not lock in C4FM
// (near-random dibits, no FSW). See issue #935 — Victoria's MMR is a
// simulcast system that is C4FM on every site, and forcing CQPSK there
// kills the decode.
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

// String returns the canonical lowercase name for the mode, matching
// the strings ParseDemodMode accepts so log lines round-trip back to
// configuration values.
func (m DemodMode) String() string {
	switch m {
	case DemodC4FM:
		return "c4fm"
	case DemodCQPSK:
		return "cqpsk"
	default:
		return "unknown"
	}
}

// ParseDemodMode maps a config / user-facing string into a DemodMode.
// Recognised values (case-insensitive): "" / "c4fm" / "fm" → DemodC4FM
// (the default — matches every previously-shipping config and the
// receiver's pre-CQPSK behaviour); "cqpsk" / "lsm" / "linear" →
// DemodCQPSK (the linear-modulation path). The "lsm" alias names the
// linear modulation itself (LSM, TIA-102.BAAA); it is *not* a "this
// site is simulcast" switch — pick it only for systems actually
// transmitting a linear waveform, not because a system is simulcast
// (see the DemodMode doc and issue #935). Unknown strings return
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
