// Package voice provides the voice-decoding plumbing that sits between the
// trunking engine and the audio output / recording layer.
//
// The package is split per the licensing reality of digital-radio vocoders:
//
//   - vocoder.go    a Vocoder interface + thread-safe Registry. Default
//                   build registers only NullVocoder (silence) and any
//                   pure-Go IMBE implementation that lands in
//                   internal/voice/imbe.
//   - wav.go        16-bit PCM mono WAV writer with length-fields patched
//                   on Close (so the file is valid even if the daemon dies).
//   - recorder.go   subscribes to CallStart / CallEnd events from the
//                   trunking engine, opens a per-call WAV file (and an
//                   optional raw-frame sidecar) under a configurable
//                   directory tree, and exposes WritePCM / WriteRawFrame
//                   for the future demod pipeline to push samples into.
//
// AMBE+2 decoding (P25 Phase 2 / DMR / NXDN) lives behind the `mbelib`
// build tag and is intentionally absent from default builds. See
// docs/vocoders.md for the full licensing picture.
package voice
