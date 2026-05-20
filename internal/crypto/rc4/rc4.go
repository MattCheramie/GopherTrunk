package rc4

import "fmt"

// Cipher is an RC4 keystream generator. A Cipher is stateful: every
// call to XORKeyStream or KeyStream advances the internal PRGA state,
// so one Cipher yields a single continuous keystream. A Cipher is not
// safe for concurrent use; construct one per call chain.
type Cipher struct {
	s    [256]byte
	i, j uint8
}

// NewCipher returns a Cipher keyed with key, which must be 1..256
// bytes. The key-scheduling algorithm (KSA) runs here; the returned
// Cipher is positioned at the first keystream byte.
func NewCipher(key []byte) (*Cipher, error) {
	k := len(key)
	if k < 1 || k > 256 {
		return nil, fmt.Errorf("rc4: invalid key length %d, must be 1..256 bytes", k)
	}
	c := &Cipher{}
	for i := range c.s {
		c.s[i] = byte(i)
	}
	var j uint8
	for i := 0; i < 256; i++ {
		j += c.s[i] + key[i%k]
		c.s[i], c.s[j] = c.s[j], c.s[i]
	}
	return c, nil
}

// XORKeyStream XORs each byte of src with the next keystream byte and
// writes the result to dst, advancing the keystream by len(src). dst
// must be at least len(src) long; dst and src may be the same slice.
func (c *Cipher) XORKeyStream(dst, src []byte) {
	if len(src) == 0 {
		return
	}
	if len(dst) < len(src) {
		panic("rc4: dst shorter than src")
	}
	i, j := c.i, c.j
	for n, v := range src {
		i++
		j += c.s[i]
		c.s[i], c.s[j] = c.s[j], c.s[i]
		dst[n] = v ^ c.s[c.s[i]+c.s[j]]
	}
	c.i, c.j = i, j
}

// KeyStream returns the next n bytes of raw keystream and advances the
// Cipher by n bytes. It is equivalent to XORKeyStream against n zero
// bytes — convenient when the caller wants the keystream itself, for
// example to XOR onto a bit-unpacked voice frame.
func (c *Cipher) KeyStream(n int) []byte {
	out := make([]byte, n)
	c.XORKeyStream(out, out)
	return out
}
