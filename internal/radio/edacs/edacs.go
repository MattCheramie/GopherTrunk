// Package edacs decodes Enhanced Digital Access Communications System
// trunked control channels (also marketed as GE-Marc / Ericsson EDACS).
// EDACS uses a continuous 9600-baud GFSK control channel over which
// 40-bit Control Channel Words (CCWs) flow, prefaced by a 24-bit sync
// pattern and protected per CCW by a shortened BCH(40, 28, 2) code
// (`internal/radio/framing/bch_edacs.go`, gated by SetBCHMode).
//
// Files:
//
//	sync.go       Standard EDACS control-channel sync word + a
//	              sliding correlator over a bit stream.
//	ccw.go        CCW = {Command, Status, Address, LCN, Aux} packed
//	              into 40 bits. Round-trip assemble / parse helpers.
//	opcodes.go    Command enum + per-command payload accessors
//	              (Group Voice Grant, Data Grant, System ID,
//	              Adjacent Site, Idle).
//	bandplan.go   LCN → Hz resolver. Linear and table strategies,
//	              same shape as the Motorola package's resolver.
//	control.go    Control-channel state machine ingesting CCWs and
//	              emitting cc.locked / grant events on the bus. The
//	              SetBCHMode / ParseBCHMode opt-in turns the BCH(40,
//	              28, 2) decode on per system via the ccdecoder
//	              connector's edacs_bch_mode YAML key.
//
// What's NOT yet wired (honest deferrals so a contributor can pick
// these up):
//
//   - EDACS ProVoice / Aegis voice frames (the digital-voice variants
//     use proprietary AMBE-derived vocoders; see docs/vocoders.md).
//     Closest open reference is `lwvmobile/edacs-fm` (pv_*.c) and
//     `szechyjs/dsd`.
//   - Vendor-specific command extensions beyond the Standard subset
//     covered by Command. GE/Ericsson networked-EDACS extensions
//     would land here.
//
// Per the canonical open reference (`lwvmobile/edacs-fm`), the
// BCH(40, 28, 2) per CCW is the only on-wire FEC layer on the
// Standard EDACS control channel — earlier package comments that
// referenced an "interleaved Reed-Solomon-derived FEC" above the
// BCH were a documentation error.
//
// All of the above slot into the existing engine + recorder + composer
// pipeline through events.KindGrant once they ship.
package edacs
