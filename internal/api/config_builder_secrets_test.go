package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestConfigBuilderHidesDecryptionKeys pins issue #1305's "keys stay out of
// reach" on the config builder API, whose reads are open: GET
// /api/v1/config/file must not send a plain-text encryption_keys key or a
// key container's password, the redacted draft must still validate, and
// saving it back must keep the stored keys (and refuse a placeholder it has
// nothing to restore from) instead of writing the placeholder to disk.
func TestConfigBuilderHidesDecryptionKeys(t *testing.T) {
	base, dir, teardown := cbServer(t)
	defer teardown()

	const aesKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	const ekcPassword = "correct horse battery"
	cfg := config.Default()
	cfg.Trunking.Systems = []config.SystemConfig{{
		Name: "county", Protocol: "p25", ControlChannels: []uint32{851_012_500},
		EncryptionKeys:     []config.EncryptionKeyConfig{{KeyID: 1, Algorithm: "aes", Key: aesKey}},
		EncryptionKeyFiles: []config.EncryptionKeyFileConfig{{Path: "keys.ekc", Password: ekcPassword}},
	}}
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// The raw response carries neither secret.
	resp, err := http.Get(base + "/api/v1/config/file?path=" + path)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(strings.ToLower(string(raw)), aesKey) || strings.Contains(string(raw), ekcPassword) {
		t.Fatalf("GET /config/file leaks a key or password:\n%s", raw)
	}
	var loaded ConfigLoadResponse
	getJSON(t, base+"/api/v1/config/file?path="+path, &loaded)
	if !loaded.Validation.OK {
		t.Errorf("the stored file should validate: %+v", loaded.Validation)
	}
	sys := loaded.Config.Trunking.Systems[0]
	if sys.EncryptionKeys[0].Key != config.RedactedSecret || sys.EncryptionKeyFiles[0].Password != config.RedactedSecret {
		t.Fatalf("loaded secrets %q / %q, want the placeholder", sys.EncryptionKeys[0].Key, sys.EncryptionKeyFiles[0].Password)
	}

	// The redacted draft validates as it is.
	var vr ValidationResult
	postJSON(t, base+"/api/v1/config/validate", ConfigValidateRequest{Config: loaded.Config}, &vr)
	if !vr.OK {
		t.Errorf("redacted draft fails validation: %+v", vr)
	}

	// Edit something else and add a key; save; the stored key and password
	// are still on disk, next to the new key.
	draft := loaded.Config
	draft.Trunking.Systems[0].ControlChannels = []uint32{851_037_500}
	draft.Trunking.Systems[0].EncryptionKeys = append(draft.Trunking.Systems[0].EncryptionKeys,
		config.EncryptionKeyConfig{KeyID: 9, Algorithm: "des", Key: "0123456789abcdef"})
	var sr ConfigSaveResponse
	if code := postJSONStatus(t, base+"/api/v1/config/file", ConfigSaveRequest{Path: path, Config: draft, Mtime: loaded.Mtime, Overwrite: true}, &sr); code != http.StatusOK {
		t.Fatalf("save status %d", code)
	}
	saved, err := config.Load(path)
	if err != nil {
		t.Fatalf("saved config does not load: %v", err)
	}
	ss := saved.Trunking.Systems[0]
	if ss.EncryptionKeys[0].Key != aesKey || ss.EncryptionKeys[1].Key != "0123456789abcdef" ||
		ss.EncryptionKeyFiles[0].Password != ekcPassword || ss.ControlChannels[0] != 851_037_500 {
		t.Errorf("saved system %+v: want the stored key and password kept, the edits applied", ss)
	}
	if b, _ := os.ReadFile(path); strings.Contains(string(b), config.RedactedSecret) {
		t.Errorf("the placeholder reached the file:\n%s", b)
	}

	// A placeholder whose key_id changed has nothing to restore: refused, file untouched.
	before, _ := os.ReadFile(path)
	var again ConfigLoadResponse
	getJSON(t, base+"/api/v1/config/file?path="+path, &again)
	again.Config.Trunking.Systems[0].EncryptionKeys[0].KeyID = 2
	if code := postJSONStatus(t, base+"/api/v1/config/file", ConfigSaveRequest{Path: path, Config: again.Config, Mtime: again.Mtime, Overwrite: true}, nil); code != http.StatusBadRequest {
		t.Errorf("save of an unrestorable placeholder: status %d, want 400", code)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Error("a refused save changed the file")
	}
}
