package ltr

// BitSink consumes the raw stream of bits an LTR receiver decodes
// from the sub-audible signalling layer. baseIdx is the absolute
// bit index of bits[0] across the stream lifetime — monotonically
// non-decreasing across calls, and reset to 0 by Receiver.Reset so
// a retune produces a fresh baseline. LTR is 2-level; the 4-level
// trunked protocols use a DibitSink instead. Wire this into a
// future ControlChannel.Process adapter (optional Manchester
// decode + 41-bit Status framing + StatusFromBits + Ingest) so the
// connector can drive the LTR per-repeater state machine on live
// IQ.
type BitSink func(bits []byte, baseIdx int)
