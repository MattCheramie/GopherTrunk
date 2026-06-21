package widebandt2

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/demod"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// steppingClock returns a monotonically increasing time, advancing by a
// fixed step on every call. The wideband engine calls e.now() once per
// IQ chunk in Run, so a step over half iqpower.Window guarantees the
// per-channel diagnostics flush fires deterministically within a few
// chunks regardless of wall-clock timing under test load.
type steppingClock struct {
	mu   sync.Mutex
	t    time.Time
	step time.Duration
}

func (c *steppingClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(c.step)
	return c.t
}

// capturedRecord is one slog record the recordingHandler kept.
type capturedRecord struct {
	level slog.Level
	msg   string
	attrs map[string]any
}

// recordingHandler captures every emitted record so a test can assert
// on the engine's diagnostic log lines. Enabled for all levels so the
// DEBUG activity summary is exercised too.
type recordingHandler struct {
	mu   *sync.Mutex
	recs *[]capturedRecord
}

func newRecordingHandler() (recordingHandler, *[]capturedRecord, *sync.Mutex) {
	var mu sync.Mutex
	recs := make([]capturedRecord, 0, 16)
	return recordingHandler{mu: &mu, recs: &recs}, &recs, &mu
}

func (h recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h recordingHandler) Handle(_ context.Context, r slog.Record) error {
	rec := capturedRecord{level: r.Level, msg: r.Message, attrs: map[string]any{}}
	r.Attrs(func(a slog.Attr) bool {
		rec.attrs[a.Key] = a.Value.Any()
		return true
	})
	h.mu.Lock()
	*h.recs = append(*h.recs, rec)
	h.mu.Unlock()
	return nil
}
func (h recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h recordingHandler) WithGroup(string) slog.Handler      { return h }

// recordingPower captures RecordIQPowerDbFS calls for the metrics test.
type recordingPower struct {
	mu    sync.Mutex
	calls []struct {
		label string
		dbfs  float64
	}
}

func (r *recordingPower) RecordIQPowerDbFS(label string, dbfs float64) {
	r.mu.Lock()
	r.calls = append(r.calls, struct {
		label string
		dbfs  float64
	}{label, dbfs})
	r.mu.Unlock()
}
func (r *recordingPower) ClearIQPowerDbFS(string) {}

// TestEngineDiagnosticsLowPowerWarns feeds a silent IQ stream and proves
// the engine emits the throttled "iq power very low" WARN — the operator
// feedback that was missing when a single wideband DMR channel silently
// decoded nothing (field report VE5XAM) — and that the Tier II channel's
// sync counter stays at zero (no false sync on silence).
func TestEngineDiagnosticsLowPowerWarns(t *testing.T) {
	const (
		rateHz       = 48_000
		centerHz     = 446_500_000
		channelHz    = centerHz // offset 0: keep the tap inside the 48 kHz test band
		chunkSamples = 4800
		numChunks    = 8
	)

	silent := make([][]complex64, numChunks)
	for i := range silent {
		silent[i] = make([]complex64, chunkSamples) // zero-valued = silence
	}
	dev := newMockDevice(silent)

	bus := events.NewBus(64)
	defer bus.Close()

	handler, recs, mu := newRecordingHandler()
	clk := &steppingClock{t: time.Unix(0, 0), step: 600 * time.Millisecond}

	e, err := New(Options{
		Log:          slog.New(handler),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: rateHz,
		CenterFreqHz: centerHz,
		Now:          clk.now,
		Channels:     []ChannelConfig{{FrequencyHz: channelHz, SystemName: "dmr-simplex"}},
		Systems:      []trunking.System{t2System("dmr-simplex")},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	mu.Lock()
	var sawLowPower bool
	for _, r := range *recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "iq power very low") {
			sawLowPower = true
			if dbfs, ok := r.attrs["dbfs"].(float64); !ok || dbfs >= -55 {
				t.Errorf("low-power WARN dbfs attr = %v (ok=%v), want < -55", r.attrs["dbfs"], ok)
			}
		}
	}
	mu.Unlock()
	if !sawLowPower {
		t.Errorf("expected an 'iq power very low' WARN for a silent stream, saw none")
	}

	if c := e.channels[0].tier2Cnt.Counters(); c.SyncHits != 0 {
		t.Errorf("SyncHits on silence = %d, want 0", c.SyncHits)
	}
}

// TestEngineDiagnosticsCountersOnDecode feeds synthesized Voice LC Header
// IQ (same chain as the end-to-end test) and proves the Tier II decode
// counters move: sync detected, FEC passes, lock declared. This is what
// lets an operator distinguish "decoding fine" from the failure modes.
func TestEngineDiagnosticsCountersOnDecode(t *testing.T) {
	const (
		widebandRateHz = 48_000.0
		centerHz       = 460_000_000
		spsWideband    = 10
		spanSymbols    = 8
		alpha          = 0.20
		colorCode      = uint8(0x7)
		groupID        = uint32(0x123)
		sourceID       = uint32(0x456789)
		burstRepeats   = 200
		chunkSamples   = 4800
	)

	dibits := buildT2VoiceLCHeaderDibits(burstRepeats, colorCode, groupID, sourceID)
	baseband := demod.ModulateC4FM(dibits, spsWideband, spanSymbols, alpha, widebandRateHz, dmrDeviationHz)
	chunks := chunkComplex(baseband, chunkSamples)
	dev := newMockDevice(chunks)

	bus := events.NewBus(256)
	defer bus.Close()
	// Drain the bus so Publish never blocks the decode path.
	sub := bus.Subscribe()
	defer sub.Close()
	go func() {
		for range sub.C {
		}
	}()

	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: uint32(widebandRateHz),
		CenterFreqHz: centerHz,
		Channels:     []ChannelConfig{{FrequencyHz: centerHz, SystemName: "regional-t2"}},
		Systems:      []trunking.System{t2System("regional-t2")},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := e.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	c := e.channels[0].tier2Cnt.Counters()
	if c.SyncHits == 0 {
		t.Errorf("SyncHits = 0, want > 0 (no DMR sync detected on a clean signal)")
	}
	if c.FECPass == 0 {
		t.Errorf("FECPass = 0, want > 0 (no Voice LC Header passed FEC)")
	}
	if c.Locks == 0 {
		t.Errorf("Locks = 0, want >= 1 (cc never locked)")
	}
}

// TestEngineDiagnosticsMetricsGauge proves a non-nil IQPowerObserver
// receives the per-channel dBFS gauge with a frequency-qualified label.
func TestEngineDiagnosticsMetricsGauge(t *testing.T) {
	const (
		rateHz       = 48_000
		centerHz     = 446_500_000
		channelHz    = centerHz // offset 0: a DC-ish constant stays in the tap passband
		chunkSamples = 4800
		numChunks    = 8
	)

	// Non-trivial constant power so the gauge reads a finite, non-floor value.
	chunks := make([][]complex64, numChunks)
	for i := range chunks {
		c := make([]complex64, chunkSamples)
		for j := range c {
			c[j] = complex(0.25, 0.25)
		}
		chunks[i] = c
	}
	dev := newMockDevice(chunks)

	bus := events.NewBus(64)
	defer bus.Close()

	obs := &recordingPower{}
	clk := &steppingClock{t: time.Unix(0, 0), step: 600 * time.Millisecond}

	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: rateHz,
		CenterFreqHz: centerHz,
		Now:          clk.now,
		Metrics:      obs,
		Channels:     []ChannelConfig{{FrequencyHz: channelHz, SystemName: "dmr-simplex"}},
		Systems:      []trunking.System{t2System("dmr-simplex")},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	obs.mu.Lock()
	defer obs.mu.Unlock()
	if len(obs.calls) == 0 {
		t.Fatalf("IQPowerObserver received no RecordIQPowerDbFS calls")
	}
	got := obs.calls[0]
	if !strings.Contains(got.label, "dmr-simplex") || !strings.Contains(got.label, "MHz") {
		t.Errorf("gauge label = %q, want it to contain the system name and frequency", got.label)
	}
	// 0.25+0.25i → |x|² = 0.125 → ~-9.03 dBFS; assert it's finite and well
	// above the dead-stream floor.
	if got.dbfs < -55 || got.dbfs > 0 {
		t.Errorf("gauge dbfs = %v, want a finite value in (-55, 0)", got.dbfs)
	}
}
