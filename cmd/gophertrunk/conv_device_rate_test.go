package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestConventionalScannerOnDeviceOwnSampleRate is #1260's AirspyHF case: a
// scanner SDR that cannot reach the shared sdr.sample_rate runs at its own
// sdr.devices[].sample_rate. Everything that SDR's scanner and calls touch
// must use that rate — the iqtap broker (spectrum/scope frame stamps), the
// scanner's LO offset search, and the composer source that sizes a call's
// decimator — while the other SDRs stay at the shared rate.
func TestConventionalScannerOnDeviceOwnSampleRate(t *testing.T) {
	dir := t.TempDir()
	var files []string
	for i := 0; i < 2; i++ {
		p := filepath.Join(dir, "voice"+string(rune('0'+i))+".cfile")
		if err := os.WriteFile(p, make([]byte, 4096), 0o644); err != nil {
			t.Fatalf("write mock IQ: %v", err)
		}
		files = append(files, p)
	}
	sdr.Register(&sdr.MockDriver{Files: files})

	const shared, hf = 2_400_000, 912_000
	cfg := config.Default()
	cfg.SDR.SampleRate = shared
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "voice"},
		{Serial: "mock-01", Role: "voice", SampleRate: hf},
	}
	cfg.Scanner.Conventional = []config.ConvChannelConfig{
		{Label: "AIR", FrequencyHz: 126_400_000, Mode: "am", Device: "mock-01"},
	}
	cfg.Recordings.Dir = t.TempDir()
	cfg.Storage.Path = ""
	cfg.Storage.CCCacheFile = ""
	cfg.API.HTTPAddr = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config rejected: %v", err)
	}

	d, err := NewDaemon(cfg, "test-conv-device-rate", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	t.Cleanup(d.Close)

	if got := d.pool.FindBySerial("mock-01").SampleRateHz(shared); got != hf {
		t.Fatalf("mock-01 opened at %d S/s, want its own %d", got, hf)
	}
	if got := d.iqBrokers["mock-01"].SampleRateHz(); got != hf {
		t.Errorf("mock-01 broker reports %d S/s, want %d", got, hf)
	}
	if got := d.iqBrokers["mock-00"].SampleRateHz(); got != shared {
		t.Errorf("mock-00 broker reports %d S/s, want the shared %d", got, shared)
	}
	wantOff, _ := convScannerLOOffsetHz(hf, cfg.Scanner.LOOffsetHz)
	sharedOff, _ := convScannerLOOffsetHz(shared, cfg.Scanner.LOOffsetHz)
	if wantOff == sharedOff {
		t.Fatalf("test needs rates whose LO offsets differ (both %v)", wantOff)
	}
	if got := d.scannerLOOffsets["mock-01"]; got != wantOff {
		t.Errorf("scanner LO offset %v Hz, want %v (the offset for %d S/s, not the shared rate's %v)", got, wantOff, hf, sharedOff)
	}

	devs := &poolDevices{pool: d.pool, rateHz: shared, scannerBrokers: d.scannerBrokers, scannerLOOffsets: d.scannerLOOffsets}
	if src := devs.FindBySerial("mock-01"); src == nil || src.SampleRateHz() != hf {
		t.Errorf("composer source for mock-01 = %v, want one reporting %d S/s", src, hf)
	}
	if src := devs.FindBySerial("mock-00"); src == nil || src.SampleRateHz() != shared {
		t.Errorf("composer source for mock-00 = %v, want one reporting %d S/s", src, shared)
	}
}
