package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/ekc"
)

// ekcReference is a container written by KFDtool's own code (see
// internal/ekc/ekc_test.go), password ekcReferencePassword. It holds an
// active keyset 1 (AES-256 id 1, DES-OFB id 0x2A, ADP id 3, TDES id 0x1234,
// a KEK id 4, DES-XL id 6) and an inactive keyset 2 (AES-128 id 1).
const (
	ekcReference         = "../../internal/ekc/testdata/kfdtool_reference.ekc"
	ekcReferencePassword = "gophertrunk-1305"
)

func openReference(t *testing.T) *ekc.Container {
	t.Helper()
	b, err := os.ReadFile(ekcReference)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ekc.Open(b, ekcReferencePassword)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func keyIDs(keys []config.EncryptionKeyConfig) map[uint16]config.EncryptionKeyConfig {
	m := map[uint16]config.EncryptionKeyConfig{}
	for _, k := range keys {
		m[k.KeyID] = k
	}
	return m
}

// TestEKCKeysTakesTEKsOfTheActiveKeyset: the TEKs GopherTrunk can decrypt
// with, the active keyset's key where two keysets share a key ID, and an
// account of what was left out.
func TestEKCKeysTakesTEKsOfTheActiveKeyset(t *testing.T) {
	load := ekcKeys(openReference(t), 0)
	got := keyIDs(load.keys)
	want := map[uint16]config.EncryptionKeyConfig{
		1:      {KeyID: 1, Algorithm: "aes", Key: "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F"},
		0x2A:   {KeyID: 0x2A, Algorithm: "des", Key: "0123456789ABCDEF"},
		3:      {KeyID: 3, Algorithm: "rc4", Key: "1234567890"},
		0x1234: {KeyID: 0x1234, Algorithm: "tdes", Key: "00112233445566778899AABBCCDDEEFF0011223344556677"},
	}
	if len(got) != len(want) {
		t.Fatalf("keys = %+v, want %+v", load.keys, want)
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("key %#x = %+v, want %+v", id, got[id], w)
		}
		if err := got[id].Validate(); err != nil {
			t.Errorf("key %#x does not validate as an encryption_keys entry: %v", id, err)
		}
	}
	sk := load.skipped()
	for _, part := range []string{"1 KEK", "1 superseded by the active keyset", "0x9F (DES-XL)"} {
		if !strings.Contains(sk, part) {
			t.Errorf("skipped = %q, missing %q", sk, part)
		}
	}
}

// TestEKCKeysKeysetFilter: keyset: 2 loads only that keyset (its AES-128 key).
func TestEKCKeysKeysetFilter(t *testing.T) {
	load := ekcKeys(openReference(t), 2)
	if len(load.keys) != 1 || load.keys[0] != (config.EncryptionKeyConfig{KeyID: 1, Algorithm: "aes", Key: "A0A1A2A3A4A5A6A7A8A9AAABACADAEAF"}) {
		t.Fatalf("keys = %+v, want keyset 2's AES-128 key id 1 only", load.keys)
	}
}

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// TestWithKeyFileKeysMergesWithoutTouchingConfig: a container's keys reach
// the key resolver, an explicit encryption_keys entry wins on its key ID,
// and the caller's config is not modified (the daemon's cfg must never hold
// decrypted keys: the API can serve and write it back).
func TestWithKeyFileKeysMergesWithoutTouchingConfig(t *testing.T) {
	t.Setenv("GT_TEST_EKC_PASSWORD", ekcReferencePassword)
	explicit := config.EncryptionKeyConfig{KeyID: 3, Algorithm: "rc4", Key: "AABBCCDDEE"}
	systems := []config.SystemConfig{{
		Name:               "county",
		EncryptionKeys:     []config.EncryptionKeyConfig{explicit},
		EncryptionKeyFiles: []config.EncryptionKeyFileConfig{{Path: ekcReference, PasswordEnv: "GT_TEST_EKC_PASSWORD"}},
	}}
	var logBuf bytes.Buffer
	merged, err := withKeyFileKeys(systems, slog.New(slog.NewTextHandler(&logBuf, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if len(systems[0].EncryptionKeys) != 1 {
		t.Fatalf("the caller's config gained keys: %+v", systems[0].EncryptionKeys)
	}
	got := keyIDs(merged[0].EncryptionKeys)
	if len(got) != 4 || got[3] != explicit {
		t.Fatalf("merged = %+v, want the 4 container TEK IDs with key 3 from encryption_keys", merged[0].EncryptionKeys)
	}
	res := buildKeyResolver(merged, quietLog())
	if k, ok := res("county", "aes", 1); !ok || len(k) != 32 {
		t.Fatalf("resolver(aes, 1) = %x, %v; want the container's AES-256 key", k, ok)
	}
	if k, ok := res("county", "rc4", 3); !ok || !bytes.Equal(k, []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE}) {
		t.Fatalf("resolver(rc4, 3) = %x, %v; want the explicit encryption_keys key", k, ok)
	}
	log := logBuf.String()
	if !strings.Contains(log, "keys=3") || !strings.Contains(log, "overridden_by_encryption_keys=1") {
		t.Errorf("load log line = %q", log)
	}
	if strings.Contains(log, "000102030405") {
		t.Errorf("key material in the log: %q", log)
	}
}

// TestOpenKeyFilePasswordSources: the password comes from the entry, its
// environment variable, or the terminal; without any of them startup fails
// with a message saying how to give one.
func TestOpenKeyFilePasswordSources(t *testing.T) {
	prompt := ekcPasswordPrompt
	defer func() { ekcPasswordPrompt = prompt }()

	if _, err := openKeyFile("s", config.EncryptionKeyFileConfig{Path: ekcReference, Password: ekcReferencePassword}); err != nil {
		t.Errorf("password: %v", err)
	}
	if _, err := openKeyFile("s", config.EncryptionKeyFileConfig{Path: ekcReference, PasswordEnv: "GT_TEST_EKC_UNSET"}); err == nil || !strings.Contains(err.Error(), "GT_TEST_EKC_UNSET is not set") {
		t.Errorf("unset password_env: err = %v", err)
	}

	ekcPasswordPrompt = func(string) (string, bool, error) { return "", false, nil } // no terminal
	if _, err := openKeyFile("s", config.EncryptionKeyFileConfig{Path: ekcReference}); err == nil || !strings.Contains(err.Error(), "password_env") {
		t.Errorf("no terminal: err = %v", err)
	}

	var asked int
	ekcPasswordPrompt = func(string) (string, bool, error) {
		asked++
		if asked == 2 {
			return ekcReferencePassword, true, nil
		}
		return "typo", true, nil
	}
	if _, err := openKeyFile("s", config.EncryptionKeyFileConfig{Path: ekcReference}); err != nil || asked != 2 {
		t.Errorf("prompt, right on the second try: err = %v after %d prompts", err, asked)
	}

	asked = 0
	ekcPasswordPrompt = func(string) (string, bool, error) { asked++; return "typo", true, nil }
	if _, err := openKeyFile("s", config.EncryptionKeyFileConfig{Path: ekcReference}); !errors.Is(err, ekc.ErrWrongPassword) || asked != ekcPromptTries {
		t.Errorf("prompt, always wrong: err = %v after %d prompts, want ErrWrongPassword after %d", err, asked, ekcPromptTries)
	}
}

// TestEKCListingShowsWhatLoadsWithoutKeyMaterial: `gophertrunk ekc` marks the
// keys the daemon would load (the active keyset's AES key, not its inactive
// twin; no KEK or DES-XL) and never prints a key.
func TestEKCListingShowsWhatLoadsWithoutKeyMaterial(t *testing.T) {
	var out bytes.Buffer
	writeEKCListing(&out, openReference(t), 0)
	s := out.String()
	for _, want := range []string{
		"County AES       1 (active)  1    1 (0x1)        0x84 (AES-256)  TEK   32     aes",
		"County AES next  2           1    1 (0x1)        0x85 (AES-128)  TEK   16     -",
		"Master KEK", "Legacy DES-XL",
		"4 key(s) would load",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("listing lacks %q:\n%s", want, s)
		}
	}
	for _, secret := range []string{"000102030405", "0123456789ABCDEF", "1234567890\n", "A0A1A2A3"} {
		if strings.Contains(s, secret) {
			t.Errorf("listing shows key material %q:\n%s", secret, s)
		}
	}
	out.Reset()
	writeEKCListing(&out, openReference(t), 2)
	if !strings.Contains(out.String(), "County AES next  2           1    1 (0x1)        0x85 (AES-128)  TEK   16     aes") ||
		!strings.Contains(out.String(), "1 key(s) would load") {
		t.Errorf("keyset 2 listing:\n%s", out.String())
	}
}
