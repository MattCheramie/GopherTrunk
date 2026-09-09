package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
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
	in := fs.String("in", "", "raw IQ input file, or - for standard input (required). Piping stdin lets an external IQ source feed the decoder, e.g. `iq-source | gophertrunk replay -in - -format f32 -sample-rate 2400000` (issue #314). stdin is a one-way stream, so -auto-tune (which must seek) is not supported with -in -; use -tune-hz.")
	format := fs.String("format", "u8", "sample format: u8 (rtl_sdr 8-bit unsigned interleaved IQ) | f32 (GNU Radio cfile, interleaved float32; aliases cf32/fc32 — the SoapySDR/OpenWebRX+ spelling) | cs16/sc16 (interleaved little-endian int16 IQ, the .raw/.cs16 SDR capture format) | wav (2-channel 16-bit baseband WAV — SDRtrunk/SDR++/GopherTrunk narrowband recording; sample rate is read from the header)")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	demod := fs.String("demod", "c4fm", "P25 Phase 1 demod mode: c4fm | cqpsk")
	protocolFlag := fs.String("protocol", "p25p1", "decoder to run: p25p1 | p25-phase2 | dmr | dmr-tier2 | nxdn | dpmr | edacs | motorola | ltr | mpt1327 | tetra | ysf | dstar (aliases: dmr-tier3)")
	conjugate := fs.Bool("conjugate", false, "conjugate IQ (negate Q) before channelization, to decode a spectrum-inverted / I-Q-swapped front-end (issue #264)")
	freq := fs.Uint64("freq", 0, "informational only: the capture's nominal centre frequency in Hz")
	nidSearchSpan := fs.Int("nid-search-span", 0, "P25 NID-alignment search radius in dibits (0 = production default; widen to bisect a stubborn capture per issue #275)")
	diag := fs.Bool("diag", false, "collect the demod-quality diagnostic report (symbol histogram + P25 FSW landscape + soft-sample eye) and print it at EOF")
	enableDDA := fs.Bool("dda", false, "enable the experimental decision-directed AFC on the P25 C4FM path (off by default; see issue #402)")
	enableAdaptiveSlicer := fs.Bool("adaptive-slicer", false, "enable the adaptive C4FM slicer on the P25 C4FM path (off by default; see issue #402)")
	enableDCBlock := fs.Bool("dc-block", false, "strip the front-end zero-IF DC spur before the C4FM discriminator (decodes on-channel HackRF voice captures whose centre spike otherwise zeroes the decode)")
	enableSoftSync := fs.Bool("soft-sync", false, "widen the P25 FSW correlator and admit looser sync words whose TSBK CRC corroborates the NID — extends lock reach on marginal-SNR captures without risking a false lock (off by default; see issue #771)")
	iqCorrect := fs.Bool("iq-correct", false, "apply blind I/Q-imbalance correction to the raw IQ before decimation (off by default; see issue #402)")
	tuneHzFlag := fs.Float64("tune-hz", 0, "frequency-shift the capture so a channel at +tune-hz lands at 0 Hz before demod (0 = no shift)")
	autoTune := fs.Bool("auto-tune", false, "find the control channel by trying the ranked carrier candidates and keeping the best lock — handles an off-centre / non-dominant control channel in a wideband capture (overrides -tune-hz)")
	recordDDC := fs.String("record-ddc", "", "also write the post-DDC narrowband IQ (the exact channelized stream the receiver decodes) to this 2-channel 16-bit baseband WAV — shrinks a fat 2.5/10 MS/s capture into a small shareable fixture that `replay -format wav` decodes identically")
	outFormat := fs.String("out-format", "text", "output format: text | json | jsonl | yaml | csv | csv-events")
	out := fs.String("out", "", "write structured output to this file (default: stdout for non-text)")
	recordVoice := fs.Bool("record-voice", false, "decode and record voice for the capture's grants, writing .wav/.raw/.json under -out-dir. Wires the production voice path (engine → composer → recorder) onto the decode, following each grant on the decoded (same) carrier. Best for conventional systems (DMR Tier II / IPSC, TETRA) whose voice rides the decoded carrier.")
	outDir := fs.String("out-dir", "", "recordings directory for -record-voice (required with it)")
	audioOut := fs.String("audio-out", "", "stream decoded voice audio as continuous raw signed 16-bit little-endian mono PCM at 8000 Hz to this file/FIFO, or - for stdout — so an external consumer (OpenWebRX+, aplay, sox) plays calls live instead of waiting for per-call WAVs (issue #314). Wires the same voice path as -record-voice (usable with or without it) and shares its constraints: requires -freq, no -auto-tune. With -audio-out -, give -out a file path so the decode result doesn't interleave with the PCM on stdout. Opening a FIFO blocks until a reader attaches.")
	voiceHangtimeMs := fs.Int("voice-hangtime-ms", 3500, "end-of-transmission hangtime for -record-voice, in ms")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk replay — decode a raw IQ capture file offline (any protocol).

USAGE:
  gophertrunk replay -in <path> [-format u8|f32] [-sample-rate Hz]
                    [-protocol p25p1|dmr|tetra|…] [-demod c4fm|cqpsk]
                    [-tune-hz Hz | -auto-tune] [-conjugate] [-diag]

EXAMPLES:
  # rtl_sdr capture of a P25 control channel, with the deep demod report
  gophertrunk replay -in mt_anakie.bin -sample-rate 2048000 -demod c4fm -diag

  # Decode a raw IQ stream piped in on stdin (e.g. from OpenWebRX+ / rtl_sdr -),
  # emitting each decoded event to stdout as a JSONL line the moment it happens
  iq-source | gophertrunk replay -in - -format f32 -sample-rate 2400000 \
                    -protocol p25p1 -tune-hz -12500 -out-format jsonl

  # OpenWebRX+ hand-off: a single channelized cf32 stream on stdin (any rate ≥
  # the protocol's channel rate is decimated, a narrower one is interpolated up),
  # live voice PCM (s16le mono 8 kHz) on stdout, decoded events to a JSONL file
  owrx-iq-source | gophertrunk replay -in - -format cf32 -sample-rate 48000 \
                    -protocol dmr-tier2 -freq 438900000 \
                    -audio-out - -out-format jsonl -out events.jsonl

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
		EnableDCBlock:        *enableDCBlock,
		EnableSoftSync:       *enableSoftSync,
		// P25 replay always wants the per-frame CCStats + receiver-state series
		// (the historical EOF summary + per-second state log); -diag adds the
		// dibit/soft-eye landscape on top.
		CollectReceiverState: true,
		Log:                  logger,
	}

	// Optional voice recording / live audio streaming: wire the production voice
	// path (engine → composer → recorder) onto the decode's grant bus +
	// channelized-IQ tap, so grants become .wav/.raw/.json recordings
	// (-record-voice) and/or a continuous live PCM stream (-audio-out).
	// -auto-tune runs several candidate decodes internally, which would each
	// drive the voice rig, so it is disallowed here — use -tune-hz for a fixed
	// offset.
	var voiceRig *replayVoiceRig
	var audioSink *pcmStreamWriter
	var audioFile *os.File
	if *recordVoice || *audioOut != "" {
		if *recordVoice && *outDir == "" {
			rep.Fatalf(2, "-record-voice requires -out-dir")
		}
		if *autoTune {
			rep.Fatalf(2, "-record-voice/-audio-out do not support -auto-tune; use -tune-hz for a fixed offset")
		}
		if *freq == 0 {
			rep.Fatalf(2, "-record-voice/-audio-out require -freq (the carrier frequency in Hz): grants carry the decode frequency, and a grant with freq 0 is dropped by the engine (no voice decodes)")
		}
		if *audioOut == "-" && *out == "" {
			rep.Fatalf(2, "-audio-out - streams raw PCM on stdout, so the decode result needs its own destination: give -out a file path")
		}
		if *audioOut != "" {
			w := os.Stdout
			if *audioOut != "-" {
				// os.Create also opens an existing FIFO for writing, blocking
				// until the consumer attaches — the pipeline shape OWRX+ uses.
				audioFile, err = os.Create(*audioOut)
				if err != nil {
					rep.Fatalf(1, "-audio-out: %v", err)
				}
				w = audioFile
			}
			audioSink = newPCMStreamWriter(w)
		}
		recDir := ""
		if *recordVoice {
			recDir = *outDir
		}
		ddcRate := *sampleRate
		if target := ccdecoder.DDCTargetForProtocol(proto); *sampleRate != target {
			ddcRate = target
		}
		voiceRig, err = setupReplayVoice(recDir, ddcRate, time.Duration(*voiceHangtimeMs)*time.Millisecond, audioSink, logger)
		if err != nil {
			rep.Fatal(1, err)
		}
		cfg.Bus = voiceRig.bus
		cfg.OnChannelIQ = voiceRig.onChannelIQ
	}

	// JSONL streams live: the output is opened up-front and every decoded event
	// is written the moment it is observed, so an unbounded stdin pipe (an
	// external IQ source that never EOFs; issue #314) delivers events as they
	// happen instead of a batch dump when the pipe finally closes. The trailing
	// summary line still lands at EOF, keeping the output byte-identical to the
	// batch export for file inputs. -auto-tune stays on the batch path: it runs
	// several candidate decodes whose failed attempts must not leak event lines.
	var jsonlStreamer *siglab.JSONLEventStreamer
	var jsonlOut *os.File
	if of == siglab.FormatJSONL && !*autoTune {
		w := os.Stdout
		if *out != "" {
			jsonlOut, err = os.Create(*out)
			if err != nil {
				rep.Fatalf(1, "create %s: %v", *out, err)
			}
			w = jsonlOut
		}
		jsonlStreamer = siglab.NewJSONLEventStreamer(w)
	}

	// Under -auto-tune, try the ranked carrier candidates and keep the best
	// lock — so a control channel that is off-centre and not the loudest carrier
	// in a wideband capture is still found (a single dominant-carrier estimate
	// would miss it). -tune-hz (without -auto-tune) still forces one offset.
	var res *siglab.Result
	if *autoTune {
		res, err = siglab.RunAutoTuneMulti(*in, cfg, 0)
	} else if jsonlStreamer != nil {
		res, err = siglab.RunStream(*in, cfg, jsonlStreamer.OnEvent)
	} else {
		res, err = siglab.Run(*in, cfg)
	}
	if voiceRig != nil {
		// Let in-flight calls reach hangtime end + finalize their recordings,
		// then tear the rig down. Done regardless of the decode error so a
		// partial capture still flushes what it recorded.
		voiceRig.finalize()
	}
	if audioFile != nil {
		_ = audioFile.Close()
	}
	if err != nil {
		rep.Fatal(1, err)
	}

	if jsonlStreamer != nil {
		// The event lines already streamed; append the summary line and close.
		w := io.Writer(os.Stdout)
		if jsonlOut != nil {
			w = jsonlOut
		}
		if serr := siglab.WriteJSONLSummary(w, res); serr != nil {
			rep.Fatal(1, serr)
		}
		if serr := jsonlStreamer.Err(); serr != nil {
			rep.Fatal(1, serr)
		}
		if jsonlOut != nil {
			if cerr := jsonlOut.Close(); cerr != nil {
				rep.Fatal(1, cerr)
			}
		}
		return
	}
	if err := emitResult(res, of, *out); err != nil {
		rep.Fatal(1, err)
	}
}
