package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr"
)

// sdrStopSignals end a foreground SDR command (capture, power, wmbus) the way
// Ctrl-C does. SIGTERM is what `kill` / `pkill` send and SIGHUP what a closed
// terminal sends; both used to kill the process without its summary. On
// Termux a root process started through `su` often never sees the terminal's
// Ctrl-C at all (#1256), and `pkill gophertrunk` from a root shell is how it
// gets stopped.
var sdrStopSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}

// sdrStopContext is signal.NotifyContext over sdrStopSignals.
func sdrStopContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), sdrStopSignals...)
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
