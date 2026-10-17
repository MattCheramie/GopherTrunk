package main

import (
	"testing"
	"time"
)

// A live `-save-frames` run must keep up with the SDR (#1256). The saver's
// history was a slice trimmed by copying the whole 0.5 s window down on
// every chunk, so feeding it 8192-sample RTL chunks at 1.6 MS/s cost ~37% of
// real time on a desktop core (measured 3.7 s per 10 s of IQ) — past real
// time on the reporter's phone, which then dropped chunks and decoded
// nothing. The bound is generous (10% of real time, even under -race); the
// ring feeds 20 s of IQ in milliseconds.
func TestWMBusSnippetSaverFeedKeepsUpWithLiveRate(t *testing.T) {
	const rate = 1_600_000
	s, err := newWMBusSnippetSaver(t.TempDir(), rate, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	chunk := make([]complex64, 8192)
	const seconds = 20
	n := seconds * rate / len(chunk)
	start := time.Now()
	for i := 0; i < n; i++ {
		s.feed(chunk)
	}
	if el := time.Since(start); el > seconds*time.Second/10 {
		t.Fatalf("feeding %d s of IQ took %v; the saver cannot keep up with a live SDR", seconds, el)
	}
}

// The ring hands back exactly the samples fed, in order, across a wrap and
// after a chunk larger than the whole window.
func TestWMBusSnippetSaverRingSpan(t *testing.T) {
	s, err := newWMBusSnippetSaver(t.TempDir(), 20, 0, 0) // a 10-sample window
	if err != nil {
		t.Fatal(err)
	}
	var next int64
	feed := func(k int) {
		iq := make([]complex64, k)
		for i := range iq {
			iq[i] = complex(float32(next), 0)
			next++
		}
		s.feed(iq)
	}
	check := func(start, end int64) {
		t.Helper()
		got := s.span(start, end)
		if int64(len(got)) != end-start {
			t.Fatalf("span(%d,%d) has %d samples", start, end, len(got))
		}
		for i, v := range got {
			if int64(real(v)) != start+int64(i) {
				t.Fatalf("span(%d,%d)[%d] = %v, want %d", start, end, i, real(v), start+int64(i))
			}
		}
	}
	feed(7)
	check(0, 7)
	feed(6) // wraps: holds 3..12
	if s.histStart() != 3 || s.histEnd != 13 {
		t.Fatalf("held [%d,%d), want [3,13)", s.histStart(), s.histEnd)
	}
	check(3, 13)
	check(8, 12)
	feed(25) // larger than the window: holds the newest 10
	if s.histStart() != 28 || s.histEnd != 38 {
		t.Fatalf("held [%d,%d), want [28,38)", s.histStart(), s.histEnd)
	}
	check(28, 38)
	check(30, 30)
}
