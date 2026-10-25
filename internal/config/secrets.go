package config

import (
	"fmt"
	"strings"
)

// RedactedSecret stands in for a decryption key or key-container password
// when a config travels to a client that only needs to edit it (the config
// builder's GET /api/v1/config/file, whose reads are open), so the key
// material in a plain-text encryption_keys entry, or a container's password,
// never leaves the machine through the API (issue #1305). It can never be a
// valid key (not hex) and RestoreSecrets puts the stored value back before a
// save, so editing a config in the builder neither leaks nor loses its keys.
const RedactedSecret = "<redacted>"

// RedactSecrets returns a copy of c with every non-empty encryption_keys key
// and encryption_key_files password replaced by RedactedSecret. c is not
// modified.
func (c Config) RedactSecrets() Config {
	out := c
	out.Trunking.Systems = make([]SystemConfig, len(c.Trunking.Systems))
	for i, s := range c.Trunking.Systems {
		if len(s.EncryptionKeys) > 0 {
			keys := make([]EncryptionKeyConfig, len(s.EncryptionKeys))
			for j, k := range s.EncryptionKeys {
				if strings.TrimSpace(k.Key) != "" {
					k.Key = RedactedSecret
				}
				keys[j] = k
			}
			s.EncryptionKeys = keys
		}
		if len(s.EncryptionKeyFiles) > 0 {
			files := make([]EncryptionKeyFileConfig, len(s.EncryptionKeyFiles))
			for j, f := range s.EncryptionKeyFiles {
				if f.Password != "" {
					f.Password = RedactedSecret
				}
				files[j] = f
			}
			s.EncryptionKeyFiles = files
		}
		out.Trunking.Systems[i] = s
	}
	return out
}

// HasRedactedSecrets reports whether c carries any RedactedSecret.
func (c Config) HasRedactedSecrets() bool {
	for _, s := range c.Trunking.Systems {
		for _, k := range s.EncryptionKeys {
			if k.Key == RedactedSecret {
				return true
			}
		}
		for _, f := range s.EncryptionKeyFiles {
			if f.Password == RedactedSecret {
				return true
			}
		}
	}
	return false
}

// RestoreSecrets replaces every RedactedSecret in c with the value stored in
// onDisk, the config file being saved over: a key by its system's name and
// its key_id, a container password by its system's name and path. A
// placeholder with no stored counterpart (a renamed system, a changed key_id
// or path, a new file) is an error naming it, so a save can never write the
// placeholder in place of a key.
func (c *Config) RestoreSecrets(onDisk Config) error {
	stored := map[string]SystemConfig{}
	for _, s := range onDisk.Trunking.Systems {
		stored[s.Name] = s
	}
	for i := range c.Trunking.Systems {
		s := &c.Trunking.Systems[i]
		disk, haveSys := stored[s.Name]
		for j := range s.EncryptionKeys {
			k := &s.EncryptionKeys[j]
			if k.Key != RedactedSecret {
				continue
			}
			found := false
			for _, dk := range disk.EncryptionKeys {
				if haveSys && dk.KeyID == k.KeyID && dk.Key != RedactedSecret {
					k.Key, found = dk.Key, true
					break
				}
			}
			if !found {
				return fmt.Errorf("trunking.systems[%d] (%s).encryption_keys[%d]: the key for key_id %d is hidden and the saved file "+
					"has none to keep for that system and key_id; enter the key again", i, s.Name, j, k.KeyID)
			}
		}
		for j := range s.EncryptionKeyFiles {
			f := &s.EncryptionKeyFiles[j]
			if f.Password != RedactedSecret {
				continue
			}
			found := false
			for _, df := range disk.EncryptionKeyFiles {
				if haveSys && df.Path == f.Path && df.Password != RedactedSecret && df.Password != "" {
					f.Password, found = df.Password, true
					break
				}
			}
			if !found {
				return fmt.Errorf("trunking.systems[%d] (%s).encryption_key_files[%d]: the password for %s is hidden and the saved "+
					"file has none to keep for that system and path; enter it again", i, s.Name, j, f.Path)
			}
		}
	}
	return nil
}

// WithRedactedPlaceholders returns a copy of c in which every redacted key
// is a zero key of a length its algorithm accepts, so validation checks the
// rest of a redacted draft (the stored key was validated when it was saved).
// A redacted container password needs no stand-in: only its presence is
// validated.
func (c Config) WithRedactedPlaceholders() Config {
	out := c
	out.Trunking.Systems = make([]SystemConfig, len(c.Trunking.Systems))
	for i, s := range c.Trunking.Systems {
		if len(s.EncryptionKeys) > 0 {
			keys := make([]EncryptionKeyConfig, len(s.EncryptionKeys))
			for j, k := range s.EncryptionKeys {
				if k.Key == RedactedSecret {
					n := 5 // the RC4 family takes any length; ADP / DMR EP keys are 40-bit
					if l := EncryptionKeyLengths(k.NormalizedAlgorithm()); len(l) > 0 {
						n = l[0]
					}
					k.Key = strings.Repeat("00", n)
				}
				keys[j] = k
			}
			s.EncryptionKeys = keys
		}
		out.Trunking.Systems[i] = s
	}
	return out
}
