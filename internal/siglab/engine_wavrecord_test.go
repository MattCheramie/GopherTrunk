package siglab

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/baseband"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestRecordDDCThenReplayWAV proves the "record the DDC output → replay it"
// loop is faithful: decoding a capture with RecordDDCPath set writes the
// post-DDC narrowband stream to a two-channel 16-bit WAV, and replaying that
// WAV with FormatWAV (rate taken from the header) reproduces the same lock and
// symbol count. This is the offline half of the IQ-record tooling — turning a
// fat capture into a small, shareable, self-describing fixture.
func TestRecordDDCThenReplayWAV(t *testing.T) {
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
	if err := os.WriteFile(capPath, EncodeCapture(iq, FormatF32), 0o644); err != nil {
		t.Fatalf("write capture: %v", err)
	}
	wavPath := filepath.Join(dir, "nb.wav")

	recCfg := cfg
	recCfg.RecordDDCPath = wavPath
	rec, err := Run(capPath, recCfg)
	if err != nil {
		t.Fatalf("Run(record): %v", err)
	}
	if !rec.Locked {
		t.Fatalf("synthesized clean P25 should lock while recording DDC output")
	}

	info, err := baseband.ReadIQWavInfo(wavPath)
	if err != nil {
		t.Fatalf("ReadIQWavInfo(%s): %v", wavPath, err)
	}
	if info.SampleRate == 0 || info.Samples == 0 {
		t.Fatalf("recorded WAV has rate=%d samples=%d, want both > 0", info.SampleRate, info.Samples)
	}

	// Replay the recorded narrowband WAV. FormatWAV takes the rate from the
	// header, so no -sample-rate is needed and the decode must match.
	replayCfg := cfg
	replayCfg.Format = FormatWAV
	rep, err := Run(wavPath, replayCfg)
	if err != nil {
		t.Fatalf("Run(replay wav): %v", err)
	}
	if !rep.Locked {
		t.Fatalf("replayed DDC WAV should lock the same as the source capture")
	}
	if rep.TotalSamples != int64(info.Samples) {
		t.Errorf("replay read %d samples, WAV holds %d", rep.TotalSamples, info.Samples)
	}
	if rep.Symbols != rec.Symbols {
		t.Errorf("symbol count changed across record→replay: record=%d replay=%d", rec.Symbols, rep.Symbols)
	}
}

// TestFormatWAVRejectsAutoTune confirms a baseband WAV (already channelized)
// cannot be auto-tuned — the run must fail with a clear error rather than
// silently mis-estimating a wideband carrier.
func TestFormatWAVRejectsAutoTune(t *testing.T) {
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
	if err := os.WriteFile(capPath, EncodeCapture(iq, FormatF32), 0o644); err != nil {
		t.Fatalf("write capture: %v", err)
	}
	wavPath := filepath.Join(dir, "nb.wav")
	recCfg := cfg
	recCfg.RecordDDCPath = wavPath
	if _, err := Run(capPath, recCfg); err != nil {
		t.Fatalf("Run(record): %v", err)
	}

	atCfg := cfg
	atCfg.Format = FormatWAV
	atCfg.AutoTune = true
	if _, err := Run(wavPath, atCfg); err == nil {
		t.Fatal("FormatWAV + AutoTune should be rejected")
	}
}

// TestParseSampleFormatWAV covers the flag spellings and the decoder shape.
func TestParseSampleFormatWAV(t *testing.T) {
	for _, s := range []string{"wav", "sw16", "s16", "WAV"} {
		f, err := ParseSampleFormat(s)
		if err != nil || f != FormatWAV {
			t.Fatalf("ParseSampleFormat(%q) = %v, %v; want FormatWAV", s, f, err)
		}
	}
	if _, n := FormatWAV.Decoder(); n != 4 {
		t.Fatalf("FormatWAV bytes-per-sample = %d, want 4", n)
	}
	if FormatWAV.String() != "wav" {
		t.Fatalf("FormatWAV.String() = %q, want wav", FormatWAV.String())
	}
}
