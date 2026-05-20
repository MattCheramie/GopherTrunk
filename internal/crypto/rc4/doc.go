// Package rc4 implements the RC4 stream cipher (also known as ARC4,
// "alleged RC4") as a keystream generator.
//
// GopherTrunk uses it to decode DMR voice traffic protected by
// ARC4-based "Enhanced Privacy" when the operator already holds the
// key. The Go standard library's crypto/rc4 is deprecated — it raises
// staticcheck SA1019 at every call site, which `make lint` rejects —
// so this small, dependency-free implementation is vendored in its
// place. It is pinned against the canonical RC4 test vectors in
// rc4_test.go so codebook drift stays detectable.
//
// RC4 is cryptographically broken and must never be used to PROTECT
// data. It exists here only to DECODE third-party transmissions whose
// key the operator is authorized to hold — the same known-key model
// SDRTrunk, DSD-FME and OP25 use. No key recovery is performed.
package rc4
