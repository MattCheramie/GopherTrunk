package siglab

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// writeF32 writes interleaved little-endian float32 IQ to path (GNU Radio
// cfile format) — the on-disk shape FormatF32 reads back.
func writeF32(t *testing.T, path string, iq []complex64) {
	t.Helper()
	buf := make([]byte, len(iq)*8)
	for i, s := range iq {
		binary.LittleEndian.PutUint32(buf[8*i:], math.Float32bits(real(s)))
		binary.LittleEndian.PutUint32(buf[8*i+4:], math.Float32bits(imag(s)))
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestEngineThroughputAndAnalysis drives a synthesized C4FM stream through
// the engine and asserts the protocol-agnostic plumbing: the receiver emits
// symbols at ~4800 baud, the effective-baud self-diagnostic lands near the
// P25 nominal, and the analyzer produces a 4-level symbol histogram. It does
// not require a full control-channel lock — that is covered by the gen→Run
// round-trip once the fixture library lands.
func TestEngineThroughputAndAnalysis(t *testing.T) {
	const (
		sampleRate = 48_000.0
		deviation  = 1800.0
	)
	// A long stream cycling through all four symbols so the Mueller-Müller
	// clock has plenty of transitions to lock onto.
	dibits := make([]uint8, 24_000)
	for i := range dibits {
		dibits[i] = uint8(i % 4)
	}
	iq := demod.ModulateP25C4FM(dibits, sampleRate, deviation)

	dir := t.TempDir()
	path := filepath.Join(dir, "synth.cfile")
	writeF32(t, path, iq)

	res, err := Run(path, Config{
		Protocol:      trunking.ProtocolP25,
		SystemName:    "test",
		SampleRateHz:  sampleRate,
		Format:        FormatF32,
		CollectIQDiag: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if res.Protocol != "p25" {
		t.Errorf("Protocol = %q, want p25", res.Protocol)
	}
	if res.Symbols == 0 {
		t.Fatal("engine recovered no symbols")
	}
	if res.ExpectedBaud != 4800 {
		t.Errorf("ExpectedBaud = %v, want 4800", res.ExpectedBaud)
	}
	// Effective baud should sit within 10% of the 4800 nominal.
	if dev := math.Abs(res.BaudDeviationPct); dev > 10 {
		t.Errorf("baud deviation %.1f%% > 10%% (effective=%.1f)", dev, res.EffectiveBaud)
	}
	if res.Signal == nil {
		t.Fatal("CollectIQDiag set but Signal is nil")
	}
	if res.Signal.SymbolCardinality != 4 {
		t.Errorf("SymbolCardinality = %d, want 4", res.Signal.SymbolCardinality)
	}
	if got := len(res.Signal.SymbolHistogram); got != 4 {
		t.Errorf("SymbolHistogram len = %d, want 4", got)
	}
	if !res.Signal.IQObserved {
		t.Error("expected IQ imbalance to be observed")
	}
	// The dibit-domain P25 detail should have been built from the buffered
	// symbols.
	if res.P25P1 == nil {
		t.Fatal("P25P1 detail not built for a P25 run with CollectIQDiag")
	}
	if res.P25P1.DibitsBuffered == 0 {
		t.Error("P25P1.DibitsBuffered = 0")
	}
}

// TestEngineRejectsBadConfig covers the guard rails.
func TestEngineRejectsBadConfig(t *testing.T) {
	if _, err := Run("/nonexistent", Config{Protocol: trunking.ProtocolP25}); err == nil {
		t.Error("expected error for zero sample rate")
	}
	if _, err := Run("/nonexistent", Config{Protocol: trunking.ProtocolUnknown, SampleRateHz: 48000}); err == nil {
		t.Error("expected error for unregistered protocol")
	}
}
