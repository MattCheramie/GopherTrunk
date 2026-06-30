package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/diag"
	"github.com/MattCheramie/GopherTrunk/internal/rfscope"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runRFScope is the entry point for `gophertrunk rfscope` — protocol-agnostic RF
// network analysis ("Wireshark for the RF physical layer"). It segments a
// wideband span (a capture file or a live SDR) into bursts and runs the
// registered analyzers (protocol hierarchy, activity timeline, burst timing, …)
// over them, with no prior knowledge of the technology, modulation, or framing.
func runRFScope(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "analyze":
			runRFScopeAnalyze(args[1:])
			return
		case "live":
			runRFScopeLive(args[1:])
			return
		case "cockpit":
			runRFScopeCockpit(args[1:])
			return
		case "list":
			runRFScopeList(args[1:])
			return
		case "help", "-h", "--help":
			rfscopeUsage(os.Stdout)
			return
		}
	}
	rfscopeUsage(os.Stderr)
	os.Exit(2)
}

func rfscopeUsage(w *os.File) {
	fmt.Fprintln(w, `gophertrunk rfscope — protocol-agnostic RF network analysis.

USAGE:
  gophertrunk rfscope analyze -in <capture> [flags]   analyze a recorded IQ capture
  gophertrunk rfscope live -serial <sdr> -freq Hz [flags]   analyze a live SDR span
  gophertrunk rfscope cockpit [-in <capture> | -serial <sdr> -freq Hz]   live scene TUI
  gophertrunk rfscope list                            list registered analyzers

Run a subcommand with -h for its flags.`)
}

// rfscopeAnalysisFlags are the segmentation/analysis knobs shared by analyze and
// live. They map directly onto rfscope.SegmentConfig.
type rfscopeAnalysisFlags struct {
	fftSize        *int
	channelRate    *float64
	peakThresholdB *float64
	minSpacing     *uint
	analyzers      *string
	outFormat      *string
	out            *string
	framesOut      *string
}

func registerAnalysisFlags(fs *flag.FlagSet) rfscopeAnalysisFlags {
	return rfscopeAnalysisFlags{
		fftSize:        fs.Int("fft", 4096, "wideband FFT size for carrier discovery (power of two)"),
		channelRate:    fs.Float64("channel-rate", 50000, "per-channel decimated baseband rate in Hz"),
		peakThresholdB: fs.Float64("peak-threshold-db", 10, "minimum dB above the noise floor for a carrier to count"),
		minSpacing:     fs.Uint("min-spacing", 12500, "minimum Hz between detected carriers / channel raster"),
		analyzers:      fs.String("analyzers", "", "comma-separated analyzers to run (default: all registered)"),
		outFormat:      fs.String("out-format", "summary", "output format: summary | json | jsonl | yaml | csv"),
		out:            fs.String("out", "", "write output to this file (default: stdout)"),
		framesOut:      fs.String("frames-out", "", "write recovered payloads as a cryptolab `ks` frames file (JSONL)"),
	}
}

func (f rfscopeAnalysisFlags) segConfig(maxSamples int) rfscope.SegmentConfig {
	return rfscope.SegmentConfig{
		FFTSize:             *f.fftSize,
		ChannelTargetRateHz: *f.channelRate,
		PeakThresholdDb:     float32(*f.peakThresholdB),
		MinSpacingHz:        uint32(*f.minSpacing),
		MaxSamples:          maxSamples,
	}
}

func (f rfscopeAnalysisFlags) selectedAnalyzers() []string {
	if strings.TrimSpace(*f.analyzers) == "" {
		return nil
	}
	var out []string
	for _, a := range strings.Split(*f.analyzers, ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func runRFScopeAnalyze(args []string) {
	fs := flag.NewFlagSet("rfscope analyze", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	in := fs.String("in", "", "raw IQ capture file (required)")
	format := fs.String("format", "f32", "sample format: u8 | f32")
	sampleRate := fs.Float64("sample-rate", 2_000_000, "IQ sample rate in Hz")
	freq := fs.Uint64("freq", 0, "capture centre frequency in Hz")
	window := fs.Float64("window", 0, "analyze only the first N seconds (0 ⇒ whole capture)")
	af := registerAnalysisFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk rfscope analyze — segment + analyze a recorded IQ capture.

USAGE:
  gophertrunk rfscope analyze -in <path> [-format u8|f32] [-sample-rate Hz] [-freq Hz]
                             [-window sec] [-analyzers a,b] [-out-format fmt] [-out path]

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("rfscope")

	if *in == "" {
		fs.Usage()
		rep.Fatalf(2, "-in is required")
	}
	if *sampleRate <= 0 {
		rep.Fatalf(2, "-sample-rate must be > 0")
	}
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}

	src, err := rfscope.OpenFile(*in, sampleFormat, uint32(*freq), *sampleRate)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer src.Close()

	maxSamples := 0
	if *window > 0 {
		maxSamples = int(*window * *sampleRate)
	}
	rfscopeRunAndExport(rep, src, af.segConfig(maxSamples), af.selectedAnalyzers(), *in, *af.outFormat, *af.out, *af.framesOut)
}

func runRFScopeLive(args []string) {
	fs := flag.NewFlagSet("rfscope live", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	serial := fs.String("serial", "", "SDR serial to capture from (default: first device)")
	freq := fs.Uint64("freq", 0, "centre frequency to tune in Hz (required)")
	sampleRate := fs.Uint64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	duration := fs.Float64("duration", 10, "seconds of live IQ to analyze")
	tui := fs.Bool("tui", false, "open the live scene cockpit (TUI) instead of a one-shot report")
	af := registerAnalysisFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk rfscope live — segment + analyze a live SDR span.

USAGE:
  gophertrunk rfscope live -serial <sdr> -freq Hz [-sample-rate Hz] [-duration sec]
                          [-analyzers a,b] [-out-format fmt] [-out path]

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("rfscope")

	if *freq == 0 {
		fs.Usage()
		rep.Fatalf(2, "-freq is required")
	}
	if *sampleRate == 0 || *duration <= 0 {
		rep.Fatalf(2, "-sample-rate and -duration must be > 0")
	}

	// -tui delegates to the cockpit, which opens its own SDR source.
	if *tui {
		cargs := []string{"-serial", *serial, "-freq", fmt.Sprint(*freq),
			"-sample-rate", fmt.Sprint(*sampleRate), "-duration", fmt.Sprint(*duration)}
		if *af.analyzers != "" {
			cargs = append(cargs, "-analyzers", *af.analyzers)
		}
		runRFScopeCockpit(cargs)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*duration*2)*time.Second+5*time.Second)
	defer cancel()

	dev, info, err := openCaptureDevice(*serial)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer dev.Close()
	fmt.Fprintf(os.Stderr, "rfscope live: %s %s @ %.6f MHz, %.3f MS/s for %.1fs\n",
		info.Driver, info.Serial, float64(*freq)/1e6, float64(*sampleRate)/1e6, *duration)

	iqsrc, err := newDeviceIQSource(ctx, dev, uint32(*sampleRate), nil)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("start IQ stream: %w", err))
	}
	src, err := rfscope.NewLiveSource(iqsrc, uint32(*freq))
	if err != nil {
		rep.Fatal(1, err)
	}

	maxSamples := int(*duration * float64(*sampleRate))
	label := fmt.Sprintf("live:%s@%.6fMHz", info.Serial, float64(*freq)/1e6)
	rfscopeRunAndExport(rep, src, af.segConfig(maxSamples), af.selectedAnalyzers(), label, *af.outFormat, *af.out, *af.framesOut)
}

func runRFScopeList(args []string) {
	fs := flag.NewFlagSet("rfscope list", flag.ExitOnError)
	_ = fs.Parse(args)
	fmt.Println("Registered rfscope analyzers:")
	for _, a := range rfscope.Analyzers() {
		deps := ""
		if d := a.DependsOn(); len(d) > 0 {
			deps = "  (after: " + strings.Join(d, ", ") + ")"
		}
		fmt.Printf("  %-12s %s%s\n", a.Name(), a.Synopsis(), deps)
	}
}

// rfscopeRunAndExport segments src, runs the selected analyzers, and writes the
// scene — the shared tail of the analyze and live subcommands.
func rfscopeRunAndExport(rep *diag.Reporter, src rfscope.Source, cfg rfscope.SegmentConfig, analyzers []string, sourceLabel, outFormat, outPath, framesOut string) {
	ctx := context.Background()
	scene, in, err := rfscope.Segment(ctx, src, cfg)
	if err != nil {
		rep.Fatal(1, err)
	}
	scene.Source = sourceLabel
	if err := rfscope.Run(ctx, scene, in, analyzers...); err != nil {
		rep.Fatal(1, err)
	}

	if framesOut != "" {
		ff, err := os.Create(framesOut)
		if err != nil {
			rep.Fatal(1, fmt.Errorf("create %s: %w", framesOut, err))
		}
		n, werr := rfscope.EmitFrames(scene, ff)
		ff.Close()
		if werr != nil {
			rep.Fatal(1, fmt.Errorf("write frames: %w", werr))
		}
		fmt.Fprintf(os.Stderr, "rfscope: wrote %d cryptolab frame(s) → %s\n", n, framesOut)
		if groups := rfscope.DetectKeystreamReuse(scene); len(groups) > 0 {
			fmt.Fprintf(os.Stderr, "rfscope: %d keystream-reuse group(s) detected — try `cryptolab ks reuse -in %s`\n", len(groups), framesOut)
		}
		// When cryptolab is linked, run the ks-reuse hand-off in-process and
		// surface its verdict; the default binary skips this (engine triage above
		// already ran).
		if rfscope.CryptolabLinked {
			if res, err := rfscope.RunCryptolabTool(ctx, "ks", "reuse", []string{"-in", framesOut}); err == nil && res != nil {
				fmt.Fprintf(os.Stderr, "rfscope: cryptolab ks reuse → %s\n", res.Summary)
			}
		}
	}

	f, err := rfscope.ParseFormat(outFormat)
	if err != nil {
		rep.Fatal(2, err)
	}
	w := os.Stdout
	if outPath != "" {
		file, err := os.Create(outPath)
		if err != nil {
			rep.Fatal(1, fmt.Errorf("create %s: %w", outPath, err))
		}
		defer file.Close()
		w = file
	}
	if err := rfscope.Write(w, scene, f); err != nil {
		rep.Fatal(1, err)
	}
}
