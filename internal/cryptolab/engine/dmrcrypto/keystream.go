// Package dmrcrypto generates the keystreams DMR's link-layer privacy
// algorithms produce, so the assessment harness and recipe pipeline can
// attempt DMR decryption the same way p25crypto handles P25.
//
// Honesty note: DMR encryption is vendor-specific. The cipher *cores* are
// standard — Motorola "Enhanced Privacy" is RC4, and the DMRA types map to
// DES/3DES/AES — but the key and IV *derivation* from the air interface is
// proprietary and reverse-engineered, and differs between Motorola and Hytera.
// This package therefore implements the standard cores keyed with the material
// the caller supplies (the reconstructed key, and the frame's IV bytes used
// directly): it does not fabricate a vendor key schedule. An analyst who has
// recovered the key can decrypt; the package makes no claim to derive that key
// from the signalling.
package dmrcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/crypto/rc4"
)

// DMRA / vendor privacy algorithm identifiers, as surfaced by common decoders.
const (
	AlgRC4    = 0x21 // Motorola "Enhanced Privacy"
	AlgDESOFB = 0x22
	AlgTDES   = 0x23
	AlgAES128 = 0x24
	AlgAES256 = 0x25
)

// KeySize returns the canonical key length in bytes for algid (RC4 keys are
// variable; 5 bytes / 40 bits is the common Enhanced-Privacy size used for the
// default-key dictionary and brute), or 0 when unsupported.
func KeySize(algid uint8) int {
	switch algid {
	case AlgRC4:
		return 5
	case AlgDESOFB:
		return 8
	case AlgTDES:
		return 24
	case AlgAES128:
		return 16
	case AlgAES256:
		return 32
	}
	return 0
}

// Supported reports whether Keystream can realise this algorithm.
func Supported(algid uint8) bool { return KeySize(algid) != 0 }

// AlgName returns a short human label for algid.
func AlgName(algid uint8) string {
	switch algid {
	case AlgRC4:
		return "RC4 (Enhanced Privacy)"
	case AlgDESOFB:
		return "DES-OFB"
	case AlgTDES:
		return "Triple-DES"
	case AlgAES128:
		return "AES-128"
	case AlgAES256:
		return "AES-256"
	}
	return fmt.Sprintf("dmr-alg-0x%02X", algid)
}

// Keystream returns n keystream bytes for the algorithm, key, and IV. For RC4
// the IV (if present) is appended to the key — the common Enhanced-Privacy
// model — and no warm-up bytes are discarded (unlike P25 ADP). For the block
// ciphers the IV seeds OFB directly. RC4 accepts a 1..256-byte key; the block
// ciphers require the exact KeySize.
func Keystream(algid uint8, key, iv []byte, n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("dmrcrypto: negative length")
	}
	switch algid {
	case AlgRC4:
		rcKey := append(append([]byte(nil), key...), iv...)
		c, err := rc4.NewCipher(rcKey)
		if err != nil {
			return nil, fmt.Errorf("dmrcrypto: rc4: %w", err)
		}
		return c.KeyStream(n), nil
	case AlgDESOFB:
		if len(key) != 8 {
			return nil, fmt.Errorf("dmrcrypto: DES needs an 8-byte key, got %d", len(key))
		}
		return ofb(func(k []byte) (cipher.Block, error) { return des.NewCipher(k) }, key, ivBlock(iv, des.BlockSize), n)
	case AlgTDES:
		if len(key) != 24 {
			return nil, fmt.Errorf("dmrcrypto: 3DES needs a 24-byte key, got %d", len(key))
		}
		return ofb(func(k []byte) (cipher.Block, error) { return des.NewTripleDESCipher(k) }, key, ivBlock(iv, des.BlockSize), n)
	case AlgAES128, AlgAES256:
		if want := KeySize(algid); len(key) != want {
			return nil, fmt.Errorf("dmrcrypto: %s needs a %d-byte key, got %d", AlgName(algid), want, len(key))
		}
		return ofb(func(k []byte) (cipher.Block, error) { return aes.NewCipher(k) }, key, ivBlock(iv, aes.BlockSize), n)
	}
	return nil, fmt.Errorf("dmrcrypto: unsupported algorithm 0x%02X", algid)
}

func ofb(newBlock func([]byte) (cipher.Block, error), key, iv []byte, n int) ([]byte, error) {
	block, err := newBlock(key)
	if err != nil {
		return nil, fmt.Errorf("dmrcrypto: %w", err)
	}
	out := make([]byte, n)
	cipher.NewOFB(block, iv).XORKeyStream(out, out)
	return out, nil
}

// ivBlock left-justifies the IV into a size-byte block (zero-padded/truncated).
func ivBlock(iv []byte, size int) []byte {
	b := make([]byte, size)
	copy(b, iv)
	return b
}

// DefaultKeys returns a small dictionary of weak/default/test keys to try for
// algid (all-zero, all-FF, and an incrementing pattern), or nil if unsupported.
func DefaultKeys(algid uint8) [][]byte {
	sz := KeySize(algid)
	if sz == 0 {
		return nil
	}
	zero := make([]byte, sz)
	ff := make([]byte, sz)
	inc := make([]byte, sz)
	for i := range ff {
		ff[i] = 0xFF
		inc[i] = byte(i + 1)
	}
	return [][]byte{zero, ff, inc}
}
