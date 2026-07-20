package lora

import "strings"

// Low Data Rate Optimization (LDRO). At high spreading factors the LoRa
// symbol becomes long enough that transmitter/receiver clock drift can
// smear the dechirp peak across the two least-significant chip bins over
// one symbol, corrupting the low bits. LDRO removes those two bits from
// use: the payload is carried in a reduced-rate mode with ppm = SF-2
// meaningful bits per symbol (the transmitted bin is always a multiple of
// four), so a +/-1 bin drift no longer changes the decoded value. This
// mirrors Semtech's SX127x `LowDataRateOptimize` bit.
//
// LDRO is not signalled in the explicit header — it is a property of the
// (SF, BW) pair the whole network agrees on. The Semtech-recommended rule
// enables it whenever the symbol time Ts = 2^SF / BW is at least 16 ms,
// i.e. SF11/SF12 at 125 kHz and SF12 at 250 kHz. GopherTrunk applies that
// rule automatically on both the modulator and the demodulator so a
// synthesized frame and a live capture agree without extra configuration;
// an operator can still force it on or off per sub-channel for a network
// that deviates from the recommendation.
//
// Scope note: LDRO here is the reduced-rate *payload*. The explicit header
// is always transmitted reduced-rate in Semtech's PHY too, but GopherTrunk
// still uses its compact self-consistent header layout, so the header's
// reduced-rate SF-2 encoding lands with the bit-exact header calibration
// (gated on captured golden vectors) rather than here. See header.go.

// LDROMode selects how Low Data Rate Optimization is decided for a
// sub-channel.
type LDROMode uint8

const (
	// LDROAuto applies the Semtech Ts >= 16 ms rule from (SF, BW). It is
	// the zero value so an unset mode auto-detects.
	LDROAuto LDROMode = iota
	// LDROOff forces the reduced-rate payload off regardless of SF/BW.
	LDROOff
	// LDROOn forces the reduced-rate payload on regardless of SF/BW.
	LDROOn
)

// ldroSymbolMs is the symbol-duration threshold (milliseconds) at or above
// which the automatic rule enables LDRO.
const ldroSymbolMs = 16.0

// AutoLDRO reports whether the automatic rule enables LDRO for the given
// spreading factor and bandwidth: symbol time Ts = 2^SF / BW >= 16 ms.
func AutoLDRO(sf int, bw Bandwidth) bool {
	if bw <= 0 || sf <= 0 {
		return false
	}
	symbolMs := float64(int(1)<<uint(sf)) / float64(bw) * 1000
	return symbolMs >= ldroSymbolMs
}

// ResolveLDRO turns an LDROMode into the concrete on/off decision for a
// sub-channel, consulting AutoLDRO only in LDROAuto mode.
func ResolveLDRO(mode LDROMode, sf int, bw Bandwidth) bool {
	switch mode {
	case LDROOn:
		return true
	case LDROOff:
		return false
	default:
		return AutoLDRO(sf, bw)
	}
}

// ParseLDROMode maps a config string to an LDROMode. Empty / "auto" ->
// LDROAuto; "on"/"true"/"1" -> LDROOn; "off"/"false"/"0" -> LDROOff.
// ok is false for an unrecognised value (the caller falls back to auto).
func ParseLDROMode(s string) (LDROMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return LDROAuto, true
	case "on", "true", "1", "yes":
		return LDROOn, true
	case "off", "false", "0", "no":
		return LDROOff, true
	default:
		return LDROAuto, false
	}
}

// payloadPPM returns the number of interleaver rows (meaningful bits per
// symbol) for a payload block: SF normally, SF-2 when LDRO reduces the
// rate. The header always uses the full SF rows (see header.go).
func payloadPPM(sf int, ldro bool) int {
	if ldro {
		return sf - 2
	}
	return sf
}
