// Package ekc reads KFDtool encrypted key containers (.ekc, issue #1305).
//
// KFDtool (github.com/omahacommsys/KFDtool, MIT) is an open keyloader whose
// containers hold an operator's P25 keys encrypted at rest under a password.
// This package only reads them: GopherTrunk decrypts with keys the operator
// already holds, and a container is a way to keep those keys out of the
// plain-text YAML.
//
// The format, from KFDtool's ContainerUtilities.cs and Shared/Utility.cs:
//
//   - file: a 4-byte little-endian length of the uncompressed data, then a
//     gzip stream of the outer XML;
//   - outer XML: <OuterContainer version="1.x"> with a <KeyDerivation>
//     (PBKDF2, HashAlgorithm SHA512, base64 Salt, IterationCount, KeyLength)
//     and a W3C XML-Encryption <EncryptedData> element (aes256-cbc);
//   - CipherValue: base64 of the 16-byte IV followed by the AES-256-CBC
//     ciphertext, padded ISO 10126 style (.NET EncryptedXml's default:
//     random filler, the last byte the pad length), so the pad bytes are
//     NOT checked as PKCS#7 — a strict PKCS#7 unpad rejects every real file;
//   - the plaintext: <InnerContainer version="1.x"> with <Keys> of
//     <KeyItem> (keyset, SLN, key ID, ALGID, TEK/KEK flags, hex key) and
//     <Groups>.
//
// Pinned by a container written by KFDtool's own C# code (testdata).
package ekc

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ErrWrongPassword is returned when the container does not decrypt to a
// key list: a wrong password (or a damaged file).
var ErrWrongPassword = errors.New("ekc: wrong password (the container did not decrypt)")

// Key is one KeyItem of a container.
type Key struct {
	// Name is the operator's label in KFDtool.
	Name string
	// KeysetID and ActiveKeyset are the KFD keyset the key belongs to.
	KeysetID     int
	ActiveKeyset bool
	// SLN is the storage location number.
	SLN int
	// KeyID is the ID a radio carries in its encryption sync / PI header.
	KeyID int
	// AlgorithmID is the TIA-102 ALGID (0x81 DES-OFB, 0x84 AES-256, …).
	AlgorithmID int
	// TEK / KEK say whether this is a traffic key (decrypts voice) or a
	// key-encryption key (wraps other keys; no use to a receiver).
	TEK, KEK bool
	// Material is the key itself.
	Material []byte
}

// Container is a decrypted .ekc.
type Container struct {
	Keys []Key
}

// maxIterations bounds the PBKDF2 work a container may ask for, so a damaged
// or hostile file cannot stall startup. KFDtool writes 100000.
const maxIterations = 10_000_000

// maxInflated bounds the decompressed outer XML (a container of many
// hundred keys is tens of KB).
const maxInflated = 16 << 20

type outerXML struct {
	XMLName       xml.Name `xml:"OuterContainer"`
	Version       string   `xml:"version,attr"`
	KeyDerivation struct {
		DerivationAlgorithm string `xml:"DerivationAlgorithm"`
		HashAlgorithm       string `xml:"HashAlgorithm"`
		Salt                string `xml:"Salt"`
		IterationCount      int    `xml:"IterationCount"`
		KeyLength           int    `xml:"KeyLength"`
	} `xml:"KeyDerivation"`
	EncryptedData struct {
		EncryptionMethod struct {
			Algorithm string `xml:"Algorithm,attr"`
		} `xml:"EncryptionMethod"`
		CipherValue string `xml:"CipherData>CipherValue"`
	} `xml:"EncryptedData"`
}

type innerXML struct {
	XMLName xml.Name `xml:"InnerContainer"`
	Version string   `xml:"version,attr"`
	Keys    []struct {
		Name         string `xml:"Name"`
		ActiveKeyset bool   `xml:"ActiveKeyset"`
		KeysetID     int    `xml:"KeysetId"`
		SLN          int    `xml:"Sln"`
		KeyTypeTek   bool   `xml:"KeyTypeTek"`
		KeyTypeKek   bool   `xml:"KeyTypeKek"`
		KeyID        int    `xml:"KeyId"`
		AlgorithmID  int    `xml:"AlgorithmId"`
		Key          string `xml:"Key"`
	} `xml:"Keys>KeyItem"`
}

const aes256CBC = "http://www.w3.org/2001/04/xmlenc#aes256-cbc"

// Open decrypts a container file's bytes with password.
func Open(file []byte, password string) (*Container, error) {
	outer, err := readOuter(file)
	if err != nil {
		return nil, err
	}
	kd := outer.KeyDerivation
	if kd.DerivationAlgorithm != "PBKDF2" || kd.HashAlgorithm != "SHA512" {
		return nil, fmt.Errorf("ekc: unsupported key derivation %s/%s (KFDtool writes PBKDF2/SHA512)", kd.DerivationAlgorithm, kd.HashAlgorithm)
	}
	if kd.KeyLength != 32 {
		return nil, fmt.Errorf("ekc: key length %d, want 32 (AES-256)", kd.KeyLength)
	}
	if kd.IterationCount < 1 || kd.IterationCount > maxIterations {
		return nil, fmt.Errorf("ekc: PBKDF2 iteration count %d out of range", kd.IterationCount)
	}
	if alg := outer.EncryptedData.EncryptionMethod.Algorithm; alg != aes256CBC {
		return nil, fmt.Errorf("ekc: unsupported encryption method %q", alg)
	}
	salt, err := base64.StdEncoding.DecodeString(strings.TrimSpace(kd.Salt))
	if err != nil {
		return nil, fmt.Errorf("ekc: bad salt: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(outer.EncryptedData.CipherValue), ""))
	if err != nil {
		return nil, fmt.Errorf("ekc: bad CipherValue: %w", err)
	}
	if len(ct) < 2*aes.BlockSize || len(ct)%aes.BlockSize != 0 {
		return nil, errors.New("ekc: CipherValue is not an IV plus whole AES blocks")
	}
	key, err := pbkdf2.Key(sha512.New, password, salt, kd.IterationCount, kd.KeyLength)
	if err != nil {
		return nil, fmt.Errorf("ekc: key derivation: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(ct)-aes.BlockSize)
	cipher.NewCBCDecrypter(block, ct[:aes.BlockSize]).CryptBlocks(plain, ct[aes.BlockSize:])
	// ISO 10126: only the last byte (the pad length) is meaningful.
	pad := int(plain[len(plain)-1])
	if pad < 1 || pad > aes.BlockSize {
		return nil, ErrWrongPassword
	}
	plain = plain[:len(plain)-pad]
	var inner innerXML
	if err := xml.Unmarshal(plain, &inner); err != nil {
		return nil, ErrWrongPassword
	}
	if major(inner.Version) != 1 {
		return nil, fmt.Errorf("ekc: inner container version %q, this reader knows 1.x", inner.Version)
	}
	c := &Container{}
	for _, k := range inner.Keys {
		m, err := hex.DecodeString(strings.TrimSpace(k.Key))
		if err != nil {
			return nil, fmt.Errorf("ekc: key %q (key ID %d): bad hex: %w", k.Name, k.KeyID, err)
		}
		c.Keys = append(c.Keys, Key{
			Name: k.Name, KeysetID: k.KeysetID, ActiveKeyset: k.ActiveKeyset, SLN: k.SLN,
			KeyID: k.KeyID, AlgorithmID: k.AlgorithmID, TEK: k.KeyTypeTek, KEK: k.KeyTypeKek, Material: m,
		})
	}
	return c, nil
}

// readOuter unwraps the length prefix and gzip stream and parses the outer XML.
func readOuter(file []byte) (*outerXML, error) {
	if len(file) < 4+2 || file[4] != 0x1f || file[5] != 0x8b {
		return nil, errors.New("ekc: not a KFDtool container (no length-prefixed gzip stream)")
	}
	want := binary.LittleEndian.Uint32(file[:4])
	if want == 0 || want > maxInflated {
		return nil, fmt.Errorf("ekc: implausible container length %d", want)
	}
	zr, err := gzip.NewReader(bytes.NewReader(file[4:]))
	if err != nil {
		return nil, fmt.Errorf("ekc: %w", err)
	}
	data, err := io.ReadAll(io.LimitReader(zr, maxInflated+1))
	if err != nil {
		return nil, fmt.Errorf("ekc: decompress: %w", err)
	}
	if len(data) > maxInflated {
		return nil, errors.New("ekc: container too large")
	}
	var outer outerXML
	if err := xml.Unmarshal(data, &outer); err != nil {
		return nil, fmt.Errorf("ekc: outer container: %w", err)
	}
	if major(outer.Version) != 1 {
		return nil, fmt.Errorf("ekc: outer container version %q, this reader knows 1.x", outer.Version)
	}
	return &outer, nil
}

// major is the major number of a "1.0"-style version, or -1.
func major(v string) int {
	n, err := strconv.Atoi(strings.SplitN(strings.TrimSpace(v), ".", 2)[0])
	if err != nil {
		return -1
	}
	return n
}
