// Package dpmr decodes dPMR (digital PMR446 / ETSI TS 102 658) Mode 3
// trunking signalling. dPMR is a 4-level FSK protocol designed for
// 6.25 kHz channel spacing — the digital successor to analogue PMR446
// in Europe and a sibling of NXDN in framing philosophy. Three
// operating modes are defined:
//
//   Mode 1   peer-to-peer (no infrastructure)
//   Mode 2   managed direct (no repeater, but a dispatch channel)
//   Mode 3   centralised trunking (a dedicated control channel that
//            grants voice / data calls onto traffic channels)
//
// This package targets Mode 3 — the only mode where the standard
// "see CC opcode → retune Voice device → follow grant" loop applies.
// Modes 1 and 2 don't have a control channel for the engine to hunt.
//
// What this package gives you:
//
//   sync.go      Frame Sync 1 / 2 / 3 constants and a tolerant
//                SyncDetector matching the shape used by the other
//                trunked-protocol packages.
//   csbk.go      CSBK (Common Signalling Block) parser — 80-bit
//                signalling unit with a 5-bit message type, source
//                and destination IDs, and opcode-specific fields.
//   opcodes.go   MessageType enum + per-message accessors
//                (VoiceServiceAllocation, IndividualCall,
//                DataServiceAllocation, Status, Release, …).
//   bandplan.go  Channel-number → Hz resolver, linear and table.
//   control.go   State machine that ingests CSBKs and publishes
//                events.KindCCLocked / events.KindGrant on the bus
//                with `trunking.Grant.Protocol = "dpmr"`.
//
// What's NOT yet wired (honest deferrals):
//
//   - The 4FSK demodulator + symbol-clock recovery for dPMR's
//     2400 sym/sec air interface. internal/dsp/demod/c4fm.go is the
//     closest fit; matched-filter parameters differ slightly.
//   - The interleaver + FEC over CSBK bits. Mode 3 CSBKs use a
//     short-block cyclic code with rate-3/4 convolutional outer
//     coding; the parsing here assumes the upstream caller has
//     already corrected errors.
//   - Voice frame extraction → AMBE+2 vocoder. The vocoder lives
//     behind the `mbelib` build tag (see internal/voice/mbelib).
//
// As with the other trunking packages: ship a clean structured
// surface now, leave the analogue / FEC pieces as named follow-ups
// so the trunking engine can consume the events end-to-end against
// fixtures.
package dpmr
