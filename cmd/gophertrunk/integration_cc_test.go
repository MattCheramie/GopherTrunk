//go:build integration

package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestDaemonCCDecodesP25Phase1 is the end-to-end "lights up live
// trunked reception" check from the roadmap. It boots the wired
// daemon with a mock SDR replaying a fully-synthesized P25 Phase
// 1 control-channel IQ stream (built by the C4FM modulator in
// internal/dsp/demod) and asserts the full chain — IQ → C4FM
// demod → MM clock recovery → 4-level slice → dibits → FSW +
// NID + TSBK trellis → CC state machine — recovers the lock:
//
//   - daemon construction (pool, supervisor, ccdecoder)
//   - cchunt supervisor publishing KindHuntProgress
//   - ccdecoder factory dispatch + pipeline construction (the
//     production newP25Phase1Pipeline; no test stubs)
//   - mock SDR's IQ chunks land in the real receiver
//   - receiver's RRC matched filter + Mueller-Müller clock
//     recovery + 4-level slicer emit dibits
//   - phase1.ControlChannel.Process drives the state machine
//   - state machine emitting cc.locked on the bus
//   - supervisor consuming cc.locked → state=locked transition
//   - /api/v1/scanner reflecting the lock
//   - gophertrunk_control_channel_locked metric reaching 1
//
// The plan documented this as the close-out for Workstream A
// ("lights up live trunked reception"). PR #147 landed an
// intermediate version that stubbed the IQ→dibit step via
// ccdecoder.SetTestFactory; PR #148 replaced that stub with the
// real C4FM modulator + RRC pulse-shaping primitive shipped in
// internal/dsp/demod/c4fm_modulator.go.
//
// TSBK / band-plan / grant publication after cc.locked is covered
// by TestDaemonCCDecodesP25Phase1GrantChain below, which uses
// ccdecoder.SetTestFactory + a dibit-direct stub so the grant
// chain assertion doesn't depend on the receiver's Mueller-Müller
// clock loop landing more than one FSW per stream (which is fine
// for cc.locked, since the state machine only needs one FSW match
// to lock, but unreliable for the multi-frame status → identifier
// → grant sequence).
func TestDaemonCCDecodesP25Phase1(t *testing.T) {
	const (
		nac           = 0x293
		controlFreqHz = 851_000_000
		sampleRateHz  = 48_000
		sps           = 10
		span          = 8
		alpha         = 0.2
		deviationHz   = 1800.0
		frameRepeats  = 30
	)

	// Build a P25 Phase 1 dibit stream: a long warmup pattern
	// (cycling through every symbol so the Mueller-Müller clock
	// recovery sees plenty of transitions and locks) followed by
	// multiple FSW + NID + trellis-encoded TSBK frames separated
	// by idle dibits. The repeats give the receiver multiple
	// sync-detect chances; with the C4FM modulator's RRC pulse
	// shaping the matched-filter cascade is ISI-free at symbol
	// centres, so any one of those frames is enough to lock.
	dibits := buildP25LockedIQDibits(nac, frameRepeats)

	// Modulate the dibit stream through the C4FM TX chain
	// (impulse train → RRC pulse shape → FM modulator → IQ).
	// 48 kHz @ 10 sps = 4800 baud, the spec rate. 1800 Hz peak
	// deviation matches TIA-102.BAAA-A; the matched
	// newP25Phase1Pipeline configures the receiver's slicer
	// thresholds against this same deviation via the
	// p25phase1rx.Options.DeviationHz knob.
	iq := demod.ModulateC4FM(dibits, sps, span, alpha, sampleRateHz, deviationHz)

	dir := t.TempDir()
	iqPath := filepath.Join(dir, "p25-cc.cfile")
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
		{Name: "Alpha", Protocol: "p25", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc", logger)
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
	waitReachable(t, base+"/api/v1/health", 5*time.Second)

	// 60 s deadline absorbs cold-start latency on the first
	// iteration of a `-count` flakiness sweep (Go runtime + RRC
	// filter init + MM clock-loop convergence + mock SDR ticker
	// warmup all run lazily on first call). PR #220's CI flake
	// at 10.31 s on a 10 s deadline drove the first bump to 30 s
	// (commit f2a728e); a subsequent flake on PR #244's CI hit
	// the full 30 s deadline under -race + GitHub-hosted runner
	// contention, so the budget is doubled again. Subsequent
	// iterations complete in under 100 ms.
	const ccLockedDeadline = 60 * time.Second
	deadline := time.After(ccLockedDeadline)
	var locked bool
WaitLoop:
	for !locked {
		select {
		case ev := <-sub.C:
			if ev.Kind != events.KindCCLocked {
				continue
			}
			ls, ok := ev.Payload.(phase1.LockState)
			if !ok {
				t.Errorf("CCLocked payload type = %T, want phase1.LockState", ev.Payload)
				continue
			}
			if ls.NAC != nac {
				t.Errorf("LockState.NAC = %#x, want %#x", ls.NAC, nac)
			}
			if ls.FrequencyHz != controlFreqHz {
				t.Errorf("LockState.FrequencyHz = %d, want %d",
					ls.FrequencyHz, controlFreqHz)
			}
			locked = true
			break WaitLoop
		case <-deadline:
			t.Fatalf("no cc.locked event arrived within %s", ccLockedDeadline)
		}
	}

	waitForScannerLock(t, base, "Alpha", 2*time.Second)

	// Verify the cc-locked gauge reaches 1 for our system. The
	// gauge is set by the events.KindCCLocked handler in
	// internal/metrics/prom.go; it's labelled by system /
	// repeater, so we check for the family + a value of 1
	// without pinning the exact label content.
	// gophertrunk_control_channel_locked{system="…"} = 1 is the
	// Prometheus-side signal that the daemon's metrics handler
	// saw the same cc.locked event and updated the gauge. The
	// system label can be "unknown" when the phase1 LockState's
	// SystemName isn't populated; that's fine — the metric
	// family + value pair is what we assert.
	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, "gophertrunk_control_channel_locked{") {
		t.Errorf("/metrics missing gophertrunk_control_channel_locked gauge family:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_control_channel_locked{system=`) ||
		!strings.Contains(body, "} 1") {
		t.Errorf("/metrics gophertrunk_control_channel_locked did not reach 1 for any system:\n%s", body)
	}
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event")
	}
}

// buildP25LockedIQDibits assembles a long P25 Phase 1 dibit
// stream suitable for the C4FM modulator + receiver chain:
//
//   - a 200-dibit warmup pattern cycling 0,1,2,3 so the
//     Mueller-Müller clock recovery sees every symbol level
//     and a transition every dibit
//   - `repeats` × (FSW + NID + trellis-encoded TSBK + 50 idle
//     dibits)
//   - a 100-dibit trailer for clean flush
//
// Mirrors the in-package phase1 test helpers' frame layout.
func buildP25LockedIQDibits(nac uint16, repeats int) []uint8 {
	frame := make([]uint8, 0, 24+32+98)
	frame = append(frame, phase1.FrameSyncWord[:]...)
	nidBits := phase1.EncodeNIDBits(nac, phase1.DUIDTrunkingSignaling)
	for i := 0; i < 32; i++ {
		frame = append(frame, (nidBits[2*i]<<1)|nidBits[2*i+1])
	}
	tsbk := phase1.AssembleTSBK(phase1.TSBK{LB: true, Opcode: phase1.OpRFSSStatusBroadcast})
	frame = append(frame, phase1.EncodeTSBKChannel(tsbk)...)

	out := make([]uint8, 0, 200+repeats*(len(frame)+50)+100)
	for i := 0; i < 200; i++ {
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

// TestDaemonCCDecodesP25Phase1GrantChain extends the cc.locked
// integration test by driving a full status → IdentifierUpdate →
// GroupVoiceChannelGrant TSBK sequence through the production
// daemon and asserting every cross-package surface the grant chain
// touches:
//
//   - phase1.ControlChannel dispatching the IdentifierUpdate TSBK
//     to its band plan
//   - phase1.ControlChannel resolving the grant's ChannelID +
//     ChannelNumber through that band plan and publishing
//     events.KindGrant with the resolved FrequencyHz
//   - trunking.Engine (subscribed via daemon.NewDaemon) accepting
//     the grant
//   - supervisor / scanner state staying locked through grant
//     dispatch
//   - /metrics' gophertrunk_events_total{kind="grant"} counter
//     incrementing
//
// The IQ → dibit chain is intentionally bypassed: the receiver's
// Mueller-Müller clock loop reliably lands the first FSW (validated
// by TestDaemonCCDecodesP25Phase1 above), but converging well
// enough to extract every subsequent FSW + NID + 98-dibit TSBK
// trellis window within one streaming pass is a tuning exercise
// orthogonal to the grant-chain wiring this test owns. Instead we
// register a stub pipeline via ccdecoder.SetTestFactory that ignores
// the IQ chunks entirely and pumps the synthesized dibit stream
// directly into the real phase1.ControlChannel on first invocation.
// Everything *above* IQ → dibit — the factory dispatch, the
// state machine, the band plan, the bus publication, the engine,
// the supervisor, the API, the metrics handler — runs through the
// real production code paths.
func TestDaemonCCDecodesP25Phase1GrantChain(t *testing.T) {
	const (
		nac                = 0x293
		controlFreqHz      = 851_000_000
		sampleRateHz       = 48_000
		grantChannelID     = 1
		grantChannelNumber = 5
		grantTalkgroup     = 0xCAFE
		grantSourceID      = 0xABCDEF
		bandBaseHz         = 851_000_000
		bandSpacingHz      = 12_500
	)

	// Build a dibit stream that drives the full status →
	// IdentifierUpdate → grant chain. Repeats are low because the
	// stub pipeline pumps every dibit into cc.Process on the
	// first IQ chunk it sees — every frame lands; we just need
	// enough to drive each opcode at least once.
	dibits := buildP25CallChainIQDibits(buildP25CallChainParams{
		NAC:                nac,
		Repeats:            2,
		BandChannelID:      grantChannelID,
		BandBaseHz:         bandBaseHz,
		BandSpacingHz:      bandSpacingHz,
		BandwidthHz:        12_500,
		TxOffsetHz:         0,
		GrantChannelID:     grantChannelID,
		GrantChannelNumber: grantChannelNumber,
		GrantTalkgroup:     grantTalkgroup,
		GrantSourceID:      grantSourceID,
	})

	// Stub factory: build the real phase1.ControlChannel, drive
	// the synthesized dibit stream into it from the first
	// Process(iq) call, then no-op on every subsequent call. The
	// state machine, bus publication, and band plan all run
	// through production code.
	restore := ccdecoder.SetTestFactory(trunking.ProtocolP25, func(opts ccdecoder.PipelineOptions) (ccdecoder.ProtocolPipeline, error) {
		cc := phase1.New(phase1.Options{
			Bus:         opts.Bus,
			Log:         opts.Log,
			SystemName:  opts.SystemName,
			FrequencyHz: opts.FrequencyHz,
		})
		return &p25Phase1StubPipeline{cc: cc, dibits: dibits}, nil
	})
	defer restore()

	// IQ stream — the stub pipeline ignores its content, but the
	// mock SDR has to keep streaming so the decoder loop sees
	// Process(iq) calls. ~5 s of zero-filled u8 IQ at 48 kHz
	// (480 000 samples × 2 bytes) gives the supervisor's hunter
	// + cchunt + ccdecoder construction goroutines plenty of
	// runway to publish HuntProgress, construct the stub
	// pipeline, and pump the first chunk before the mock SDR EOFs.
	dir := t.TempDir()
	iqPath := filepath.Join(dir, "p25-cc-grant.cfile")
	if err := os.WriteFile(iqPath, make([]byte, 960_000), 0o600); err != nil {
		t.Fatalf("write IQ: %v", err)
	}
	sdr.Register(&sdr.MockDriver{Files: []string{iqPath}})

	cfg := config.Default()
	cfg.SDR.SampleRate = sampleRateHz
	cfg.SDR.Devices = []config.DeviceConfig{
		{Serial: "mock-00", Role: "control"},
	}
	cfg.Trunking.Systems = []config.SystemConfig{
		{Name: "Alpha", Protocol: "p25", ControlChannels: []uint32{controlFreqHz}},
	}
	cfg.API.HTTPAddr = freeAddr(t)
	cfg.Metrics.Enabled = true

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	d, err := NewDaemon(cfg, "integration-cc-grant", logger)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
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
	waitReachable(t, base+"/api/v1/health", 5*time.Second)

	// 60 s deadline absorbs first-iteration cold-start in `-count`
	// flakiness sweeps — same pattern + same budget as
	// TestDaemonCCDecodesP25Phase1 (which flaked again at 30 s
	// under -race + GitHub-hosted runner contention on PR #244,
	// driving the doubled budget).
	const grantChainDeadline = 60 * time.Second
	deadline := time.After(grantChainDeadline)
	var locked, granted bool
	var grantPayload trunking.Grant
WaitLoop:
	for !locked || !granted {
		select {
		case ev := <-sub.C:
			switch ev.Kind {
			case events.KindCCLocked:
				ls, ok := ev.Payload.(phase1.LockState)
				if !ok {
					t.Errorf("CCLocked payload type = %T, want phase1.LockState", ev.Payload)
					continue
				}
				if ls.NAC != nac {
					t.Errorf("LockState.NAC = %#x, want %#x", ls.NAC, nac)
				}
				if ls.FrequencyHz != controlFreqHz {
					t.Errorf("LockState.FrequencyHz = %d, want %d",
						ls.FrequencyHz, controlFreqHz)
				}
				locked = true
			case events.KindGrant:
				g, ok := ev.Payload.(trunking.Grant)
				if !ok {
					t.Errorf("Grant payload type = %T, want trunking.Grant", ev.Payload)
					continue
				}
				grantPayload = g
				granted = true
			}
		case <-deadline:
			if !locked {
				t.Fatalf("no cc.locked event arrived within %s", grantChainDeadline)
			}
			if !granted {
				t.Logf("metrics at timeout:\n%s", scrape(t, base+"/metrics"))
				t.Fatalf("no KindGrant event arrived within %s", grantChainDeadline)
			}
			break WaitLoop
		}
	}

	if grantPayload.Protocol != "p25" {
		t.Errorf("Grant.Protocol = %q, want p25", grantPayload.Protocol)
	}
	if grantPayload.GroupID != grantTalkgroup {
		t.Errorf("Grant.GroupID = %#x, want %#x", grantPayload.GroupID, grantTalkgroup)
	}
	if grantPayload.SourceID != grantSourceID {
		t.Errorf("Grant.SourceID = %#x, want %#x", grantPayload.SourceID, grantSourceID)
	}
	// Resolved frequency = base + spacing × channelNumber
	// = 851_000_000 + 12_500 × 5 = 851_062_500.
	const wantGrantFreq = 851_062_500
	if grantPayload.FrequencyHz != wantGrantFreq {
		t.Errorf("Grant.FrequencyHz = %d, want %d (base + spacing × channel)",
			grantPayload.FrequencyHz, wantGrantFreq)
	}

	waitForScannerLock(t, base, "Alpha", 2*time.Second)

	body := scrape(t, base+"/metrics")
	if !strings.Contains(body, `gophertrunk_events_total{kind="cc.locked"} 1`) {
		t.Errorf("/metrics did not count one cc.locked event:\n%s", body)
	}
	// At least one grant event — the stub fires every grant frame
	// in `Repeats` rounds, so the counter sits at >=1.
	if !strings.Contains(body, `gophertrunk_events_total{kind="grant"}`) {
		t.Errorf("/metrics missing gophertrunk_events_total grant counter:\n%s", body)
	}
}

// p25Phase1StubPipeline implements ccdecoder.ProtocolPipeline by
// pumping a fixed dibit stream into a real phase1.ControlChannel
// on the first Process call. The IQ chunks themselves are ignored
// — this stub is meant to test everything *above* the IQ → dibit
// step, the grant chain's chunk-boundary-sensitive multi-frame
// sequence.
type p25Phase1StubPipeline struct {
	cc       *phase1.ControlChannel
	dibits   []uint8
	consumed bool
}

func (p *p25Phase1StubPipeline) Process(iq []complex64) {
	if p.consumed {
		return
	}
	p.consumed = true
	p.cc.Process(p.dibits, 0)
}

func (p *p25Phase1StubPipeline) Reset()       {}
func (p *p25Phase1StubPipeline) Close() error { return nil }

// buildP25CallChainParams configures buildP25CallChainIQDibits.
type buildP25CallChainParams struct {
	NAC     uint16
	Repeats int

	// Band-plan parameters carried in the IdentifierUpdate TSBK.
	// The receiver populates its band plan from these so the
	// subsequent GroupVoiceChannelGrant's ChannelID + ChannelNumber
	// can resolve to a real frequency.
	BandChannelID uint8
	BandBaseHz    uint64
	BandSpacingHz uint32
	BandwidthHz   uint32
	TxOffsetHz    int64

	// GroupVoiceChannelGrant fields.
	GrantChannelID     uint8
	GrantChannelNumber uint16
	GrantTalkgroup     uint16
	GrantSourceID      uint32
}

// buildP25CallChainIQDibits assembles a P25 Phase 1 dibit stream
// that drives the full cc.locked → grant chain on the receiver:
//
//   - 200-dibit warmup cycling 0..3
//   - `Repeats` × RFSSStatusBroadcast frames (drive cc.locked)
//   - `Repeats` × IdentifierUpdate frames (prime the band plan
//     so subsequent grants can resolve to a real frequency)
//   - `Repeats` × GroupVoiceChannelGrant frames (resolve through
//     the now-primed band plan; drive KindGrant on each frame)
//   - 100-dibit trailer for clean flush
//
// Frames are grouped rather than interleaved so the band plan is
// primed by the IdentifierUpdate batch BEFORE the grant batch
// arrives.
func buildP25CallChainIQDibits(p buildP25CallChainParams) []uint8 {
	statusFrame := buildP25TSBKFrame(p.NAC, phase1.TSBK{
		LB:     true,
		Opcode: phase1.OpRFSSStatusBroadcast,
	})
	identFrame := buildP25TSBKFrame(p.NAC, phase1.TSBK{
		LB:     true,
		Opcode: phase1.OpIdentifierUpdate,
		Payload: phase1.AssembleIdentifierUpdate(phase1.IdentifierUpdate{
			ChannelID:   p.BandChannelID,
			BandwidthHz: p.BandwidthHz,
			SpacingHz:   p.BandSpacingHz,
			TxOffsetHz:  p.TxOffsetHz,
			BaseHz:      p.BandBaseHz,
		}),
	})
	grantFrame := buildP25TSBKFrame(p.NAC, phase1.TSBK{
		LB:      true,
		Opcode:  phase1.OpGroupVoiceChannelGrant,
		Payload: assembleGroupVoiceChannelGrant(p),
	})

	frameWithGap := func(out []uint8, frame []uint8) []uint8 {
		out = append(out, frame...)
		for i := 0; i < 50; i++ {
			out = append(out, uint8(i&3))
		}
		return out
	}

	out := make([]uint8, 0, 200+p.Repeats*(len(statusFrame)+len(identFrame)+len(grantFrame)+150)+100)
	for i := 0; i < 200; i++ {
		out = append(out, uint8(i&3))
	}
	for r := 0; r < p.Repeats; r++ {
		out = frameWithGap(out, statusFrame)
	}
	for r := 0; r < p.Repeats; r++ {
		out = frameWithGap(out, identFrame)
	}
	for r := 0; r < p.Repeats; r++ {
		out = frameWithGap(out, grantFrame)
	}
	for i := 0; i < 100; i++ {
		out = append(out, uint8(i&3))
	}
	return out
}

// buildP25TSBKFrame returns the FSW + NID + trellis-encoded TSBK
// dibits for a single frame.
func buildP25TSBKFrame(nac uint16, tsbk phase1.TSBK) []uint8 {
	frame := make([]uint8, 0, 24+32+98)
	frame = append(frame, phase1.FrameSyncWord[:]...)
	nidBits := phase1.EncodeNIDBits(nac, phase1.DUIDTrunkingSignaling)
	for i := 0; i < 32; i++ {
		frame = append(frame, (nidBits[2*i]<<1)|nidBits[2*i+1])
	}
	frame = append(frame, phase1.EncodeTSBKChannel(phase1.AssembleTSBK(tsbk))...)
	return frame
}

// assembleGroupVoiceChannelGrant packs the 8-byte payload of a
// GroupVoiceChannelGrant TSBK matching the layout
// phase1.ParseGroupVoiceChannelGrant expects:
//
//	byte 0:    service options
//	byte 1-2:  channel (4-bit ID + 12-bit number, big-endian)
//	byte 3-4:  group address
//	byte 5-7:  source unit (24-bit)
func assembleGroupVoiceChannelGrant(p buildP25CallChainParams) [8]byte {
	var out [8]byte
	out[0] = 0 // service options: not encrypted, not emergency
	chanField := (uint16(p.GrantChannelID)&0x0F)<<12 | (p.GrantChannelNumber & 0x0FFF)
	out[1] = byte(chanField >> 8)
	out[2] = byte(chanField & 0xFF)
	out[3] = byte(p.GrantTalkgroup >> 8)
	out[4] = byte(p.GrantTalkgroup & 0xFF)
	out[5] = byte((p.GrantSourceID >> 16) & 0xFF)
	out[6] = byte((p.GrantSourceID >> 8) & 0xFF)
	out[7] = byte(p.GrantSourceID & 0xFF)
	return out
}

// writeIQToU8File serialises a complex64 IQ buffer to u8
// interleaved pairs (the format sdr.MockDriver consumes). Each
// complex64 sample becomes 2 bytes: I and Q each scaled from
// [-1, 1] to [0, 255] with 127.5 offset.
func writeIQToU8File(path string, iq []complex64) error {
	out := make([]byte, len(iq)*2)
	for i, s := range iq {
		out[2*i] = floatToU8(real(s))
		out[2*i+1] = floatToU8(imag(s))
	}
	return os.WriteFile(path, out, 0o600)
}

func floatToU8(v float32) byte {
	scaled := float64(v)*127.0 + 127.5
	if scaled < 0 {
		return 0
	}
	if scaled > 255 {
		return 255
	}
	return byte(scaled)
}

// waitForScannerLock polls /api/v1/scanner until the named system
// reports state=locked or the timeout fires.
func waitForScannerLock(t *testing.T, base, system string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/v1/scanner")
		if err == nil {
			var status struct {
				Systems []struct {
					Name  string `json:"name"`
					State string `json:"state"`
				} `json:"systems"`
			}
			err := json.NewDecoder(resp.Body).Decode(&status)
			resp.Body.Close()
			if err == nil {
				for _, s := range status.Systems {
					if s.Name == system && s.State == "locked" {
						return
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("/api/v1/scanner did not report state=locked for %q within %v", system, timeout)
}
