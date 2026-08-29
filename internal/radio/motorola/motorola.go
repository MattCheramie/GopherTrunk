// Package motorola decodes Motorola Type II / SmartNet / SmartZone
// trunked control channels. The control channel transmits 84-bit
// frames (8-bit sync + 76 coded bits) at 3600 baud over ~±1.2 kHz
// binary FSK; each frame's payload deinterleaves and error-corrects
// to one 27-bit Outbound Status Word (OSW): a 16-bit address, a
// group/individual bit and a 10-bit command. Grants, system-ID
// broadcasts and extended functions span one to three consecutive
// OSWs.
//
// The wire format and OSW semantics are ported from OP25's
// rx_smartnet and trunk-recorder's SmartnetParser — implementations
// proven against live systems (issue #1143 established that the
// package's original spec-derived framing matched no real signal).
//
// Layout:
//
//	frame.go     Sync word, 76-bit payload codec (interleave ↔
//	             convolutional-parity ECC ↔ CRC-10 ↔ field
//	             inversion), OSW encode/decode.
//	osw.go       The 27-bit OSW model + command constants and
//	             talkgroup/status-flag accessors.
//	bandplan.go  Channel number → frequency for the 800 MHz
//	             standard / rebanded / splinter and 900 MHz plans.
//	control.go   Multi-OSW sequencer ingesting OSWs and emitting
//	             cc.locked / grant events on the bus.
//	process.go   Bit-stream framer: sync-bracketed frame capture
//	             feeding the payload codec and the sequencer.
//
// The IQ → bit front end lives in the receiver subpackage
// (internal/radio/motorola/receiver), wired to this package by
// ccdecoder.newMotorolaPipeline. Voice on SmartNet is analog FM on
// the granted channel; the composer's analog-trunk chain records it.
//
// Not yet wired: Type I fleet/subfleet decoding, OBT (VHF/UHF custom
// band plans), digital (ASTRO) voice channels, and the long tail of
// extended-function OSWs — all layer on top of the OSW sequencer.
package motorola

// BitSink consumes the raw stream of bits a Motorola receiver
// decodes from IQ. baseIdx is the absolute bit index of bits[0]
// across the stream lifetime — monotonically non-decreasing across
// calls, and reset to 0 by Receiver.Reset so a retune produces a
// fresh baseline. The control channel is 2-level FSK, so bits arrive
// one per symbol; the 4-level trunked protocols use a DibitSink
// instead.
type BitSink func(bits []byte, baseIdx int)
