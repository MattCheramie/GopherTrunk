package main

import (
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/voice/composer"
)

// TestConventionalChannelKeysResolveByFrequency: scan-list channel keys
// (scanner.conventional and a wideband dongle's conventional list) reach
// the composer's resolver under ConventionalKeySystem(hz) as 2-byte
// big-endian scrambler keys, and the recorder's KeyConfigured predicate
// counts them for a call whose system ("scanner") has no keys of its own.
func TestConventionalChannelKeysResolveByFrequency(t *testing.T) {
	cfg := config.Config{}
	cfg.Scanner.Conventional = []config.ConvChannelConfig{
		{FrequencyHz: 447_100_000, Mode: "nxdn", EncryptionKeys: []config.EncryptionKeyConfig{
			{KeyID: 3, Algorithm: "scrambler", Key: "32767"},
			{KeyID: 1, Algorithm: "scrambler", Key: "12345"},
		}},
		{FrequencyHz: 155_000_000, Mode: "fm"},
	}
	cfg.SDR.Devices = []config.DeviceConfig{{Serial: "WB", Conventional: []config.ConvChannelConfig{
		{FrequencyHz: 451_000_000, Mode: "dmr", EncryptionKeys: []config.EncryptionKeyConfig{{KeyID: 7, Algorithm: "rc4", Key: "0123456789"}}},
	}}}
	systems := conventionalKeySystems(cfg)
	if len(systems) != 2 {
		t.Fatalf("%d pseudo systems, want 2: %+v", len(systems), systems)
	}
	log := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	res := buildKeyResolver(systems, log)
	if res == nil {
		t.Fatal("nil resolver")
	}
	k, ok := res(composer.ConventionalKeySystem(447_100_000), "scrambler", 3, 0)
	if !ok || len(k) != 2 || uint16(k[0])<<8|uint16(k[1]) != 32767 {
		t.Fatalf("key 3 → %x %v, want 7fff", k, ok)
	}
	if k, ok := res(composer.ConventionalKeySystem(447_100_000), "scrambler", 1, 0); !ok || uint16(k[0])<<8|uint16(k[1]) != 12345 {
		t.Fatalf("key 1 → %x %v", k, ok)
	}
	if _, ok := res(composer.ConventionalKeySystem(447_100_000), "rc4", 3, 0); ok {
		t.Fatal("algorithm mismatch must not resolve")
	}
	if _, ok := res("scanner", "scrambler", 3, 0); ok {
		t.Fatal("the bare scanner system holds no keys")
	}
	if k, ok := res(composer.ConventionalKeySystem(451_000_000), "rc4", 7, 0); !ok || len(k) != 5 {
		t.Fatalf("wideband channel key → %x %v", k, ok)
	}
	kc := buildKeyConfigured(systems)
	if kc == nil || !kc("scanner", 1, 3, 0) || !kc("scanner", 1, 0, 0) || kc("scanner", 1, 9, 0) {
		t.Fatal("KeyConfigured must cover a scanner call by the channels' key ids")
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }
