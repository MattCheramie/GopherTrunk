//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/mpt1327"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesMPT1327 is the per-protocol sibling of the
// earlier integration-cc tests. First protocol to exercise
// audio-band FFSK modulation. Boots the daemon with a mock SDR
// replaying synthesized MPT 1327 IQ (CCIR FFSK at 1200 baud
// inside an FM audio channel — mark = 1200 Hz, space = 1800 Hz)
// carrying BCH(63, 38)-encoded ALH (Aloha) codewords, and
// asserts the production newMPT1327Pipeline + mpt1327_bch_mode +
// supervisor + API + metrics chain recovers the lock.
//
// Exercises the FFSK modulator primitive shipped alongside this
// PR — first integration test to cover the audio-FSK family.
func TestDaemonCCDecodesMPT1327(t *testing.T) {
	const (
		controlFreqHz = 169_212_500
		// MPT 1327 receiver expects ≥ 2 × max tone (1800 Hz) =
		// 3600 Hz. 48 kHz is a comfortable margin and matches the
		// audio-rate floor.
		sampleRate            = 48_000.0
		markHz                = 1200.0
		spaceHz               = 1800.0
		symbolRate            = 1200.0
		prefix          uint8 = 0x5
		codewordRepeats       = 100
	)

	bits := buildMPT1327AlohaStream(codewordRepeats, prefix)
	iq := demod.ModulateFFSK(bits, sampleRate, symbolRate, markHz, spaceHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "mpt1327-cc.cfile")
	if err := writeIQToU8File(iqPath, iq); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = uint32(sampleRate)
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{
			Name:            "MPTSite",
			Protocol:        "mpt1327",
			ControlChannels: []uint32{controlFreqHz},
			MPT1327BCHMode:  "on",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-mpt1327", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if d.ccDecoder == nil {
		t.Fatalf("ccDecoder is nil; daemon should have constructed one")
	}

	sub := d.Bus().Subscribe()
	defer sub.Close()

	ctx, cancel := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- d.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-runErrCh:
		case <-time.After(3 * time.Second):
		}
	})

	base := "http://" + cfg.API.HTTPAddr
	waitReachable(t, base+"/api/v1/health", 3*time.Second)

	deadline := time.After(5 * time.Second)
	var locked bool
WaitLoop:
	for !locked {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(mpt1327.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want mpt1327.LockState", ev.Payload)
				continue
			}
			if ls.Prefix != prefix {
				t.Errorf("LockState.Prefix = %d, want %d", ls.Prefix, prefix)
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d", ls.FrequencyHz, controlFreqHz)
			}
			locked = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within 5s")
		}
	}

	waitForScannerLock(t, base, "MPTSite", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildMPT1327AlohaStream assembles an MPT 1327 bit stream for
// the FFSK modulator + receiver chain:
//
//   - 200-bit warmup alternating 0/1 so the Mueller-Müller clock
//     recovery sees a transition every bit
//   - `repeats` × (64-bit BCH(63, 38)-encoded ALH codeword + 16
//     idle bits)
//   - 100-bit trailer for clean flush
//
// Each codeword is an ALH (Aloha) carrying the requested Prefix.
// The 48-bit info field runs through framing.BCHEncodeMPT1327 to
// produce a 64-bit on-wire codeword that the production receiver
// recognises under `mpt1327_bch_mode: on`.
func buildMPT1327AlohaStream(repeats int, prefix uint8) []byte {
	aloha := mpt1327.Codeword{
		Type:     mpt1327.TypeAddress,
		Prefix:   prefix,
		Function: uint32(mpt1327.KindAloha) << 13,
	}
	wire48 := mpt1327.CodewordBits48(aloha)
	var info48 uint64
	for i := 0; i < 48; i++ {
		if wire48[i]&1 != 0 {
			info48 |= uint64(1) << uint(i)
		}
	}
	cw := framing.BCHEncodeMPT1327(info48)
	codeword := make([]byte, 64)
	for i := 0; i < 64; i++ {
		codeword[i] = byte((cw >> uint(i)) & 1)
	}

	out := make([]byte, 0, 200+repeats*(len(codeword)+16)+100)
	for i := 0; i < 200; i++ {
		out = append(out, byte(i&1))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, codeword...)
		for i := 0; i < 16; i++ {
			out = append(out, byte(i&1))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, byte(i&1))
	}
	return out
}
