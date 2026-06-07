package siglab

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRunReaderMatchesRun proves that decoding a capture from an in-memory
// io.Reader produces the same headline Result as decoding the same bytes from
// a file — i.e. the in-memory path shares the engine's read loop faithfully.
func TestRunReaderMatchesRun(t *testing.T) {
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
	fromReader, err := RunReader(bytes.NewReader(data), "p25.cfile", cfg)
	if err != nil {
		t.Fatalf("RunReader: %v", err)
	}

	if fromFile.Locked != fromReader.Locked {
		t.Errorf("Locked mismatch: file=%v reader=%v", fromFile.Locked, fromReader.Locked)
	}
	if fromFile.TotalSamples != fromReader.TotalSamples {
		t.Errorf("TotalSamples mismatch: file=%d reader=%d", fromFile.TotalSamples, fromReader.TotalSamples)
	}
	if fromFile.Symbols != fromReader.Symbols {
		t.Errorf("Symbols mismatch: file=%d reader=%d", fromFile.Symbols, fromReader.Symbols)
	}
	if fromFile.Source != fromReader.Source {
		t.Errorf("Source mismatch: file=%q reader=%q", fromFile.Source, fromReader.Source)
	}
	if !fromReader.Locked {
		t.Errorf("expected synthesized clean P25 to lock via RunReader")
	}
}

// TestRunReaderCaptureIQ confirms the decimated-IQ + soft-sample taps are
// populated, bounded by the cap, and report a sane decimated rate.
func TestRunReaderCaptureIQ(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	cfg, err := meta.Config(true)
	if err != nil {
		t.Fatalf("meta.Config: %v", err)
	}
	cfg.CaptureIQ = true
	cfg.CaptureIQMaxPoints = 5000

	data := EncodeCapture(iq, FormatF32)
	res, err := RunReader(bytes.NewReader(data), "p25.cfile", cfg)
	if err != nil {
		t.Fatalf("RunReader: %v", err)
	}
	if res.IQTaps == nil {
		t.Fatal("expected IQTaps when CaptureIQ is set")
	}
	if len(res.IQTaps.DecimatedIQ) == 0 {
		t.Error("expected decimated IQ points")
	}
	if len(res.IQTaps.DecimatedIQ) > cfg.CaptureIQMaxPoints {
		t.Errorf("decimated IQ exceeds cap: %d > %d", len(res.IQTaps.DecimatedIQ), cfg.CaptureIQMaxPoints)
	}
	if len(res.IQTaps.SoftSamples) > cfg.CaptureIQMaxPoints {
		t.Errorf("soft samples exceed cap: %d > %d", len(res.IQTaps.SoftSamples), cfg.CaptureIQMaxPoints)
	}
	if res.IQTaps.Stride < 1 {
		t.Errorf("stride must be >= 1, got %d", res.IQTaps.Stride)
	}
	if res.IQTaps.DecimatedRateHz <= 0 {
		t.Errorf("decimated rate must be > 0, got %v", res.IQTaps.DecimatedRateHz)
	}
}

// TestRunReaderCaptureSymbolSeries confirms the aligned symbol series
// (dibits + soft) that backs the offline Symbol-scope viz is populated,
// aligned, bounded by the cap, and reports cardinality 4 for P25 C4FM.
func TestRunReaderCaptureSymbolSeries(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	cfg, err := meta.Config(true)
	if err != nil {
		t.Fatalf("meta.Config: %v", err)
	}
	cfg.CollectIQDiag = true
	cfg.CaptureIQ = true
	cfg.CaptureIQMaxPoints = 4000

	data := EncodeCapture(iq, FormatF32)
	res, err := RunReader(bytes.NewReader(data), "p25.cfile", cfg)
	if err != nil {
		t.Fatalf("RunReader: %v", err)
	}
	if res.IQTaps == nil {
		t.Fatal("expected IQTaps when CaptureIQ is set")
	}
	if len(res.IQTaps.SymbolDibits) == 0 {
		t.Fatal("expected a symbol series when CollectIQDiag + CaptureIQ are set on P25")
	}
	if res.IQTaps.SymbolCardinality != 4 {
		t.Errorf("SymbolCardinality = %d, want 4 for P25 C4FM", res.IQTaps.SymbolCardinality)
	}
	if len(res.IQTaps.SymbolDibits) > cfg.CaptureIQMaxPoints {
		t.Errorf("symbol series exceeds cap: %d > %d", len(res.IQTaps.SymbolDibits), cfg.CaptureIQMaxPoints)
	}
	if n := len(res.IQTaps.SymbolSoft); n != 0 && n != len(res.IQTaps.SymbolDibits) {
		t.Errorf("soft track len %d not aligned with dibits len %d", n, len(res.IQTaps.SymbolDibits))
	}
	for i, d := range res.IQTaps.SymbolDibits {
		if d > 3 {
			t.Fatalf("dibit %d at %d out of range", d, i)
		}
	}
}

// TestRunReaderAutoTuneNonSeekable confirms auto-tune over a non-seekable
// reader fails with a clear error rather than silently mis-decoding.
func TestRunReaderAutoTuneNonSeekable(t *testing.T) {
	cfg := Config{
		Protocol:     trunking.ProtocolP25,
		SampleRateHz: 48000,
		Format:       FormatF32,
		AutoTune:     true,
	}
	// A bare io.Reader (not a ReadSeeker).
	r := readerOnly{bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})}
	if _, err := RunReader(r, "x", cfg); err == nil {
		t.Fatal("expected auto-tune over non-seekable reader to error")
	}
}

// readerOnly hides the ReadSeeker methods of its embedded reader.
type readerOnly struct{ r *bytes.Reader }

func (ro readerOnly) Read(p []byte) (int, error) { return ro.r.Read(p) }
