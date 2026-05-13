// Package dvsi implements the DVSI USB-3000 / AMBE-3003 hardware
// vocoder backend.
//
// # Build tag
//
// The Vocoder, Transport, USB enumeration, and registration into
// voice.DefaultRegistry all live under the `dvsi` build tag. Default
// builds (`go build`, `go test`) compile only the patent-surface-free
// packet framing — VocoderName, the AMBE-3003 wire protocol encoder/
// decoder, and this package documentation — so nothing in the binary
// pulls in the DVSI codepath unless an operator has opted in.
//
//	go build -tags dvsi ./cmd/gophertrunk     # links the DVSI backend
//	go test  -tags dvsi ./internal/voice/dvsi # runs the tagged tests
//
// # Patent posture
//
// AMBE+2 decoding is patent-encumbered. The default GopherTrunk build
// ships the pure-Go ambe2.Decoder which produces real audio under the
// open-source-license posture documented in docs/vocoders.md. The
// DVSI backend exists for operators in jurisdictions where vendor-
// blessed AMBE+2 decode is required; it expects a connected DVSI
// USB-3000 (FTDI-FT2232H-based USB device, VID 0x0403 / PID 0x6010)
// or AMBE-3003 module reachable through a compatible FTDI transport.
//
// # Loopback mode (testing only)
//
// Options{LoopbackOnly: true} substitutes a software loopback for the
// USB transport so the AMBE-3003 wire protocol, factory plumbing, and
// voice.Vocoder interface conformance can be exercised in CI without a
// real chip. The loopback transport returns synthesized PCM-silence
// SpeechData packets; it does NOT decode AMBE+2 (the whole point of
// the DVSI backend is to outsource the patent surface to hardware).
// Operators must never enable LoopbackOnly outside tests — it would
// produce silence on a real call.
//
// # Registration
//
// Under `-tags dvsi` the package init() registers "dvsi" in
// voice.DefaultRegistry pointing at Open(DefaultOptions()). The
// recorder picks the factory by name from per-system config; if no
// USB device matches the configured VID/PID, Open returns
// ErrNoDevice and the recorder falls back to the operator's chosen
// non-DVSI vocoder.
package dvsi

// VocoderName is the registry key the recorder uses to select the
// DVSI backend. Lives outside the build-tagged Vocoder so callers can
// reference the name from non-DVSI builds (e.g. for config validation).
const VocoderName = "dvsi"
