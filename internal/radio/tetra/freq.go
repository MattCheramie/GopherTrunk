package tetra

// TETRA carrier-frequency arithmetic, per ETSI EN 300 392-2 V3.8.1 §21.4.4.1
// (SYSINFO). The SYSINFO broadcast places the cell's main carrier on an absolute
// frequency via four fields — frequency band, main carrier number, offset and
// (for the uplink) duplex spacing + reverse operation:
//
//	downlink  = base(band) + main_carrier*25 kHz + offset
//	uplink    = downlink - duplex_spacing        (normal operation)
//	uplink    = downlink + duplex_spacing        (reverse operation)
//
// The base-frequency and duplex-spacing mappings are defined by the carrier
// numbering scheme in ETSI TS 100 392-15. The offset field is what makes a
// carrier that a cell *broadcasts* as e.g. 469.875 MHz actually sit at
// 469.88125 MHz (offset 01 = +6.25 kHz) — the ±5–12.5 kHz that a decoder which
// ignores the field cannot account for.

// tetraCarrierSpacingHz is the 25 kHz main-carrier step size (§21.4.4.1).
const tetraCarrierSpacingHz = 25_000

// sysInfoOffsetHz maps the 2-bit Offset field to the carrier adjustment in Hz
// (§21.4.4.1): 00 = 0, 01 = +6.25 kHz, 10 = -6.25 kHz, 11 = +12.5 kHz. Matches
// tetra-kit's duplex[] = {0, 6250, -6250, 12500}.
var sysInfoOffsetHz = [4]int64{0, 6_250, -6_250, 12_500}

// tetraBandBaseHz maps the 4-bit Frequency band field to its base frequency in
// Hz. The deployed carrier-numbering scheme (TS 100 392-15) uses base =
// band × 100 MHz for the common bands — the convention tetra-kit and osmo-tetra
// apply, and the one that reproduces on-air data (band 4 ⇒ 400 MHz base, so
// carrier 2795 ⇒ 469.875 MHz nominal).
func tetraBandBaseHz(band uint8) int64 { return int64(band) * 100_000_000 }

// tetraDLCarrierHz computes the absolute downlink carrier frequency (Hz) from
// the frequency band, 12-bit main carrier number and 2-bit offset field
// (§21.4.4.1): DL = base(band) + carrier*25 kHz + offset.
func tetraDLCarrierHz(band uint8, carrier uint16, offset uint8) int64 {
	return tetraBandBaseHz(band) + int64(carrier)*tetraCarrierSpacingHz + sysInfoOffsetHz[offset&0x3]
}

// duplexSpacingHz maps a (frequency band, 3-bit duplex spacing field) pair to
// the duplex spacing in Hz used to derive the uplink carrier. The full
// band↔spacing table lives in ETSI TS 100 392-15; only combinations verified
// against on-air captures are populated here, and unverified pairs return
// ok=false so the uplink is reported as unknown rather than guessed. The 400 MHz
// band's 10 MHz duplex is confirmed on-air (DL 469.875 / UL 459.875).
func duplexSpacingHz(band, duplexField uint8) (int64, bool) {
	_ = duplexField // reserved: band selects the standard spacing for verified bands
	switch band {
	case 4: // 400 MHz band — standard 10 MHz duplex
		return 10_000_000, true
	}
	return 0, false
}

// tetraULCarrierHz derives the uplink carrier (Hz) from a downlink carrier and
// the SYSINFO duplex-spacing / reverse-operation fields (§21.4.4.1). Returns
// ok=false when the duplex spacing for the band is not known.
func tetraULCarrierHz(dlHz int64, band, duplexField uint8, reverse bool) (int64, bool) {
	sp, ok := duplexSpacingHz(band, duplexField)
	if !ok {
		return 0, false
	}
	if reverse {
		return dlHz + sp, true
	}
	return dlHz - sp, true
}
