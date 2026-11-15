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

// FECMode selects how the Process adapter interprets the bits after
// each frame sync match.
//
//   - FECOn (the default): the 660-bit on-air FEC section, decoded by
//     framing.DecodeDStarHeaderFEC. This is what every D-STAR radio
//     sends.
//   - FECOff: the 41 header bytes unencoded (328 bits, each byte LSB
//     first). No radio sends that; it exists for input that has been
//     de-FEC'd elsewhere.
type FECMode uint8

const (
	FECOn FECMode = iota
	FECOff
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
// Recognised values (case-insensitive): "" / "on" / "true" / "1" →
// FECOn (the default, the on-air format); "off" / "false" / "0" →
// FECOff. Unknown strings return FECOn with `ok = false` so callers
// can surface the misconfiguration.
func ParseFECMode(s string) (FECMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "on", "true", "1":
		return FECOn, true
	case "off", "false", "0":
		return FECOff, true
	default:
		return FECOn, false
	}
}
