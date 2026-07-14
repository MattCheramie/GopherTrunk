package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runAnalyze is the entry point for `gophertrunk analyze`. It runs an offline
// IQ capture through the production pipeline for any protocol GopherTrunk
// decodes, collects the protocol-agnostic signal-quality analysis, and emits
// a clean structured report (JSON / JSONL / YAML / CSV) or a human-readable
// summary. It is the all-protocol structured-output front door that
// complements `replay` (whose P25/DMR paths carry deeper demod diagnostics).
func runAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	in := fs.String("in", "", "raw IQ input file (required)")
	format := fs.String("format", "u8", "sample format: u8 | f32")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	protocolFlag := fs.String("protocol", "p25p1", "protocol to decode (p25p1|p25-phase2|dmr|dmr-tier2|nxdn|dpmr|edacs|motorola|ltr|mpt1327|tetra|ysf|dstar)")
	freq := fs.Uint64("freq", 0, "informational: the capture's nominal centre frequency in Hz")
	tuneHz := fs.Float64("tune-hz", 0, "frequency-shift the capture so a channel at +tune-hz lands at 0 Hz before demod")
	autoTune := fs.Bool("auto-tune", false, "find the signal by trying the ranked carrier candidates and keeping the best lock — handles an off-centre / non-dominant carrier in a wideband capture")
	conjugate := fs.Bool("conjugate", false, "conjugate IQ (negate Q) before channelization (spectrum-inverted front-end, issue #264)")
	iqCorrect := fs.Bool("iq-correct", false, "apply blind I/Q-imbalance correction to the raw IQ before decimation")
	out := fs.String("out", "", "write structured output to this file (default: stdout)")
	outFormat := fs.String("out-format", "text", "output format: text | json | jsonl | yaml | csv | csv-events")
	bundlePath := fs.String("bundle", "", "read the capture from this GopherTrunk Bundle (.gtb.tar.gz) and write the result back into it (siglab/result.yaml + events.jsonl)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk analyze — decode + analyze a raw IQ capture, with structured export.

USAGE:
  gophertrunk analyze -in <path> -protocol <p> [-format u8|f32] [-sample-rate Hz]
                     [-out-format text|json|jsonl|yaml|csv|csv-events] [-out <path>]

EXAMPLES:
  # P25 Phase 1 capture, JSON report to a file
  gophertrunk analyze -in cc.cfile -format f32 -sample-rate 2400000 -protocol p25p1 -out-format json -out cc.json

  # TETRA capture, human-readable summary to stdout
  gophertrunk analyze -in tetra.cfile -format f32 -sample-rate 2400000 -protocol tetra -auto-tune

  # Streaming per-event JSONL
  gophertrunk analyze -in dmr.cfile -format f32 -sample-rate 2400000 -protocol dmr -out-format jsonl

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("analyze")

	// When a bundle is given, source the capture (and defaults) from it so a
	// bundled case analyzes without unpacking by hand.
	var bundleCleanup func()
	if *bundlePath != "" {
		iqPath, meta, cleanup, err := extractBundleCapture(*bundlePath)
		if err != nil {
			rep.Fatal(1, err)
		}
		bundleCleanup = cleanup
		defer bundleCleanup()
		*in = iqPath
		if meta != nil {
			if !flagPassed(fs, "sample-rate") && meta.SampleRateHz > 0 {
				*sampleRate = meta.SampleRateHz
			}
			if !flagPassed(fs, "format") && meta.Format != "" {
				*format = meta.Format
			}
			if !flagPassed(fs, "protocol") && meta.Protocol != "" {
				*protocolFlag = meta.Protocol
			}
			if !flagPassed(fs, "tune-hz") && meta.TuneHz != 0 {
				*tuneHz = meta.TuneHz
			}
		}
	}

	if *in == "" {
		fs.Usage()
		rep.Fatalf(2, "-in is required (or pass -bundle to read the capture from a GopherTrunk Bundle)")
	}
	proto, err := siglab.ParseProtocolCLI(*protocolFlag)
	if err != nil {
		rep.Fatal(2, err)
	}
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}
	of, err := siglab.ParseFormat(*outFormat)
	if err != nil {
		rep.Fatal(2, err)
	}

	cfg := siglab.Config{
		Protocol:      proto,
		SystemName:    "analyze",
		FrequencyHz:   uint32(*freq),
		SampleRateHz:  *sampleRate,
		Format:        sampleFormat,
		TuneHz:        *tuneHz,
		AutoTune:      *autoTune,
		Conjugate:     *conjugate,
		IQCorrect:     *iqCorrect,
		CollectIQDiag: true,
	}

	// -auto-tune tries the ranked carrier candidates and keeps the best lock,
	// so an off-centre / non-dominant control channel in a wideband capture is
	// still found.
	var res *siglab.Result
	if *autoTune {
		res, err = siglab.RunAutoTuneMulti(*in, cfg, 0)
	} else {
		res, err = siglab.Run(*in, cfg)
	}
	if err != nil {
		rep.Fatal(1, err)
	}

	if err := emitResult(res, of, *out); err != nil {
		rep.Fatal(1, err)
	}

	if *bundlePath != "" {
		if err := writeResultToBundle(*bundlePath, res); err != nil {
			rep.Fatal(1, fmt.Errorf("write result to bundle: %w", err))
		}
		fmt.Fprintf(os.Stderr, "analyze: wrote result → %s (siglab/result.yaml)\n", *bundlePath)
	}
}

// flagPassed reports whether the named flag was explicitly set on the command
// line (as opposed to left at its default), so a bundle's metadata can supply a
// default only when the operator did not override it.
func flagPassed(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// emitResult writes res in the requested format to outPath (or stdout when
// empty). FormatTextSummary uses the shared human renderer; the rest go
// through siglab.WriteResult.
func emitResult(res *siglab.Result, of siglab.Format, outPath string) error {
	w := os.Stdout
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", outPath, err)
		}
		defer f.Close()
		w = f
	}
	if of == siglab.FormatTextSummary {
		renderResultText(w, res)
		return nil
	}
	return siglab.WriteResult(w, res, of)
}
