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
	"strconv"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/powersweep"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/iio"
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
	iioAddr := fs.String("iio", "", "address of a Pluto / PlutoSky board's iiod (e.g. 192.168.2.1) to sweep instead of a local SDR")
	settle := fs.Duration("settle", 0, "samples discarded after each retune (default 50ms local, 250ms rtl_tcp)")
	keepDC := fs.Bool("keep-dc", false, "leave the tuner's own DC line in the output: a spike at the centre of every hop that is not on the air, removed by default (which also takes out an unmodulated carrier within a sixteenth of a bin of a hop's centre)")
	fillDC := fs.Bool("fill-dc", false, "when a line like the tuner's DC line is still at a hop's centre after its removal (the tuner's offset moved during the hop, e.g. an AGC stepping its gain), fill those three bins in from the bins beside them, as rtl_power does; this also hides a narrowband signal that sits exactly on a hop's centre")
	dcReport := fs.Bool("dc-report", false, "print what each hop measured of the tuner's DC line (on stderr, every sweep)")
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk power — rtl_power-style sweep logger (CSV) for a local or rtl_tcp SDR.

USAGE:
  gophertrunk power -f lower:upper:bin [-i interval] [-e duration | -1]
                    [-rtltcp host:port | -iio addr | -serial S] [-gain G] [-ppm N]
                    [-rate S/s] [-crop F] [-out file.csv | file.csv]

OUTPUT (rtl_power's CSV layout, one line per hop):
  date, time, Hz low, Hz high, Hz step, samples, dB, dB, ...
  Bin k of a line is at Hz low + k·Hz step. Levels are uncalibrated dBFS.
  The tuner's own DC line, which would show as a carrier at the centre of
  every hop, is removed (-keep-dc leaves it in). If the tuner's offset moves
  while a hop is measured, part of the line stays: power says so, and
  -fill-dc fills it in.

EXAMPLES:
  # FM broadcast band in 10 kHz bins, one sweep every 10 s, to a file
  gophertrunk power -f 88M:108M:10k -out fm.csv

  # Same, from a dongle shared by an rtl_tcp server (e.g. on an Android phone)
  gophertrunk power -rtltcp 127.0.0.1:1234 -f 88M:108M:10k -i 30s -out fm.csv

  # From an ADALM-Pluto / PlutoSky over its iiod (no SoapySDR or libiio needed)
  gophertrunk power -iio 192.168.2.1 -f 88M:108M:10k -gain 400 -out fm.csv

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
	dev, name, err := openPowerDevice(*rtlTCP, *iioAddr, *serial, log)
	if err != nil {
		rep.Fatal(1, noSDRWithConfigHint("power", err))
	}
	defer closeSDRBounded(dev, sdrCloseTimeout, os.Stderr, "power")
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

	ctx, stop := sdrStopContext("power", os.Stderr)
	defer stop()
	settleFor := *settle
	if settleFor <= 0 {
		settleFor = powerSettleLocal
		if *rtlTCP != "" || *iioAddr != "" {
			// A network radio: the retune and the first samples both
			// cross a socket.
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
	integ.KeepDC = *keepDC
	integ.FillDC = *fillDC
	fmt.Fprint(os.Stderr, powerPlanSummary(name, r, plan, every))

	// IQ dropped anywhere between the dongle and the FFT: by the driver (its
	// queue full) or by the source's slack queue (the FFT behind). It tells
	// a slow CPU from a dongle that delivered too few samples.
	driverDrops := &captureDropCounter{serial: dev.Info().Serial}
	sdr.SetIQDropObserver(driverDrops.observe)
	defer sdr.SetIQDropObserver(nil)
	drops := func() uint64 { return driverDrops.count() + src.dropped.Load() }

	err = runPowerSweeps(ctx, src, plan, integ, powerSchedule{every: every, stopAfter: stopAfter, single: *single, dcReport: *dcReport, fixedGain: gainTenthDB >= 0}, bw, os.Stderr, time.Now, sleepCtx, drops)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		err = iqStreamEnded(dev, err)
	}
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
	// dcReport prints every hop's DC line measurements (-dc-report).
	dcReport bool
	// fixedGain says the tuner's gain is not automatic, so a DC line left
	// after the removal is not put down to an AGC.
	fixedGain bool
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

// powerIntegration splits one sweep interval across its hops: each hop may
// spend its share of the interval, less the retune settle, integrating, and
// asks for the samples real time delivers in that time. On a CPU that cannot
// FFT every sample in real time the budget ends the hop first, so sweeps keep
// to the interval instead of stretching (#1230).
//
// The request must leave the settle out too. It used to ask for the whole
// share (every/hops × rate) while the settle had already spent 50 ms of it, so
// no hop could ever fill it: an idle PC integrated 98 % (rtl_tcp's 250 ms
// settle: 90 %) and runPowerSweeps blamed the CPU.
func powerIntegration(every time.Duration, rate uint32, hops int, settle time.Duration) powersweep.Integration {
	window := every / time.Duration(hops)
	in := powersweep.Integration{Samples: int(window.Seconds() * float64(rate))}
	if b := window - settle; b > 0 {
		in.Budget = b
		in.Samples = int(b.Seconds() * float64(rate))
	}
	return in
}

// powerSlowCPUFraction is how much of its request a sweep must integrate
// before runPowerSweeps says it fell short. A hop that keeps up with real
// time still ends a little short of it: the budget is checked once per capture
// block (~27 ms at 2.4 MS/s) and the clock starts after a retune that costs a
// few ms of USB or network I/O. A CPU that genuinely falls behind (the #1230
// phones FFT at about half real time) lands far below.
const powerSlowCPUFraction = 0.9

// runPowerSweeps runs sweeps on sched, writing (and flushing) each one's rows
// as it completes so a long log can be tailed and a Ctrl-C loses at most the
// sweep in progress. A sweep that integrates clearly fewer samples than asked
// (under powerSlowCPUFraction) prints a note to notes, once per cause.
//
// drops, when non-nil, counts the IQ chunks dropped between the dongle and the
// FFT so far, and settles the cause. A hop's source queue is emptied on every
// retune and holds about 0.2 s of IQ, so an FFT under 90 % of real time falls
// further behind than that within a hop and drops chunks: a short sweep with
// drops is the CPU. One with none got every sample the dongle delivered, so
// the dongle (its USB link or driver) delivered fewer than its rate. Without
// that split, the S5 in #1230 was told its CPU could not keep up when only
// the first three hops of its first sweep came up short and every later hop
// filled. Without drops (nil) a short sweep is put down to the CPU, as before.
func runPowerSweeps(ctx context.Context, src powersweep.Source, plan powersweep.Plan, in powersweep.Integration,
	sched powerSchedule, w *bufio.Writer, notes io.Writer, now func() time.Time, sleep func(context.Context, time.Duration) error,
	drops func() uint64) error {
	if in.Now == nil {
		in.Now = now
	}
	start := now()
	var notedCPU, notedSource, notedDC bool
	var lastDrops uint64
	if drops != nil {
		lastDrops = drops()
	}
	for sweep := 1; ; sweep++ {
		t0 := now()
		rows, err := powersweep.Sweep(ctx, src, plan, in, t0)
		if err != nil {
			return err
		}
		var dropped uint64
		if drops != nil {
			d := drops()
			dropped, lastDrops = d-lastDrops, d
		}
		if notes != nil {
			got, want := sweepSamples(rows), len(rows)*in.Samples
			if pct := 100 * float64(got) / float64(want); float64(got) < powerSlowCPUFraction*float64(want) {
				switch {
				case drops == nil || dropped > 0:
					if !notedCPU {
						notedCPU = true
						fmt.Fprintf(notes, "power: note — the FFT can't keep up with %.0f S/s on this CPU, so each hop of sweep %d averaged "+
							"%.0f%% of the samples real time delivers, to keep sweeps on the -i schedule%s. The CSV's samples column has each hop's "+
							"count. A lower -rate (e.g. 1024000) needs less CPU.\n",
							plan.SampleRateHz, sweep, pct, droppedClause(dropped))
					}
				default:
					if !notedSource {
						notedSource = true
						fmt.Fprintf(notes, "power: note — sweep %d integrated %.0f%% of the samples real time delivers at %.0f S/s, "+
							"but no IQ was dropped on the way to the FFT, so the CPU kept up: the SDR delivered fewer samples than its rate "+
							"(a USB link or driver that was slow to deliver). The CSV's samples column has each hop's count.\n",
							sweep, pct, plan.SampleRateHz)
					}
				}
			}
		}
		if notes != nil {
			if sched.dcReport {
				for _, row := range rows {
					fmt.Fprintln(notes, powerDCReportLine(row, plan))
				}
			}
			if note := powerDCLineNote(sweep, rows, plan, sched.fixedGain); note != "" && !notedDC {
				notedDC = true
				fmt.Fprint(notes, note)
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

// powerHopCentreHz is the frequency a row's hop was tuned to: the bin
// HopBins/2 of the row, whether or not a short last row reaches it.
func powerHopCentreHz(row powersweep.Row, plan powersweep.Plan) float64 {
	return row.LowHz + float64(plan.HopBins/2)*row.StepHz
}

// powerDCLineNote is what `power` says, once per run, when the removal of
// the tuner's DC line left a line of its shape at the centre of some hops
// (#1230: the v1.4.1 retest on a PlutoSky still showed it in the busy hops,
// with nothing to say why). It is empty when no hop of the sweep has one.
//
// The removal takes out what stands still from frame to frame. A line is
// left when the tuner's offset moved during the hop (an AGC stepping its
// gain moves it), and also when a narrowband signal sits exactly on the
// hop's centre; the spectrum cannot tell which, so the note names both and
// what to do about each.
func powerDCLineNote(sweep int, rows []powersweep.Row, plan powersweep.Plan, fixedGain bool) string {
	var at []string
	var worst float64
	filled := false
	for _, row := range rows {
		if row.DC.Finding != powersweep.DCLineLeft {
			continue
		}
		at = append(at, fmt.Sprintf("%.4f", powerHopCentreHz(row, plan)/1e6))
		worst = math.Max(worst, row.DC.LeftDB)
		filled = filled || row.DC.Filled
	}
	if len(at) == 0 {
		return ""
	}
	where := strings.Join(at, ", ")
	if len(at) > 4 {
		where = strings.Join(at[:4], ", ") + ", …"
	}
	if filled {
		return fmt.Sprintf("power: note — sweep %d: a line the shape of the tuner's DC line was left at the centre of %d of %d hops "+
			"(%s MHz; up to %.1f dB over the bins beside it) and is filled in from those bins (-fill-dc). "+
			"A narrowband signal exactly on a hop's centre is filled in with it. -dc-report prints what each hop measured.\n",
			sweep, len(at), len(rows), where, worst)
	}
	agc := " (an AGC changing its gain does that: a fixed -gain keeps it still)"
	if fixedGain {
		agc = ""
	}
	return fmt.Sprintf("power: note — sweep %d: after the tuner's DC line was removed, a line of its shape is still at the centre of "+
		"%d of %d hops (%s MHz; up to %.1f dB over the bins beside it). Either the tuner's offset moved while those hops were measured%s, "+
		"or a narrowband signal sits exactly on a hop's centre. "+
		"-fill-dc fills those three bins in from the bins beside them, as rtl_power does. -dc-report prints what each hop measured.\n",
		sweep, len(at), len(rows), where, worst, agc)
}

// powerDCReportLine is one hop's line of -dc-report: what was removed as
// the tuner's DC line, what was left at the centre, and how that behaves
// from one FFT frame to the next. Posted back from a field run, it tells a
// moving offset (alike from frame to frame, no turn) from a carrier off the
// centre (a steady turn) without a capture.
func powerDCReportLine(row powersweep.Row, plan powersweep.Plan) string {
	head := fmt.Sprintf("power: dc %.4f–%.4f MHz (centre %.4f):", row.LowHz/1e6, row.HighHz/1e6, powerHopCentreHz(row, plan)/1e6)
	d := row.DC
	if !d.Measured {
		return head + " not measured"
	}
	still := "nothing stood still"
	if !math.IsInf(d.StillDB, -1) {
		still = fmt.Sprintf("%.1f dBFS stood still and was removed", d.StillDB)
	}
	turnHz := d.PhaseRad / (2 * math.Pi) * plan.SampleRateHz / float64(plan.FFTSize)
	s := fmt.Sprintf("%s %s; centre %+.1f dB over the bins beside it, shape %.2f, alike %.2f, turn %+.1f° a frame (%+.0f Hz): %s",
		head, still, d.LeftDB, d.Shape, d.Coherence, d.PhaseRad*180/math.Pi, turnHz, d.Finding)
	if d.Filled {
		s += ", filled in"
	}
	return s
}

// droppedClause names the IQ chunks a short sweep dropped, for the slow-CPU
// note; it is empty when the count is unknown.
func droppedClause(n uint64) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf(" (%d IQ chunks dropped)", n)
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

// iqStreamEnded explains an IQ stream that stopped before the command was
// done. A closed channel says only that it ended, so `power` printed a bare
// "unexpected EOF" and `wmbus` exited silently when the reporter's rtl_tcp
// driver app dropped the session (#1230, #1256); the device's own reason
// (sdr.StreamErr: the server hung up, after how long and how much IQ) is
// what the operator can act on. Without one, fallback is returned as is.
func iqStreamEnded(dev sdr.Device, fallback error) error {
	if err := sdr.StreamErr(dev); err != nil {
		return fmt.Errorf("IQ stream ended: %w", err)
	}
	return fallback
}

// openPowerDevice opens the rtl_tcp server at addr when given, the Pluto-style
// board whose iiod is at iioAddr when given (issue #1329; the #1230 reporter's
// `sdr.iio` config does nothing here, since the one-shot tools read no config
// file), else the local SDR matching serial. It returns the device and a name
// for the banner.
func openPowerDevice(addr, iioAddr, serial string, log *slog.Logger) (sdr.Device, string, error) {
	n := 0
	for _, s := range []string{addr, iioAddr, serial} {
		if s != "" {
			n++
		}
	}
	if n > 1 {
		return nil, "", errors.New("-rtltcp, -iio and -serial are mutually exclusive")
	}
	if addr != "" {
		dev, err := rtltcp.New([]rtltcp.Spec{{Addr: addr}}, log).Open(0)
		if err != nil {
			return nil, "", fmt.Errorf("rtl_tcp %s: %w", addr, err)
		}
		return dev, "rtl_tcp " + addr, nil
	}
	if iioAddr != "" {
		dev, err := iio.New([]iio.Spec{{Addr: iioAddr}}, log).Open(0)
		if err != nil {
			return nil, "", err
		}
		return dev, "iio " + iio.NormalizeAddr(iioAddr), nil
	}
	dev, info, err := openCaptureDevice(serial)
	if err != nil {
		return nil, "", err
	}
	return dev, fmt.Sprintf("%s[%s]", info.Driver, info.Serial), nil
}
