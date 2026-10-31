package main

import (
	"log/slog"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// buildKeyResolver turns trunking.systems[].encryption_keys into the
// composer's KeyResolver: (system name, algorithm, key ID, talkgroup) → raw
// key bytes. An entry scoped to the call's talkgroup wins over an unscoped
// entry with the same key ID (issue #1261: one key ID, a different key per
// talkgroup); an unscoped entry matches every talkgroup. The config has
// already been validated (algorithm, hex, length, duplicate key ID /
// talkgroup pairs), so a decode failure here is logged and the entry skipped
// rather than failing startup. Returns nil when no system configures a key,
// which leaves the voice path exactly as it was (issue #1187).
func buildKeyResolver(systems []config.SystemConfig, log *slog.Logger) composer.KeyResolver {
	type entry struct {
		alg string
		key []byte
	}
	keys := map[string]map[config.EncryptionKeyScope]entry{}
	for _, s := range systems {
		for _, k := range s.EncryptionKeys {
			b, err := k.KeyBytes()
			if err != nil {
				log.Warn("daemon: encryption key skipped", "system", s.Name, "key_id", k.KeyID, "talkgroup", k.Talkgroup, "err", err)
				continue
			}
			if keys[s.Name] == nil {
				keys[s.Name] = map[config.EncryptionKeyScope]entry{}
			}
			keys[s.Name][k.Scope()] = entry{alg: k.NormalizedAlgorithm(), key: b}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return func(system, algorithm string, keyID uint16, group uint32) ([]byte, bool) {
		sys := keys[system]
		if group != 0 {
			if e, ok := sys[config.EncryptionKeyScope{KeyID: keyID, Talkgroup: group}]; ok && e.alg == algorithm {
				return e.key, true
			}
		}
		e, ok := sys[config.EncryptionKeyScope{KeyID: keyID}]
		if !ok || e.alg != algorithm {
			return nil, false
		}
		return e.key, true
	}
}

// buildKeyConfigured turns trunking.systems[].encryption_keys into the
// recorder's "is this call decryptable" predicate (RecorderOptions.
// KeyConfigured): true when the system lists the call's key id — unscoped,
// or scoped to the call's talkgroup (issue #1261) — or, while the key id is
// still unknown (0, before the first Encryption Sync / PI header), when the
// system has any key at all, mirroring the engine's systemHasKeys deferral.
// The algorithm is not checked here: the voice chain refuses a key whose
// length disagrees with the ALGID on air, and a mismatched entry is an
// operator error better seen in the log than silently muted. A call from a
// system with no keys of its own (the conventional scanner's synthetic
// grants, system "scanner") is covered by the conventional channels' keys as
// a whole — the recorder knows the call's system, not its frequency, so a
// key id configured on any channel counts (issue #1261). Returns nil when
// nothing configures a key.
func buildKeyConfigured(systems []config.SystemConfig) func(system string, algID uint8, keyID uint16, group uint32) bool {
	keys := map[string]map[config.EncryptionKeyScope]bool{}
	conv := map[config.EncryptionKeyScope]bool{}
	for _, s := range systems {
		for _, k := range s.EncryptionKeys {
			if keys[s.Name] == nil {
				keys[s.Name] = map[config.EncryptionKeyScope]bool{}
			}
			keys[s.Name][k.Scope()] = true
			if strings.HasPrefix(s.Name, conventionalKeySystemPrefix) {
				conv[k.Scope()] = true
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return func(system string, _ uint8, keyID uint16, group uint32) bool {
		ids := keys[system]
		if len(ids) == 0 {
			ids = conv
		}
		if len(ids) == 0 {
			return false
		}
		if keyID == 0 {
			return true
		}
		if ids[config.EncryptionKeyScope{KeyID: keyID}] {
			return true
		}
		return group != 0 && ids[config.EncryptionKeyScope{KeyID: keyID, Talkgroup: group}]
	}
}

// conventionalKeySystemPrefix is the prefix of composer.ConventionalKeySystem
// names.
var conventionalKeySystemPrefix = composer.ConventionalKeySystem(0)[:len("conventional:")]
