package main

import (
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// writeSilentIQWav writes a tiny 16-bit stereo IQ WAV the replay driver can
// mount as a virtual tuner.
func writeSilentIQWav(t *testing.T, path string, rate uint32) {
	t.Helper()
	const frames = 4096
	body := make([]byte, frames*4)
	h := make([]byte, 44)
	copy(h[0:4], "RIFF")
	binary.LittleEndian.PutUint32(h[4:8], uint32(36+len(body)))
	copy(h[8:12], "WAVE")
	copy(h[12:16], "fmt ")
	binary.LittleEndian.PutUint32(h[16:20], 16)
	binary.LittleEndian.PutUint16(h[20:22], 1)
	binary.LittleEndian.PutUint16(h[22:24], 2)
	binary.LittleEndian.PutUint32(h[24:28], rate)
	binary.LittleEndian.PutUint32(h[28:32], rate*4)
	binary.LittleEndian.PutUint16(h[32:34], 4)
	binary.LittleEndian.PutUint16(h[34:36], 16)
	copy(h[36:40], "data")
	binary.LittleEndian.PutUint32(h[40:44], uint32(len(body)))
	if err := os.WriteFile(path, append(h, body...), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestConventionalScannerDoesNotRequireRecordingsDir pins that a config with
// only scanner.conventional channels and a voice SDR builds the scanner even
// with recordings.dir unset. The scanner used to be gated on the FILE
// recorder, which only exists when recordings.dir is set, so the web UI
// reported the scanner as disabled with no log line explaining why — while
// trunked systems already fell back to a decode-only recorder.
func TestConventionalScannerDoesNotRequireRecordingsDir(t *testing.T) {
	dir := t.TempDir()
	wav := filepath.Join(dir, "voice.wav")
	const rate = 2_400_000
	writeSilentIQWav(t, wav, rate)

	cfg := config.Default()
	cfg.API.HTTPAddr = ""
	cfg.SDR.Devices = nil
	cfg.SDR.SampleRate = rate
	cfg.Recordings.Dir = ""
	cfg.Baseband.Replay = []config.BasebandReplayConfig{{File: wav, Serial: "conv-voice", Role: "voice"}}
	cfg.Scanner.Conventional = []config.ConvChannelConfig{{
		Label: "Test", FrequencyHz: 154_280_000, Mode: "fm",
	}}

	d, err := NewDaemon(cfg, "conv-no-recordings", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	if d.recorder != nil {
		t.Fatal("precondition: recordings.dir is unset, so no file recorder should exist")
	}
	if d.convScan == nil {
		t.Fatal("conventional scanner was not built without recordings.dir")
	}
}
