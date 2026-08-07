package ccdecoder

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestTETRAEffectiveBaud pins the effective-baud helper: symbols over the
// window duration, plus percentage deviation from the nominal 18000 sym/s.
func TestTETRAEffectiveBaud(t *testing.T) {
	cases := []struct {
		name    string
		dibits  int64
		elapsed time.Duration
		baud    float64
		dev     float64
	}{
		{"nominal", 108_000, 6 * time.Second, 18000, 0},
		{"plus_0.1pct", 90_070, 5 * time.Second, 18014, (18014.0 - 18000) / 18000 * 100},
		{"empty_window", 0, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baud, dev := tetraEffectiveBaud(tc.dibits, tc.elapsed)
			if baud != tc.baud {
				t.Errorf("baud = %v, want %v", baud, tc.baud)
			}
			if (dev-tc.dev) > 1e-9 || (tc.dev-dev) > 1e-9 {
				t.Errorf("devPct = %v, want %v", dev, tc.dev)
			}
		})
	}
}

// TestTETRAStatusLineThrottled drives the pipeline's decode-status line with a
// deterministic clock: it primes on the first call, stays silent until the
// throttle interval elapses, then emits one line with the effective baud
// derived from the symbols processed in the window. A second call within the
// same interval must not re-emit.
func TestTETRAStatusLineThrottled(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus := events.NewBus(8)
	defer bus.Close()

	pp, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, Log: logger, SystemName: "Test", FrequencyHz: 412_000_000,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{412_000_000},
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	p := pp.(*tetraPipeline)

	now := time.Unix(1000, 0)
	p.now = func() time.Time { return now }

	// First call primes the window; no status line yet.
	p.maybeLogStatus()
	if strings.Contains(buf.String(), "tetra: decode status") {
		t.Fatalf("status line emitted before the interval elapsed\n%s", buf.String())
	}

	// Accumulate 108000 symbols so baud = 108000 / 6 s = 18000.
	p.cc.Process(make([]uint8, 108_000), 0)

	now = now.Add(6 * time.Second)
	p.maybeLogStatus()
	out := buf.String()
	if !strings.Contains(out, "tetra: decode status") {
		t.Fatalf("expected a status line after the interval\n%s", out)
	}
	if !strings.Contains(out, "baud=18000") {
		t.Errorf("expected baud=18000 in status line\n%s", out)
	}
	for _, key := range []string{"carrier_off_hz=", "locked=", "sb_bursts=", "sch_pdus=", "sch_pdus_fail=", "colour_code="} {
		if !strings.Contains(out, key) {
			t.Errorf("status line missing key %q\n%s", key, out)
		}
	}

	// A second call within the same interval must not re-emit.
	buf.Reset()
	p.maybeLogStatus()
	if strings.Contains(buf.String(), "tetra: decode status") {
		t.Errorf("status line emitted twice within one interval\n%s", buf.String())
	}
}

// TestTETRAStatusIntervalConfigurable confirms tetra_status_interval_secs sets the
// emission cadence: a 10 s interval stays silent at 6 s (where the 5 s default
// would already have emitted) and emits once 10 s have elapsed. An unset interval
// falls back to the 5 s default.
func TestTETRAStatusIntervalConfigurable(t *testing.T) {
	newPipe := func(t *testing.T, intervalSecs float64) (*tetraPipeline, *bytes.Buffer) {
		t.Helper()
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		bus := events.NewBus(8)
		t.Cleanup(bus.Close)
		pp, err := newTETRAPipeline(PipelineOptions{
			Bus: bus, Log: logger, SystemName: "Test", FrequencyHz: 412_000_000,
			SampleRateHz: 144_000,
			System: trunking.System{
				Name: "Test", Protocol: trunking.ProtocolTETRA,
				ControlChannels:         []uint32{412_000_000},
				TETRAStatusIntervalSecs: intervalSecs,
			},
		})
		if err != nil {
			t.Fatalf("newTETRAPipeline: %v", err)
		}
		return pp.(*tetraPipeline), &buf
	}

	t.Run("10s interval waits past the 5s default", func(t *testing.T) {
		p, buf := newPipe(t, 10)
		if p.statusInterval != 10*time.Second {
			t.Fatalf("statusInterval = %v, want 10s", p.statusInterval)
		}
		now := time.Unix(2000, 0)
		p.now = func() time.Time { return now }
		p.maybeLogStatus() // prime

		now = now.Add(6 * time.Second) // past the 5 s default, before 10 s
		p.maybeLogStatus()
		if strings.Contains(buf.String(), "tetra: decode status") {
			t.Fatalf("status line emitted at 6 s with a 10 s interval\n%s", buf.String())
		}

		now = now.Add(5 * time.Second) // now 11 s since prime
		p.maybeLogStatus()
		if !strings.Contains(buf.String(), "tetra: decode status") {
			t.Fatalf("status line not emitted after 10 s interval elapsed\n%s", buf.String())
		}
	})

	t.Run("unset falls back to 5s default", func(t *testing.T) {
		p, _ := newPipe(t, 0)
		if p.statusInterval != tetraStatusInterval {
			t.Fatalf("statusInterval = %v, want default %v", p.statusInterval, tetraStatusInterval)
		}
	})
}

// TestTETRAStatusLineSilentAtInfo confirms the status line is gated on debug —
// an info-level logger emits nothing even past the interval.
func TestTETRAStatusLineSilentAtInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bus := events.NewBus(8)
	defer bus.Close()

	pp, err := newTETRAPipeline(PipelineOptions{
		Bus: bus, Log: logger, SystemName: "Test", FrequencyHz: 412_000_000,
		SampleRateHz: 144_000,
		System: trunking.System{
			Name: "Test", Protocol: trunking.ProtocolTETRA,
			ControlChannels: []uint32{412_000_000},
		},
	})
	if err != nil {
		t.Fatalf("newTETRAPipeline: %v", err)
	}
	p := pp.(*tetraPipeline)
	now := time.Unix(1000, 0)
	p.now = func() time.Time { return now }
	p.maybeLogStatus()
	now = now.Add(6 * time.Second)
	p.maybeLogStatus()
	if strings.Contains(buf.String(), "tetra: decode status") {
		t.Errorf("status line should be gated off at info level\n%s", buf.String())
	}
}
