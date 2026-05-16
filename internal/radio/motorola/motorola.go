// Package motorola decodes Motorola Type II / SmartZone trunked control
// channels. The SmartZone control channel transmits 84-bit Outbound
// Status Words (OSWs) at 3600 baud over an MSK-modulated carrier;
// after sync detection and BCH(64,16,11) error correction, each OSW
// reduces to 32 information bits split as a 16-bit address and a
// 16-bit command/opcode field.
//
// What's wired here:
//
//	sync.go      Standard Motorola control-channel sync words and a
//	             sliding correlator over a bit stream.
//	osw.go       OSW assemble/parse over the 32 information bits.
//	opcodes.go   Opcode constants + per-opcode payload accessors
//	             (Group Voice Channel Grant, Adjacent Site,
//	             System ID, Idle).
//	bandplan.go  Logical Channel Number → frequency (Hz) resolver
//	             with linear and table-backed strategies.
//	control.go   Control-channel state machine ingesting OSWs and
//	             emitting cc.locked / grant events on the bus.
//
// What's NOT yet wired (honest deferrals so a contributor can pick
// these up):
//
//   - The MSK demodulator that turns IQ from a 3600-baud control
//     channel into the bits this package consumes. The DSP package
//     has FM / C4FM / H-DQPSK; MSK is a special-case CPFSK that fits
//     in the same shape.
//   - BCH(64,16,11) decoder for the OSW FEC. The information-bit
//     parser here assumes the upstream caller has already corrected
//     errors.
//   - Type I sub-band decoding. Type II is the modern variant and
//     the focus here; Type I support layers on top.
//   - Vendor-specific opcode coverage. The opcode list focuses on
//     the SmartZone subset that carries voice grants and system
//     identification.
//
// All of the above slot into the existing engine + recorder + composer
// pipeline through events.KindGrant once they ship.
package motorola
