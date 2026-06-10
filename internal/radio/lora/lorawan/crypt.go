package lorawan

import (
	"crypto/aes"
	"encoding/binary"
)

// DecryptFRMPayload decrypts (or encrypts — the operation is symmetric) the
// frame payload under key, following the LoRaWAN 1.0 §4.3.3 AES-CTR-style
// scheme. key is the AppSKey for FPort > 0, or the NwkSKey for FPort == 0
// (MAC-command payloads). The returned slice is freshly allocated.
func (m *Message) DecryptFRMPayload(key [16]byte) []byte {
	if len(m.FRMPayload) == 0 {
		return nil
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		panic("lorawan: aes.NewCipher: " + err.Error())
	}
	dir := byte(0)
	if !m.MType.IsUplink() {
		dir = 1
	}

	out := make([]byte, len(m.FRMPayload))
	var a, s [16]byte
	a[0] = 0x01
	a[5] = dir
	binary.LittleEndian.PutUint32(a[6:10], m.DevAddr)
	binary.LittleEndian.PutUint32(a[10:14], m.FCnt32())

	blocks := (len(m.FRMPayload) + 15) / 16
	for i := 1; i <= blocks; i++ {
		a[15] = byte(i)
		block.Encrypt(s[:], a[:])
		start := (i - 1) * 16
		end := start + 16
		if end > len(m.FRMPayload) {
			end = len(m.FRMPayload)
		}
		for j := start; j < end; j++ {
			out[j] = m.FRMPayload[j] ^ s[j-start]
		}
	}
	return out
}
