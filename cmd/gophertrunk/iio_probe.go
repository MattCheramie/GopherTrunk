package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/sdr/iio"
)

// runIIO dispatches `gophertrunk iio <subcommand>`.
func runIIO(args []string) {
	if len(args) == 0 || args[0] != "probe" {
		fmt.Fprintln(os.Stderr, "usage: gophertrunk iio probe [-addr 192.168.2.1] [-out board.xml] [-samples N]")
		os.Exit(2)
	}
	os.Exit(runIIOProbe(args[1:]))
}

// runIIOProbe checks a Pluto-style board over its iiod without changing any
// setting (#1329): the same device and sample-format check sdr.iio makes at
// open, the current LO / rate / gain read back, one buffer of samples for the
// level, and the board's context XML saved for the issue tracker. It needs no
// libiio, SoapySDR or netcat on the host.
func runIIOProbe(args []string) int {
	fs := flag.NewFlagSet("iio probe", flag.ExitOnError)
	addr := fs.String("addr", "192.168.2.1", `the board's address: host, host:port (default port 30431) or "ip:<host>"`)
	out := fs.String("out", "", "write the board's context XML (iiod PRINT) to this file")
	samples := fs.Int("samples", 65536, "samples to read once to check the stream and measure the level (0 skips; the board's current settings are used)")
	timeout := fs.Duration("timeout", 5*time.Second, "dial and per-command timeout")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk iio probe — check an ADALM-Pluto / PlutoSky (or other Pluto-firmware
AD936x board) over the iiod daemon running on it, without changing any setting.

It prints the iiod version, board model and firmware, the IIO devices and their
sample formats, whether sdr.iio can use the board (and why not), the current RX LO,
sample rate, bandwidth and gain, and the level of one buffer of samples.

USAGE:
  gophertrunk iio probe [-addr 192.168.2.1] [-out board.xml] [-samples 65536]

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	rep, err := iio.Probe(*addr, *timeout, *samples)
	if rep != nil && rep.Version != "" {
		fmt.Print(rep.String())
	}
	if *out != "" && rep != nil && len(rep.XML) > 0 {
		if werr := os.WriteFile(*out, rep.XML, 0o644); werr != nil {
			fmt.Fprintf(os.Stderr, "iio probe: write %s: %v\n", *out, werr)
			return 1
		}
		fmt.Printf("context XML (%d bytes) written to %s\n", len(rep.XML), *out)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "iio probe: %v\n", err)
		return 1
	}
	if rep.BindErr != nil {
		return 1
	}
	return 0
}
