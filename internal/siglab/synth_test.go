package siglab

import (
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestSynthDecodeRoundTrip is the closed synthesize→decode→grade loop: for
// every registered fixture, generate a clean capture + metadata, run it back
// through the engine, and assert the metadata's acceptance criteria pass.
// This is the strongest end-to-end check that the engine drives each
// protocol's production pipeline correctly.
func TestSynthDecodeRoundTrip(t *testing.T) {
	for _, proto := range Fixtures() {
		proto := proto
		t.Run(proto.String(), func(t *testing.T) {
			iq, meta, err := Synthesize(SynthOptions{Protocol: proto, Format: FormatF32})
			if err != nil {
				t.Fatalf("Synthesize(%s): %v", proto, err)
			}
			dir := t.TempDir()
			capPath := filepath.Join(dir, proto.String()+".cfile")
			if err := WriteCapture(capPath, iq, FormatF32); err != nil {
				t.Fatalf("WriteCapture: %v", err)
			}

			cfg, err := meta.Config(true)
			if err != nil {
				t.Fatalf("meta.Config: %v", err)
			}
			res, err := Run(capPath, cfg)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if res.Verdict == nil {
				t.Fatal("expected a verdict (acceptance was set)")
			}
			if !res.Verdict.Pass {
				t.Errorf("synth round-trip failed acceptance for %s: %v\n  locked=%v lock=%+v",
					proto, res.Verdict.Failures, res.Locked, res.Lock)
			}
		})
	}
}

// TestSynthUnsupportedProtocol confirms a protocol without a fixture reports
// a helpful error rather than panicking.
func TestSynthUnsupportedProtocol(t *testing.T) {
	if _, _, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolLTR}); HasFixture(trunking.ProtocolLTR) == (err != nil) {
		// If LTR has no fixture, err must be non-nil; if it has one, err nil.
		t.Errorf("Synthesize/HasFixture disagree for LTR: hasFixture=%v err=%v", HasFixture(trunking.ProtocolLTR), err)
	}
}

// TestMetadataRoundTrip writes a synthesized fixture's metadata to JSON and
// reloads it, confirming the loader and Config builder agree.
func TestMetadataRoundTrip(t *testing.T) {
	_, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	dir := t.TempDir()
	mp := filepath.Join(dir, "p25.metadata.json")
	if err := WriteMetadata(mp, meta); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}
	got, err := LoadMetadata(mp)
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}
	if got.Protocol != "p25" || got.SampleRateHz != 48000 {
		t.Errorf("metadata mismatch: %+v", got)
	}
	if got.Expected.Lock == nil || !*got.Expected.Lock {
		t.Errorf("expected lock acceptance preserved, got %+v", got.Expected)
	}
}

// TestDiscoverMetadata confirms sidecar discovery by stem.
func TestDiscoverMetadata(t *testing.T) {
	dir := t.TempDir()
	capPath := filepath.Join(dir, "cap.cfile")
	metaPath := filepath.Join(dir, "cap.metadata.json")
	if err := WriteMetadata(metaPath, &Metadata{Protocol: "p25", SampleRateHz: 48000}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}
	if got := DiscoverMetadata(capPath); got != metaPath {
		t.Errorf("DiscoverMetadata = %q, want %q", got, metaPath)
	}
}
