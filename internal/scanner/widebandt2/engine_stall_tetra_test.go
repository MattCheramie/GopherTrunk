package widebandt2

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestEngineTETRAPumpStalls is the #1321 instrument for the residual "dropping
// live IQ chunks" on an Airspy Mini with several TETRA cells. It is skipped
// unless GT_WB_STALL_SECONDS is set.
//
// An SDR driver sheds live IQ when the wideband engine's pump (one goroutine:
// shared decimator, then every tap's DDC + receiver on the worker pool, waited
// for per chunk) falls further behind the USB cadence than the driver's
// output buffer holds. Mean CPU says nothing about that: the pump can average
// 0.3x real time and still run a backlog of tens of milliseconds when a few
// slow chunks cluster. So this test times bank.Process per chunk, at the
// Airspy's URB size (4096 IQ samples = 1.37 ms at 3 MS/s), and replays the
// timings through a queue fed at the USB cadence to report the worst backlog —
// scaled by GT_WB_STALL_SLOWDOWN factors to stand in for a slower CPU — against
// the two Airspy buffer sizes: 8 chunks (v1.3.6 and earlier, ~11 ms at 3 MS/s)
// and 400 ms (#1335).
//
// Measured on a 2.8 GHz Xeon (4 cores), 4 hunting cells, 60 s of noise:
// p50 0.34 ms, p99 4.0 ms, max 24 ms per chunk. The backlog passed 8 chunks at
// every slowdown (10 chunks at 1x, 791 at 2.5x) and never passed the 400 ms
// buffer up to 3x (worst 284 ms). That is the v1.3.6 symptom — drops spread
// through the run with most cores idle, even with no cell locked — and the
// margin #1335 buys.
//
// Knobs: GT_WB_STALL_CELLS (default 4, cells 37.5 kHz apart around the
// centre), GT_WB_BENCH_RATE (default 3000000), GT_WB_STALL_SLOWDOWN (comma
// list, default "1,1.5,2,2.5,3"), GT_WB_STALL_SIGNAL=1 to loop the committed
// 2 s TETRA control-channel capture (ccdecoder testdata) onto the second cell,
// so locks, resyncs and decodes are timed too (each loop seam forces a resync).
// Run it on the machine that drops: the per-chunk times are what matter, and
// `go test` reports them directly.
func TestEngineTETRAPumpStalls(t *testing.T) {
	secs := atoiOr(os.Getenv("GT_WB_STALL_SECONDS"), 0)
	if secs <= 0 {
		t.Skip("set GT_WB_STALL_SECONDS")
	}
	rate := atoiOr(os.Getenv("GT_WB_BENCH_RATE"), 3_000_000)
	cells := atoiOr(os.Getenv("GT_WB_STALL_CELLS"), 4)
	chunk := atoiOr(os.Getenv("GT_WB_STALL_CHUNK"), 4096) // one Airspy URB of 16 KiB real samples, as IQ
	center := uint32(467_912_500)

	// Cells alternate either side of the centre, 37.5 kHz apart (all within
	// the reporter's ±90 kHz for 4 cells).
	var chs []ChannelConfig
	var sys []trunking.System
	for i := 0; i < cells; i++ {
		step := int64((i + 1) / 2)
		if i%2 == 1 {
			step = -step
		}
		f := uint32(int64(center) + step*37_500)
		name := fmt.Sprintf("c%d", i)
		chs = append(chs, ChannelConfig{FrequencyHz: f, SystemName: name})
		sys = append(sys, tetraSystem(name, f))
	}

	// One period of input: noise, optionally plus the CC capture on cell 1.
	period := buildStallInput(t, rate, chunk, chs, center)

	bus := events.NewBus(1024)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       newMockDevice(nil),
		Bus:          bus,
		SampleRateHz: uint32(rate),
		CenterFreqHz: center,
		Channels:     chs,
		Systems:      sys,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := e.bank.(interface{ Close() }); ok {
		defer c.Close()
	}

	n := int(int64(secs) * int64(rate) / int64(chunk))
	lat := make([]time.Duration, n)
	type ev struct {
		at   int
		kind events.Kind
	}
	var evs []ev
	start := time.Now()
	for i := 0; i < n; i++ {
		c := period[i%len(period)]
		t0 := time.Now()
		e.widebandPwr.Add(c)
		e.foldInputClip(c)
		e.bank.Process(c)
		e.maybeLogDiagnostics(e.now())
		lat[i] = time.Since(t0)
		// Publish enqueues synchronously, so draining here tags each event
		// with the chunk that produced it.
		for drained := false; !drained; {
			select {
			case x := <-sub.C:
				if x.Kind == events.KindCCLocked || x.Kind == events.KindCCLost {
					evs = append(evs, ev{i, x.Kind})
				}
			default:
				drained = true
			}
		}
	}
	wall := time.Since(start)

	chunkDur := time.Duration(float64(chunk) / float64(rate) * float64(time.Second))
	sorted := append([]time.Duration(nil), lat...)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a] < sorted[b] })
	pct := func(p float64) time.Duration { return sorted[min(n-1, int(p*float64(n)))] }
	t.Logf("%d cells, %d S/s, %d s (%d chunks of %v): %.2fx real time; per chunk p50=%v p99=%v p99.9=%v max=%v",
		cells, rate, secs, n, chunkDur, wall.Seconds()/float64(secs), pct(0.5), pct(0.99), pct(0.999), sorted[n-1])
	locks, lost := 0, 0
	for _, x := range evs {
		if x.kind == events.KindCCLocked {
			locks++
		} else {
			lost++
		}
	}
	t.Logf("cc.locked=%d cc.lost=%d", locks, lost)

	oldDepth := 8
	newDepth := int(math.Ceil(0.4 * float64(rate) / float64(chunk)))
	for _, k := range parseSlowdowns(os.Getenv("GT_WB_STALL_SLOWDOWN")) {
		worst, overOld, overNew := stallBacklog(lat, chunkDur, k, oldDepth, newDepth)
		t.Logf("slowdown x%.1f: worst backlog %d chunks (%.0f ms); chunks shed by an %d-deep buffer=%d, by a %d-deep (400 ms) buffer=%d",
			k, worst, float64(worst)*chunkDur.Seconds()*1000, oldDepth, overOld, newDepth, overNew)
	}

	// The slowest chunks, with the nearest lock/loss event.
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return lat[idx[a]] > lat[idx[b]] })
	for _, i := range idx[:min(10, n)] {
		near := ""
		for _, x := range evs {
			if d := i - x.at; d >= 0 && d < int(time.Second/chunkDur) {
				near = fmt.Sprintf(" (%v after %s)", time.Duration(d)*chunkDur, x.kind)
			}
		}
		t.Logf("  stream t=%.2f s: %v%s", float64(i)*chunkDur.Seconds(), lat[i], near)
	}
}

// stallBacklog replays per-chunk service times (scaled by k) through a FIFO
// fed one chunk per chunkDur and returns the worst queue depth and how many
// chunks a buffer of each depth would have shed. The engine's own primary
// hand-off (iqtap's 2 chunks) is counted as part of either buffer.
func stallBacklog(lat []time.Duration, chunkDur time.Duration, k float64, depthA, depthB int) (worst, shedA, shedB int) {
	var finish time.Duration
	for i, d := range lat {
		arr := time.Duration(i) * chunkDur
		if finish < arr {
			finish = arr
		}
		finish += time.Duration(float64(d) * k)
		q := int((finish - arr) / chunkDur)
		worst = max(worst, q)
		if q > depthA+2 {
			shedA++
		}
		if q > depthB+2 {
			shedB++
		}
	}
	return worst, shedA, shedB
}

func parseSlowdowns(s string) []float64 {
	if s == "" {
		return []float64{1, 1.5, 2, 2.5, 3}
	}
	var out []float64
	for _, f := range strings.Split(s, ",") {
		var v float64
		if _, err := fmt.Sscanf(strings.TrimSpace(f), "%g", &v); err == nil && v > 0 {
			out = append(out, v)
		}
	}
	return out
}

// buildStallInput returns one looping period of wideband input chunks: white
// noise, plus (GT_WB_STALL_SIGNAL=1) the committed 144 kHz TETRA CC capture
// interpolated to the capture rate and mixed onto the second cell.
func buildStallInput(t *testing.T, rate, chunk int, chs []ChannelConfig, center uint32) [][]complex64 {
	t.Helper()
	rng := rand.New(rand.NewSource(1))
	periodSamples := 257 * chunk // noise only: a prime count of distinct chunks
	var sig []complex64
	if os.Getenv("GT_WB_STALL_SIGNAL") == "1" && len(chs) > 1 {
		raw, err := os.ReadFile("../ccdecoder/testdata/tetra_cc_sync_loss_2s_144k.cs16")
		if err != nil {
			t.Fatalf("GT_WB_STALL_SIGNAL: %v", err)
		}
		in := make([]complex64, len(raw)/4)
		for i := range in {
			re := int16(binary.LittleEndian.Uint16(raw[4*i:]))
			im := int16(binary.LittleEndian.Uint16(raw[4*i+2:]))
			in[i] = complex(float32(re)/32768, float32(im)/32768)
		}
		g := gcd(rate, 144_000)
		rs := dsp.NewResampler(rate/g, 144_000/g, 16, 8)
		sig = rs.Process(nil, in)
		off := float64(int64(chs[1].FrequencyHz) - int64(center))
		for i := range sig {
			ph := 2 * math.Pi * off * float64(i) / float64(rate)
			sig[i] *= complex(float32(math.Cos(ph)), float32(math.Sin(ph)))
		}
		periodSamples = len(sig) / chunk * chunk
	}
	out := make([][]complex64, periodSamples/chunk)
	for c := range out {
		buf := make([]complex64, chunk)
		for j := range buf {
			v := complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
			if sig != nil {
				v += sig[c*chunk+j]
			}
			buf[j] = v
		}
		out[c] = buf
	}
	return out
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
