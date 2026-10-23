package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// sdrStopSignals end a foreground SDR command (capture, power, wmbus) the way
// Ctrl-C does. SIGTERM is what `kill` / `pkill` send and SIGHUP what a closed
// terminal sends; both used to kill the process without its summary. On
// Termux a root process started through `su` often never sees the terminal's
// Ctrl-C at all (#1256), and `pkill gophertrunk` from a root shell is how it
// gets stopped.
var sdrStopSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}

// sdrStopContext is signal.NotifyContext over sdrStopSignals, plus a stop
// typed on stdin where a signal may never arrive (see stdinStopMode). prog
// prefixes the hint it writes to w when it says how to stop.
func sdrStopContext(prog string, w io.Writer) (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), sdrStopSignals...)
	st, err := os.Stdin.Stat()
	if err != nil {
		return ctx, stop
	}
	pipe := st.Mode()&(os.ModeNamedPipe|os.ModeSocket) != 0
	watch, hint := stdinStopMode(isatty.IsTerminal(os.Stdin.Fd()), pipe, hasControllingTerminal())
	if !watch {
		return ctx, stop
	}
	ctx, cancel := context.WithCancel(ctx)
	go watchStopInput(os.Stdin, cancel)
	if hint {
		fmt.Fprintf(w, "%s: this process has no controlling terminal (started through su?), so Ctrl-C may not reach it; "+
			"type q and Enter to stop, or kill %d\n", prog, os.Getpid())
	}
	return ctx, func() { cancel(); stop() }
}

// stdinStopMode decides whether a foreground SDR command also stops on a
// "q" typed on stdin (watch), and whether to tell the operator so (hint).
//
// The #1256 reporter runs gophertrunk as root through `su` in Termux, and
// Ctrl-C never stopped it: their `ps` shows the process with no terminal
// (TTY "?"). su hands it the terminal's file descriptors but not the
// terminal itself, so the Ctrl-C the terminal turns into SIGINT goes to su,
// and gophertrunk keeps running behind a shell that is now sharing its
// input ("the shell becomes strange after Ctrl-C"). What gophertrunk does
// still get is the typed text, so it reads it, where that is safe:
//
//   - no controlling terminal: reading stdin can never stop the process
//     with SIGTTIN, and Ctrl-C cannot work, so watch and say so;
//   - stdin a pipe or socket: reading is safe too, but Ctrl-C works;
//   - a terminal that is the controlling one: Ctrl-C works, and reading it
//     from a background job (`… &`) would stop the job, so leave it.
//
// The hint is for a terminal or a pipe someone may type into; a daemon or
// cron job with stdin on /dev/null has nobody to tell.
func stdinStopMode(isTTY, isPipe, hasCtty bool) (watch, hint bool) {
	switch {
	case !hasCtty:
		return true, isTTY || isPipe
	case isPipe:
		return true, false
	}
	return false, false
}

// watchStopInput calls stop on a line reading q, quit or stop (any case),
// or on a Ctrl-C byte (a terminal in raw mode passes it through as 0x03).
// It returns at the end of the input without stopping: a closed stdin is
// not a request to stop.
// It reads byte by byte, so a raw Ctrl-C stops at once, without an Enter.
func watchStopInput(r io.Reader, stop func()) {
	br := bufio.NewReader(r)
	var line []byte
	isStop := func() bool {
		switch strings.ToLower(strings.TrimSpace(string(line))) {
		case "q", "quit", "stop":
			return true
		}
		return false
	}
	for {
		b, err := br.ReadByte()
		if err != nil {
			if isStop() {
				stop()
			}
			return
		}
		switch b {
		case 0x03:
			stop()
			return
		case '\n':
			if isStop() {
				stop()
				return
			}
			line = line[:0]
		default:
			if len(line) < 64 {
				line = append(line, b)
			}
		}
	}
}

// sdrCloseTimeout bounds how long a command waits for the SDR to close once
// it has finished.
const sdrCloseTimeout = 5 * time.Second

// closeSDRBounded closes dev but waits at most timeout. Closing an RTL-SDR
// stops its bulk-IN transfers and puts the tuner in standby over USB control
// transfers; on a phone's USB stack either can stall, and a stalled close
// left the process alive after Ctrl-C, holding the dongle, so the next run
// failed with "IQ stream ended" until it was killed with kill -9 (#1256). On a
// timeout it says so and returns; the command then exits, which releases the
// device anyway.
func closeSDRBounded(dev sdr.Device, timeout time.Duration, w io.Writer, prog string) {
	done := make(chan struct{})
	go func() {
		_ = dev.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		fmt.Fprintf(w, "%s: the SDR did not close within %s (a USB transfer is stuck); exiting anyway. "+
			"If the next run cannot open the dongle, unplug and replug it.\n", prog, timeout)
	}
}
