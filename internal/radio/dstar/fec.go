package dstar

import (
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// encodeFECForTest mirrors framing.EncodeDStarHeaderFEC under a
// package-local name so test helpers can synthesize FEC-encoded
// fixtures without importing the framing package directly.
// Returns 660 channel bits on success, nil on bad input.
func encodeFECForTest(info []byte) []byte {
	return framing.EncodeDStarHeaderFEC(info)
}

// FECMode selects how the Process adapter interprets the bit stream
// it slices after each Frame Sync match.
//
//   - FECOff: the adapter reads HeaderInfoBits (328) bits straight
//     off the wire and parses them as the 41-byte PCH header.
//     Useful for synthesized fixtures whose conv encoder + scrambler
//
//   - interleaver layers aren't applied. Default — matches the
//     in-package fixture path and operator captures that have been
//     pre-decoded out of band.
//
//   - FECOn: the adapter reads framing.DStarHeaderChannelBits (660)
//     bits, runs the full JARL DV-mode chain (deinterleave →
//     descramble → depuncture → K=5 Viterbi → 328 info bits), and
//     parses the recovered information field as the 41-byte header.
//     This is the path that lights up on a live-air capture.
type FECMode uint8

const (
	FECOff FECMode = iota
	FECOn
)

// FECOnHeaderBits is the on-wire bit count the Process adapter
// collects per Frame Sync match under FECOn. Mirrors
// framing.DStarHeaderChannelBits.
const FECOnHeaderBits = framing.DStarHeaderChannelBits

// SetFECMode toggles the full JARL DV-mode FEC chain on the Process
// adapter. See FECMode for the trade-offs.
func (c *ControlChannel) SetFECMode(mode FECMode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fecMode = mode
}

// FECMode returns the configured FECMode. Mirrors the Set* family so
// callers (and tests) can introspect the configured mode without
// poking at unexported state.
func (c *ControlChannel) FECMode() FECMode {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fecMode
}

// ParseFECMode maps a config / user-facing string into an FECMode.
// Recognised values (case-insensitive): "" / "off" / "false" / "0"
// → FECOff (the default — read 328 info bits straight off the wire);
// "on" / "true" / "1" → FECOn (full chain). Unknown strings return
// FECOff with `ok = false` so callers can surface the
// misconfiguration.
func ParseFECMode(s string) (FECMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "false", "0":
		return FECOff, true
	case "on", "true", "1":
		return FECOn, true
	default:
		return FECOff, false
	}
}
