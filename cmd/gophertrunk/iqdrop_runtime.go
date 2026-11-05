package main

import (
	"math"
	"runtime/metrics"
	"time"
)

// iqDropRuntime is a snapshot of the Go runtime counters that tell apart
// the causes of a live IQ drop. A drop means the consumer of an SDR's
// stream stopped draining it for longer than the driver buffer holds, and
// on a busy multi-cell rig there are three candidates: the decode work
// itself is too slow, the process is short of CPU (the pump goroutine was
// runnable but not running), or the garbage collector stalled it. #1321's
// reporter saw a 10 s burst of drops right after a TETRA voice call ended,
// and the drop line alone could not say which it was. Each drop WARN now
// carries the deltas since the previous WARN for that SDR.
//
// The /cpu/classes counters are refreshed by the runtime only at the end of
// a GC cycle, so the CPU shares describe the window up to the last GC in
// it; a daemon streaming IQ collects many times a second, so that is close
// to the WARN window. They are left at 0 when no GC finished in between.
type iqDropRuntime struct {
	at       time.Time
	totalCPU float64 // /cpu/classes/total:cpu-seconds (GOMAXPROCS × wall)
	idleCPU  float64 // /cpu/classes/idle:cpu-seconds
	gcCPU    float64 // /cpu/classes/gc/total:cpu-seconds
	gcCycles uint64  // /gc/cycles/total:gc-cycles
	schedLat *metrics.Float64Histogram
	gcPauses *metrics.Float64Histogram
}

var iqDropRuntimeMetrics = []string{
	"/cpu/classes/total:cpu-seconds",
	"/cpu/classes/idle:cpu-seconds",
	"/cpu/classes/gc/total:cpu-seconds",
	"/gc/cycles/total:gc-cycles",
	"/sched/latencies:seconds",
	"/sched/pauses/total/gc:seconds",
}

// readIQDropRuntime samples the runtime. It is called only when a drop
// WARN is about to be logged (at most once a second per SDR), never per
// dropped chunk.
func readIQDropRuntime() iqDropRuntime {
	s := make([]metrics.Sample, len(iqDropRuntimeMetrics))
	for i, n := range iqDropRuntimeMetrics {
		s[i].Name = n
	}
	metrics.Read(s)
	r := iqDropRuntime{at: time.Now()}
	f := func(i int) float64 {
		if s[i].Value.Kind() == metrics.KindFloat64 {
			return s[i].Value.Float64()
		}
		return 0
	}
	h := func(i int) *metrics.Float64Histogram {
		if s[i].Value.Kind() == metrics.KindFloat64Histogram {
			return s[i].Value.Float64Histogram()
		}
		return nil
	}
	r.totalCPU, r.idleCPU, r.gcCPU = f(0), f(1), f(2)
	if s[3].Value.Kind() == metrics.KindUint64 {
		r.gcCycles = s[3].Value.Uint64()
	}
	r.schedLat, r.gcPauses = h(4), h(5)
	return r
}

// iqDropRuntimeDelta is what changed between two snapshots.
type iqDropRuntimeDelta struct {
	CPUBusyPct    float64 // share of all cores' time spent not idle
	GCCPUPct      float64 // share of all cores' time spent in the GC
	GCCycles      uint64
	GCPauseMaxMs  float64 // longest stop-the-world GC pause (bucket upper bound)
	SchedP99Ms    float64 // 99th percentile time a runnable goroutine waited to run
	SchedMaxMs    float64 // longest such wait (bucket upper bound)
	WindowSeconds float64 // wall time since the previous snapshot
}

// since returns cur − prev. With no previous snapshot it reports nothing
// (zero window) rather than the whole process lifetime.
func (cur iqDropRuntime) since(prev iqDropRuntime) iqDropRuntimeDelta {
	var d iqDropRuntimeDelta
	if prev.at.IsZero() || !cur.at.After(prev.at) {
		return d
	}
	d.WindowSeconds = cur.at.Sub(prev.at).Seconds()
	if total := cur.totalCPU - prev.totalCPU; total > 0 {
		d.CPUBusyPct = clampPct(100 * (1 - (cur.idleCPU-prev.idleCPU)/total))
		d.GCCPUPct = clampPct(100 * (cur.gcCPU - prev.gcCPU) / total)
	}
	if cur.gcCycles >= prev.gcCycles {
		d.GCCycles = cur.gcCycles - prev.gcCycles
	}
	if p, m, ok := histDeltaStats(cur.schedLat, prev.schedLat, 0.99); ok {
		d.SchedP99Ms, d.SchedMaxMs = p*1e3, m*1e3
	}
	if _, m, ok := histDeltaStats(cur.gcPauses, prev.gcPauses, 1); ok {
		d.GCPauseMaxMs = m * 1e3
	}
	return d
}

func clampPct(v float64) float64 {
	switch {
	case math.IsNaN(v) || v < 0:
		return 0
	case v > 100:
		return 100
	}
	return v
}

// histDeltaStats subtracts two snapshots of the same cumulative histogram
// and returns the q-quantile and the maximum of the new observations, each
// as the upper bound of the bucket it falls in (the runtime only gives
// bucket counts). ok is false when nothing was observed in between or the
// bucket layouts differ.
func histDeltaStats(cur, prev *metrics.Float64Histogram, q float64) (quantile, max float64, ok bool) {
	if cur == nil || prev == nil || len(cur.Counts) != len(prev.Counts) || len(cur.Buckets) != len(cur.Counts)+1 {
		return 0, 0, false
	}
	var n uint64
	delta := make([]uint64, len(cur.Counts))
	for i := range cur.Counts {
		if cur.Counts[i] > prev.Counts[i] {
			delta[i] = cur.Counts[i] - prev.Counts[i]
			n += delta[i]
		}
	}
	if n == 0 {
		return 0, 0, false
	}
	upper := func(i int) float64 {
		// The last bucket's upper bound can be +Inf; report its lower bound
		// instead so the log line stays a number.
		if u := cur.Buckets[i+1]; !math.IsInf(u, 1) {
			return u
		}
		return cur.Buckets[i]
	}
	want := uint64(math.Ceil(q * float64(n)))
	if want == 0 {
		want = 1
	}
	var seen uint64
	quantile = math.NaN()
	for i, c := range delta {
		if c == 0 {
			continue
		}
		seen += c
		if math.IsNaN(quantile) && seen >= want {
			quantile = upper(i)
		}
		max = upper(i)
	}
	return quantile, max, true
}

// round1 rounds to one decimal place for a log field.
func round1(v float64) float64 { return math.Round(v*10) / 10 }
