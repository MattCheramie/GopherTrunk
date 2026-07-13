package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runReplay is the entry point for `gophertrunk replay`. It runs an offline
// raw IQ capture through the production receiver + control-channel chain for
// any protocol GopherTrunk decodes and prints lock / grant / decode-error
// results plus — for P25 Phase 1 — the deep demod diagnostics (per-frame NID
// breakdown, FSW-correlation landscape, true-symbol eye, receiver-state
// series) that issues #275/#402 produced.
//
// All decode/analysis logic lives in the shared internal/siglab engine; this
// command is a thin front-end that maps flags onto a siglab.Config and renders
// the structured Result. What it decodes is what the daemon would decode, so a
// replay lock implies an on-air lock and a replay failure makes the offline
// capture a reproducible fixture.
//
// Usage:
//
//	gophertrunk replay -in <path> [-format u8|f32] [-sample-rate Hz]
//	                  [-protocol p25p1|dmr-tier3|tetra|…] [-demod c4fm|cqpsk]
//	                  [-tune-hz Hz | -auto-tune] [-conjugate] [-diag]
func runReplay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	in := fs.String("in", "", "raw IQ input file (required)")
	format := fs.String("format", "u8", "sample format: u8 (rtl_sdr 8-bit unsigned interleaved IQ) | f32 (GNU Radio cfile, interleaved float32) | wav (2-channel 16-bit baseband WAV — SDRtrunk/SDR++/GopherTrunk narrowband recording; sample rate is read from the header)")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	demod := fs.String("demod", "c4fm", "P25 Phase 1 demod mode: c4fm | cqpsk")
	protocolFlag := fs.String("protocol", "p25p1", "decoder to run: p25p1 | p25-phase2 | dmr | dmr-tier2 | nxdn | dpmr | edacs | motorola | ltr | mpt1327 | tetra | ysf | dstar (aliases: dmr-tier3)")
	conjugate := fs.Bool("conjugate", false, "conjugate IQ (negate Q) before channelization, to decode a spectrum-inverted / I-Q-swapped front-end (issue #264)")
	freq := fs.Uint64("freq", 0, "informational only: the capture's nominal centre frequency in Hz")
	nidSearchSpan := fs.Int("nid-search-span", 0, "P25 NID-alignment search radius in dibits (0 = production default; widen to bisect a stubborn capture per issue #275)")
	diag := fs.Bool("diag", false, "collect the demod-quality diagnostic report (symbol histogram + P25 FSW landscape + soft-sample eye) and print it at EOF")
	enableDDA := fs.Bool("dda", false, "enable the experimental decision-directed AFC on the P25 C4FM path (off by default; see issue #402)")
	enableAdaptiveSlicer := fs.Bool("adaptive-slicer", false, "enable the adaptive C4FM slicer on the P25 C4FM path (off by default; see issue #402)")
	enableSoftSync := fs.Bool("soft-sync", false, "widen the P25 FSW correlator and admit looser sync words whose TSBK CRC corroborates the NID — extends lock reach on marginal-SNR captures without risking a false lock (off by default; see issue #771)")
	iqCorrect := fs.Bool("iq-correct", false, "apply blind I/Q-imbalance correction to the raw IQ before decimation (off by default; see issue #402)")
	tuneHzFlag := fs.Float64("tune-hz", 0, "frequency-shift the capture so a channel at +tune-hz lands at 0 Hz before demod (0 = no shift)")
	autoTune := fs.Bool("auto-tune", false, "find the control channel by trying the ranked carrier candidates and keeping the best lock — handles an off-centre / non-dominant control channel in a wideband capture (overrides -tune-hz)")
	recordDDC := fs.String("record-ddc", "", "also write the post-DDC narrowband IQ (the exact channelized stream the receiver decodes) to this 2-channel 16-bit baseband WAV — shrinks a fat 2.5/10 MS/s capture into a small shareable fixture that `replay -format wav` decodes identically")
	outFormat := fs.String("out-format", "text", "output format: text | json | jsonl | yaml | csv | csv-events")
	out := fs.String("out", "", "write structured output to this file (default: stdout for non-text)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk replay — decode a raw IQ capture file offline (any protocol).

USAGE:
  gophertrunk replay -in <path> [-format u8|f32] [-sample-rate Hz]
                    [-protocol p25p1|dmr|tetra|…] [-demod c4fm|cqpsk]
                    [-tune-hz Hz | -auto-tune] [-conjugate] [-diag]

EXAMPLES:
  # rtl_sdr capture of a P25 control channel, with the deep demod report
  gophertrunk replay -in mt_anakie.bin -sample-rate 2048000 -demod c4fm -diag

  # Wideband cfile whose control channel is off-centre: auto-tune to 0 Hz
  gophertrunk replay -in mmr-s9.cfile -format f32 -sample-rate 2400000 -auto-tune

  # DMR Tier III control channel from a wideband cfile
  gophertrunk replay -in dmr.cfile -format f32 -sample-rate 2400000 -protocol dmr -auto-tune

  # Any other protocol GopherTrunk decodes (e.g. TETRA), with structured export
  gophertrunk replay -in tetra.cfile -format f32 -sample-rate 2400000 -protocol tetra -out-format json -out out.json

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("replay")

	if *in == "" {
		fs.Usage()
		rep.Fatalf(2, "-in is required")
	}
	if *sampleRate <= 0 {
		rep.Fatalf(2, "-sample-rate must be > 0")
	}
	if *nidSearchSpan < 0 {
		rep.Fatalf(2, "-nid-search-span must be >= 0")
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

	// Run the production decoder at debug log level so the receiver's internal
	// diagnostics (`nid corrected` / `nid parse failed` / `at_boundary`, the
	// FSW-miss throttle) still surface live on stderr — the value the operator
	// is here for. The structured analysis is collected into the Result.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := siglab.Config{
		Protocol:             proto,
		SystemName:           "replay",
		FrequencyHz:          uint32(*freq),
		SampleRateHz:         *sampleRate,
		Format:               sampleFormat,
		TuneHz:               *tuneHzFlag,
		AutoTune:             *autoTune,
		Conjugate:            *conjugate,
		IQCorrect:            *iqCorrect,
		CollectIQDiag:        *diag,
		DemodMode:            *demod,
		RecordDDCPath:        *recordDDC,
		NIDSearchSpan:        *nidSearchSpan,
		EnableDDA:            *enableDDA,
		EnableAdaptiveSlicer: *enableAdaptiveSlicer,
		EnableSoftSync:       *enableSoftSync,
		// P25 replay always wants the per-frame CCStats + receiver-state series
		// (the historical EOF summary + per-second state log); -diag adds the
		// dibit/soft-eye landscape on top.
		CollectReceiverState: true,
		Log:                  logger,
	}

	// Under -auto-tune, try the ranked carrier candidates and keep the best
	// lock — so a control channel that is off-centre and not the loudest carrier
	// in a wideband capture is still found (a single dominant-carrier estimate
	// would miss it). -tune-hz (without -auto-tune) still forces one offset.
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
}
