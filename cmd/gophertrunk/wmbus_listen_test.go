package main

import (
	"context"
	"encoding/hex"
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
