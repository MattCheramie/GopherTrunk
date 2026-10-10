package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/powersweep"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtltcp"
)

// Retune settle defaults for `gophertrunk power`. A local dongle's samples
// follow a retune within a few USB transfers. An rtl_tcp server keeps
// streaming while the retune command crosses the network, so the samples
// already in its queue and the socket buffers still belong to the old
// frequency and need a longer discard.
const (
	powerSettleLocal  = 50 * time.Millisecond
	powerSettleRTLTCP = 250 * time.Millisecond
)

// runPower is `gophertrunk power` (issue #1230): an rtl_power-style sweep
// logger. It steps a live SDR — local USB, or a remote dongle behind an
// rtl_tcp server — across a frequency range and writes the averaged power
// spectrum in rtl_power's CSV layout.
func runPower(args []string) {
	fs := flag.NewFlagSet("power", flag.ExitOnError)
	freqRange := fs.String("f", "", "frequency range lower:upper:bin_size, with k/M/G suffixes (e.g. 88M:108M:10k); required")
	interval := fs.String("i", "10s", "integration interval: one sweep per interval, split across its hops (seconds, or a duration like 500ms / 1m)")
	exitAfter := fs.String("e", "", "stop after this long (seconds or a duration); empty runs until Ctrl-C")
	single := fs.Bool("1", false, "one sweep, then exit")
	out := fs.String("out", "", "CSV output path (default: stdout; a trailing positional argument works too, as with rtl_power)")
	rate := fs.Uint("rate", 2_400_000, "sample rate in S/s; each hop covers rate·(1−crop) Hz")
	crop := fs.Float64("crop", 0.25, "fraction of each FFT dropped at the band edges, where the tuner filter rolls off (0..0.9)")
	gain := fs.String("gain", "auto", `tuner gain: "auto", or tenths of a dB ("280" = 28 dB; "28.0" also reads as 28 dB)`)
	ppm := fs.Int("ppm", 0, "frequency correction in ppm")
	biasTee := fs.Bool("bias-tee", false, "enable the bias-tee (local dongles and rtl_tcp servers that support it)")
	serial := fs.String("serial", "", "local SDR serial (required when more than one is attached)")
	rtlTCP := fs.String("rtltcp", "", "host:port of an rtl_tcp server to sweep instead of a local SDR")
	settle := fs.Duration("settle", 0, "samples discarded after each retune (default 50ms local, 250ms rtl_tcp)")
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk power — rtl_power-style sweep logger (CSV) for a local or rtl_tcp SDR.

USAGE:
  gophertrunk power -f lower:upper:bin [-i interval] [-e duration | -1]
                    [-rtltcp host:port | -serial S] [-gain G] [-ppm N]
                    [-rate S/s] [-crop F] [-out file.csv | file.csv]

OUTPUT (rtl_power's CSV layout, one line per hop):
  date, time, Hz low, Hz high, Hz step, samples, dB, dB, ...
  Bin k of a line is at Hz low + k·Hz step. Levels are uncalibrated dBFS.

EXAMPLES:
  # FM broadcast band in 10 kHz bins, one sweep every 10 s, to a file
  gophertrunk power -f 88M:108M:10k -out fm.csv

  # Same, from a dongle shared by an rtl_tcp server (e.g. on an Android phone)
  gophertrunk power -rtltcp 127.0.0.1:1234 -f 88M:108M:10k -i 30s -out fm.csv

  # One quick sweep of the 2 m band to the terminal
  gophertrunk power -f 144M:148M:5k -i 1 -1

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("power")

	if *out == "" && fs.NArg() == 1 {
		*out = fs.Arg(0)
	} else if fs.NArg() > 0 {
		fs.Usage()
		rep.Fatalf(2, "unexpected arguments %q", fs.Args())
	}
	if *freqRange == "" {
		fs.Usage()
		rep.Fatalf(2, "-f is required")
	}
	r, err := powersweep.ParseRange(*freqRange)
	if err != nil {
		rep.Fatal(2, err)
	}
	if *crop < 0 || *crop > 0.9 {
		rep.Fatalf(2, "-crop must be in 0..0.9")
	}
	plan, err := powersweep.NewPlan(r, float64(*rate), *crop)
	if err != nil {
		rep.Fatal(2, err)
	}
	every, err := parsePowerDuration(*interval)
	if err != nil || every <= 0 {
		rep.Fatalf(2, "-i: want a positive number of seconds or a duration, got %q", *interval)
	}
	var stopAfter time.Duration
	if *exitAfter != "" {
		if stopAfter, err = parsePowerDuration(*exitAfter); err != nil || stopAfter <= 0 {
			rep.Fatalf(2, "-e: want a positive number of seconds or a duration, got %q", *exitAfter)
		}
	}
	gainTenthDB, ok := parseGain(*gain)
	if !ok {
		rep.Fatalf(2, `-gain: want "auto" or tenths of a dB, got %q`, *gain)
	}
	if gainLooksLikeDBMistake(*gain, gainTenthDB) {
		fmt.Fprintf(os.Stderr, "power: note — -gain %s is %.1f dB (gain is tenths of a dB; write %s0 or %s.0 for %s dB)\n",
			*gain, float64(gainTenthDB)/10, *gain, *gain, *gain)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	dev, name, err := openPowerDevice(*rtlTCP, *serial, log)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer dev.Close()
	if err := dev.SetSampleRate(uint32(*rate)); err != nil {
		rep.Fatal(1, fmt.Errorf("set sample rate: %w", err))
	}
	// The plan's bin frequencies come from the sample rate, so lay it out at
	// the rate the hardware actually delivers.
	if hw, note := captureEffectiveRate(dev, uint32(*rate)); note != "" {
		fmt.Fprintln(os.Stderr, strings.Replace(note, "capture:", "power:", 1))
		*rate = uint(hw)
		if plan, err = powersweep.NewPlan(r, float64(hw), *crop); err != nil {
			rep.Fatal(2, err)
		}
	}
	if *ppm != 0 {
		if err := dev.SetPPM(*ppm); err != nil {
			rep.Fatal(1, fmt.Errorf("set ppm: %w", err))
		}
	}
	if err := dev.SetGain(gainTenthDB); err != nil {
		rep.Fatal(1, fmt.Errorf("set gain: %w", err))
	}
	if *biasTee {
		if err := dev.SetBiasTee(true); err != nil {
			rep.Fatal(1, fmt.Errorf("set bias-tee: %w", err))
		}
	}
	if err := dev.SetCenterFreq(plan.Hops[0].CenterHz); err != nil {
		rep.Fatal(1, fmt.Errorf("tune %d Hz: %w", plan.Hops[0].CenterHz, err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	settleFor := *settle
	if settleFor <= 0 {
		settleFor = powerSettleLocal
		if *rtlTCP != "" {
			settleFor = powerSettleRTLTCP
		}
	}
	src, err := newPowerIQSource(ctx, dev, uint32(*rate), settleFor, log)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("start IQ stream: %w", err))
	}

	w := io.Writer(os.Stdout)
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			rep.Fatal(1, err)
		}
		defer f.Close()
		w = f
	}
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	integ := powerIntegration(every, uint32(*rate), len(plan.Hops), settleFor)
	fmt.Fprint(os.Stderr, powerPlanSummary(name, r, plan, every))

	err = runPowerSweeps(ctx, src, plan, integ, powerSchedule{every: every, stopAfter: stopAfter, single: *single}, bw, os.Stderr, time.Now, sleepCtx)
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = bw.Flush()
		rep.Fatal(1, err)
	}
}

// powerPlanSummary is the banner `power` prints before sweeping. The first
// line is the layout; when the bins come out narrower than -f asked for, a
// second line shows where the numbers come from (#1230: "10k" gave 192 ×
// 9375 Hz bins per line, where a reader expected 180 × 10 kHz). Like
// rtl_power, the requested bin size is a maximum: the FFT is the smallest
// power of two whose bins are no wider, and each hop keeps the centre
// (1 − crop) of it.
func powerPlanSummary(name string, r powersweep.Range, plan powersweep.Plan, every time.Duration) string {
	s := fmt.Sprintf("power: %s, %.0f–%.0f Hz in %d hops of %d × %.1f Hz bins (%d-point FFT), %s per sweep\n",
		name, r.LowHz, r.HighHz, len(plan.Hops), plan.HopBins, plan.BinHz, plan.FFTSize, every)
	if plan.BinHz < r.BinHz {
		hopHz := float64(plan.HopBins) * plan.BinHz
		s += fmt.Sprintf("power: bins are %.0f S/s ÷ %d = %.1f Hz (the largest width not over the %.0f Hz asked for); "+
			"each hop keeps the centre %d of %d bins (%.1f kHz, -crop %.2f), so a CSV line has %d values\n",
			plan.SampleRateHz, plan.FFTSize, plan.BinHz, r.BinHz,
			plan.HopBins, plan.FFTSize, hopHz/1e3, 1-float64(plan.HopBins)/float64(plan.FFTSize), plan.HopBins)
	}
	return s
}

// powerSchedule is when sweeps run: one per every, until stopAfter has
// elapsed (0 = never) or after one sweep when single is set.
type powerSchedule struct {
	every     time.Duration
	stopAfter time.Duration
	single    bool
}

// newPowerIQSource opens the device's IQ stream for `power`. Falling behind
// real time is expected there on a slow CPU (each hop integrates what it can
// within its time budget, and runPowerSweeps says so once), so the stream's
// per-second "hunt: decode falling behind" WARN is silenced (#1230).
func newPowerIQSource(ctx context.Context, dev sdr.Device, rate uint32, settle time.Duration, log *slog.Logger) (*streamIQSource, error) {
	src, err := newDeviceIQSource(ctx, dev, rate, log)
	if err != nil {
		return nil, err
	}
	src.settle = settle
	src.quietDrops = true
	return src, nil
}

// powerIntegration splits one sweep interval across its hops: each hop asks
// for its share of the interval's samples, and may spend its share of the
// interval (less the retune settle) doing so. On a CPU that cannot FFT every
// sample in real time the budget ends the hop first, so sweeps keep to the
// interval instead of stretching (#1230).
func powerIntegration(every time.Duration, rate uint32, hops int, settle time.Duration) powersweep.Integration {
	in := powersweep.Integration{Samples: int(every.Seconds() * float64(rate) / float64(hops))}
	if b := every/time.Duration(hops) - settle; b > 0 {
		in.Budget = b
	}
	return in
}

// runPowerSweeps runs sweeps on sched, writing (and flushing) each one's rows
// as it completes so a long log can be tailed and a Ctrl-C loses at most the
// sweep in progress. The first sweep that integrates fewer samples than asked
// (the FFT could not keep up, see powerIntegration) prints one note to notes.
func runPowerSweeps(ctx context.Context, src powersweep.Source, plan powersweep.Plan, in powersweep.Integration,
	sched powerSchedule, w *bufio.Writer, notes io.Writer, now func() time.Time, sleep func(context.Context, time.Duration) error) error {
	if in.Now == nil {
		in.Now = now
	}
	start := now()
	noted := false
	for {
		t0 := now()
		rows, err := powersweep.Sweep(ctx, src, plan, in, t0)
		if err != nil {
			return err
		}
		if !noted && notes != nil {
			if got, want := sweepSamples(rows), len(rows)*in.Samples; got < want {
				noted = true
				fmt.Fprintf(notes, "power: note — the FFT can't keep up with %.0f S/s on this CPU, so each hop averaged "+
					"%.0f%% of its samples to keep sweeps on the -i schedule. The CSV's samples column has each hop's "+
					"count. A lower -rate (e.g. 1024000) needs less CPU.\n",
					plan.SampleRateHz, 100*float64(got)/float64(want))
			}
		}
		if err := powersweep.WriteCSV(w, rows); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		if sched.single {
			return nil
		}
		next := t0.Add(sched.every)
		if sched.stopAfter > 0 && !next.Before(start.Add(sched.stopAfter)) {
			return nil
		}
		if err := sleep(ctx, next.Sub(now())); err != nil {
			return err
		}
	}
}

// sweepSamples is the number of IQ samples a sweep's rows integrated.
func sweepSamples(rows []powersweep.Row) int {
	n := 0
	for _, r := range rows {
		n += r.Samples
	}
	return n
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// parsePowerDuration reads a bare number as seconds (rtl_power's -i / -e)
// and anything else as a Go duration ("500ms", "1m30s").
func parsePowerDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, fmt.Errorf("bad duration %q", s)
		}
		return time.Duration(f * float64(time.Second)), nil
	}
	return time.ParseDuration(s)
}

// openPowerDevice opens the rtl_tcp server at addr when given, else the local
// SDR matching serial. It returns the device and a name for the banner.
func openPowerDevice(addr, serial string, log *slog.Logger) (sdr.Device, string, error) {
	if addr != "" {
		if serial != "" {
			return nil, "", errors.New("-rtltcp and -serial are mutually exclusive")
		}
		dev, err := rtltcp.New([]rtltcp.Spec{{Addr: addr}}, log).Open(0)
		if err != nil {
			return nil, "", fmt.Errorf("rtl_tcp %s: %w", addr, err)
		}
		return dev, "rtl_tcp " + addr, nil
	}
	dev, info, err := openCaptureDevice(serial)
	if err != nil {
		return nil, "", err
	}
	return dev, fmt.Sprintf("%s[%s]", info.Driver, info.Serial), nil
}
