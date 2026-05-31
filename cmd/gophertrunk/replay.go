package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/dsp"
	"github.com/MattCheramie/GopherTrunk/internal/events"
	p25phase1 "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1"
	p25phase1rx "github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1/receiver"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr/rtlsdr"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// runReplay is the entry point for `gophertrunk replay`. It runs an
// offline raw IQ capture file through the real P25 Phase 1 receiver +
// control-channel chain and prints lock / grant / decode-error events
// plus the per-frame NID-decoder diagnostics — including the
// at_boundary flag the bounded-search fix emits (issue #275). The
// effective-baud summary at EOF self-diagnoses a mislabeled capture
// (the symptom that made anakie_short.zip unusable as ground truth):
// if numDibits / (numSamples / sampleRate) ≠ ~4800, the file's true
// sample rate is not what the operator passed via -sample-rate.
//
// Usage:
//
//	gophertrunk replay -in <path> [-format u8|f32] [-sample-rate Hz]
//	                  [-demod c4fm|cqpsk] [-freq Hz]
//
// The decoder runs at debug log level so `nid corrected`, `nid parse
// failed`, and the new `at_boundary` field all surface in the output.
// Reuses the production receiver + control-channel constructors — what
// it decodes is what the daemon would decode, so a replay-lock
// implies an on-air lock and a replay-fail makes the offline capture a
// reproducible test fixture.
func runReplay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	in := fs.String("in", "", "raw IQ input file (required)")
	format := fs.String("format", "u8", "sample format: u8 (rtl_sdr 8-bit unsigned interleaved IQ) | f32 (GNU Radio cfile, interleaved float32)")
	sampleRate := fs.Float64("sample-rate", 2_400_000, "IQ sample rate in Hz")
	demod := fs.String("demod", "c4fm", "P25 Phase 1 demod mode: c4fm | cqpsk")
	freq := fs.Uint64("freq", 0, "informational only: the capture's nominal centre frequency in Hz")
	// Issue #275 bisect knob. The default ±6 grid is the production
	// value; widening to ±12/±18/±36 on a stubborn capture tells a
	// span-bounded failure (errs drop at the new optimum) from a
	// demod-quality-bounded one (errs stay at the BCH(63,16,11)
	// correction ceiling regardless of alignment). Both acceptance
	// tiers (BCH+parity + TSBK CRC) reject wrong alignments at any
	// span, so widening cannot manufacture a false lock.
	nidSearchSpan := fs.Int("nid-search-span", p25phase1.NIDSearchSpan, "NID-alignment search radius in dibits (default matches the production ccdecoder; widen to bisect a stubborn capture per issue #275)")
	// Issue #275 Phase B knob — after Phase A's widening ruled out
	// alignment, this surfaces what the NID-failure diag cannot: the
	// dibit-value histogram and the per-rotation FSW-correlation
	// landscape across the whole capture. Off by default since it
	// allocates an O(numDibits) buffer.
	diag := fs.Bool("diag", false, "print a demod-quality diagnostic report (dibit histogram + per-rotation FSW correlation landscape) at EOF")
	// Issue #402: the decision-directed AFC is off by default (it can
	// stably false-lock and broke CC lock on the original capture), so
	// the daemon and replay run CoarseAFC-alone. This knob re-enables it
	// for A/B experimentation on a capture without turning it on in
	// production. C4FM only.
	enableDDA := fs.Bool("dda", false, "enable the experimental decision-directed AFC on the C4FM path (off by default; see issue #402)")
	// Issue #402: the adaptive C4FM slicer is off by default (the fixed
	// slicer outperformed it on the original capture's closed/asymmetric
	// eye). This knob enables it for A/B experimentation. C4FM only.
	enableAdaptiveSlicer := fs.Bool("adaptive-slicer", false, "enable the adaptive C4FM slicer on the C4FM path (off by default; see issue #402)")
	// Issue #402: blind I/Q-imbalance correction on the raw IQ before the DDC.
	// Off by default; the chain audit found an uncorrected RTL-SDR imbalance is
	// the leading explanation for the asymmetric demodulated eye on MMR Site 9.
	iqCorrect := fs.Bool("iq-correct", false, "apply blind I/Q-imbalance correction to the raw IQ before decimation (off by default; see issue #402)")
	// Channel tuning for an off-centre capture. The live pipeline gets a
	// channel centred at 0 Hz from the SDR tuner; a wideband file replay
	// must shift the channel down itself. -tune-hz applies a fixed offset;
	// -auto-tune estimates the dominant carrier from a prefix of the file.
	// Needed for captures (e.g. MMR Site 9) whose control channel sits a
	// few hundred Hz off centre — at 4800 baud even ~50 Hz is several
	// degrees of differential rotation per symbol, enough to break lock.
	tuneHzFlag := fs.Float64("tune-hz", 0, "frequency-shift the capture so a channel at +tune-hz lands at 0 Hz before demod (0 = no shift)")
	autoTune := fs.Bool("auto-tune", false, "estimate the dominant carrier offset from the start of the file and tune it to 0 Hz (overrides -tune-hz)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk replay — decode a raw IQ capture file offline.

USAGE:
  gophertrunk replay -in <path> [-format u8|f32] [-sample-rate Hz] [-demod c4fm|cqpsk]

EXAMPLES:
  # rtl_sdr capture of a P25 control channel
  gophertrunk replay -in mt_anakie.bin -sample-rate 2048000 -demod c4fm

  # GNU Radio float32 cfile of an LSM simulcast site
  gophertrunk replay -in cbd.cfile -format f32 -sample-rate 960000 -demod cqpsk

  # Wideband capture whose control channel is off-centre: auto-estimate the
  # carrier and tune it to 0 Hz before the down-converter (the SDR tuner does
  # this live; a recorded file does not). MMR Site 9 sits ~+37 kHz off centre.
  gophertrunk replay -in mmr-s9.cfile -format f32 -sample-rate 2400000 -demod c4fm -auto-tune

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if *in == "" {
		fmt.Fprintln(os.Stderr, "replay: -in is required")
		fs.Usage()
		os.Exit(2)
	}
	if *sampleRate <= 0 {
		fmt.Fprintln(os.Stderr, "replay: -sample-rate must be > 0")
		os.Exit(2)
	}
	demodMode, ok := p25phase1rx.ParseDemodMode(*demod)
	if !ok {
		fmt.Fprintf(os.Stderr, "replay: unknown -demod %q (want c4fm or cqpsk)\n", *demod)
		os.Exit(2)
	}
	if *nidSearchSpan <= 0 {
		fmt.Fprintln(os.Stderr, "replay: -nid-search-span must be > 0")
		os.Exit(2)
	}

	decode, bytesPerSample, err := pickSampleDecoder(*format)
	if err != nil {
		fmt.Fprintln(os.Stderr, "replay:", err)
		os.Exit(2)
	}

	f, err := os.Open(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: open %s: %v\n", *in, err)
		os.Exit(1)
	}
	defer f.Close()

	// Resolve the tuning offset. -auto-tune estimates the dominant carrier
	// from a prefix of the file (then rewinds so the prefix is still
	// decoded); -tune-hz is the manual override. Estimation runs on the
	// raw IQ, before any decimation, so the offset is in input-rate Hz.
	tuneHz := *tuneHzFlag
	if *autoTune {
		est, err := estimateCaptureCarrierHz(f, decode, bytesPerSample, *sampleRate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "replay: auto-tune failed: %v\n", err)
			os.Exit(1)
		}
		tuneHz = est
		fmt.Fprintf(os.Stderr, "replay: auto-tune  carrier_offset_hz=%.1f\n", tuneHz)
	}

	// Logger at debug so `nid corrected` (with at_boundary), `nid
	// parse failed` (with diag), and the FSW miss throttle all
	// surface — the diagnostic value the operator is here for.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	bus := events.NewBus(1024)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()

	// Mirror ccdecoder/pipelines.go: restrict the rotation set on the
	// C4FM path so the search cannot converge on a non-physical
	// rot 1 / rot 3 miscorrection (issue #275 post-#321).
	rotations := p25phase1.RotationsAll
	if demodMode == p25phase1rx.DemodC4FM {
		rotations = p25phase1.RotationsC4FM
	}
	cc := p25phase1.New(p25phase1.Options{
		Bus:           bus,
		Log:           logger,
		SystemName:    "replay",
		FrequencyHz:   uint32(*freq),
		Rotations:     rotations,
		NIDSearchSpan: *nidSearchSpan,
	})

	// Mirror the production ccdecoder DDC for the C4FM path (issue
	// #402 Phase 2): the daemon decimates raw SDR IQ down to ~48 kHz
	// before the receiver sees it; without the same step here replay
	// feeds the receiver wideband IQ (e.g. 2.4 MHz), the matched
	// filter sizes for ~500 samples per symbol, and AFC / AGC time
	// constants are off by ~50× compared to what runs in production.
	// A capture that fails in the daemon would then decode (or fail)
	// differently in replay, defeating the whole point of using
	// replay as a reproducer. The DDC is only enabled when the demod
	// is C4FM and the supplied -sample-rate exceeds the production
	// target — CQPSK and already-channelized captures are unchanged.
	// Decimate only on the C4FM path when the input exceeds the production
	// target; CQPSK and already-channelised captures keep their rate. The
	// tuning shift (if any) applies on either path — it just centres the
	// channel and never changes the rate.
	ddcTarget := *sampleRate
	if demodMode == p25phase1rx.DemodC4FM && *sampleRate > ccdecoder.DDCTargetRateHz {
		ddcTarget = ccdecoder.DDCTargetRateHz
	}
	var ddc *ccdecoder.Downconverter
	receiverRate := *sampleRate
	if tuneHz != 0 || ddcTarget < *sampleRate {
		ddc = ccdecoder.NewDownconverterWithOffset(*sampleRate, ddcTarget, tuneHz)
		receiverRate = ddc.OutRateHz()
	}

	// Surface the active configuration the same way the ccdecoder
	// pipeline does, so the replay log line is directly comparable
	// to a daemon's startup line — and a non-default span (the
	// bisect knob) is visible without re-reading the command.
	fmt.Fprintf(os.Stderr, "replay: p25/phase1 configured  demod=%s  rotations=%v  nid_search_span=%d  nid_accept_errs=%d  nid_marginal_max=%d  dda=%t  adaptive_slicer=%t  iq_correct=%t\n",
		*demod, rotations, *nidSearchSpan, p25phase1.NIDAcceptErrs, p25phase1.NIDMarginalMaxErrs, *enableDDA, *enableAdaptiveSlicer, *iqCorrect)
	if ddc != nil {
		fmt.Fprintf(os.Stderr, "replay: ddc enabled  sdr_rate_hz=%g  pipeline_rate_hz=%g\n",
			*sampleRate, receiverRate)
	} else {
		fmt.Fprintf(os.Stderr, "replay: ddc bypassed  pipeline_rate_hz=%g  (sample rate already at or below the C4FM target, or demod=cqpsk)\n",
			receiverRate)
	}

	var dibitCount int64
	var diagAcc *iqDiag
	if *diag {
		diagAcc = &iqDiag{}
	}
	// Blind I/Q-imbalance correction on the raw IQ (issue #402), opt-in.
	var iqCorrector *rtlsdr.IQImbalanceCorrector
	if *iqCorrect {
		iqCorrector = rtlsdr.NewIQImbalanceCorrector()
	}
	rxOpts := p25phase1rx.Options{
		SampleRateHz:              receiverRate,
		DeviationHz:               1800.0,
		DemodMode:                 demodMode,
		EnableDecisionDirectedAFC: *enableDDA,
		EnableAdaptiveC4FMSlicer:  *enableAdaptiveSlicer,
		DibitSink: func(dibits []uint8, baseIdx int) {
			dibitCount += int64(len(dibits))
			if diagAcc != nil {
				diagAcc.observe(dibits)
			}
			cc.Process(dibits, baseIdx)
		},
	}
	if diagAcc != nil {
		rxOpts.SoftSink = diagAcc.observeSoft
	}
	rx := p25phase1rx.New(rxOpts)

	// Drain bus events to stdout in the background so they print
	// interleaved with the decoder log going to stderr.
	doneEvents := make(chan struct{})
	var stats replayStats
	go func() {
		defer close(doneEvents)
		for ev := range sub.C {
			handleEvent(ev, &stats)
		}
	}()

	// Per-second receiver-state observer (issue #402 Phase 2): after
	// every secondsPerStateLog seconds of wall-clock IQ have flowed
	// through the receiver, log the AFC bias, symbol-AGC level/target,
	// and Mueller-Müller clock state. Lets the reporter (and us) see
	// whether one stage's internal state slowly drifts after the CC
	// locks, which is what the disproven-DC-spike-hypothesis pivot now
	// points at. The cadence is measured in IQ-stream seconds (totalSamples
	// / sampleRate), not wall clock — so the same capture produces the
	// same log lines regardless of how fast replay can chew through it.
	const stateLogIntervalSec = 1.0
	var nextStateLogAt float64 = stateLogIntervalSec
	logReceiverState := func(at float64) {
		// afc_hz_est is the TRUE carrier offset (rx.AFCOffsetHz), which
		// divides out the matched filter's sps DC-gain. The earlier
		// AFCBias·Fs/(2π) form over-reported by ≈sps (≈10× at 48 kHz /
		// 4800 baud) and sent issue #402 chasing a phantom ~10 kHz
		// offset that was really ~1 kHz. afc_bias_rad_per_sample is the
		// raw matched-output-unit value behind it.
		//
		// slicer_levels are the adaptive slicer's tracked −3/−1/+1/+3
		// levels; slicer_thresholds are the actual decision boundaries
		// (neg-outer/zero/pos-outer) the slicer decides on (issue #402: the
		// OP asked to see the thresholds, not just the levels — on a spread
		// eye the variance-aware boundary sits below the level midpoint).
		// Both read zero on the fixed-slicer path (no -adaptive-slicer).
		lv := rx.SlicerLevels()
		th := rx.SlicerThresholds()
		fmt.Fprintf(os.Stderr,
			"replay: receiver state  t=%.2fs  afc_bias_rad_per_sample=%.6g  afc_hz_est=%.3f  agc_level=%.6g  agc_target=%.6g  agc_gain=%.4g  mm_mu=%.4f  mm_sps=%.2f  dda_active=%t  dda_rearms=%d  slicer_levels=[%.4f %.4f %.4f %.4f]  slicer_thresholds=[%.4f %.4f %.4f]\n",
			at,
			rx.AFCBiasRadPerSample(),
			rx.AFCOffsetHz(),
			rx.AGCLevel(),
			rx.AGCTarget(),
			ratioOrZero(rx.AGCTarget(), rx.AGCLevel()),
			rx.MMClockMu(),
			rx.MMClockSPS(),
			rx.DDAActive(),
			rx.DDARearms(),
			lv[0], lv[1], lv[2], lv[3],
			th[0], th[1], th[2])
	}

	const chunkSamples = 8192
	buf := make([]byte, chunkSamples*bytesPerSample)
	samples := make([]complex64, chunkSamples)
	var ddcOut []complex64 // reused across the read loop
	var totalSamples int64
	for {
		n, rerr := io.ReadFull(f, buf)
		if n > 0 {
			pairBytes := bytesPerSample
			pairs := n / pairBytes
			if pairs*pairBytes != n {
				fmt.Fprintf(os.Stderr, "replay: trailing %d bytes ignored (not a whole IQ pair at %d bytes/sample)\n", n-pairs*pairBytes, pairBytes)
			}
			if pairs > len(samples) {
				samples = make([]complex64, pairs)
			}
			decode(buf[:pairs*pairBytes], samples[:pairs])
			feed := samples[:pairs]
			// Measure the raw I/Q imbalance for the diag report, then (opt-in)
			// correct it — both on the raw IQ, before the DDC (issue #402).
			if diagAcc != nil {
				diagAcc.observeIQ(feed)
			}
			if iqCorrector != nil {
				iqCorrector.Process(feed)
			}
			if ddc != nil {
				ddcOut = ddc.Process(ddcOut, feed)
				feed = ddcOut
			}
			rx.Process(feed)
			totalSamples += int64(pairs)

			// Throttle the state log on IQ-stream time, not wall
			// clock. *sampleRate is the input rate even when the
			// DDC is active — that's the rate totalSamples counts.
			if t := float64(totalSamples) / *sampleRate; t >= nextStateLogAt {
				logReceiverState(t)
				nextStateLogAt = t + stateLogIntervalSec
			}
		}
		if errors.Is(rerr, io.EOF) || errors.Is(rerr, io.ErrUnexpectedEOF) {
			break
		}
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "replay: read %s: %v\n", *in, rerr)
			os.Exit(1)
		}
	}

	// Close the bus to release the event-drainer goroutine, then
	// wait for it to finish so the final stats are accurate.
	bus.Close()
	<-doneEvents

	printSummary(filepath.Base(*in), totalSamples, *sampleRate, dibitCount, stats, cc.Stats())
	if diagAcc != nil {
		diagAcc.printReport(os.Stdout)
	}
}

// ratioOrZero returns num / den, or 0 if den is too small to divide
// safely. Used by the state log so a not-yet-seeded AGC level
// (level=0) renders as gain=0 rather than panicking on divide-by-zero
// or printing +Inf.
func ratioOrZero(num, den float64) float64 {
	if den < 1e-12 && den > -1e-12 {
		return 0
	}
	return num / den
}

// replayStats accumulates bus events seen during the replay so the
// EOF summary can render lock / grant / decode-error totals.
type replayStats struct {
	locked         bool
	nac            uint16
	grants         int
	grantFreqs     map[uint32]int
	decodeErrors   map[events.Stage]int
	otherEventKind map[events.Kind]int
}

func handleEvent(ev events.Event, s *replayStats) {
	switch ev.Kind {
	case events.KindCCLocked:
		if ls, ok := ev.Payload.(p25phase1.LockState); ok {
			s.locked = true
			s.nac = ls.NAC
			fmt.Printf("replay: cc.locked  nac=%#x  freq=%d  duid=%s\n", ls.NAC, ls.FrequencyHz, ls.DUID)
		}
	case events.KindGrant:
		if g, ok := ev.Payload.(trunking.Grant); ok {
			s.grants++
			if s.grantFreqs == nil {
				s.grantFreqs = make(map[uint32]int)
			}
			s.grantFreqs[g.FrequencyHz]++
			fmt.Printf("replay: grant      tg=%d  src=%d  ch=%d/%d  freq=%d  enc=%v  emer=%v\n",
				g.GroupID, g.SourceID, g.ChannelID, g.ChannelNum, g.FrequencyHz, g.Encrypted, g.Emergency)
		}
	case events.KindDecodeError:
		if de, ok := ev.Payload.(events.DecodeError); ok {
			if s.decodeErrors == nil {
				s.decodeErrors = make(map[events.Stage]int)
			}
			s.decodeErrors[de.Stage]++
		}
	default:
		if s.otherEventKind == nil {
			s.otherEventKind = make(map[events.Kind]int)
		}
		s.otherEventKind[ev.Kind]++
	}
}

// printSummary writes the EOF report — what the operator pastes into
// the GitHub issue alongside the live log. Includes the effective-baud
// self-diagnostic the anakie_short.zip mislabel taught us we need, plus
// the per-frame outcome breakdown the ControlChannel keeps in CCStats
// (issue #402 Phase 2: lets the reporter answer "did anything decode?"
// without scrolling through every debug line).
func printSummary(name string, samples int64, sampleRate float64, dibits int64, s replayStats, cc p25phase1.CCStats) {
	fmt.Fprintln(os.Stdout, "----")
	duration := float64(samples) / sampleRate
	fmt.Fprintf(os.Stdout, "replay: %s — %d samples (%.2fs at %.0f Hz), %d dibits emitted\n",
		name, samples, duration, sampleRate, dibits)

	const expectedBaud = 4800.0
	if duration > 0 && dibits > 0 {
		effective := float64(dibits) / duration
		dev := (effective - expectedBaud) / expectedBaud * 100
		warning := ""
		if math.Abs(dev) > 2 {
			warning = "  (>2% — capture sample rate may not match -sample-rate)"
		}
		fmt.Fprintf(os.Stdout, "replay: effective baud %.1f (expected %.0f, deviation %+.1f%%)%s\n",
			effective, expectedBaud, dev, warning)
	}

	if s.locked {
		fmt.Fprintf(os.Stdout, "replay: locked  nac=%#x\n", s.nac)
	} else {
		fmt.Fprintln(os.Stdout, "replay: did NOT lock the control channel")
	}
	if s.grants > 0 {
		fmt.Fprintf(os.Stdout, "replay: %d grant(s) across %d frequencies\n", s.grants, len(s.grantFreqs))
	}

	// Per-frame outcome breakdown from CCStats. NID-tier counts split
	// "passed BCH+parity cleanly" from "needed TSBK corroboration",
	// and TSBK counts split the failure mode (trellis vs CRC). On a
	// healthy site every FSW hit becomes NIDTrusted + TSBKDecoded;
	// the issue-#402 shape is high NIDFailed + high TSBKCRCFailed
	// (gross dibit corruption that BCH can't recover and Viterbi
	// barely can).
	nidAttempts := cc.NIDTrusted + cc.NIDMarginal + cc.NIDFailed
	tsbkAttempts := cc.TSBKDecoded + cc.TSBKTrellisFailed + cc.TSBKCRCFailed
	fmt.Fprintf(os.Stdout, "replay: nid   trusted=%d  marginal=%d  uncorrectable=%d  (of %d FSW-hit attempts; %s ok)\n",
		cc.NIDTrusted, cc.NIDMarginal, cc.NIDFailed, nidAttempts,
		pctOf(cc.NIDTrusted+cc.NIDMarginal, nidAttempts))
	fmt.Fprintf(os.Stdout, "replay: tsbk  decoded=%d  trellis_failed=%d  crc_failed=%d  (of %d NID-passed frames; %s ok)\n",
		cc.TSBKDecoded, cc.TSBKTrellisFailed, cc.TSBKCRCFailed, tsbkAttempts,
		pctOf(cc.TSBKDecoded, tsbkAttempts))

	if len(s.decodeErrors) > 0 {
		parts := make([]string, 0, len(s.decodeErrors))
		for stage, n := range s.decodeErrors {
			parts = append(parts, fmt.Sprintf("%s=%d", stage, n))
		}
		fmt.Fprintf(os.Stdout, "replay: decode errors (from bus): %s\n", strings.Join(parts, " "))
	}
}

// pctOf renders num/den as a percentage string, or "n/a" when den is
// zero. Used by the EOF summary so "0 / 0 = NaN%" doesn't appear in
// the operator's pasted output.
func pctOf(num, den int64) string {
	if den <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(num)/float64(den))
}

// pickSampleDecoder maps the -format flag to a (decoder, bytes-per-IQ-pair)
// pair the read loop drives. u8 is the rtl_sdr default; f32 is GNU Radio's
// interleaved float32 cfile.
// estimateCaptureCarrierHz reads a prefix of f, estimates the dominant
// carrier offset across the full band, then rewinds f to the start so the
// prefix is still decoded by the main read loop. Used by -auto-tune.
func estimateCaptureCarrierHz(f *os.File, decode func([]byte, []complex64), bytesPerSample int, sampleRateHz float64) (float64, error) {
	const prefixSamples = 262144
	buf := make([]byte, prefixSamples*bytesPerSample)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return 0, err
	}
	if _, serr := f.Seek(0, io.SeekStart); serr != nil {
		return 0, serr
	}
	pairs := n / bytesPerSample
	if pairs == 0 {
		return 0, fmt.Errorf("capture has no samples to estimate from")
	}
	iq := make([]complex64, pairs)
	decode(buf[:pairs*bytesPerSample], iq)
	return dsp.EstimateCarrierOffsetHz(iq, sampleRateHz, sampleRateHz*0.5), nil
}

func pickSampleDecoder(format string) (func([]byte, []complex64), int, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "u8", "":
		return decodeU8Replay, 2, nil
	case "f32", "float32", "cfile":
		return decodeF32Replay, 8, nil
	default:
		return nil, 0, fmt.Errorf("unknown -format %q (want u8 or f32)", format)
	}
}

// decodeU8Replay converts rtl_sdr 8-bit unsigned interleaved IQ to
// complex64 in [-1, +1]. Mirrors internal/sdr/mock.go's decodeU8 but
// kept package-local so replay has no dependency on the mock SDR
// driver layer.
func decodeU8Replay(buf []byte, out []complex64) {
	n := len(buf) / 2
	for i := 0; i < n; i++ {
		ir := float32(buf[2*i]) - 127.5
		qr := float32(buf[2*i+1]) - 127.5
		out[i] = complex(ir/127.5, qr/127.5)
	}
}

// decodeF32Replay converts an interleaved-float32 GNU Radio cfile to
// complex64. The file is read as little-endian, matching the format
// gnuradio-companion emits on every platform GopherTrunk supports.
func decodeF32Replay(buf []byte, out []complex64) {
	n := len(buf) / 8
	for i := 0; i < n; i++ {
		ir := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i:]))
		qr := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i+4:]))
		out[i] = complex(ir, qr)
	}
}
