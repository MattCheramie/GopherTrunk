package config

import (
	"strings"
	"testing"
)

func secretConfig() Config {
	c := Default()
	c.Trunking.Systems = []SystemConfig{{
		Name: "county", Protocol: "p25", ControlChannels: []uint32{851_012_500},
		EncryptionKeys: []EncryptionKeyConfig{
			{KeyID: 1, Algorithm: "aes", Key: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"},
			{KeyID: 7, Algorithm: "adp", Key: "1234567890"},
		},
		EncryptionKeyFiles: []EncryptionKeyFileConfig{
			{Path: "keys.ekc", Password: "correct horse"},
			{Path: "other.ekc", PasswordEnv: "GT_EKC"},
		},
	}}
	return c
}

func TestRedactSecretsHidesKeysWithoutTouchingTheOriginal(t *testing.T) {
	c := secretConfig()
	r := c.RedactSecrets()
	for _, k := range r.Trunking.Systems[0].EncryptionKeys {
		if k.Key != RedactedSecret {
			t.Errorf("key ID %d: %q, want redacted", k.KeyID, k.Key)
		}
	}
	files := r.Trunking.Systems[0].EncryptionKeyFiles
	if files[0].Password != RedactedSecret || files[1].Password != "" || files[1].PasswordEnv != "GT_EKC" {
		t.Errorf("key files %+v: want the password redacted, an empty one left empty, password_env kept", files)
	}
	if c.Trunking.Systems[0].EncryptionKeys[0].Key == RedactedSecret || c.Trunking.Systems[0].EncryptionKeyFiles[0].Password == RedactedSecret {
		t.Error("RedactSecrets modified the original config")
	}
	if !r.HasRedactedSecrets() || c.HasRedactedSecrets() {
		t.Error("HasRedactedSecrets wrong")
	}
	// The placeholder is never a valid key: a file holding it does not load.
	if err := (EncryptionKeyConfig{KeyID: 1, Algorithm: "aes", Key: RedactedSecret}).Validate(); err == nil {
		t.Error("the placeholder validated as a key")
	}
}

func TestRestoreSecretsFromTheSavedFile(t *testing.T) {
	disk := secretConfig()
	draft := disk.RedactSecrets()
	// The operator adds a key and edits another field; the redacted ones
	// stay as loaded.
	s := &draft.Trunking.Systems[0]
	s.EncryptionKeys = append(s.EncryptionKeys, EncryptionKeyConfig{KeyID: 9, Algorithm: "des", Key: "0123456789abcdef"})
	s.ControlChannels = []uint32{851_037_500}
	if err := draft.RestoreSecrets(disk); err != nil {
		t.Fatal(err)
	}
	got := draft.Trunking.Systems[0]
	if got.EncryptionKeys[0].Key != disk.Trunking.Systems[0].EncryptionKeys[0].Key ||
		got.EncryptionKeys[1].Key != "1234567890" || got.EncryptionKeys[2].Key != "0123456789abcdef" ||
		got.EncryptionKeyFiles[0].Password != "correct horse" {
		t.Errorf("restored %+v", got)
	}
	if draft.HasRedactedSecrets() {
		t.Error("a placeholder survived the restore")
	}
}

func TestRestoreSecretsRefusesWhatItCannotRestore(t *testing.T) {
	disk := secretConfig()
	for name, edit := range map[string]func(*Config){
		"changed key ID":   func(c *Config) { c.Trunking.Systems[0].EncryptionKeys[0].KeyID = 2 },
		"renamed system":   func(c *Config) { c.Trunking.Systems[0].Name = "county2" },
		"changed key path": func(c *Config) { c.Trunking.Systems[0].EncryptionKeyFiles[0].Path = "moved.ekc" },
	} {
		draft := disk.RedactSecrets()
		edit(&draft)
		err := draft.RestoreSecrets(disk)
		if err == nil || !strings.Contains(err.Error(), "hidden") {
			t.Errorf("%s: err = %v, want a refusal", name, err)
		}
	}
	// A new file has nothing on disk to restore from.
	draft := disk.RedactSecrets()
	if err := draft.RestoreSecrets(Config{}); err == nil {
		t.Error("restoring against an empty file succeeded")
	}
}

func TestRedactedDraftStillValidates(t *testing.T) {
	r := secretConfig().RedactSecrets().WithRedactedPlaceholders()
	for _, err := range r.ValidateAll() {
		if strings.Contains(err.Error(), "key") {
			t.Errorf("redacted draft fails key validation: %v", err)
		}
	}
	for _, k := range r.Trunking.Systems[0].EncryptionKeys {
		if err := k.Validate(); err != nil {
			t.Errorf("placeholder for key ID %d invalid: %v", k.KeyID, err)
		}
	}
}
