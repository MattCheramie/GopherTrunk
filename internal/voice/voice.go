// Package voice provides the voice-decoding plumbing that sits between the
// trunking engine and the audio output / recording layer.
//
// The package layout:
//
//   - vocoder.go    a Vocoder interface + thread-safe Registry. Default
//     build registers NullVocoder (silence), the pure-Go
//     IMBE decoder from internal/voice/imbe, and the
//     pure-Go AMBE+2 decoder from internal/voice/ambe2.
//   - wav.go        16-bit PCM mono WAV writer with length-fields patched
//     on Close (so the file is valid even if the daemon dies).
//   - recorder.go   subscribes to CallStart / CallEnd events from the
//     trunking engine, opens a per-call WAV file (and an
//     optional raw-frame sidecar) under a configurable
//     directory tree, and exposes WritePCM / WriteRawFrame
//     for the demod pipeline to push samples into.
//
// IMBE patents have expired; AMBE+2 carries active patents in some
// jurisdictions and re-implementing it in pure Go does not change that
// posture. Operators in licence-restrictive jurisdictions should evaluate
// before deploying. See docs/vocoders.md for the full picture.
package voice
