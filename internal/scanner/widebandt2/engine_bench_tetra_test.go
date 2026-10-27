package widebandt2

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestEngineDualTETRA200kThroughput is a CPU-budget instrument, skipped unless
// GT_WB_BENCH_SECONDS is set: it pumps that many seconds of noise through the
// wideband engine with two TETRA DDC channels (the 10 Sep dual-TETRA X310
// config that reported host overruns) and logs the ratio of wall-clock
// processing time to stream time. GT_WB_BENCH_RATE sets the capture rate
// (default 200 kS/s; 3_000_000 is the #1321 Airspy Mini / Pi 4 case),
// GT_WB_BENCH_PROFILE writes a CPU profile. Before the RM(30,14) codebook +
// FIR window fixes this measured 0.44x real time on a 2.1 GHz Xeon core (65%
// of it re-encoding the AACH codebook per slot); after, 0.12x. At 3 MS/s it
// measured 0.39x on a 2.8 GHz Xeon before #1321 (two full-rate 3 MS/s →
// 144 kHz resamplers on one goroutine) and 0.18x after (shared decimation to
// 375 kHz, taps on two workers); the 200 kS/s case went 0.12x → 0.08x.
func TestEngineDualTETRA200kThroughput(t *testing.T) {
	secs := 20
	if v := os.Getenv("GT_WB_BENCH_SECONDS"); v == "" {
		t.Skip("set GT_WB_BENCH_SECONDS")
	} else {
		secs = atoiOr(v, 20)
	}
	rate := atoiOr(os.Getenv("GT_WB_BENCH_RATE"), 200_000)
	chunk := rate / 200 // ~5 ms chunks, the live USB cadence
	rng := rand.New(rand.NewSource(1))
	n := secs * rate / chunk
	chunks := make([][]complex64, n)
	for i := range chunks {
		c := make([]complex64, chunk)
		for j := range c {
			c[j] = complex(float32(rng.NormFloat64()*0.01), float32(rng.NormFloat64()*0.01))
		}
		chunks[i] = c
	}
	dev := newMockDevice(chunks)
	bus := events.NewBus(64)
	defer bus.Close()
	center := uint32(467_912_500)
	e, err := New(Options{
		Log:          slog.New(slog.NewTextHandler(discardWriter{}, nil)),
		Device:       dev,
		Bus:          bus,
		SampleRateHz: uint32(rate),
		CenterFreqHz: center,
		Channels: []ChannelConfig{
			{FrequencyHz: 467_912_500, SystemName: "a"},
			{FrequencyHz: 467_875_000, SystemName: "b"},
		},
		Systems: []trunking.System{tetraSystem("a", 467_912_500), tetraSystem("b", 467_875_000)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if p := os.Getenv("GT_WB_BENCH_PROFILE"); p != "" {
		f, _ := os.Create(p)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	start := time.Now()
	_ = e.Run(ctx) // the mock closes the stream when its chunks run out
	el := time.Since(start)
	t.Logf("processed %d s of %d S/s dual-TETRA IQ in %v (%.2fx real time, strategy %s)", secs, rate, el, el.Seconds()/float64(secs), e.Strategy())
}

func atoiOr(s string, d int) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return d
		}
		n = n*10 + int(c-'0')
	}
	return n
}
