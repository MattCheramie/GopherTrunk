package framing

// CRCCCITT returns the 16-bit CRC-CCITT/FALSE of msg using polynomial
// 0x1021, initial value 0xFFFF, no input/output reflection, no final XOR.
// This is the variant used by the P25 TSBK trailer (the trailer stores the
// bitwise complement; callers handle that at protocol level).
func CRCCCITT(msg []byte) uint16 {
	return CRCCCITTWithInit(msg, 0xFFFF)
}

// CRCCCITTWithInit is the same CRC-CCITT (poly 0x1021, no reflection,
// no final XOR) but with a caller-supplied initial value. Pass
// 0xFFFF for the CCITT/FALSE variant the P25 TSBK trailer uses; pass
// 0x0000 for the XMODEM / YSF FICH variant.
func CRCCCITTWithInit(msg []byte, init uint16) uint16 {
	const poly uint16 = 0x1021
	crc := init
	for _, b := range msg {
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

// CRCCCITTBits computes CRC-CCITT over a bit slice (each entry 0/1, MSB-first
// nibbles within each byte). Useful when the caller already has a bit array.
func CRCCCITTBits(bits []byte) uint16 {
	return CRCCCITT(PackBitsMSB(bits))
}
