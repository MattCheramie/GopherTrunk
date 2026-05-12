//go:build integration

package main

import (
	"context"
	"encoding/binary"
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
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// TestDaemonCCDecodesNXDN is the per-protocol sibling of
// TestDaemonCCDecodesP25Phase1: boot the daemon with a mock SDR
// replaying a fully-synthesized NXDN-TS-1-A §4.6 RCCH outbound
// frame (FSW + LICH + 150-dibit CAC carrying a SITE_INFO RCCH
// message through the §4.5.1.1 spec-correct FEC chain), assert
// the production newNXDNPipeline + `nxdn_viterbi_mode: spec`
// recovers the lock and surfaces it through the bus + API +
// metrics.
//
// The synthesized IQ is the same 9600-baud / 4-FSK / α = 0.20 /
// 1800 Hz peak deviation modulation used by P25 Phase 1 — NXDN
// shares the modulation params, just differs in the framing and
// channel-coding layers above the demod. The newly-shipped C4FM
// modulator (PR #148) carries straight over; the receiver-side
// slicer calibration via Options.DeviationHz is the only other
// piece the test needs to round-trip cleanly.
func TestDaemonCCDecodesNXDN(t *testing.T) {
	const (
		controlFreqHz = 851_062_500
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.20
		deviationHz   = 1800.0
		frameRepeats  = 20
		siteID        = 0xBEEF
		systemID      = 0x0042
	)

	dibits := buildNXDNSpecEncodedDibits(frameRepeats, siteID, systemID)
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "nxdn-cc.cfile")
	if err := writeIQToU8File(iqPath, iq); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = sampleRateHz
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{
			Name:            "NXDNSite",
			Protocol:        "nxdn",
			ControlChannels: []uint32{controlFreqHz},
			NXDNViterbiMode: "spec",
		},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-nxdn", logger)
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
			ls, ok := ev.Payload.(nxdn.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want nxdn.LockState", ev.Payload)
				continue
			}
			if ls.SiteID != siteID {
				t.Errorf("LockState.SiteID = %#x, want %#x", ls.SiteID, siteID)
			}
			if ls.SystemID != systemID {
				t.Errorf("LockState.SystemID = %#x, want %#x", ls.SystemID, systemID)
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

	waitForScannerLock(t, base, "NXDNSite", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
}

// buildNXDNSpecEncodedDibits assembles a long NXDN dibit stream
// for the C4FM modulator + receiver chain:
//
//   - 300-dibit warmup cycling 0..3 so the Mueller-Müller clock
//     recovery sees every symbol level
//   - `repeats` × (FSWDibitsOutbound + LICH wire dibits + 150
//     spec-encoded CAC dibits + 50 idle dibits)
//   - 100-dibit trailer for clean flush
//
// Each frame layout matches the production process.go
// ViterbiSpec post-sync window (8 LICH + 150 CAC = 158 dibits)
// per §4.6 RCCH outbound. The CAC carries a SITE_INFO RCCH
// message via the §4.5.1.1 spec FEC chain (EncodeCACChannel):
// 155 info bits (8 SR + 144 L3 + 3 Null) → CRC-16 → 4 tail →
// K=5 R=1/2 → puncture(50/350) → 25×12 interleave → 300 channel
// bits = 150 dibits.
func buildNXDNSpecEncodedDibits(repeats int, siteID, systemID uint16) []uint8 {
	// LICH for RCCH outbound control channel.
	lichInfo := nxdn.AssembleLICH(nxdn.LICH{RFCh: nxdn.RFChControl})
	lichWire := nxdn.EncodeLICHWire(lichInfo)
	lichDibits := framing.BitsToDibits(lichWire)

	// SITE_INFO L3 prefix: 1 byte RCCH type + 8 bytes payload.
	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA)
	binary.BigEndian.PutUint16(payload[2:4], siteID)
	binary.BigEndian.PutUint16(payload[4:6], systemID)
	l3 := make([]byte, 9)
	l3[0] = byte(nxdn.RCCHSITEINFO)
	copy(l3[1:9], payload[:])

	// Build the 155-bit spec info block: SR zero + L3 prefix +
	// trailing zeros + 3 Null bits.
	info := make([]byte, nxdn.CACInfoBits)
	l3Bits := framing.UnpackBitsMSB(l3, 72)
	copy(info[8:8+72], l3Bits)

	channel := nxdn.EncodeCACChannel(info)
	cacDibits := framing.BitsToDibits(channel)

	frame := make([]uint8, 0, 8+len(lichDibits)+len(cacDibits))
	frame = append(frame, nxdn.FSWDibitsOutbound...)
	frame = append(frame, lichDibits...)
	frame = append(frame, cacDibits...)

	out := make([]uint8, 0, 300+repeats*(len(frame)+50)+100)
	for i := 0; i < 300; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < repeats; r++ {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}
