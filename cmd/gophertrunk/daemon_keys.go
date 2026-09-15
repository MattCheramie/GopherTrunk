package main

import (
	"log/slog"

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
