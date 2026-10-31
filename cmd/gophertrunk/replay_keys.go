package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// replayKeyFlags is `replay -key`, repeatable (issue #1187): decryption keys
// for the -record-voice / -audio-out voice path, so an encrypted capture can
// be checked offline through the exact composer path the daemon runs with
// trunking.systems[].encryption_keys — with the release binary, no config
// file and no Go toolchain.
//
// Each value is ALG:KEYID:HEXKEY, or KEYID:HEXKEY for RC4:
//
//	-key rc4:11:4E77AD0B51     DMR Enhanced Privacy, key ID 11
//	-key adp:1:1234567890      P25 ADP (the same RC4 family)
//
// KEYID is decimal (or 0x-prefixed hex). Each key is validated exactly like a
// config entry (EncryptionKeyConfig.Validate), and a repeated key ID is
// rejected as it is in a system's list.
type replayKeyFlags []config.EncryptionKeyConfig

func (k *replayKeyFlags) String() string {
	parts := make([]string, 0, len(*k))
	for _, e := range *k {
		parts = append(parts, fmt.Sprintf("%s:%d", e.NormalizedAlgorithm(), e.KeyID))
	}
	return strings.Join(parts, ",")
}

func (k *replayKeyFlags) Set(v string) error {
	e, err := parseReplayKey(v)
	if err != nil {
		return err
	}
	for _, have := range *k {
		if have.KeyID == e.KeyID {
			return fmt.Errorf("-key %q: key ID %d given twice", v, e.KeyID)
		}
	}
	*k = append(*k, e)
	return nil
}

func parseReplayKey(v string) (config.EncryptionKeyConfig, error) {
	parts := strings.Split(strings.TrimSpace(v), ":")
	var alg, kid, key string
	switch len(parts) {
	case 2:
		alg, kid, key = "rc4", parts[0], parts[1]
	case 3:
		alg, kid, key = parts[0], parts[1], parts[2]
	default:
		return config.EncryptionKeyConfig{}, fmt.Errorf("-key %q: want ALG:KEYID:HEXKEY (e.g. rc4:11:4E77AD0B51) or KEYID:HEXKEY", v)
	}
	id, err := strconv.ParseUint(strings.TrimSpace(kid), 0, 16)
	if err != nil {
		return config.EncryptionKeyConfig{}, fmt.Errorf("-key %q: key ID %q: want 0..65535", v, kid)
	}
	e := config.EncryptionKeyConfig{KeyID: uint16(id), Algorithm: alg, Key: key}
	if err := e.Validate(); err != nil {
		return config.EncryptionKeyConfig{}, fmt.Errorf("-key %q: %w", v, err)
	}
	return e, nil
}

// replayKeyResolver builds the composer's KeyResolver from -key values,
// through the daemon's own buildKeyResolver so a key matches in replay
// exactly when it would match live. A replay decodes one system, so the
// grant's system name is not used to pick the key. nil when no key is given
// (the voice path is then byte-identical to before).
func replayKeyResolver(keys []config.EncryptionKeyConfig, log *slog.Logger) composer.KeyResolver {
	if len(keys) == 0 {
		return nil
	}
	const sys = "replay"
	res := buildKeyResolver([]config.SystemConfig{{Name: sys, EncryptionKeys: keys}}, log)
	if res == nil {
		return nil
	}
	return func(_, algorithm string, keyID uint16, group uint32) ([]byte, bool) {
		return res(sys, algorithm, keyID, group)
	}
}
