package phase1

import "github.com/MattCheramie/GopherTrunk/internal/radio/p25/motorola"

// The Motorola talker-alias per-byte cipher and its 256-byte
// substitution table now live in the shared internal/radio/p25/motorola
// package so the Phase 1 (LDU1 / TDULC link control) and Phase 2
// (FACCH-S MAC) decode paths share one implementation. decodeAliasBytes
// is kept as a thin package-local alias so existing phase1 call sites
// and tests are unchanged.
func decodeAliasBytes(encoded []byte) []byte {
	return motorola.DecodeAliasBytes(encoded)
}
