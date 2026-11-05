package main

import (
	"bytes"
	"log/slog"
	"math"
	"regexp"
	"runtime"
	"runtime/metrics"
	"strconv"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

func TestHistDeltaStatsQuantileAndMax(t *testing.T) {
	buckets := []float64{0, 0.001, 0.002, 0.004, 0.008, math.Inf(1)}
	prev := &metrics.Float64Histogram{Buckets: buckets, Counts: []uint64{5, 5, 5, 5, 5}}
	// New observations: 98 in [0,1ms), 1 in [2,4ms), 1 in [8ms,+Inf).
	cur := &metrics.Float64Histogram{Buckets: buckets, Counts: []uint64{103, 5, 6, 5, 6}}
	q, m, ok := histDeltaStats(cur, prev, 0.99)
	if !ok {
		t.Fatal("ok=false with new observations")
	}
	if q != 0.004 {
		t.Errorf("p99 = %v, want 0.004 (the 99th of 100 new samples is in [2,4ms))", q)
	}
	if m != 0.008 {
		t.Errorf("max = %v, want 0.008 (+Inf bucket reported by its lower bound)", m)
	}
	if _, _, ok := histDeltaStats(cur, cur, 0.99); ok {
		t.Error("ok=true with no new observations")
	}
	short := &metrics.Float64Histogram{Buckets: buckets[:3], Counts: []uint64{1, 1}}
	if _, _, ok := histDeltaStats(cur, short, 0.99); ok {
		t.Error("ok=true with mismatched bucket layouts")
	}
}

func TestIQDropRuntimeSinceFirstSnapshotIsEmpty(t *testing.T) {
	cur := readIQDropRuntime()
	if d := cur.since(iqDropRuntime{}); d != (iqDropRuntimeDelta{}) {
		t.Errorf("first WARN must not report the process lifetime: %+v", d)
	}
}

func TestIQDropRuntimeSinceComputesShares(t *testing.T) {
	t0 := time.Unix(1000, 0)
	prev := iqDropRuntime{at: t0, totalCPU: 100, idleCPU: 60, gcCPU: 1, gcCycles: 10}
	cur := iqDropRuntime{at: t0.Add(time.Second), totalCPU: 104, idleCPU: 61, gcCPU: 2, gcCycles: 13}
	d := cur.since(prev)
	if d.WindowSeconds != 1 || d.CPUBusyPct != 75 || d.GCCPUPct != 25 || d.GCCycles != 3 {
		t.Errorf("delta = %+v, want window 1 s, busy 75 %%, gc 25 %%, 3 cycles", d)
	}
	// No GC finished in the window: the CPU counters did not move, so the
	// shares stay 0 rather than reading as an idle process.
	stale := iqDropRuntime{at: t0.Add(2 * time.Second), totalCPU: 104, idleCPU: 61, gcCPU: 2, gcCycles: 13}
	if d := stale.since(cur); d.CPUBusyPct != 0 || d.GCCPUPct != 0 || d.WindowSeconds != 1 {
		t.Errorf("stale CPU counters: %+v", d)
	}
}

// TestIQDropWarnCarriesRuntimeEvidence drives the real observer: the
// second WARN for an SDR must report the GC cycles forced between the two,
// so a field log can tell a GC stall from a slow consumer (#1321).
func TestIQDropWarnCarriesRuntimeEvidence(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	d := &Daemon{}
	d.installIQDropObserver(log)
	t.Cleanup(func() { sdr.SetIQDropObserver(nil) })

	info := sdr.Info{Driver: "test", Serial: "iqdrop-runtime-test"}
	sdr.NotifyIQDrop(info)
	runtime.GC()
	runtime.GC()
	time.Sleep(1100 * time.Millisecond) // past the 1 s WARN throttle
	sdr.NotifyIQDrop(info)

	lines := regexp.MustCompile(`(?m)^.*dropping live IQ chunks.*$`).FindAllString(buf.String(), -1)
	if len(lines) != 2 {
		t.Fatalf("got %d drop WARNs, want 2:\n%s", len(lines), buf.String())
	}
	field := func(line, key string) float64 {
		m := regexp.MustCompile(`\b` + key + `=([0-9.]+)`).FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("%s missing from %q", key, line)
		}
		v, _ := strconv.ParseFloat(m[1], 64)
		return v
	}
	for _, k := range []string{"cpu_busy_pct", "gc_cpu_pct", "gc_pause_max_ms", "sched_wait_p99_ms", "sched_wait_max_ms"} {
		field(lines[1], k)
	}
	if n := field(lines[1], "gc_cycles"); n < 2 {
		t.Errorf("gc_cycles = %v, want ≥ 2 (two forced GCs in the window)", n)
	}
	if w := field(lines[1], "window_s"); w < 1 {
		t.Errorf("window_s = %v, want ≥ 1", w)
	}
	if w := field(lines[0], "window_s"); w != 0 {
		t.Errorf("first WARN window_s = %v, want 0 (no previous snapshot)", w)
	}
}
