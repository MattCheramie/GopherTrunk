package main

import (
	"log/slog"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// buildKeyResolver turns trunking.systems[].encryption_keys into the
// composer's KeyResolver: (system name, algorithm, key ID) → raw key bytes.
// The config has already been validated (algorithm, hex, length, duplicate
// IDs), so a decode failure here is logged and the entry skipped rather
// than failing startup. Returns nil when no system configures a key, which
// leaves the voice path exactly as it was (issue #1187).
func buildKeyResolver(systems []config.SystemConfig, log *slog.Logger) composer.KeyResolver {
	type entry struct {
		alg string
		key []byte
	}
	keys := map[string]map[uint16]entry{}
	for _, s := range systems {
		for _, k := range s.EncryptionKeys {
			b, err := k.KeyBytes()
			if err != nil {
				log.Warn("daemon: encryption key skipped", "system", s.Name, "key_id", k.KeyID, "err", err)
				continue
			}
			if keys[s.Name] == nil {
				keys[s.Name] = map[uint16]entry{}
			}
			keys[s.Name][k.KeyID] = entry{alg: k.NormalizedAlgorithm(), key: b}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return func(system, algorithm string, keyID uint16) ([]byte, bool) {
		e, ok := keys[system][keyID]
		if !ok || e.alg != algorithm {
			return nil, false
		}
		return e.key, true
	}
}

// buildKeyConfigured turns trunking.systems[].encryption_keys into the
// recorder's "is this call decryptable" predicate (RecorderOptions.
// KeyConfigured): true when the system lists the call's key id, or — while
// the key id is still unknown (0, before the first Encryption Sync / PI
// header) — when the system has any key at all, mirroring the engine's
// systemHasKeys deferral. The algorithm is not checked here: the voice
// chain refuses a key whose length disagrees with the ALGID on air, and a
// mismatched entry is an operator error better seen in the log than
// silently muted. A call from a system with no keys of its own (the
// conventional scanner's synthetic grants, system "scanner") is covered by
// the conventional channels' keys as a whole — the recorder knows the
// call's system, not its frequency, so a key id configured on any channel
// counts (issue #1261). Returns nil when nothing configures a key.
func buildKeyConfigured(systems []config.SystemConfig) func(system string, algID uint8, keyID uint16) bool {
	keys := map[string]map[uint16]bool{}
	conv := map[uint16]bool{}
	for _, s := range systems {
		for _, k := range s.EncryptionKeys {
			if keys[s.Name] == nil {
				keys[s.Name] = map[uint16]bool{}
			}
			keys[s.Name][k.KeyID] = true
			if strings.HasPrefix(s.Name, conventionalKeySystemPrefix) {
				conv[k.KeyID] = true
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return func(system string, _ uint8, keyID uint16) bool {
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
		return ids[keyID]
	}
}

// conventionalKeySystemPrefix is the prefix of composer.ConventionalKeySystem
// names.
var conventionalKeySystemPrefix = composer.ConventionalKeySystem(0)[:len("conventional:")]
