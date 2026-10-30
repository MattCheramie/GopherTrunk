package main

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// TestKeyResolverTalkgroupScopedKeys pins issue #1261's field case: an NXDN
// site programs TG 21 and TG 10 with the SAME key id (0) but different
// scrambler keys. Before talkgroup scoping only one key id 0 entry could
// exist, so only one talkgroup ever decoded. A talkgroup-scoped entry must
// resolve for its own talkgroup, win over an unscoped entry with the same
// key id, and never leak to another talkgroup, which falls back to the
// unscoped entry (or nothing).
func TestKeyResolverTalkgroupScopedKeys(t *testing.T) {
	systems := []config.SystemConfig{{Name: "SJPR NXDN", EncryptionKeys: []config.EncryptionKeyConfig{
		{KeyID: 0, Talkgroup: 21, Algorithm: "scrambler", Key: "12345"},
		{KeyID: 0, Talkgroup: 10, Algorithm: "scrambler", Key: "22122"},
		{KeyID: 0, Algorithm: "scrambler", Key: "100"},
		{KeyID: 5, Talkgroup: 30, Algorithm: "scrambler", Key: "777"},
	}}}
	res := buildKeyResolver(systems, slog.New(slog.NewTextHandler(testWriter{t}, nil)))
	if res == nil {
		t.Fatal("nil resolver")
	}
	key := func(group uint32, keyID uint16) (uint16, bool) {
		k, ok := res("SJPR NXDN", "scrambler", keyID, group)
		if !ok {
			return 0, false
		}
		return uint16(k[0])<<8 | uint16(k[1]), true
	}
	for _, tc := range []struct {
		group uint32
		keyID uint16
		want  uint16
		ok    bool
	}{
		{21, 0, 12345, true}, // talkgroup entry
		{10, 0, 22122, true}, // the other talkgroup's entry, same key id
		{99, 0, 100, true},   // unlisted talkgroup → unscoped entry
		{0, 0, 100, true},    // talkgroup unknown → unscoped entry
		{30, 5, 777, true},
		{31, 5, 0, false}, // a scoped key never leaks to another talkgroup
		{0, 5, 0, false},
	} {
		got, ok := key(tc.group, tc.keyID)
		if ok != tc.ok || got != tc.want {
			t.Errorf("tg %d key_id %d → %d %v, want %d %v", tc.group, tc.keyID, got, ok, tc.want, tc.ok)
		}
	}
	if _, ok := res("SJPR NXDN", "rc4", 0, 21); ok {
		t.Error("algorithm mismatch must not resolve")
	}

	kc := buildKeyConfigured(systems)
	if !kc("SJPR NXDN", 1, 5, 30) {
		t.Error("KeyConfigured: tg 30 key_id 5 is configured")
	}
	if kc("SJPR NXDN", 1, 5, 31) {
		t.Error("KeyConfigured: key_id 5 is configured only for tg 30, so a tg 31 call is not decryptable")
	}
}
