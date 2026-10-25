package ekc

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Iterations is the PBKDF2-SHA512 work factor Encode writes: KFDtool's own.
const Iterations = 100_000

// saltLen is the PBKDF2 salt length KFDtool writes.
const saltLen = 32

// Encode encrypts c under password into .ekc file bytes, in KFDtool's format:
// PBKDF2-SHA512 (Iterations, 32-byte salt) → AES-256-CBC with a random IV
// and ISO 10126 padding, inside the outer XML, gzipped behind a 4-byte
// little-endian length. Every field Open reads is written back, so a
// container read and re-encoded keeps its item IDs, groups and counters.
func Encode(c *Container, password string) ([]byte, error) {
	return encode(c, password, rand.Reader)
}

// encode is Encode with the salt / IV / padding randomness supplied, so a
// test can make it deterministic.
func encode(c *Container, password string, rnd io.Reader) ([]byte, error) {
	if password == "" {
		return nil, errors.New("ekc: a container needs a password")
	}
	plain, err := marshalInner(c)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, saltLen)
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rnd, salt); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(rnd, iv); err != nil {
		return nil, err
	}
	// ISO 10126, as .NET EncryptedXml writes it: random filler, the last
	// byte the pad length, a whole block when the data is block-aligned.
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	filler := make([]byte, pad-1)
	if _, err := io.ReadFull(rnd, filler); err != nil {
		return nil, err
	}
	padded := append(append(append([]byte(nil), plain...), filler...), byte(pad))
	key, err := pbkdf2.Key(sha512.New, password, salt, Iterations, 32)
	if err != nil {
		return nil, fmt.Errorf("ekc: key derivation: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ct := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, padded)
	outer := outerDocument(base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(append(iv, ct...)))

	var out bytes.Buffer
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(len(outer)))
	out.Write(n[:])
	zw := gzip.NewWriter(&out)
	if _, err := zw.Write(outer); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// outerDocument is the outer XML exactly as KFDtool writes it (pinned by
// TestEncodeOuterMatchesKFDtool), with the given base64 salt and cipher value.
func outerDocument(salt, cipherValue string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?><OuterContainer version="1.0"><KeyDerivation>` +
		`<DerivationAlgorithm>PBKDF2</DerivationAlgorithm><HashAlgorithm>SHA512</HashAlgorithm>` +
		`<Salt>` + salt + `</Salt><IterationCount>` + strconv.Itoa(Iterations) + `</IterationCount>` +
		`<KeyLength>32</KeyLength></KeyDerivation>` +
		`<EncryptedData Type="http://www.w3.org/2001/04/xmlenc#Element" xmlns="http://www.w3.org/2001/04/xmlenc#">` +
		`<EncryptionMethod Algorithm="` + aes256CBC + `" /><CipherData><CipherValue>` + cipherValue +
		`</CipherValue></CipherData></EncryptedData></OuterContainer>`)
}

// marshalInner serialises c as KFDtool's inner container (no XML
// declaration, KFDtool's element order; pinned byte for byte by
// TestEncodeInnerMatchesKFDtool).
func marshalInner(c *Container) ([]byte, error) {
	in := innerXML{Version: "1.0", NextKeyNumber: c.NextKeyNumber, NextGroupNumber: c.NextGroupNumber}
	for _, k := range c.Keys {
		in.Keys.Items = append(in.Keys.Items, innerKey{
			ID: k.ID, Name: k.Name, ActiveKeyset: k.ActiveKeyset, KeysetID: k.KeysetID, SLN: k.SLN,
			KeyTypeAuto: k.Auto, KeyTypeTek: k.TEK, KeyTypeKek: k.KEK, KeyID: k.KeyID, AlgorithmID: k.AlgorithmID,
			Key: fmt.Sprintf("%X", k.Material),
		})
	}
	for _, g := range c.Groups {
		in.Groups.Items = append(in.Groups.Items, innerGroup{ID: g.ID, Name: g.Name, Keys: g.Keys})
	}
	return xml.Marshal(in)
}

// New returns an empty container, its counters where KFDtool starts them.
func New() *Container {
	return &Container{NextKeyNumber: 1, NextGroupNumber: 1}
}

// Add appends k as a new item: its ID is the container's next key number,
// and its SLN, when 0, the lowest one no key in its keyset uses. It refuses a
// second traffic key with the same keyset and key ID (a radio would not know
// which to use) and returns the item ID.
func (c *Container) Add(k Key) (int, error) {
	if len(k.Material) == 0 {
		return 0, errors.New("ekc: key material is empty")
	}
	if k.KeyID < 0 || k.KeyID > 0xFFFF {
		return 0, fmt.Errorf("ekc: key ID %d outside 0..65535", k.KeyID)
	}
	if k.KeysetID == 0 {
		k.KeysetID = 1
	}
	usedSLN := map[int]bool{}
	for _, e := range c.Keys {
		if e.KeysetID != k.KeysetID {
			continue
		}
		usedSLN[e.SLN] = true
		if e.KeyID == k.KeyID && e.KEK == k.KEK {
			return 0, fmt.Errorf("ekc: keyset %d already has a %s with key ID %d (item %d, %q)",
				k.KeysetID, keyType(k.KEK), k.KeyID, e.ID, e.Name)
		}
	}
	if k.SLN == 0 {
		for k.SLN = 1; usedSLN[k.SLN]; k.SLN++ {
		}
	}
	if c.NextKeyNumber < 1 {
		c.NextKeyNumber = 1
	}
	for _, e := range c.Keys { // a hand-edited counter must not reuse an ID
		if e.ID >= c.NextKeyNumber {
			c.NextKeyNumber = e.ID + 1
		}
	}
	k.ID = c.NextKeyNumber
	c.NextKeyNumber++
	if !k.KEK {
		k.TEK = true
	}
	c.Keys = append(c.Keys, k)
	return k.ID, nil
}

// Remove deletes the item with KFDtool ID id, and its references from every
// group. It reports whether such an item existed.
func (c *Container) Remove(id int) bool {
	for i, k := range c.Keys {
		if k.ID != id {
			continue
		}
		c.Keys = append(c.Keys[:i], c.Keys[i+1:]...)
		for gi := range c.Groups {
			keep := c.Groups[gi].Keys[:0]
			for _, ref := range c.Groups[gi].Keys {
				if ref != id {
					keep = append(keep, ref)
				}
			}
			c.Groups[gi].Keys = keep
		}
		return true
	}
	return false
}

func keyType(kek bool) string {
	if kek {
		return "KEK"
	}
	return "traffic key"
}
