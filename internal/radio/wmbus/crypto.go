package wmbus

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"errors"
	"fmt"
)

// Decryption of OMS / EN 13757 telegrams with the meter's AES-128 key (issue
// #1256). Most installed OMS meters encrypt their readings; the key belongs to
// whoever owns or reads the meter, and with it the telegram decodes like an
// unencrypted one. Three schemes are handled, the ones meters in the field use:
//
//   - TPL security mode 5: AES-128-CBC, IV = M-field (2) + A-field (6: ID,
//     version, device type) + 8 copies of the access number. The A-field and
//     M-field are the long transport header's when there is one (a radio
//     converter's meter), else the data link layer's.
//   - TPL security mode 7: AES-128-CBC with a zero IV, under an ephemeral key
//     derived from the meter key with AES-CMAC (key derivation function 1):
//     Kenc = CMAC(K, 0x00 | message counter (4) | ID (4) | 0x07 × 7). The
//     counter comes from the authentication and fragmentation layer (CI 0x90).
//   - Extended link layer AES-CTR (CI 0x8D with encryption set in its SN
//     field): counter block = M (2) + A (6) + CC (the hop-count and repeated
//     bits 0x10 / 0x02 cleared) + SN (4) + FN (2, zero) + BC (1, zero),
//     incremented big-endian per block. The decrypted data opens with a
//     CRC-16 (EN 13757, little-endian) over the rest.
//
// Every scheme has its own check that the key was right: modes 5 and 7 open
// with the 0x2F 0x2F filler, the ELL with its payload CRC. A wrong key fails
// that check and the telegram stays encrypted, with DecryptErr saying why.
//
// The layouts follow wmbusmeters (wmbus_utils.cc decrypt_TPL_AES_CBC_IV,
// decrypt_TPL_AES_CBC_NO_IV, decrypt_ELL_AES_CTR and wmbus.cc parseTPLConfig,
// parseAFL), read for the facts only; the tests decode real encrypted
// telegrams from its test set, with their published keys, to the readings it
// publishes for them.

// KeyLookup returns the AES-128 key for the meter with identification number
// id (8 digits, as Telegram.ID prints it), if one is known.
type KeyLookup func(id string) ([]byte, bool)

// KeyMap is a KeyLookup over a map of identification number to key.
func KeyMap(m map[string][]byte) KeyLookup {
	return func(id string) ([]byte, bool) {
		k, ok := m[id]
		return k, ok
	}
}

var errWrongKey = errors.New("wrong key")

// decryptELL decrypts an extended link layer's AES-CTR payload and checks its
// payload CRC. iv is the counter block's first 13 bytes (M, A, CC, SN).
func decryptELL(key, ivHead, enc []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	var iv [16]byte
	copy(iv[:], ivHead)
	out := make([]byte, len(enc))
	cipher.NewCTR(block, iv[:]).XORKeyStream(out, enc)
	if len(out) < 3 {
		return nil, fmt.Errorf("ELL payload %d bytes", len(out))
	}
	want := uint16(out[0]) | uint16(out[1])<<8
	if CRC16(out[2:]) != want {
		return nil, fmt.Errorf("payload CRC failed: %w", errWrongKey)
	}
	return out[2:], nil
}

// decryptCBC decrypts the first n bytes of p (n a multiple of 16; 0 means as
// many whole blocks as p holds) with AES-128-CBC and checks the 0x2F 0x2F the
// plaintext must open with. Bytes after the encrypted ones are clear and are
// returned after the plaintext.
func decryptCBC(key, iv, p []byte, n int) ([]byte, error) {
	if n == 0 || n > len(p) {
		n = len(p)
	}
	n -= n % aes.BlockSize
	if n == 0 {
		return nil, fmt.Errorf("no whole AES block to decrypt (%d bytes)", len(p))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, n, len(p))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, p[:n])
	if out[0] != 0x2F || out[1] != 0x2F {
		return nil, fmt.Errorf("plaintext does not open with 2F 2F: %w", errWrongKey)
	}
	return append(out, p[n:]...), nil
}

// cbcDecrypt is plain AES-128-CBC over whole blocks.
func cbcDecrypt(key, iv, dst, src []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(dst, src)
	return nil
}

// mode7Key derives security mode 7's ephemeral encryption key.
func mode7Key(key, counter, id []byte) ([]byte, error) {
	in := make([]byte, 0, 16)
	in = append(in, 0x00) // derivation constant: encryption, from the meter
	in = append(in, counter...)
	in = append(in, id...)
	for len(in) < 16 {
		in = append(in, 0x07)
	}
	return aesCMAC(key, in)
}

// aesCMAC is AES-CMAC (RFC 4493).
func aesCMAC(key, msg []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	var l [16]byte
	block.Encrypt(l[:], l[:])
	k1 := cmacShift(l)
	k2 := cmacShift(k1)
	n := (len(msg) + 15) / 16
	complete := n > 0 && len(msg)%16 == 0
	if n == 0 {
		n = 1
	}
	var last [16]byte
	tail := msg[(n-1)*16:]
	if complete {
		subtle.XORBytes(last[:], tail, k1[:])
	} else {
		copy(last[:], tail)
		last[len(tail)] = 0x80
		subtle.XORBytes(last[:], last[:], k2[:])
	}
	var x [16]byte
	for i := 0; i < n-1; i++ {
		subtle.XORBytes(x[:], x[:], msg[16*i:16*i+16])
		block.Encrypt(x[:], x[:])
	}
	subtle.XORBytes(x[:], x[:], last[:])
	block.Encrypt(x[:], x[:])
	return x[:], nil
}

func cmacShift(b [16]byte) [16]byte {
	var o [16]byte
	for i := 0; i < 15; i++ {
		o[i] = b[i]<<1 | b[i+1]>>7
	}
	o[15] = b[15] << 1
	if b[0]&0x80 != 0 {
		o[15] ^= 0x87
	}
	return o
}
