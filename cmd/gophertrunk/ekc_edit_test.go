package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/ekc"
)

// ekcRun runs one `gophertrunk ekc` command and returns its report.
func ekcRun(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := ekcMain(args, &out)
	return out.String(), err
}

// noPrompt makes any password / key prompt fail the test: these commands
// must take everything from the environment variables they were given.
func noPrompt(t *testing.T) {
	t.Helper()
	prompt := ekcPasswordPrompt
	t.Cleanup(func() { ekcPasswordPrompt = prompt })
	ekcPasswordPrompt = func(p string) (string, bool, error) {
		t.Errorf("unexpected prompt %q", p)
		return "", false, nil
	}
}

// The algorithm ↔ ALGID mapping round-trips for every algorithm and key
// length GopherTrunk decrypts, so a key comes back as the algorithm it went
// in as.
func TestEKCAlgorithmIDRoundTrips(t *testing.T) {
	for _, tc := range []struct {
		alg  string
		n    int
		id   int
		back string
	}{
		{"des", 8, 0x81, "des"}, {"tdes", 16, 0x83, "tdes"}, {"3des", 24, 0x86, "tdes"},
		{"aes-128", 16, 0x85, "aes"}, {"aes", 32, 0x84, "aes"},
		{"rc4", 5, 0xAA, "rc4"}, {"adp", 5, 0xAA, "rc4"}, {"arc4", 13, 0xAA, "rc4"},
	} {
		id, err := ekcAlgorithmID(tc.alg, tc.n)
		if err != nil || id != tc.id || ekcAlgorithm(id) != tc.back {
			t.Errorf("%s/%d bytes: ALGID %#x err %v (back %q), want %#x → %q", tc.alg, tc.n, id, err, ekcAlgorithm(id), tc.id, tc.back)
		}
	}
	if _, err := ekcAlgorithmID("des-xl", 8); err == nil {
		t.Error("an algorithm GopherTrunk does not decrypt was accepted")
	}
}

// TestEKCCreateAddRemoveAndDaemonLoad is the keystore's whole loop: create a
// container, add an AES, a DES and a DMR Enhanced Privacy (RC4) key, list it,
// remove one, change the password; then the daemon's own loader
// (withKeyFileKeys) reads the container and yields exactly the keys that went
// in. Key material is never printed and never comes from the command line.
func TestEKCCreateAddRemoveAndDaemonLoad(t *testing.T) {
	noPrompt(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.ekc")
	t.Setenv("GT_TEST_EKC_PW", "correct horse")
	t.Setenv("GT_TEST_EKC_NEWPW", "battery staple")

	if _, err := ekcRun(t, "create", "-out", path, "-password-env", "GT_TEST_EKC_PW"); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("container mode %v (err %v), want 0600", fi.Mode().Perm(), err)
	}
	if _, err := ekcRun(t, "create", "-out", path, "-password-env", "GT_TEST_EKC_PW"); err == nil {
		t.Error("create overwrote an existing container")
	}

	keys := []struct{ id, alg, name, hex string }{
		{"1", "aes", "County AES", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"},
		{"42", "des", "Fire DES", "0123456789ABCDEF"},
		{"11", "rc4", "Ops DMR EP", "12 34 56 78 90"},
		{"9", "tdes", "gone", "00112233445566778899AABBCCDDEEFF"},
	}
	for _, k := range keys {
		t.Setenv("GT_TEST_EKC_KEY", k.hex)
		out, err := ekcRun(t, "add", "-in", path, "-password-env", "GT_TEST_EKC_PW", "-key-env", "GT_TEST_EKC_KEY",
			"-key-id", k.id, "-algorithm", k.alg, "-name", k.name)
		if err != nil {
			t.Fatalf("add %s: %v", k.name, err)
		}
		if strings.Contains(strings.ToUpper(out), strings.ToUpper(strings.ReplaceAll(k.hex, " ", ""))) {
			t.Errorf("add printed the key: %q", out)
		}
	}
	// A bad key length is refused before anything is written.
	t.Setenv("GT_TEST_EKC_KEY", "0011")
	if _, err := ekcRun(t, "add", "-in", path, "-password-env", "GT_TEST_EKC_PW", "-key-env", "GT_TEST_EKC_KEY",
		"-key-id", "5", "-algorithm", "aes"); err == nil || !strings.Contains(err.Error(), "aes key is 16 or 32 bytes") {
		t.Errorf("short AES key: err = %v", err)
	}

	list, err := ekcRun(t, "list", "-in", path, "-password-env", "GT_TEST_EKC_PW")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"1     County AES", "3     Ops DMR EP", "0xAA (ADP/RC4)", "4 key(s) would load"} {
		if !strings.Contains(list, want) {
			t.Errorf("listing lacks %q:\n%s", want, list)
		}
	}
	if out, err := ekcRun(t, "remove", "-in", path, "-password-env", "GT_TEST_EKC_PW", "-item", "4"); err != nil ||
		!strings.Contains(out, `removed item 4`) {
		t.Fatalf("remove: %q, %v", out, err)
	}
	if _, err := ekcRun(t, "remove", "-in", path, "-password-env", "GT_TEST_EKC_PW", "-item", "4"); err == nil {
		t.Error("removing a missing item succeeded")
	}
	if _, err := ekcRun(t, "passwd", "-in", path, "-password-env", "GT_TEST_EKC_PW", "-new-password-env", "GT_TEST_EKC_NEWPW"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if _, err := ekc.Open(data, "correct horse"); err == nil {
		t.Error("the old password still opens the container")
	}

	// The daemon's loader takes the container like any KFDtool one.
	systems, err := withKeyFileKeys([]config.SystemConfig{{
		Name:               "county",
		EncryptionKeyFiles: []config.EncryptionKeyFileConfig{{Path: path, PasswordEnv: "GT_TEST_EKC_NEWPW"}},
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	got := keyIDs(systems[0].EncryptionKeys)
	want := map[uint16]config.EncryptionKeyConfig{
		1:  {KeyID: 1, Algorithm: "aes", Key: "000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F"},
		42: {KeyID: 42, Algorithm: "des", Key: "0123456789ABCDEF"},
		11: {KeyID: 11, Algorithm: "rc4", Key: "1234567890"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("daemon loaded %+v\nwant %+v", got, want)
	}
}

// TestEKCImportMovesYAMLKeysIntoAContainer: `ekc import` takes a system's
// plain-text encryption_keys into a container and says what to put in the
// config instead; a config using that container loads the same keys the YAML
// held.
func TestEKCImportMovesYAMLKeysIntoAContainer(t *testing.T) {
	noPrompt(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "gophertrunk.yaml")
	yaml := `trunking:
  systems:
    - name: county
      protocol: p25
      control_channels: [851012500]
      encryption_keys:
        - { key_id: 1, algorithm: aes-256, key: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f" }
        - { key_id: 42, algorithm: des, key: "0123456789abcdef" }
        - { key_id: 7, algorithm: adp, key: "1234567890" }
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "county.ekc")
	t.Setenv("GT_TEST_EKC_PW", "correct horse")
	out, err := ekcRun(t, "import", "-config", cfgPath, "-system", "county", "-out", path, "-password-env", "GT_TEST_EKC_PW")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`wrote 3 of 3 key(s) from system "county"`, "encryption_key_files:", "path: " + path, "password_env: GT_TEST_EKC_PW"} {
		if !strings.Contains(out, want) {
			t.Errorf("import report lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "0123456789") {
		t.Errorf("import printed key material:\n%s", out)
	}
	// Importing again adds nothing: every key ID is already in keyset 1.
	if out, err := ekcRun(t, "import", "-config", cfgPath, "-system", "county", "-out", path, "-password-env", "GT_TEST_EKC_PW"); err != nil ||
		!strings.Contains(out, "wrote 0 of 3") || !strings.Contains(out, "skipped key ID 42") {
		t.Errorf("second import: %q, %v", out, err)
	}

	before := keyIDs(cfg.Trunking.Systems[0].EncryptionKeys)
	sys := cfg.Trunking.Systems[0]
	sys.EncryptionKeys = nil
	sys.EncryptionKeyFiles = []config.EncryptionKeyFileConfig{{Path: path, PasswordEnv: "GT_TEST_EKC_PW"}}
	systems, err := withKeyFileKeys([]config.SystemConfig{sys}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	after := keyIDs(systems[0].EncryptionKeys)
	if len(after) != len(before) {
		t.Fatalf("container gives %d keys, the YAML had %d", len(after), len(before))
	}
	for id, b := range before {
		a := after[id]
		bb, _ := b.KeyBytes()
		ab, _ := a.KeyBytes()
		if a.NormalizedAlgorithm() != b.NormalizedAlgorithm() || !bytes.Equal(ab, bb) {
			t.Errorf("key ID %d: container %s/%X, YAML %s/%X", id, a.NormalizedAlgorithm(), ab, b.NormalizedAlgorithm(), bb)
		}
	}

	if _, err := ekcRun(t, "import", "-config", cfgPath, "-system", "nope", "-out", path, "-password-env", "GT_TEST_EKC_PW"); err == nil {
		t.Error("an unknown system was accepted")
	}
}

// A new container's password must be confirmed and long enough.
func TestEKCNewPasswordRules(t *testing.T) {
	prompt := ekcPasswordPrompt
	defer func() { ekcPasswordPrompt = prompt }()
	answers := []string{"correct horse", "correct hors"}
	ekcPasswordPrompt = func(string) (string, bool, error) {
		a := answers[0]
		answers = answers[1:]
		return a, true, nil
	}
	if _, err := ekcNewPassword("", "x.ekc"); err == nil || !strings.Contains(err.Error(), "differ") {
		t.Errorf("mismatched passwords: err = %v", err)
	}
	t.Setenv("GT_TEST_EKC_SHORT", "short")
	if _, err := ekcNewPassword("GT_TEST_EKC_SHORT", "x.ekc"); err == nil || !strings.Contains(err.Error(), "at least 8") {
		t.Errorf("short password: err = %v", err)
	}
}

// The bare `ekc -in file` form still lists, as before the subcommands.
func TestEKCBareFormStillLists(t *testing.T) {
	noPrompt(t)
	t.Setenv("GT_TEST_EKC_REF", ekcReferencePassword)
	out, err := ekcRun(t, "-in", ekcReference, "-password-env", "GT_TEST_EKC_REF")
	if err != nil || !strings.Contains(out, "ITEM") || !strings.Contains(out, "4 key(s) would load") {
		t.Errorf("bare form: %v\n%s", err, out)
	}
	if _, err := ekcRun(t, "frobnicate"); err == nil {
		t.Error("an unknown subcommand was accepted")
	}
}
