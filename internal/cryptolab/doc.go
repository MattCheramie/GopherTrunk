// Package cryptolab is GopherTrunk's optional cryptographic-research
// toolkit: a set of byte-oriented analysis and brute-force tools for the
// kinds of obfuscation and weak/keyless ciphers found in RF traffic. It
// ships statistical triage (entropy/IC plus autocorrelation period
// detection), a NIST SP 800-22 randomness battery, an obfuscation-class
// classifier, keyspace brute force, LFSR/keystream analysis,
// keystream-reuse / many-time-pad recovery, CRC parameter recovery, analog
// voice descrambling, and a pluggable "subject" framework for studying
// specific byte-oriented obfuscators.
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
// These are research tools for security testing. The toolkit actively
// attempts to break captured encryption — the assess harness runs every
// applicable method (keystream reuse, known-plaintext recovery, default/weak
// keys against the real ADP/DES/AES ciphers, keystream-LFSR prediction) and
// grades each one's effectiveness, because attempting decryption IS the
// security test. A complete decryption means the deployment FAILED the test;
// recovering nothing means it held.
//
// Honesty about limits: the toolkit cannot conjure a strong key out of a
// strong cipher. AES/DES with a non-default key and correctly-rotated IVs has
// an infeasible keyspace, and the assess harness reports that plainly as
// RESISTANT rather than pretending otherwise. What it does break is the things
// that actually fail in the field: reused IVs, default/test keys, keyless
// obfuscation, and structurally weak keystreams.
package cryptolab

// SubcommandName is the gophertrunk subcommand under which the toolkit is
// exposed (`gophertrunk cryptolab <verb>`). It is referenced from the
// always-linked CLI usage text so the command name is documented even in a
// default build that does not link the toolkit itself.
const SubcommandName = "cryptolab"
