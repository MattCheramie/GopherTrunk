// Package tetra decodes TETRA (Terrestrial Trunked Radio) Trunked-
// Mode Operation (TMO) signalling per ETSI EN 300 392-2. TETRA is a
// professional-mobile-radio standard widely deployed by European
// public-safety, transport, and military networks. Air interface:
// π/4-DQPSK at 18 ksym/sec, organised as a 4-slot TDMA frame with
// 14.167 ms slots, 18 frames per multiframe, and 60 multiframes per
// hyperframe.
//
// This package targets TMO — the centralised-trunking mode where a
// dedicated control channel grants voice / data calls onto traffic
// channels. The two non-trunked modes (DMO direct, and Repeater
// mode) don't have a CC for the engine to hunt and are out of scope.
//
// What this package gives you:
//
//	sync.go      Synchronisation burst sync words and a tolerant
//	             SyncDetector matching the shape used by the other
//	             trunked-protocol packages.
//	pdu.go       TETRA Layer-3 PDU parser — discriminator + PDU
//	             type + payload bytes, mirroring the L3 message
//	             framing across MLE / MM / CMCE / SDS sub-protocols.
//	cmce.go      CMCE (Circuit-Mode Control Entity) PDU accessors
//	             for the trunking-grant subset: D-CONNECT,
//	             D-RELEASE, plus SYSINFO broadcasts that identify
//	             the network. Extracts the assigned carrier
//	             number / timeslot / call identifier / encryption
//	             + emergency flags / source + destination SSIs.
//	bandplan.go  Carrier-number → Hz resolver, linear and table.
//	control.go   State machine that ingests PDUs and publishes
//	             events.KindCCLocked / events.KindGrant on the bus
//	             with `trunking.Grant.Protocol = "tetra"`.
//
// What's NOT yet wired (honest deferrals):
//
//   - The π/4-DQPSK demodulator + symbol-clock recovery for TETRA's
//     18 ksym/sec air interface. internal/dsp/demod/dqpsk.go is the
//     closest fit; matched-filter parameters differ from P25 Phase 2.
//   - TDMA timing recovery + slot demarcation. The 4-slot frame +
//     18-frame multiframe + 60-multiframe hyperframe scheduling
//     belongs in a separate sync stage.
//   - Lower-layer FEC: RCPC convolutional + reed-muller block coding
//     across the various logical channels (BSCH, AACH, SCH/F,
//     SCH/HD, BNCH). Parsing here assumes upstream FEC has corrected
//     errors.
//   - End-to-end air-interface encryption (TEA1/2/3/4) — TETRA voice
//     traffic on most operational networks is encrypted; the
//     `Encrypted` flag on a grant just records what the CC said.
//   - Voice frame extraction → AMBE+2 vocoder. The vocoder lives
//     in internal/voice/ambe2 (pure-Go, default-on).
//
// As with the other trunking packages: ship a clean structured
// surface now, leave the analogue / FEC / encryption pieces as named
// follow-ups so the trunking engine can consume the events
// end-to-end against fixtures.
package tetra
