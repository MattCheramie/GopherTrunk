package ccdecoder

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestResyncGuardContract pins the shared guard's behaviour: no-op until the
// first decode ever lands, a heartbeat change clears the budget, a full
// window of samples fires exactly once, and the fire resets the budget (the
// inherent throttle).
func TestResyncGuardContract(t *testing.T) {
	g := resyncGuard{rateHz: 48_000, window: 2 * time.Second}
	budget := int64(g.window.Seconds() * g.rateHz)

	// Before the first decode (activity == 0) the guard never fires, no
	// matter how much signal accumulates.
	for i := 0; i < 4; i++ {
		if g.check(int(budget), 0) {
			t.Fatal("guard fired before the first decode ever landed")
		}
	}

	// First sighting of a heartbeat syncs the tracker without firing.
	if g.check(0, 100) {
		t.Fatal("guard fired on the first heartbeat sighting")
	}

	// A held heartbeat accumulates; one sample short of the budget must not
	// fire, crossing it fires exactly once.
	if g.check(int(budget-1), 100) {
		t.Fatal("guard fired one sample short of the budget")
	}
	if !g.check(1, 100) {
		t.Fatal("guard did not fire once the budget was crossed")
	}
	// The fire cleared the budget: another sub-budget feed must not fire.
	if g.check(int(budget-1), 100) {
		t.Fatal("guard fired again before another full budget (throttle failed)")
	}

	// A heartbeat advance clears the accumulated budget entirely.
	if g.check(0, 200) {
		t.Fatal("guard fired on a heartbeat advance")
	}
	if g.samplesSinceDecode != 0 {
		t.Fatalf("heartbeat advance did not clear the budget: %d", g.samplesSinceDecode)
	}

	// reset() restarts everything, back to the no-op-until-decode state.
	g.check(int(budget/2), 200)
	g.reset()
	if g.samplesSinceDecode != 0 || g.lastSeenActivity != 0 {
		t.Fatalf("reset left state behind: samples=%d lastSeen=%d", g.samplesSinceDecode, g.lastSeenActivity)
	}
}

// newP25P1ResyncTestPipeline builds a P25 Phase 1 C4FM pipeline at 48 kHz
// with a debug logger, arms its decode heartbeat by feeding real modulated
// control-channel IQ (CRC-clean TSBKs), and returns the pipeline, log buffer
// and the drought sample budget.
func newP25P1ResyncTestPipeline(t *testing.T) (*p25Phase1Pipeline, *bytes.Buffer, int64) {
	t.Helper()
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)

	pp, err := newP25Phase1Pipeline(PipelineOptions{
		Bus: bus, Log: logger, SystemName: "Test", FrequencyHz: 851_000_000,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolP25,
			ControlChannels: []uint32{851_000_000},
		},
	})
	if err != nil {
		t.Fatalf("newP25Phase1Pipeline: %v", err)
	}
	p := pp.(*p25Phase1Pipeline)

	// Arm the heartbeat with real modulated CC frames through the full
	// receiver (FM mod at the spec 1800 Hz deviation, scaled for AGC
	// headroom like the decoder tests' fixtures).
	dibits := buildP25CCDibits(0x293, 6)
	iq := demod.ModulateP25C4FM(dibits, 48_000, 1800.0)
	for i := range iq {
		iq[i] *= 100
	}
	p.Process(iq)
	if decoded, _ := p.TSBKCounts(); decoded == 0 {
		t.Fatal("arming fixture decoded no TSBKs")
	}
	if p.cc.LastActivityNano() == 0 {
		t.Fatal("TSBK decode did not stamp the activity heartbeat")
	}
	if strings.Contains(buf.String(), "dsp resync") {
		t.Fatalf("resync fired during the clean arming feed (no-harm broken)\n%s", buf.String())
	}
	buf.Reset()

	budget := int64(p25Phase1ResyncWindow.Seconds() * 48_000)
	return p, buf, budget
}

// feedPipelineNoise drives a pipeline's real Process path with n samples of a
// fixed low-level constant — nonzero (no zero-RMS AGC divide) but carrying no
// frame sync, so nothing decodes and the drought budget accumulates.
func feedPipelineNoise(p ProtocolPipeline, n int64) {
	const chunk = 8192
	for n > 0 {
		c := n
		if c > chunk {
			c = chunk
		}
		buf := make([]complex64, c)
		for i := range buf {
			buf[i] = complex(0.01, 0.01)
		}
		p.Process(buf)
		n -= c
	}
}

// TestP25P1ResyncFiresOnDecodeDrought is the failing-first regression for
// the generalised watchdog on the P25 Phase 1 pipeline (before it, the
// pipeline had NO drought recovery at all — this test's resync assertion is
// red against the old bare Process). It also pins recovery: after the
// destructive reset the same clean fixture must decode again through the
// CC's non-contiguous-index resync path.
func TestP25P1ResyncFiresOnDecodeDrought(t *testing.T) {
	p, buf, budget := newP25P1ResyncTestPipeline(t)

	// A full budget of decode-free signal fires exactly one resync.
	feedPipelineNoise(p, budget)
	if got := strings.Count(buf.String(), "p25/phase1: dsp resync"); got != 1 {
		t.Fatalf("want exactly one resync after a full-budget drought, got %d\n%s", got, buf.String())
	}

	// Sub-budget feed after the fire must not fire again (throttle).
	buf.Reset()
	feedPipelineNoise(p, budget-1)
	if strings.Contains(buf.String(), "dsp resync") {
		t.Fatalf("resync fired again before another full budget (throttle failed)\n%s", buf.String())
	}

	// Recovery: the receiver was reset mid-stream (dibit index restarted),
	// so the CC must reacquire on clean signal via its buffer-resync path.
	// The receiver's slow loops (symbol clock, AGC) need a bounded warm-up
	// after the long noise run — decode must return within a few fixture
	// repetitions, proving the CC is not permanently wedged by the index
	// restart (the failure mode ResyncReset exists to prevent on DMR/NXDN).
	before, _ := p.TSBKCounts()
	dibits := buildP25CCDibits(0x293, 6)
	iq := demod.ModulateP25C4FM(dibits, 48_000, 1800.0)
	for i := range iq {
		iq[i] *= 100
	}
	recovered := false
	for pass := 0; pass < 4 && !recovered; pass++ {
		p.Process(iq)
		after, _ := p.TSBKCounts()
		recovered = after > before
	}
	if !recovered {
		t.Fatalf("no TSBKs decoded across 4 clean fixture passes after the drought reset (reacquire broken)")
	}
}

// TestP25P1ResyncNoHarmOnHealthyStream pins the no-harm side: a continuous
// clean control-channel stream far longer than the drought window never
// fires a resync, because every window contains decodes that clear the
// budget.
func TestP25P1ResyncNoHarmOnHealthyStream(t *testing.T) {
	p, buf, budget := newP25P1ResyncTestPipeline(t)

	dibits := buildP25CCDibits(0x293, 8)
	iq := demod.ModulateP25C4FM(dibits, 48_000, 1800.0)
	for i := range iq {
		iq[i] *= 100
	}
	var fed int64
	for fed < 3*budget {
		p.Process(iq)
		fed += int64(len(iq))
	}
	if strings.Contains(buf.String(), "dsp resync") {
		t.Fatalf("resync fired on a healthy continuously-decoding stream\n%s", buf.String())
	}
}

// TestNXDNResyncFiresOnDecodeDrought pins the watchdog wiring on the NXDN
// pipeline (failing-first: the old bare Process never resynced). The
// heartbeat is armed through the public IngestFrame path — the same
// CRC-clean-CAC stamping the on-air chain reaches.
func TestNXDNResyncFiresOnDecodeDrought(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	t.Cleanup(bus.Close)

	pp, err := newNXDNPipeline(PipelineOptions{
		Bus: bus, Log: logger, SystemName: "Test", FrequencyHz: 157_000_000,
		SampleRateHz: 48_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolNXDN,
			ControlChannels: []uint32{157_000_000},
		},
	})
	if err != nil {
		t.Fatalf("newNXDNPipeline: %v", err)
	}
	p := pp.(*nxdnPipeline)

	// Arm the heartbeat via the public CRC-clean-frame entry point.
	p.cc.IngestFrame(nxdn.LICH{ParityOK: true, RFCh: nxdn.RFChControl}, &nxdn.CACMessage{Type: nxdn.RCCHCCH})
	if p.cc.LastActivityNano() == 0 {
		t.Fatal("IngestFrame did not stamp the activity heartbeat")
	}
	p.Process(nil) // sync the guard to the armed heartbeat

	budget := int64(nxdnResyncWindow.Seconds() * 48_000)
	feedPipelineNoise(p, budget)
	if got := strings.Count(buf.String(), "nxdn: dsp resync"); got != 1 {
		t.Fatalf("want exactly one resync after a full-budget drought, got %d\n%s", got, buf.String())
	}
}
