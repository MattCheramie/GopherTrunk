package lora

// LoRa payload whitening and CRC. Whitening XORs the payload with a PN
// sequence from an 8-bit LFSR; being an XOR it is its own inverse. The
// generator polynomial / seed here produce a deterministic, self-consistent
// sequence; matching the exact Semtech sequence byte-for-byte is a
// calibration step gated behind a captured golden packet (see plan).

// whitenSeed / whitenPoly drive the LFSR. Poly 0xB8 (x^8+x^6+x^5+x^4+1
// reflected) with seed 0xFF is the de-facto sequence used by several open
// LoRa decoders.
const (
	whitenSeed = 0xFF
	whitenPoly = 0xB8
)

// Whiten XORs data in place with the LoRa PN sequence. Calling it twice
// restores the original bytes (XOR is self-inverse), so the same function
// whitens on TX and de-whitens on RX.
func Whiten(data []byte) {
	lfsr := uint8(whitenSeed)
	for i := range data {
		data[i] ^= lfsr
		// advance the LFSR 8 times (one full byte).
		for b := 0; b < 8; b++ {
			lsb := lfsr & 1
			lfsr >>= 1
			if lsb != 0 {
				lfsr ^= whitenPoly
			}
		}
	}
}

// PayloadCRC computes the LoRa payload CRC-16 (CCITT polynomial 0x1021,
// init 0x0000) over the de-whitened payload bytes (excluding the trailing
// CRC field). Byte order / seed compatibility with Semtech silicon is
// validated against a golden packet.
func PayloadCRC(payload []byte) uint16 {
	const poly = 0x1021
	crc := uint16(0x0000)
	for _, b := range payload {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// bytesToNibbles expands bytes into low-nibble-first nibble pairs:
// byte b -> [b&0x0f, b>>4].
func bytesToNibbles(data []byte) []uint8 {
	out := make([]uint8, 0, len(data)*2)
	for _, b := range data {
		out = append(out, b&0x0f, b>>4)
	}
	return out
}

// nibblesToBytes is the inverse of bytesToNibbles. A trailing odd nibble
// is dropped.
func nibblesToBytes(nibs []uint8) []byte {
	out := make([]byte, 0, len(nibs)/2)
	for i := 0; i+1 < len(nibs); i += 2 {
		out = append(out, (nibs[i]&0x0f)|(nibs[i+1]&0x0f)<<4)
	}
	return out
}
