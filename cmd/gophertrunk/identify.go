package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runIdentify is the entry point for `gophertrunk identify`. It scans a capture
// against every protocol GopherTrunk can decode (over a fast prefix), ranks
// them by lock + sync-landscape + FEC evidence, names the most likely one, and
// then runs the full analysis for that protocol over the whole capture — so a
// single command answers "what is this signal, and how does it decode?".
func runIdentify(args []string) {
	fs := flag.NewFlagSet("identify", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	in := fs.String("in", "", "raw IQ input file (required)")
	format := fs.String("format", "u8", "sample format: u8 | f32")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	autoTune := fs.Bool("auto-tune", false, "estimate the dominant carrier offset and tune it to 0 Hz before demod")
	conjugate := fs.Bool("conjugate", false, "conjugate IQ (negate Q) before channelization (spectrum-inverted front-end)")
	iqCorrect := fs.Bool("iq-correct", false, "apply blind I/Q-imbalance correction to the raw IQ before decimation")
	maxSeconds := fs.Float64("max-seconds", 3.0, "seconds of capture to scan per candidate (0 = whole capture)")
	noAnalyze := fs.Bool("no-analyze", false, "only identify; skip the full analysis of the winning protocol")
	out := fs.String("out", "", "write the structured report to this file (default: stdout)")
	outFormat := fs.String("out-format", "text", "output format: text | json | yaml")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk identify — auto-detect the protocol in a capture, then analyze it.

USAGE:
  gophertrunk identify -in <path> [-format u8|f32] [-sample-rate Hz]
                      [-max-seconds N] [-out-format text|json|yaml] [-out <file>]

EXAMPLES:
  # Identify an unknown wideband capture and analyze the winning protocol
  gophertrunk identify -in unknown.cfile -format f32 -sample-rate 2400000 -auto-tune

  # Just the ranked identification, as JSON
  gophertrunk identify -in unknown.cfile -format f32 -sample-rate 2400000 -no-analyze -out-format json

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("identify")

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

	var maxSamples int64
	if *maxSeconds > 0 {
		maxSamples = int64(*maxSeconds * *sampleRate)
	}

	idr, err := siglab.Identify(*in, siglab.IdentifyConfig{
		SampleRateHz: *sampleRate,
		Format:       sampleFormat,
		AutoTune:     *autoTune,
		Conjugate:    *conjugate,
		IQCorrect:    *iqCorrect,
		MaxSamples:   maxSamples,
	})
	if err != nil {
		rep.Fatal(1, err)
	}

	// Run the full analysis for the winner over the whole capture, unless the
	// identification was inconclusive or the operator opted out.
	var analysis *siglab.Result
	if !*noAnalyze && !idr.Inconclusive {
		proto, perr := idr.WinnerProtocol()
		if perr != nil {
			rep.Fatal(1, perr)
		}
		analysis, err = siglab.Run(*in, siglab.Config{
			Protocol:      proto,
			SystemName:    "identify",
			SampleRateHz:  *sampleRate,
			Format:        sampleFormat,
			AutoTune:      *autoTune,
			Conjugate:     *conjugate,
			IQCorrect:     *iqCorrect,
			CollectIQDiag: true,
		})
		if err != nil {
			rep.Fatal(1, err)
		}
	}

	w := os.Stdout
	if *out != "" {
		f, cerr := os.Create(*out)
		if cerr != nil {
			rep.Fatal(1, fmt.Errorf("create %s: %w", *out, cerr))
		}
		defer f.Close()
		w = f
	}
	if err := emitIdentify(w, idr, analysis, *outFormat); err != nil {
		rep.Fatal(1, err)
	}
}

// emitIdentify renders the identification verdict (and the winner's analysis,
// when present) as text/json/yaml.
func emitIdentify(w io.Writer, idr *siglab.IdentifyResult, analysis *siglab.Result, outFormat string) error {
	switch outFormat {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Identify *siglab.IdentifyResult `json:"identify"`
			Analysis *siglab.Result         `json:"analysis,omitempty"`
		}{idr, analysis})
	case "yaml", "yml":
		// Text verdict + the winner's analysis as YAML (the structured part
		// operators pipe onward).
		renderIdentifyText(w, idr)
		if analysis != nil {
			return siglab.WriteResult(w, analysis, siglab.FormatYAML)
		}
		return nil
	case "text", "":
		renderIdentifyText(w, idr)
		if analysis != nil {
			fmt.Fprintln(w, "----")
			fmt.Fprintf(w, "identify: full analysis of %s ↓\n", idr.Winner)
			renderResultText(w, analysis)
		}
		return nil
	default:
		return fmt.Errorf("unknown -out-format %q (want text|json|yaml)", outFormat)
	}
}

// renderIdentifyText prints the ranked candidate table + verdict.
func renderIdentifyText(w io.Writer, idr *siglab.IdentifyResult) {
	fmt.Fprintf(w, "identify: %s @ %.0f Hz — %d candidates\n", idr.Source, idr.SampleRateHz, len(idr.Candidates))
	fmt.Fprintf(w, "identify: %-12s %-7s %-7s %-14s %-9s %-6s %s\n",
		"protocol", "locked", "hits", "sync", "fec_pass", "modal", "score")
	for _, c := range idr.Candidates {
		fmt.Fprintf(w, "identify:   %-10s %-7v %-7d %-14s %-9.2f %-6d %.3f\n",
			c.Protocol, c.Locked, c.SyncHits, truncStr(c.SyncVariant, 14), c.FECPassRate, c.ModalSpacing, c.Score)
	}
	verdict := idr.Winner
	if idr.Inconclusive {
		verdict = "INCONCLUSIVE (best guess: " + idr.Winner + ")"
	}
	fmt.Fprintf(w, "identify: → %s  (confidence %.2f)\n", verdict, idr.Confidence)
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
