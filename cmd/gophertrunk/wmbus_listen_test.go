package main

import (
	"context"
	"encoding/hex"
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/radio/wmbus"
)

// The live loop (#1256): with nothing on screen for the minutes between a
// meter's telegrams, an operator could not tell a working decoder from a
// dead one, and the live -save-frames run said nothing until Ctrl-C. Driven
// here with RTL-sized chunks, it prints a progress line every -status of
// stream time with the CRC-valid / CRC-failed split and the SDR's drops,
// warns once per new drop count, announces each snippet as it is saved, and
// reports a stream that closed on its own.
func TestListenWMBusReportsProgressDropsAndSaves(t *testing.T) {
	const rate = 1_600_000.0
	r := rand.New(rand.NewSource(3))
	noise := func(n int) []complex64 {
		out := make([]complex64, n)
		for i := range out {
			out[i] = complex(float32(r.NormFloat64()*0.03), float32(r.NormFloat64()*0.03))
		}
		return out
	}
	data, _ := hex.DecodeString(wmbusQcaloric)
	iq := noise(int(0.4 * rate))
	iq = append(iq, wmbus.Synthesize(data, wmbus.SynthOptions{Mode: wmbus.ModeT1, SampleRateHz: rate, NoiseRMS: 0.03, Seed: 9})...)
	iq = append(iq, noise(int(1.7*rate))...)

	saver, err := newWMBusSnippetSaver(t.TempDir(), rate, wmbus.ChannelHz, 0)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	saver.onSaved = func(p string) { lines = append(lines, "wmbus: saved "+p) }
	rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: rate, OnFrame: func(f wmbus.Frame, info wmbus.FrameInfo) {
		tg, _ := wmbus.ParseTelegram(f)
		saver.onFrame("t", f, tg, info)
	}})
	if err != nil {
		t.Fatal(err)
	}

	stream := make(chan []complex64, len(iq)/8192+1)
	for len(iq) > 0 {
		n := min(len(iq), 8192)
		stream <- iq[:n]
		iq = iq[n:]
	}
	close(stream)
	var drops uint64
	chunks := 0
	ended := listenWMBus(context.Background(), stream, rx, saver, wmbusListenOptions{
		RateHz: rate, Status: 500 * time.Millisecond,
		Drops: func() uint64 {
			chunks++
			if chunks > 2 {
				drops = 3 // the SDR drops chunks after the first two lines
			}
			return drops
		},
		Statusf: func(l string) { lines = append(lines, l) },
	})
	if !ended {
		t.Error("a stream that closed on its own was not reported as ended")
	}
	text := strings.Join(lines, "\n")
	t.Log(text)
	if n := strings.Count(text, "wmbus: listening "); n != 4 {
		t.Errorf("%d progress lines over ~2.1 s at a 0.5 s -status, want 4", n)
	}
	if !strings.Contains(text, "wmbus: stopped after 2s: 1 frames (1 CRC-valid, 0 CRC-failed)") {
		t.Error("the final summary does not count the telegram")
	}
	if n := strings.Count(text, "dropped 3 IQ chunk(s)"); n != 1 {
		t.Errorf("drop warning printed %d times, want once for the new count", n)
	}
	if !strings.Contains(text, "3 dropped chunks") {
		t.Error("progress lines do not carry the drop count")
	}
	if strings.Count(text, "wmbus: saved ") != 1 || saver.saved != 1 {
		t.Errorf("saved %d snippets, want the one telegram announced as it was written", saver.saved)
	}
}

// The 1 Nov 2026 walk (#1256): the phone's logs showed 30 s of stream time
// taking 3–4 minutes of wall time while 40 000 chunks were dropped, i.e.
// the decoder kept up with ~13 % of real time and ~10 of 21 minutes went
// unheard, but the warning only said "the decoder is not keeping up … try
// -rate". With the wall clock, a progress line carries the time of day (so
// it lines up with the telegram timestamps) and the drop warning says how
// much of real time the decoder got and what starves it.
func TestListenWMBusDropWarningMeasuresRealTime(t *testing.T) {
	const rate = 1_600_000.0
	rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: rate})
	if err != nil {
		t.Fatal(err)
	}
	// 30 s of stream time (silence) in 3m44s of wall time, as on the walk.
	const chunk = 160_000 // 0.1 s
	stream := make(chan []complex64, 300)
	for i := 0; i < 300; i++ {
		stream <- make([]complex64, chunk)
	}
	close(stream)
	wall := time.Date(2026, 11, 1, 13, 8, 0, 0, time.UTC)
	var drops uint64
	var lines []string
	listenWMBus(context.Background(), stream, rx, nil, wmbusListenOptions{
		RateHz: rate, Status: 30 * time.Second,
		Drops: func() uint64 { return drops },
		Clock: func() time.Time {
			// Each call moves the clock as the decoder crawls: 224 s per
			// 30 s of IQ, with 43 805 chunks of 5.12 ms dropped meanwhile.
			wall = wall.Add(224 * time.Second)
			drops += 43_805
			return wall
		},
		Statusf: func(l string) { lines = append(lines, l) },
	})
	text := strings.Join(lines, "\n")
	t.Log(text)
	if !strings.Contains(text, "wmbus: listening 30s (13:15:28): ") {
		t.Error("the progress line does not carry the time of day")
	}
	for _, want := range []string{"13% of real time", "about 3m14s of air went unheard", "termux-wake-lock"} {
		if !strings.Contains(text, want) {
			t.Errorf("drop warning lacks %q", want)
		}
	}
	if strings.Contains(text, "-rate") {
		t.Error("a starved decoder is not a -rate problem; the warning must not point there")
	}
}

// A reopened stream continues the progress lines where they were instead of
// printing one per chunk until it catches up with the receiver's sample count.
func TestListenWMBusContinuesProgressAfterReopen(t *testing.T) {
	const rate = 1_600_000.0
	rx, err := wmbus.NewReceiver(wmbus.ReceiverOptions{SampleRateHz: rate})
	if err != nil {
		t.Fatal(err)
	}
	feed := func(seconds float64) <-chan []complex64 {
		n := int(seconds * 10)
		ch := make(chan []complex64, n)
		for i := 0; i < n; i++ {
			ch <- make([]complex64, 160_000)
		}
		close(ch)
		return ch
	}
	var lines []string
	o := wmbusListenOptions{RateHz: rate, Status: time.Second, Reconnect: true,
		Statusf: func(l string) { lines = append(lines, l) }}
	if !listenWMBus(context.Background(), feed(3.05), rx, nil, o) {
		t.Fatal("first stream did not end")
	}
	if strings.Contains(strings.Join(lines, "\n"), "stopped after") {
		t.Error("a lost SDR about to be reopened printed the final summary")
	}
	lines = nil
	listenWMBus(context.Background(), feed(2), rx, nil, o)
	if n := strings.Count(strings.Join(lines, "\n"), "wmbus: listening "); n != 2 {
		t.Errorf("%d progress lines over 2 s of the reopened stream at a 1 s -status, want 2:\n%s", n, strings.Join(lines, "\n"))
	}
}

// wmbusReopen retries until the SDR is back, names the reason while it waits,
// and gives up only when the run is stopped.
func TestWMBusReopenRetriesUntilBackOrStopped(t *testing.T) {
	var lines []string
	o := wmbusReopenOptions{Retry: time.Millisecond, Note: time.Nanosecond, Statusf: func(l string) { lines = append(lines, l) }}
	tries := 0
	v, ok := wmbusReopen(context.Background(), func() (int, error) {
		tries++
		if tries < 3 {
			return 0, errors.New("capture: no SDRs found")
		}
		return 42, nil
	}, o)
	if !ok || v != 42 || tries != 3 {
		t.Fatalf("got %d, %v after %d tries; want the SDR back on the third", v, ok, tries)
	}
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "still waiting for the SDR") || !strings.Contains(text, "no SDRs found") || !strings.Contains(text, "SDR back after") {
		t.Errorf("progress lines:\n%s", text)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := wmbusReopen(ctx, func() (int, error) { return 0, errors.New("gone") }, o); ok {
		t.Error("a stopped run kept waiting for the SDR")
	}
}
