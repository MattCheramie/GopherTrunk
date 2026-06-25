package widebandt2

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp/iqpower"
)

// loudIQ returns a chunk near full scale; quietIQ returns near silence. Both
// feed iqpower.Accumulator directly so the test controls the per-channel vs
// wideband-input power split without going through the DSP chain.
func loudIQ(n int) []complex64 {
	c := make([]complex64, n)
	for i := range c {
		c[i] = complex(0.5, 0.5)
	}
	return c
}

func quietIQ(n int) []complex64 {
	c := make([]complex64, n)
	for i := range c {
		c[i] = complex(1e-4, 0)
	}
	return c
}

// runDiag builds a one-channel Engine, loads the accumulators, and flushes one
// diagnostics window, returning the captured WARN record (if any).
func runDiag(t *testing.T, chanIQ, widebandIQ []complex64) (capturedRecord, bool) {
	t.Helper()
	handler, recs, mu := newRecordingHandler()
	ec := &engineChannel{freqHz: 450_125_000, sysName: "Main_Site_1", protoTag: "p25-phase1"}
	ec.pwr.Add(chanIQ)
	e := &Engine{
		log:      slog.New(handler),
		channels: []*engineChannel{ec},
		now:      time.Now,
	}
	e.widebandPwr.Add(widebandIQ)

	// First call seeds lastDiagAt; the second (a full Window later) flushes.
	base := time.Unix(1_700_000_000, 0)
	e.maybeLogDiagnostics(base)
	e.maybeLogDiagnostics(base.Add(iqpower.Window + time.Second))

	mu.Lock()
	defer mu.Unlock()
	for _, r := range *recs {
		if r.level == slog.LevelWarn && strings.Contains(r.msg, "iq power very low") {
			return r, true
		}
	}
	return capturedRecord{}, false
}

// TestLowPowerHintOffBand: a healthy wideband input but a dead channel points
// the operator at an off-band / wrong-frequency carrier, not the antenna — and
// the WARN now carries the wideband input level for context.
func TestLowPowerHintOffBand(t *testing.T) {
	const n = 4096
	r, ok := runDiag(t, quietIQ(n), loudIQ(n))
	if !ok {
		t.Fatal("expected a low-power WARN, got none")
	}
	if !strings.Contains(r.msg, "outside the captured passband") {
		t.Errorf("WARN msg = %q, want the off-band hint", r.msg)
	}
	// On a shared front end a healthy capture with a dead tap can also be a
	// weaker co-tenant under-amplified by a fixed gain; the hint now points
	// at 'gain: auto' (AGC) as the remedy (issue #749).
	if !strings.Contains(r.msg, "gain: auto") {
		t.Errorf("WARN msg = %q, want the gain: auto (AGC) hint", r.msg)
	}
	if _, has := r.attrs["wideband_input_dbfs"]; !has {
		t.Errorf("WARN missing wideband_input_dbfs attr; attrs=%v", r.attrs)
	}
	if wb, ok := r.attrs["wideband_input_dbfs"].(float64); !ok || wb < -55 {
		t.Errorf("wideband_input_dbfs = %v, want a healthy (>= -55) level", r.attrs["wideband_input_dbfs"])
	}
}

// TestLowPowerHintDeadFrontEnd: when the wideband input is also dead, the hint
// stays on the antenna/gain/cable front end.
func TestLowPowerHintDeadFrontEnd(t *testing.T) {
	const n = 4096
	r, ok := runDiag(t, quietIQ(n), quietIQ(n))
	if !ok {
		t.Fatal("expected a low-power WARN, got none")
	}
	if !strings.Contains(r.msg, "antenna") {
		t.Errorf("WARN msg = %q, want the antenna/gain/cable hint", r.msg)
	}
}
