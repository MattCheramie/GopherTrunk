package lorawan

import (
	"crypto/aes"
	"crypto/subtle"
	"encoding/binary"
)

// aesCMAC computes the AES-CMAC (RFC 4493) of msg under a 16-byte key.
func aesCMAC(key [16]byte, msg []byte) [16]byte {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		// key is fixed-size [16]byte, so NewCipher cannot fail.
		panic("lorawan: aes.NewCipher: " + err.Error())
	}
	const blockSize = 16

	// Subkey generation (RFC 4493 §2.3).
	var l [16]byte
	block.Encrypt(l[:], l[:]) // L = AES_K(0^128)
	k1 := shiftLeftXorRb(l)
	k2 := shiftLeftXorRb(k1)

	// Split into blocks; build the last block per the padding rules.
	var last [16]byte
	complete := len(msg) > 0 && len(msg)%blockSize == 0
	n := (len(msg) + blockSize - 1) / blockSize
	if n == 0 {
		n = 1
	}
	if complete {
		copy(last[:], msg[(n-1)*blockSize:])
		xor16(&last, k1)
	} else {
		rem := msg[(n-1)*blockSize:]
		copy(last[:], rem)
		last[len(rem)] = 0x80 // padding
		xor16(&last, k2)
	}

	var x, y [16]byte
	for i := 0; i < n-1; i++ {
		copy(y[:], msg[i*blockSize:(i+1)*blockSize])
		xorInto(&y, x)
		block.Encrypt(x[:], y[:])
	}
	xorInto(&last, x)
	block.Encrypt(x[:], last[:])
	return x
}

// shiftLeftXorRb left-shifts a 128-bit block by one bit and conditionally
// XORs the CMAC Rb constant (0x87) when the high bit was set.
func shiftLeftXorRb(in [16]byte) [16]byte {
	var out [16]byte
	overflow := byte(0)
	for i := 15; i >= 0; i-- {
		out[i] = in[i]<<1 | overflow
		overflow = in[i] >> 7
	}
	if in[0]&0x80 != 0 {
		out[15] ^= 0x87
	}
	return out
}

func xor16(dst *[16]byte, k [16]byte) {
	for i := range dst {
		dst[i] ^= k[i]
	}
}

func xorInto(dst *[16]byte, x [16]byte) {
	for i := range dst {
		dst[i] ^= x[i]
	}
}

// micBlockB0 builds the B0 block prepended to the MIC computation
// (LoRaWAN 1.0 §4.4). dir is 0 uplink, 1 downlink.
func micBlockB0(dir byte, devAddr uint32, fcnt32 uint32, msgLen int) [16]byte {
	var b [16]byte
	b[0] = 0x49
	// b[1..4] = 0x00000000
	b[5] = dir
	binary.LittleEndian.PutUint32(b[6:10], devAddr)
	binary.LittleEndian.PutUint32(b[10:14], fcnt32)
	// b[14] = 0x00
	b[15] = byte(msgLen)
	return b
}

// ComputeMIC returns the 4-byte data-frame MIC under nwkSKey.
func (m *Message) ComputeMIC(nwkSKey [16]byte) [4]byte {
	dir := byte(0)
	if !m.MType.IsUplink() {
		dir = 1
	}
	b0 := micBlockB0(dir, m.DevAddr, m.FCnt32(), len(m.mac))
	buf := make([]byte, 0, 16+len(m.mac))
	buf = append(buf, b0[:]...)
	buf = append(buf, m.mac...)
	full := aesCMAC(nwkSKey, buf)
	var out [4]byte
	copy(out[:], full[:4])
	return out
}

// VerifyMIC reports whether the received MIC matches the one computed under
// nwkSKey. Only meaningful for data frames.
func (m *Message) VerifyMIC(nwkSKey [16]byte) bool {
	if !m.MType.IsData() {
		return false
	}
	want := m.ComputeMIC(nwkSKey)
	return subtle.ConstantTimeCompare(want[:], m.MIC[:]) == 1
}
