package siglab

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRunStdinMatchesFile proves `-in -` (stdin) decodes a raw IQ stream identically
// to decoding the same bytes from a file — the offline half of issue #314 (pipe an
// IQ stream into the decoder). It swaps os.Stdin for a file handle over a
// synthesized P25 capture and checks the headline Result matches the file path.
func TestRunStdinMatchesFile(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	cfg, err := meta.Config(true)
	if err != nil {
		t.Fatalf("meta.Config: %v", err)
	}

	dir := t.TempDir()
	capPath := filepath.Join(dir, "p25.cfile")
	data := EncodeCapture(iq, FormatF32)
	if err := os.WriteFile(capPath, data, 0o644); err != nil {
		t.Fatalf("write capture: %v", err)
	}

	fromFile, err := Run(capPath, cfg)
	if err != nil {
		t.Fatalf("Run(path): %v", err)
	}
	if !fromFile.Locked {
		t.Fatal("expected synthesized clean P25 to lock from the file path")
	}

	// Feed the identical bytes through os.Stdin and decode with the "-" sentinel.
	f, err := os.Open(capPath)
	if err != nil {
		t.Fatalf("open capture: %v", err)
	}
	orig := os.Stdin
	os.Stdin = f
	defer func() {
		os.Stdin = orig
		f.Close()
	}()

	fromStdin, err := Run("-", cfg)
	if err != nil {
		t.Fatalf("Run(\"-\"): %v", err)
	}
	if !fromStdin.Locked {
		t.Error("expected the same capture to lock when piped through stdin")
	}
	if fromStdin.Source != "stdin" {
		t.Errorf("Source = %q, want \"stdin\"", fromStdin.Source)
	}
	if fromStdin.Locked != fromFile.Locked {
		t.Errorf("Locked mismatch: file=%v stdin=%v", fromFile.Locked, fromStdin.Locked)
	}
	if fromStdin.TotalSamples != fromFile.TotalSamples {
		t.Errorf("TotalSamples mismatch: file=%d stdin=%d", fromFile.TotalSamples, fromStdin.TotalSamples)
	}
	if fromStdin.Symbols != fromFile.Symbols {
		t.Errorf("Symbols mismatch: file=%d stdin=%d", fromFile.Symbols, fromStdin.Symbols)
	}
}

// TestOpenCaptureStdin checks the "-" sentinel resolves to standard input with a
// no-op closer (so decoding does not close the process's real stdin), and that a
// real path still opens normally while a missing one errors.
func TestOpenCaptureStdin(t *testing.T) {
	rc, source, err := openCapture("-")
	if err != nil {
		t.Fatalf("openCapture(\"-\"): %v", err)
	}
	if source != "stdin" {
		t.Errorf("source = %q, want \"stdin\"", source)
	}
	if err := rc.Close(); err != nil {
		t.Errorf("stdin Close should be a no-op, got %v", err)
	}
	// The no-op close must leave the real stdin usable.
	if os.Stdin == nil {
		t.Fatal("os.Stdin was closed by openCapture's Closer")
	}

	dir := t.TempDir()
	p := filepath.Join(dir, "cap.bin")
	if err := os.WriteFile(p, []byte{1, 2, 3, 4}, 0o644); err != nil {
		t.Fatal(err)
	}
	rc2, source2, err := openCapture(p)
	if err != nil {
		t.Fatalf("openCapture(path): %v", err)
	}
	rc2.Close()
	if source2 != p {
		t.Errorf("source = %q, want %q", source2, p)
	}
	if _, _, err := openCapture(filepath.Join(dir, "missing.bin")); err == nil {
		t.Error("expected an error opening a missing file")
	}
}

// TestRunAutoTuneStdinRejected confirms auto-tune over stdin fails with a clear
// error rather than a confusing os.Open("-") failure — stdin cannot seek.
func TestRunAutoTuneStdinRejected(t *testing.T) {
	cfg := Config{Protocol: trunking.ProtocolP25, SampleRateHz: 2_400_000, Format: FormatF32, AutoTune: true}
	if _, err := RunAutoTuneMulti("-", cfg, 0); err == nil {
		t.Fatal("expected auto-tune over stdin to be rejected")
	}
}
