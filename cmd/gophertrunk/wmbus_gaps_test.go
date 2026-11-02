package main

import (
	"context"
	"encoding/hex"
	"math/rand"
	"strings"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
)

// wmbusQDSWarmWater is a real C1 telegram from the #1256 1 Nov walk (Qundis
// warm-water meter 52927709, CRC-valid at -24 dBFS), as Frame.Data.
const wmbusQDSWarmWater = "49449344097792521a06780dff5f350082bb00007f0007c113ffff351902003f3c166900005f3a2719020094009400b400ac00a000810082008c00a4006a00ae0092002f046d000f413b"

// runWMBusGapStream plays noise, one C1 telegram and noise through
// listenWMBus the way a starved phone's SDR delivers it: a bounded queue the
// producer keeps full, and, like the RTL-SDR and Airspy drivers, the chunk
// dropAt dropped (the newest) when offered to a full queue. dropAt -1 drops
// nothing; -2 drops two thirds of the way through the telegram. It returns
// the telegram lines and the status lines.
func runWMBusGapStream(t *testing.T, dropAt int) (telegrams, status []string) {
	t.Helper()
	const rate = 1_600_000.0
	const chunk = 1024 // 0.64 ms, so a hole can land mid-telegram
	r := rand.New(rand.NewSource(5))
	noise := func(n int) []complex64 {
		out := make([]complex64, n)
		for i := range out {
			out[i] = complex(float32(r.NormFloat64()*0.02), float32(r.NormFloat64()*0.02))
		}
		return out
	}
	data, _ := hex.DecodeString(wmbusQDSWarmWater)
	iq := noise(int(0.05 * rate))
	tgStart := len(iq)
	tgIQ := wmbus.Synthesize(data, wmbus.SynthOptions{Mode: wmbus.ModeC1, SampleRateHz: rate, NoiseRMS: 0.02, Seed: 4})
	iq = append(iq, tgIQ...)
	iq = append(iq, noise(int(0.05*rate))...)
	if dropAt == -2 {
		// Past the header block, as in the reporter's snippet (header OK,
		// later blocks failed).
		dropAt = (tgStart + 2*len(tgIQ)/3) / chunk
	}
	var chunks [][]complex64
	for len(iq) > 0 {
		c := iq[:min(len(iq), chunk)]
		iq = iq[len(c):]
		chunks = append(chunks, c)
	}

	gaps := &wmbusGapLog{}
	var buf strings.Builder
	rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: rate, OnFrame: func(f wmbus.Frame, info wmbus.FrameInfo) {
		tg, perr := wmbus.ParseTelegram(f)
		writeWMBusTelegram(&buf, "t", tg, perr, f, info, gaps.cutByGap(f, info, rate), false)
	}})
	if err != nil {
		t.Fatal(err)
	}

	// The producer runs in the Drops call, which listenWMBus makes once per
	// chunk it reads, so it is deterministic: each read frees one slot, the
	// producer (faster than the decoder, as on a starved phone) refills the
	// queue and offers the next chunk to a full one. Only dropAt is dropped
	// there; any other chunk waits for room, so dropAt is offered exactly
	// when the queue has just filled, where a real driver drops.
	stream := make(chan []complex64, 8)
	var drops uint64
	next, closed := 0, false
	produce := func() {
		for next < len(chunks) {
			full := len(stream) == cap(stream)
			if next == dropAt {
				if !full {
					break
				}
				drops++
				next++
				continue
			}
			if full {
				break
			}
			stream <- chunks[next]
			next++
		}
		if next == len(chunks) && !closed {
			close(stream)
			closed = true
		}
	}
	produce()
	listenWMBus(context.Background(), stream, rx, nil, wmbusListenOptions{
		RateHz: rate, Gaps: gaps,
		Drops:   func() uint64 { produce(); return drops },
		Statusf: func(l string) { status = append(status, l) },
	})
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(l, "t  ") {
			telegrams = append(telegrams, l)
		}
	}
	return telegrams, status
}

// Issue #1256: a telegram the phone's SDR cut with dropped IQ used to read
// exactly like a meter too weak to decode ("CRC FAILED (blocks 3,4,5 of 5;
// header OK)"). It now says the dropped IQ cut it, and the summary counts
// such failures; the same telegram with nothing dropped decodes, unmarked.
func TestWMBusMarksTelegramCutByDroppedIQ(t *testing.T) {
	clean, _ := runWMBusGapStream(t, -1)
	if len(clean) != 1 || strings.Contains(clean[0], "CRC FAILED") || strings.Contains(clean[0], "dropped IQ") {
		t.Fatalf("with nothing dropped, want one clean telegram, got %q", clean)
	}

	cut, status := runWMBusGapStream(t, -2)
	t.Logf("%q\n%q", cut, status)
	if len(cut) != 1 || !strings.Contains(cut[0], "CRC FAILED") {
		t.Fatalf("with a chunk dropped mid-telegram, want one CRC-failed telegram, got %q", cut)
	}
	if !strings.Contains(cut[0], "cut by dropped IQ") {
		t.Errorf("the CRC failure is not attributed to the dropped IQ: %q", cut[0])
	}
	if !strings.Contains(strings.Join(status, "\n"), "1 dropped chunks; 1 of the CRC failures were cut by dropped IQ") {
		t.Errorf("the summary does not count the cut telegram: %q", status)
	}
}

// A hole well clear of a frame does not mark it, and the position estimate
// follows the queue: the hole sits after the current chunk and everything
// still queued behind it.
func TestWMBusGapLogNearAndPosition(t *testing.T) {
	if got := wmbusGapPosition(10_000, 1024, 3); got != 10_000+4*1024 {
		t.Errorf("hole position %d, want %d", got, 10_000+4*1024)
	}
	g := &wmbusGapLog{}
	g.mark(50_000, 1024)
	for _, c := range []struct {
		start, end int64
		want       bool
	}{
		{40_000, 49_000, true},  // within one chunk before
		{51_000, 60_000, true},  // within one chunk after
		{45_000, 55_000, true},  // straddles it
		{10_000, 40_000, false}, // ends 10 000 samples before
		{60_000, 70_000, false}, // starts 10 000 samples after
	} {
		if got := g.near(c.start, c.end); got != c.want {
			t.Errorf("near(%d, %d) = %v, want %v", c.start, c.end, got, c.want)
		}
	}
	for range wmbusGapKeep + 10 {
		g.mark(1, 1024)
	}
	if len(g.gaps) != wmbusGapKeep {
		t.Errorf("log holds %d holes, want it bounded at %d", len(g.gaps), wmbusGapKeep)
	}
}
