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
//   - End-to-end air-interface encryption (TEA1/2/3/4) — TETRA voice
//     traffic on most operational networks is encrypted; the
//     `Encrypted` flag on a grant just records what the CC said. Encrypted
//     TCH/S fails the class-2 CRC and produces no decoded audio.
//
// Now wired (voice path): the composer's TETRA chain retunes to the granted
// carrier, demodulates it, and — via traffic.go — recovers each Normal
// Continuous Downlink Burst tagged with its TDMA timeslot (numbered from the
// synchronisation burst). TCH/S channel decode (§5.5: RCPC + interleave +
// class-2 CRC) turns each burst into 137-bit speech frames, which the
// clean-room ACELP vocoder (internal/voice/acelp, bit-exact to ETSI
// EN 300 395-2) renders to PCM. Each burst also carries its AACH downlink usage
// marker; the voice chain routes bursts by matching that marker to the granted
// call's, so up to four concurrent same-carrier calls decode into independent
// recordings.
//
// As with the other trunking packages: ship a clean structured
// surface now, leave the analogue / FEC / encryption pieces as named
// follow-ups so the trunking engine can consume the events
// end-to-end against fixtures.
package tetra
