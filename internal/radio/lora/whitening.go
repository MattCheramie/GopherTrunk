package lora

// LoRa payload whitening and CRC, in the Semtech air format. Both are
// pinned by literal vectors from two independent open implementations
// (gr-lora_sdr, validated against SX1276 hardware, and jkadbear/LoRaPHY,
// validated against SX1278) — see bitexact_test.go.
//
// Whitening XORs each payload byte with one byte of a PN sequence from an
// 8-bit Fibonacci LFSR (x^8+x^6+x^5+x^4+1, seed 0xFF) that advances ONE
// bit per payload byte: 0xFF, 0xFE, 0xFC, 0xF8, 0xF0, 0xE1, 0xC2, 0x85, …
// Only the payload is whitened — not the explicit header and not the CRC.

// Whiten XORs data in place with the LoRa whitening sequence. Calling it
// twice restores the original bytes (XOR is self-inverse), so the same
// function whitens on TX and de-whitens on RX.
func Whiten(data []byte) {
	lfsr := uint8(0xFF)
	for i := range data {
		data[i] ^= lfsr
		fb := (lfsr>>7 ^ lfsr>>5 ^ lfsr>>4 ^ lfsr>>3) & 1
		lfsr = lfsr<<1 | fb
	}
}

// PayloadCRC computes the LoRa payload CRC over the plaintext payload
// (excluding the CRC field itself). LoRa's construction is not a plain
// CRC-16 over the whole payload: it is CRC-16 (poly 0x1021, init 0,
// MSB-first, no reflection) over all but the last two bytes, XORed with
// those last two bytes as a big-endian word. For a payload shorter than
// two bytes the CRC over the empty prefix is 0 and only the bytes present
// are XORed in (LoRaPHY's calc_crc cases 0 and 1).
func PayloadCRC(payload []byte) uint16 {
	n := len(payload)
	crc := uint16(0)
	for i := 0; i < n-2; i++ {
		crc ^= uint16(payload[i]) << 8
		for b := 0; b < 8; b++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	if n >= 1 {
		crc ^= uint16(payload[n-1])
	}
	if n >= 2 {
		crc ^= uint16(payload[n-2]) << 8
	}
	return crc
}

// crcNibbles is the on-air order of the CRC field: least-significant
// nibble first.
func crcNibbles(crc uint16) []uint8 {
	return []uint8{uint8(crc) & 0x0f, uint8(crc>>4) & 0x0f, uint8(crc>>8) & 0x0f, uint8(crc >> 12)}
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
