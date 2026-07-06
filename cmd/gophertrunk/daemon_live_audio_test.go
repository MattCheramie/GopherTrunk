//go:build integration

package main

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonWiresLiveAudioWithoutRecordings pins the fix that live browser
// audio no longer requires a recordings directory. A DMR user who never
// configured recordings still needs decoded voice to reach the web/gRPC audio
// stream — before the fix the whole composer + decoded-PCM tap was gated on the
// recorder, which itself required recordings.dir, so calls appeared in the UI
// but no audio ever played.
//
// With no recordings.dir the daemon must build a decode-only voice decoder,
// wire the composer through it, and keep the audio publisher live; with a
// recordings.dir the file recorder doubles as the decoder (unchanged behaviour).
func TestDaemonWiresLiveAudioWithoutRecordings(t *testing.T) {
	const (
		controlFreqHz = 460_500_000
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.20
		deviationHz   = 1944.0
		colorCode     = 0x7
		groupID       = 0x123
		sourceID      = 0x456789
		burstRepeats  = 4
	)

	dibits := buildDMRTier2VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)
	iqPath := filepath.Join(t.TempDir(), "dmr-tier2-cc.cfile")
	if err := writeIQToU8File(iqPath, iq); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	newDaemon := func(t *testing.T, recordingsDir string) *Daemon {
		t.Helper()
		cfg := config.Default()
		cfg.SDR.SampleRate = sampleRateHz
		cfg.SDR.Devices = []config.DeviceConfig{{Serial: "mock-00", Role: "control"}}
		cfg.Trunking.Systems = []config.SystemConfig{
			{Name: "DMRTier2Repeater", Protocol: "dmr-tier2", ControlChannels: []uint32{controlFreqHz}},
		}
		cfg.API.HTTPAddr = freeAddr(t)
		cfg.Recordings.Dir = recordingsDir

		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		d, err := NewDaemon(cfg, "test-live-audio", logger)
		if err != nil {
			t.Fatalf("NewDaemon: %v", err)
		}
		t.Cleanup(d.Close)
		return d
	}

	t.Run("no recordings dir builds a decode-only decoder", func(t *testing.T) {
		d := newDaemon(t, "")
		if d.recorder != nil {
			t.Error("recorder should be nil without a recordings dir (nothing to persist)")
		}
		if d.voiceDecoder == nil {
			t.Fatal("voiceDecoder is nil: digital voice would never be decoded, so live audio is silent")
		}
		if d.composer == nil {
			t.Error("composer is nil: no voice demod would run, so no PCM reaches the stream")
		}
		if d.audioPub == nil {
			t.Error("audioPub is nil: the /api/v1/audio/stream endpoint has no source")
		}
	})

	t.Run("recordings dir reuses the file recorder as the decoder", func(t *testing.T) {
		d := newDaemon(t, t.TempDir())
		if d.recorder == nil {
			t.Fatal("recorder should be built when a recordings dir is set")
		}
		if d.voiceDecoder != d.recorder {
			t.Error("voiceDecoder must be the file recorder when recording (no separate decoder)")
		}
		if d.composer == nil {
			t.Error("composer is nil: recording + live audio both need it")
		}
	})
}
