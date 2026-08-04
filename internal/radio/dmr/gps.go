package dmr

// DMR GPS Info decode. A subscriber unit can report its position in the
// voice superframe's embedded Link Control as a "GPS Info" Full LC PDU
// (FLCOGPS / 0x08), the same embedded-LC framing the group-voice and
// talker-alias LCs use. This file decodes the position fields; the voice
// chain binds the fix to the call's source radio (the LC itself carries no
// address) and publishes it on the location bus.
//
// Wire layout (the 72-bit / 9-octet embedded-LC info block, bit indices
// MSB-first as recovered by ReassembleEmbeddedLCInfo), per ETSI
// TS 102 361-2 §7.1.1.3 and cross-checked against the OK-DMR ok-dmrlib
// reference decoder (okdmr/dmrlib Full Link Control):
//
//	bits  0..15 : PF/FLCO (octet 0) + FID (octet 1)
//	bits 16..19 : reserved
//	bits 20..22 : Position Error (3 bits)
//	bits 23..47 : Longitude — 25-bit two's-complement, deg = raw·360/2^25
//	bits 48..71 : Latitude  — 24-bit two's-complement, deg = raw·180/2^24
//
// Like the other DMR embedded-LC constants (see emb.go / talker_alias.go),
// this is a working model pending validation against a real capture.

// GPSInfo is a decoded DMR GPS Info Full LC: a geographic fix a subscriber
// reported over the air. Latitude is positive north, Longitude positive
// east, both in decimal degrees.
type GPSInfo struct {
	Latitude      float64
	Longitude     float64
	PositionError uint8 // 3-bit reported position-error class (0 = best)
}

// gpsLonScale / gpsLatScale convert the raw two's-complement integers to
// decimal degrees.
const (
	gpsLonScale = 360.0 / (1 << 25)
	gpsLatScale = 180.0 / (1 << 24)
)

// Valid reports whether the fix is a usable position — within range and
// not the (0,0) "no fix yet" placeholder (Null Island is open ocean), the
// same rule the location package's NMEA parser applies.
func (g GPSInfo) Valid() bool {
	if g.Latitude == 0 && g.Longitude == 0 {
		return false
	}
	return g.Latitude >= -90 && g.Latitude <= 90 &&
		g.Longitude >= -180 && g.Longitude <= 180
}

// ParseGPSInfo decodes a GPS Info fix from the leading 9 octets of an
// embedded-LC info block whose FLCO is FLCOGPS. It returns ok=false for a
// non-GPS FLCO or a short buffer.
func ParseGPSInfo(info []byte) (GPSInfo, bool) {
	if len(info) < 9 {
		return GPSInfo{}, false
	}
	if FLCO(info[0]&0x3F) != FLCOGPS {
		return GPSInfo{}, false
	}
	lonRaw := int32(readBitsMSB(info, 23, 25))
	if lonRaw >= 1<<24 { // sign-extend the 25-bit two's-complement value
		lonRaw -= 1 << 25
	}
	latRaw := int32(readBitsMSB(info, 48, 24))
	if latRaw >= 1<<23 { // sign-extend the 24-bit two's-complement value
		latRaw -= 1 << 24
	}
	return GPSInfo{
		Latitude:      float64(latRaw) * gpsLatScale,
		Longitude:     float64(lonRaw) * gpsLonScale,
		PositionError: uint8(readBitsMSB(info, 20, 3)),
	}, true
}

// AssembleGPSInfo packs a fix into the 9-octet GPS Info info block. It is
// the inverse of ParseGPSInfo, used to build synthetic test vectors. The
// latitude/longitude round-trip to within one quantisation step.
func AssembleGPSInfo(g GPSInfo) []byte {
	info := make([]byte, 9)
	info[0] = byte(FLCOGPS)
	lonRaw := int32(round(g.Longitude / gpsLonScale))
	latRaw := int32(round(g.Latitude / gpsLatScale))
	writeBitsMSB(info, 20, 3, uint32(g.PositionError))
	writeBitsMSB(info, 23, 25, uint32(lonRaw)&0x1FFFFFF)
	writeBitsMSB(info, 48, 24, uint32(latRaw)&0xFFFFFF)
	return info
}

// readBitsMSB reads n bits (n<=32) starting at bit offset start from an
// MSB-first octet slice.
func readBitsMSB(b []byte, start, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		idx := start + i
		v <<= 1
		if idx/8 < len(b) && b[idx/8]&(1<<uint(7-(idx%8))) != 0 {
			v |= 1
		}
	}
	return v
}

// writeBitsMSB writes the low n bits of val into an MSB-first octet slice
// starting at bit offset start.
func writeBitsMSB(b []byte, start, n int, val uint32) {
	for i := 0; i < n; i++ {
		if val&(1<<uint(n-1-i)) != 0 {
			idx := start + i
			if idx/8 < len(b) {
				b[idx/8] |= 1 << uint(7-(idx%8))
			}
		}
	}
}

// round rounds a float to the nearest integer (half away from zero)
// without pulling in math for one call site.
func round(f float64) float64 {
	if f < 0 {
		return float64(int64(f - 0.5))
	}
	return float64(int64(f + 0.5))
}
