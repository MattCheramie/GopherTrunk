package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/hunt"
)

// writeSmokeBundle packs a small bundle with a discovered system and crypto
// frames, returning its path.
func writeSmokeBundle(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "smoke.gtb.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	w, err := gtbundle.NewWriter(f, gtbundle.WriterOptions{Name: "smoke", CaptureIntent: gtbundle.IntentCrypto})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	sys := &hunt.DiscoveredSystem{
		Name:     "Smoke",
		Protocol: "p25",
		Talkgroups: []hunt.DiscoveredTalkgroup{
			{Dec: 101, Hex: "65", Count: 3},
			{Dec: 202, Hex: "ca", Count: 1},
		},
	}
	if _, err := w.AddJSON(gtbundle.RoleMappingSystem, "system.json", sys, "hunt"); err != nil {
		t.Fatalf("add system: %v", err)
	}
	// TG 101 under AES-256 (0x84) encrypted; TG 202 under CLEAR (0x80).
	frames := `{"tg":101,"algid":132,"iv":"aa","ct":"bb"}` + "\n" +
		`{"tg":202,"algid":128,"iv":"cc","ct":"dd"}` + "\n"
	if _, err := w.AddBytes(gtbundle.RoleCryptolabFrames, "frames.jsonl", []byte(frames), "cryptolab"); err != nil {
		t.Fatalf("add frames: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return path
}

// TestEnrichFromCryptoVerdict verifies the commit-time enrichment marks only the
// talkgroup carrying a non-clear algorithm.
func TestEnrichFromCryptoVerdict(t *testing.T) {
	path := writeSmokeBundle(t)
	r, err := gtbundle.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sys, err := r.MappingSystem()
	if err != nil {
		t.Fatalf("MappingSystem: %v", err)
	}
	n := enrichFromCryptoVerdict(r, sys)
	if n != 1 {
		t.Fatalf("enriched %d talkgroups, want 1", n)
	}
	byDec := map[uint32]bool{}
	for _, tg := range sys.Talkgroups {
		byDec[tg.Dec] = tg.Encrypted
	}
	if !byDec[101] {
		t.Errorf("TG 101 (AES-256) not marked encrypted")
	}
	if byDec[202] {
		t.Errorf("TG 202 (CLEAR) wrongly marked encrypted")
	}
}

// TestEnrichNoFramesIsNoOp confirms a bundle without crypto frames enriches
// nothing rather than erroring.
func TestEnrichNoFramesIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nf.gtb.tar.gz")
	f, _ := os.Create(path)
	w, err := gtbundle.NewWriter(f, gtbundle.WriterOptions{Name: "nf"})
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	sys := &hunt.DiscoveredSystem{Name: "nf", Talkgroups: []hunt.DiscoveredTalkgroup{{Dec: 1}}}
	if _, err := w.AddJSON(gtbundle.RoleMappingSystem, "system.json", sys, "hunt"); err != nil {
		t.Fatalf("add: %v", err)
	}
	w.Close()
	f.Close()

	r, err := gtbundle.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := r.MappingSystem()
	if n := enrichFromCryptoVerdict(r, got); n != 0 {
		t.Errorf("enriched %d, want 0 for a frameless bundle", n)
	}
}

// TestRoleForRelPath spot-checks the from-dir role inference.
func TestRoleForRelPath(t *testing.T) {
	cases := map[string]gtbundle.Role{
		"capture/cc.cfile":          gtbundle.RoleCaptureIQ,
		"capture/cc.slice.wav":      gtbundle.RoleCaptureSlice,
		"capture/cc.meta.yaml":      gtbundle.RoleCaptureMeta,
		"logs/events.jsonl":         gtbundle.RoleLogEvents,
		"logs/messages.log":         gtbundle.RoleLogMessages,
		"mapping/system.json":       gtbundle.RoleMappingSystem,
		"mapping/survey.jsonl":      gtbundle.RoleMappingSurvey,
		"mapping/system.bundle.csv": gtbundle.RoleMappingExport,
		"siglab/result.yaml":        gtbundle.RoleSiglabResult,
		"cryptolab/frames.jsonl":    gtbundle.RoleCryptolabFrames,
		"cryptolab/assess.yaml":     gtbundle.RoleCryptolabResult,
		"notes.md":                  gtbundle.RoleNotes,
	}
	for rel, want := range cases {
		got, ok := roleForRelPath(rel)
		if !ok || got != want {
			t.Errorf("roleForRelPath(%q) = %q,%v; want %q", rel, got, ok, want)
		}
	}
	if _, ok := roleForRelPath("random/thing.txt"); ok {
		t.Errorf("roleForRelPath accepted an unrecognized path")
	}
}
