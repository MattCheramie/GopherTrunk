package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

type stuckCloseDevice struct {
	sdr.Device
	release chan struct{}
}

func (d *stuckCloseDevice) Close() error { <-d.release; return nil }

// TestCloseSDRBoundedGivesUpOnStuckClose: a close that never returns (a USB
// transfer stalled on a phone, #1256) used to keep wmbus alive after Ctrl-C,
// holding the dongle. The bounded close must return after its timeout and say
// why.
func TestCloseSDRBoundedGivesUpOnStuckClose(t *testing.T) {
	dev := &stuckCloseDevice{release: make(chan struct{})}
	defer close(dev.release)
	var out bytes.Buffer
	start := time.Now()
	closeSDRBounded(dev, 50*time.Millisecond, &out, "wmbus")
	if el := time.Since(start); el > 2*time.Second {
		t.Fatalf("closeSDRBounded blocked %s on a stuck close", el)
	}
	if !strings.Contains(out.String(), "wmbus: the SDR did not close within 50ms") {
		t.Fatalf("output = %q", out.String())
	}
}

// TestCloseSDRBoundedQuietOnNormalClose: a close that completes says nothing.
func TestCloseSDRBoundedQuietOnNormalClose(t *testing.T) {
	dev := &stuckCloseDevice{release: make(chan struct{})}
	close(dev.release)
	var out bytes.Buffer
	closeSDRBounded(dev, time.Second, &out, "wmbus")
	if out.Len() != 0 {
		t.Fatalf("output = %q, want none", out.String())
	}
}

// TestStdinStopModeCoversSuWithoutTerminal: the #1256 reporter's root
// process, started through su, has stdin on the terminal but no controlling
// terminal (ps TTY "?"), so Ctrl-C reaches su and never gophertrunk. It must
// watch stdin for a stop and say so. With a controlling terminal Ctrl-C
// works, and reading the terminal from a background job would stop it with
// SIGTTIN, so it must not read; a pipe is safe either way; /dev/null under a
// daemon or cron job is watched (harmlessly) but gets no hint.
func TestStdinStopModeCoversSuWithoutTerminal(t *testing.T) {
	for _, c := range []struct {
		name                string
		tty, pipe, ctty     bool
		wantWatch, wantHint bool
	}{
		{"su: terminal, no controlling terminal", true, false, false, true, true},
		{"normal shell: controlling terminal", true, false, true, false, false},
		{"piped stdin in a shell", false, true, true, true, false},
		{"piped stdin, no controlling terminal", false, true, false, true, true},
		{"daemon: /dev/null, no controlling terminal", false, false, false, true, false},
		{"/dev/null in a shell", false, false, true, false, false},
	} {
		w, h := stdinStopMode(c.tty, c.pipe, c.ctty)
		if w != c.wantWatch || h != c.wantHint {
			t.Errorf("%s: watch, hint = %v, %v; want %v, %v", c.name, w, h, c.wantWatch, c.wantHint)
		}
	}
}

// TestWatchStopInput: "q" + Enter (or quit / stop, any case) and a raw
// Ctrl-C byte stop; other lines and the end of the input do not.
func TestWatchStopInput(t *testing.T) {
	for _, c := range []struct {
		in   string
		stop bool
	}{
		{"q\n", true},
		{"  Quit \n", true},
		{"hello\nSTOP\n", true},
		{"\x03", true},
		{"query\n", false},
		{"hello\n", false},
		{"", false},
		{"q", true}, // the last line before EOF, no newline
	} {
		stopped := false
		watchStopInput(strings.NewReader(c.in), func() { stopped = true })
		if stopped != c.stop {
			t.Errorf("input %q: stopped = %v, want %v", c.in, stopped, c.stop)
		}
	}
}

// TestWatchStopInputRawCtrlCNeedsNoEnter: a terminal in raw mode passes
// Ctrl-C through as a lone 0x03 byte with no newline after it; it must stop
// at once, not wait for an Enter that never comes.
func TestWatchStopInputRawCtrlCNeedsNoEnter(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	done := make(chan struct{})
	go watchStopInput(pr, func() { close(done) })
	if _, err := pw.Write([]byte{0x03}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a raw Ctrl-C byte did not stop the watch")
	}
}

// Once the stop watcher reads stdin, a CLI reporter must not prompt on it
// (#1256: "y" to "Show full diagnostic trace?" printed nothing, because the
// watcher took the line).
func TestReporterDoesNotPromptOnceStopWatcherOwnsStdin(t *testing.T) {
	prev := stdinStopWatched.Load()
	defer stdinStopWatched.Store(prev)
	stdinStopWatched.Store(false)
	rep := newReporter("wmbus")
	if rep.PromptAllowed == nil || !rep.PromptAllowed() {
		t.Fatal("a reporter must still prompt while no watcher reads stdin")
	}
	startStopWatcher(strings.NewReader(""), func() {})
	if rep.PromptAllowed() {
		t.Error("the reporter still prompts on a stdin the stop watcher reads")
	}
}
