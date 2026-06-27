// Package cryptolab is GopherTrunk's optional cryptographic-research
// toolkit: a set of byte-oriented analysis and brute-force tools aimed at
// the kinds of obfuscation and weak/keyless ciphers found in RF trunking
// traffic (analog voice inversion, LFSR scramblers, CRC/FEC framing, and
// the Motorola P25 talker-alias obfuscator).
//
// # Clean-room
//
// Everything here is derived only from observed data and from public
// structural descriptions. No GPL decoder source (SDRTrunk / Trunk
// Recorder) was read or ported; the toolkit is Apache-2.0 like the rest of
// GopherTrunk. Nothing in this package flips motorola.CipherVerified — the
// alias decode in internal/radio/p25/motorola stays gated until a real
// alias is confirmed end-to-end.
//
// # Optional at install
//
// The toolkit is gated out of the default gophertrunk binary. The CLI
// wiring in cmd/gophertrunk is split into a //go:build cryptolab pair: the
// default build links a stub for the "cryptolab" subcommand that prints how
// to rebuild with the toolkit; a build with -tags cryptolab links the real
// dispatch. The generic engine and subject packages under this directory
// carry no build tag so they always compile and are exercised by the test
// suite, but they are only linked into the shipped binary when the operator
// opts in (see the Makefile's test-cryptolab target and `make build
// TAGS=cryptolab`).
//
// # Honesty
//
// These are research tools. They recover obfuscation and weak/keyless
// constructions and triage whether a captured payload is even breakable;
// they make no claim to break strong keyed encryption (P25 DES-OFB / AES,
// ADP/RC4). For those, the toolkit offers known-plaintext keystream
// extraction and breakability triage only.
package cryptolab

// SubcommandName is the gophertrunk subcommand under which the toolkit is
// exposed (`gophertrunk cryptolab <verb>`). It is referenced from the
// always-linked CLI usage text so the command name is documented even in a
// default build that does not link the toolkit itself.
const SubcommandName = "cryptolab"
