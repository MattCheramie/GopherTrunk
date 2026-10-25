package ekc

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// referenceDocs returns the KFDtool-written reference container's outer XML
// and its decrypted inner XML (padding stripped), straight from the file.
func referenceDocs(t *testing.T) (outer, inner []byte, salt, cipherValue string) {
	t.Helper()
	b := readReference(t)
	zr, err := gzip.NewReader(bytes.NewReader(b[4:]))
	if err != nil {
		t.Fatal(err)
	}
	outer, _ = io.ReadAll(zr)
	o, err := readOuter(b)
	if err != nil {
		t.Fatal(err)
	}
	salt, cipherValue = o.KeyDerivation.Salt, o.EncryptedData.CipherValue
	rawSalt, _ := base64.StdEncoding.DecodeString(salt)
	ct, _ := base64.StdEncoding.DecodeString(cipherValue)
	key, _ := pbkdf2.Key(sha512.New, referencePassword, rawSalt, o.KeyDerivation.IterationCount, 32)
	blk, _ := aes.NewCipher(key)
	p := make([]byte, len(ct)-aes.BlockSize)
	cipher.NewCBCDecrypter(blk, ct[:aes.BlockSize]).CryptBlocks(p, ct[aes.BlockSize:])
	return outer, p[:len(p)-int(p[len(p)-1])], salt, cipherValue
}

// The inner container Encode writes for the reference's keys is byte for
// byte the one KFDtool's own code wrote: same elements, order, spelling.
func TestEncodeInnerMatchesKFDtool(t *testing.T) {
	_, want, _, _ := referenceDocs(t)
	c, err := Open(readReference(t), referencePassword)
	if err != nil {
		t.Fatal(err)
	}
	got, err := marshalInner(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("inner XML differs from KFDtool's\n got: %s\nwant: %s", got, want)
	}
}

// The outer document is KFDtool's apart from the salt and cipher value.
func TestEncodeOuterMatchesKFDtool(t *testing.T) {
	want, _, salt, cv := referenceDocs(t)
	if got := outerDocument(salt, cv); !bytes.Equal(got, want) {
		t.Errorf("outer XML differs from KFDtool's\n got: %s\nwant: %s", got, want)
	}
}

// What Encode writes, Open reads back unchanged, with the right password only.
func TestEncodeRoundTrip(t *testing.T) {
	c, err := Open(readReference(t), referencePassword)
	if err != nil {
		t.Fatal(err)
	}
	file, err := Encode(c, "new-password")
	if err != nil {
		t.Fatal(err)
	}
	back, err := Open(file, "new-password")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, c) {
		t.Errorf("round trip changed the container\n got %+v\nwant %+v", back, c)
	}
	if _, err := Open(file, referencePassword); !errors.Is(err, ErrWrongPassword) {
		t.Errorf("old password on the re-encoded file: err = %v, want ErrWrongPassword", err)
	}
	if _, err := Encode(c, ""); err == nil {
		t.Error("an empty password was accepted")
	}
}

// ISO 10126 as .NET writes it: the last plaintext byte is the pad length, a
// whole block when the inner XML is block-aligned; two encodings differ (fresh
// salt and IV each time).
func TestEncodePaddingAndFreshRandomness(t *testing.T) {
	c := New()
	if _, err := c.Add(Key{Name: "A", KeyID: 1, AlgorithmID: 0x81, Material: []byte{1, 2, 3, 4, 5, 6, 7, 8}}); err != nil {
		t.Fatal(err)
	}
	plain, _ := marshalInner(c)
	file, err := encode(c, "pw", rand.New(rand.NewSource(1)))
	if err != nil {
		t.Fatal(err)
	}
	o, _ := readOuter(file)
	rawSalt, _ := base64.StdEncoding.DecodeString(o.KeyDerivation.Salt)
	ct, _ := base64.StdEncoding.DecodeString(o.EncryptedData.CipherValue)
	key, _ := pbkdf2.Key(sha512.New, "pw", rawSalt, Iterations, 32)
	blk, _ := aes.NewCipher(key)
	p := make([]byte, len(ct)-aes.BlockSize)
	cipher.NewCBCDecrypter(blk, ct[:aes.BlockSize]).CryptBlocks(p, ct[aes.BlockSize:])
	wantPad := aes.BlockSize - len(plain)%aes.BlockSize
	if int(p[len(p)-1]) != wantPad || !bytes.Equal(p[:len(p)-wantPad], plain) {
		t.Errorf("pad byte %d, want %d, over the exact inner XML", p[len(p)-1], wantPad)
	}
	if o.KeyDerivation.IterationCount != Iterations || len(rawSalt) != saltLen {
		t.Errorf("KDF: %d iterations, %d-byte salt", o.KeyDerivation.IterationCount, len(rawSalt))
	}
	a, _ := Encode(c, "pw")
	b, _ := Encode(c, "pw")
	if bytes.Equal(a, b) {
		t.Error("two encodings are identical: salt / IV are not fresh")
	}
}

func TestAddAndRemove(t *testing.T) {
	c, err := Open(readReference(t), referencePassword)
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.Add(Key{Name: "DMR EP", KeyID: 11, AlgorithmID: 0xAA, Material: []byte{1, 2, 3, 4, 5}})
	if err != nil {
		t.Fatal(err)
	}
	if id != 8 || c.NextKeyNumber != 9 {
		t.Errorf("new item id %d, next %d; want 8 and 9 (the reference's NextKeyNumber is 8)", id, c.NextKeyNumber)
	}
	added := c.Keys[len(c.Keys)-1]
	// Keyset 1 uses SLNs 1,2,3,4,6,7: the first free one is 5.
	if added.KeysetID != 1 || added.SLN != 5 || !added.TEK || added.KEK {
		t.Errorf("added %+v, want keyset 1, SLN 5, a TEK", added)
	}
	if _, err := c.Add(Key{Name: "dup", KeyID: 11, AlgorithmID: 0xAA, Material: []byte{9}}); err == nil ||
		!strings.Contains(err.Error(), "already has a traffic key with key ID 11") {
		t.Errorf("duplicate key ID in a keyset: err = %v", err)
	}
	// The same key ID in another keyset is how a KFD stages the next key.
	if _, err := c.Add(Key{Name: "next", KeysetID: 2, KeyID: 11, AlgorithmID: 0xAA, Material: []byte{9}}); err != nil {
		t.Errorf("same key ID in keyset 2: %v", err)
	}
	if !c.Remove(2) || c.Remove(2) {
		t.Error("Remove(2) should succeed once")
	}
	if g := c.Groups[0].Keys; !reflect.DeepEqual(g, []int{1, 3}) {
		t.Errorf("group keys after removing item 2: %v, want [1 3]", g)
	}
	if _, err := c.Add(Key{Name: "empty", KeyID: 12, AlgorithmID: 0x81}); err == nil {
		t.Error("a key with no material was accepted")
	}
}

// An empty container still writes its <Keys> and <Groups> elements (KFDtool
// writes <Keys /> and <Groups />): a missing element would deserialise to a
// null list on the KFDtool side.
func TestEncodeEmptyContainerKeepsListElements(t *testing.T) {
	got, err := marshalInner(New())
	if err != nil {
		t.Fatal(err)
	}
	want := `<InnerContainer version="1.0"><Keys></Keys><NextKeyNumber>1</NextKeyNumber><Groups></Groups><NextGroupNumber>1</NextGroupNumber></InnerContainer>`
	if string(got) != want {
		t.Errorf("empty container\n got %s\nwant %s", got, want)
	}
	file, err := Encode(New(), "pw")
	if err != nil {
		t.Fatal(err)
	}
	if c, err := Open(file, "pw"); err != nil || len(c.Keys) != 0 || c.NextKeyNumber != 1 {
		t.Errorf("empty container round trip: %+v, %v", c, err)
	}
}
