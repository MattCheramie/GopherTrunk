package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/gtbundle"
	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/sdr"
	"github.com/MattCheramie/GopherTrunk/internal/siglab"
)

// runCapture is the entry point for `gophertrunk capture`. It opens a live
// SDR directly (not through the daemon's pool), records the requested number
// of seconds of raw IQ to a GNU Radio cfile (interleaved little-endian
// float32) or the rtl_sdr-native unsigned-8-bit shape, and writes a siglab
// metadata sidecar so the capture is a drop-in fixture for the
// replay/analyze/test subcommands and the samples/ acceptance harness.
//
// Unlike the daemon's --iq-capture flag — which taps a control SDR already in
// the running pool (issue #402) — this is a standalone one-shot for grabbing a
// capture off a dedicated dongle without bringing the whole daemon up. Both
// paths stream chunk-by-chunk through siglab's shared capture encoder.
func runCapture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	verboseFlag := fs.Bool("verbose-errors", false, "print full error chain + stack on failures")
	serial := fs.String("serial", "", "SDR serial to capture from (default: the sole enumerated device)")
	freq := fs.Uint("freq", 0, "centre frequency in Hz (required)")
	sampleRate := fs.Uint("sample-rate", 2_400_000, "sample rate in Hz")
	gain := fs.Int("gain", -1, "tuner gain in tenths of a dB (-1 selects automatic gain control)")
	ppm := fs.Int("ppm", 0, "frequency-correction in PPM")
	seconds := fs.Float64("seconds", 10, "capture length in seconds (required, > 0)")
	out := fs.String("out", "capture.cfile", "output capture path")
	format := fs.String("format", "f32", "capture sample format: u8 | f32 | cs16 (f32 = GNU Radio cfile; cs16 = headerless 16-bit raw)")
	centerHz := fs.Uint("center", 0, "narrowband slice centre in Hz (default: -freq); with -bandwidth, carves a channel from the captured wideband without retuning")
	bandwidthHz := fs.Uint("bandwidth", 0, "narrowband slice bandwidth in Hz; when > 0 the capture is decimated to ~this rate around -center (must fit inside -freq ± sample-rate/2)")
	decimate := fs.Uint("decimate", 0, "integer software-decimation factor: record the full band anti-alias decimated to sample-rate/decimate (a manageable long capture off a source like the USRP B210 whose hardware rate floor is ~1 MS/s). Mutually exclusive with -bandwidth")
	protocol := fs.String("protocol", "", "protocol name written to the metadata sidecar (enables `test`; see `gophertrunk gen -list`)")
	source := fs.String("source", "", "free-text provenance written to the metadata sidecar")
	tune := fs.Float64("tune", 0, "fine software tune offset in Hz written to the metadata sidecar")
	autoTune := fs.Bool("auto-tune", false, "set auto_tune in the metadata sidecar")
	conjugate := fs.Bool("conjugate", false, "set conjugate (spectrum-inverted / I-Q-swapped front end) in the metadata sidecar")
	metaOut := fs.String("meta", "", "metadata sidecar path (default: <out stem>.metadata.json; \"none\" skips it)")
	bundleOut := fs.String("bundle", "", "also package the capture (+ metadata + a carved narrowband slice) into a GopherTrunk Bundle (.gtb.tar.gz) at this path")
	bundleIntent := fs.String("intent", "general", "bundle capture intent: cc-map | crypto | survey | general (sets the slice length)")
	bundleSigMF := fs.Bool("sigmf", false, "with -bundle, also emit a SigMF .sigmf-meta sidecar for the capture (interop with SigMF tooling)")
	listDevices := fs.Bool("list", false, "list the SDRs available to capture from and exit")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `gophertrunk capture — record raw IQ off a live SDR to a .cfile + metadata sidecar.

USAGE:
  gophertrunk capture -freq <hz> [-serial <s>] [-sample-rate <hz>] -seconds <n> -out <path>

EXAMPLES:
  # 30 s of P25 control-channel IQ at 2.4 MS/s → cfile + sidecar
  gophertrunk capture -freq 460000000 -sample-rate 2400000 -seconds 30 \
    -protocol p25 -out p25-cc.cfile

  # rtl_sdr-native u8 capture from a specific dongle
  gophertrunk capture -serial 00000001 -freq 453000000 -seconds 10 \
    -format u8 -out dmr.bin

  # Small 16-bit narrowband slice: carve a 50 kHz channel at 467.913 MHz out of
  # a 2.4 MS/s grab centred there — a shareable .raw a fraction of the full size
  gophertrunk capture -freq 467913000 -sample-rate 2400000 -seconds 30 \
    -center 467913000 -bandwidth 50000 -format cs16 -out cc.raw

  # USRP B210 (hardware rate floor ~1 MS/s): record the full band but software-
  # decimate ×5 → an alias-free 200 kS/s file, a fifth the size, still replayable
  gophertrunk capture -freq 467913000 -sample-rate 1000000 -seconds 60 \
    -decimate 5 -format cs16 -protocol tetra -out cc.raw

  # List the SDRs this binary can capture from
  gophertrunk capture -list

FLAGS:`)
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	resolveVerbose(*verboseFlag, false)
	rep := newReporter("capture")

	if *listDevices {
		infos, errs := sdr.EnumerateAll()
		for _, in := range infos {
			fmt.Printf("%s\t%s\t%s\n", in.Serial, in.Driver, in.Product)
		}
		if len(infos) == 0 {
			fmt.Fprintln(os.Stderr, "capture: no SDRs found")
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  driver error: %v\n", e)
			}
		}
		return
	}

	if *freq == 0 {
		fs.Usage()
		rep.Fatalf(2, "-freq is required")
	}
	if *seconds <= 0 {
		fs.Usage()
		rep.Fatalf(2, "-seconds must be > 0")
	}
	sampleFormat, err := siglab.ParseSampleFormat(*format)
	if err != nil {
		rep.Fatal(2, err)
	}

	dev, info, err := openCaptureDevice(*serial)
	if err != nil {
		rep.Fatal(1, err)
	}
	defer dev.Close()

	if err := dev.SetSampleRate(uint32(*sampleRate)); err != nil {
		rep.Fatal(1, fmt.Errorf("set sample rate: %w", err))
	}
	if err := dev.SetCenterFreq(uint32(*freq)); err != nil {
		rep.Fatal(1, fmt.Errorf("set centre frequency: %w", err))
	}
	if *ppm != 0 {
		if err := dev.SetPPM(*ppm); err != nil {
			rep.Fatal(1, fmt.Errorf("set ppm: %w", err))
		}
	}
	if err := dev.SetGain(*gain); err != nil {
		rep.Fatal(1, fmt.Errorf("set gain: %w", err))
	}

	// Ctrl-C ends the capture early but keeps whatever was written.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Count IQ chunks the driver drops while this capture runs. A dropped
	// chunk is a contiguous time gap in the recording — enough to slip a
	// downstream decoder's symbol clock and make the carrier appear to
	// "jump". The daemon's --iq-capture path already surfaces its drops
	// (iqcapture.go); this standalone path did not, so a corrupt capture was
	// silent to the operator. Install a device-scoped counter and warn at the
	// end if any dropped.
	drops := &captureDropCounter{serial: info.Serial}
	sdr.SetIQDropObserver(drops.observe)
	defer sdr.SetIQDropObserver(nil)

	stream, err := dev.StreamIQ(ctx)
	if err != nil {
		rep.Fatal(1, fmt.Errorf("start IQ stream: %w", err))
	}

	fmt.Printf("capture: %s[%s] @ %d Hz, %g MS/s → %s for %gs…\n",
		info.Driver, info.Serial, *freq, float64(*sampleRate)/1e6, *out, *seconds)

	if hint := captureSampleRateHint(uint32(*sampleRate), *protocol); hint != "" {
		fmt.Fprintln(os.Stderr, hint)
	}

	// Optional narrowband slice: decimate the captured wideband to a channel at
	// -center ± -bandwidth without retuning the SDR (the same DDC the daemon and
	// replay path use). The recorded rate + centre become the channel's.
	var ddc *ccdecoder.Downconverter
	recRate := float64(*sampleRate)
	recCenter := uint32(*freq)
	if *bandwidthHz > 0 && *decimate > 1 {
		rep.Fatalf(2, "-bandwidth and -decimate are mutually exclusive (both decimate; pick one)")
	}
	if *decimate > 1 {
		// Full-band integer decimation: keep the whole captured span, just at
		// a lower rate. Same down-converter as the -bandwidth slice, targeted
		// at sample-rate/decimate with no tuning offset (centre stays -freq).
		ddc = ccdecoder.NewDownconverter(float64(*sampleRate), float64(*sampleRate)/float64(*decimate))
		recRate = ddc.OutRateHz()
		fmt.Printf("capture: software decimation ×%d → %.1f kHz effective rate\n",
			*decimate, recRate/1e3)
	} else if *bandwidthHz > 0 {
		center := uint32(*centerHz)
		if center == 0 {
			center = uint32(*freq)
		}
		offsetHz := int64(center) - int64(*freq)
		half := int64(*sampleRate) / 2
		if absInt64(offsetHz)+int64(*bandwidthHz)/2 > half {
			rep.Fatalf(2, "-center %d + -bandwidth %d falls outside the captured span %d ± %d Hz",
				center, *bandwidthHz, *freq, half)
		}
		ddc = ccdecoder.NewDownconverterWithOffset(float64(*sampleRate), float64(*bandwidthHz), float64(offsetHz))
		recRate = ddc.OutRateHz()
		recCenter = center
		fmt.Printf("capture: narrowband slice → centre %.3f MHz, %.1f kHz channel rate\n",
			float64(recCenter)/1e6, recRate/1e3)
	}

	written, probe, capErr := captureToFile(ctx, *out, sampleFormat, stream, uint32(*sampleRate), *seconds, ddc)
	if capErr != nil && !errors.Is(capErr, context.Canceled) {
		rep.Fatal(1, fmt.Errorf("capture: %w (wrote %d samples to %s)", capErr, written, *out))
	}
	// Warn if a ppm error has pulled the recorded carrier off centre — the
	// other half of the "capture won't lock" story alongside dropped chunks.
	if w := carrierOffsetWarning(probe, recRate, recCenter); w != "" {
		fmt.Fprintln(os.Stderr, w)
	}
	if d := drops.count(); d > 0 {
		// Loud + actionable: a dropped-chunk capture looks fine on disk but
		// decodes as a clock-slipping, carrier-jumping mess. The bottleneck is
		// almost always the drain not keeping up with the native rate — the DDC
		// (-bandwidth) decimation cost or slow storage — so the driver sheds
		// whole chunks.
		fmt.Fprintf(os.Stderr,
			"capture: WARNING — the SDR dropped %d IQ chunk(s) during this capture. "+
				"Each is a time gap that corrupts downstream decode (symbol-clock slip / carrier jumps). "+
				"Remedy: lower -sample-rate, drop or widen -bandwidth (the DDC decimation is CPU-heavy), "+
				"or write to faster storage; then re-capture.\n", d)
	}

	meta := &siglab.Metadata{
		Protocol:     *protocol,
		Source:       *source,
		SampleRateHz: recRate,
		CenterFreqHz: recCenter,
		Format:       sampleFormat.String(),
		TuneHz:       *tune,
		AutoTune:     *autoTune,
		Conjugate:    *conjugate,
	}

	metaPath := *metaOut
	switch {
	case strings.EqualFold(metaPath, "none"):
		metaPath = ""
	case metaPath == "":
		metaPath = strings.TrimSuffix(*out, ext(*out)) + ".metadata.json"
	}
	if metaPath != "" {
		if err := siglab.WriteMetadata(metaPath, meta); err != nil {
			rep.Fatal(1, fmt.Errorf("write metadata: %w", err))
		}
	}

	if metaPath != "" {
		fmt.Printf("capture: wrote %d samples → %s  (metadata → %s)\n", written, *out, metaPath)
	} else {
		fmt.Printf("capture: wrote %d samples → %s\n", written, *out)
	}
	if *protocol == "" && metaPath != "" {
		fmt.Fprintln(os.Stderr, "capture: note — no -protocol set; sidecar is informational only (the `test` harness needs a protocol).")
	}

	if *bundleOut != "" {
		name := strings.TrimSuffix(baseName(*bundleOut), gtbundle.Ext)
		name = strings.TrimSuffix(strings.TrimSuffix(name, ".tar.gz"), ".gtb")
		if err := writeCaptureBundle(captureBundleParams{
			bundlePath:   *bundleOut,
			name:         name,
			intent:       gtbundle.CaptureIntent(*bundleIntent),
			capturePath:  *out,
			format:       sampleFormat,
			sampleRateHz: recRate,
			centerHz:     recCenter,
			tuneHz:       *tune,
			protocol:     *protocol,
			source:       *source,
			meta:         meta,
			emitSigMF:    *bundleSigMF,
		}); err != nil {
			rep.Fatal(1, fmt.Errorf("write bundle: %w", err))
		}
		fmt.Printf("capture: packaged → %s\n", *bundleOut)
	}
}

// openCaptureDevice enumerates the registered drivers and opens the device
// matching serial (or the sole device when serial is empty). Returns the open
// handle plus the chosen Info so the caller can report what it grabbed.
func openCaptureDevice(serial string) (sdr.Device, sdr.Info, error) {
	infos, errs := sdr.EnumerateAll()
	if len(infos) == 0 {
		if len(errs) > 0 {
			return nil, sdr.Info{}, fmt.Errorf("capture: no SDRs found (%v)", errs[0])
		}
		return nil, sdr.Info{}, errors.New("capture: no SDRs found")
	}
	var chosen sdr.Info
	if serial == "" {
		if len(infos) > 1 {
			return nil, sdr.Info{}, fmt.Errorf("capture: %d SDRs present; specify -serial (have: %s)", len(infos), serialsOf(infos))
		}
		chosen = infos[0]
	} else {
		found := false
		for _, in := range infos {
			if in.Serial == serial {
				chosen, found = in, true
				break
			}
		}
		if !found {
			return nil, sdr.Info{}, fmt.Errorf("capture: no SDR with serial %q (have: %s)", serial, serialsOf(infos))
		}
	}
	drv, err := sdr.DriverByName(chosen.Driver)
	if err != nil {
		return nil, sdr.Info{}, err
	}
	dev, err := drv.Open(chosen.Index)
	if err != nil {
		return nil, sdr.Info{}, fmt.Errorf("capture: open %s[%d]: %w", chosen.Driver, chosen.Index, err)
	}
	return dev, chosen, nil
}

// captureWriterDepth is how many encoded chunks captureStream buffers between
// the IQ-drain goroutine and the disk-writer goroutine. A storage-latency
// spike up to this many chunks is absorbed here instead of stalling the drain
// — which, on a live SDR, would make the driver shed whole IQ chunks
// (NotifyIQDrop), punching time gaps into the capture that slip a downstream
// decoder's symbol clock. Each entry is one post-DDC encoded chunk (a few KB
// for a narrowband slice), so this is trivial memory.
const captureWriterDepth = 256

// captureBufWriter is the disk write-buffer size. Batching many small chunk
// writes into one syscall keeps the writer goroutine from falling behind.
const captureBufWriter = 1 << 20 // 1 MiB

// captureToFile streams complex64 chunks from src, optionally decimates each
// chunk to a narrowband channel through ddc (nil = full band), encodes the
// result in the requested on-disk format with siglab's shared capture encoder,
// and writes to path until it has collected seconds*rate
// INPUT samples, the stream ends, ctx cancels, or a wall-clock safety deadline
// elapses. Returns the number of IQ samples written (post-decimation when ddc
// is set).
// captureToFile returns the samples written plus the first captureProbeSamples
// of recorded (post-DDC) IQ, which the caller FFTs for the carrier-offset
// warning.
func captureToFile(ctx context.Context, path string, format siglab.SampleFormat, src <-chan []complex64, rate uint32, seconds float64, ddc *ccdecoder.Downconverter) (int64, []complex64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, nil, fmt.Errorf("create %s: %w", path, err)
	}
	bw := bufio.NewWriterSize(f, captureBufWriter)
	written, probe, loopErr := captureStream(ctx, bw, format, src, rate, seconds, ddc)
	// captureStream has joined its writer goroutine before returning, so bw is
	// no longer touched concurrently — flush + close on this goroutine.
	if ferr := bw.Flush(); ferr != nil && loopErr == nil {
		loopErr = fmt.Errorf("flush: %w", ferr)
	}
	if cerr := f.Close(); cerr != nil && loopErr == nil {
		loopErr = cerr
	}
	return written, probe, loopErr
}

// captureStream is the format-encode + write loop behind captureToFile,
// decoupled from the file so it is testable with any io.Writer.
//
// The disk write runs on its own goroutine fed by a deep buffered channel, so
// a slow/stalling writer never stalls the goroutine draining src. That
// matters because src is the SDR driver's bounded delivery channel: if this
// goroutine blocks on a disk write, the driver's channel backs up and the
// driver drops whole IQ chunks (non-blocking, silent), corrupting the capture.
// Decoupling keeps the drain running so a transient stall costs latency, not
// samples. The stateful DDC stays on the drain goroutine (it must run in
// order); only the encoded bytes cross to the writer.
func captureStream(ctx context.Context, w io.Writer, format siglab.SampleFormat, src <-chan []complex64, rate uint32, seconds float64, ddc *ccdecoder.Downconverter) (int64, []complex64, error) {
	probe := make([]complex64, 0, captureProbeSamples)

	// Stop once seconds worth of INPUT samples have been read; the written
	// count may be far smaller when ddc decimates to a narrow channel.
	target := int64(seconds * float64(rate))
	// Safety deadline so a stalled/under-delivering device doesn't hang the
	// command forever waiting to reach the sample target. The timer is an
	// explicit select arm (not a post-receive check) so a source that never
	// delivers a chunk can't block the receive forever.
	timer := time.NewTimer(time.Duration(seconds*float64(time.Second)) + 5*time.Second)
	defer timer.Stop()

	writerCh := make(chan []byte, captureWriterDepth)
	writeErrCh := make(chan error, 1)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for b := range writerCh {
			if _, werr := w.Write(b); werr != nil {
				select {
				case writeErrCh <- werr:
				default:
				}
				// Keep draining so the producer never blocks on a full channel
				// after a write failure; stop writing.
				for range writerCh {
				}
				return
			}
		}
	}()

	var ddcBuf []complex64
	var inputRead, written int64
	var loopErr error
loop:
	for inputRead < target {
		select {
		case <-ctx.Done():
			loopErr = ctx.Err()
			break loop
		case <-timer.C:
			break loop
		case werr := <-writeErrCh:
			loopErr = fmt.Errorf("write: %w", werr)
			break loop
		case chunk, ok := <-src:
			if !ok {
				loopErr = errors.New("IQ stream ended before capture finished")
				break loop
			}
			inputRead += int64(len(chunk))
			samples := chunk
			if ddc != nil {
				ddcBuf = ddc.Process(ddcBuf, chunk)
				samples = ddcBuf
			}
			if len(samples) == 0 {
				continue
			}
			// Collect the first window of recorded samples for the caller's
			// carrier-offset probe (append copies, so it's safe against the
			// reused ddcBuf / the src-owned chunk).
			if len(probe) < captureProbeSamples {
				take := samples
				if n := captureProbeSamples - len(probe); len(take) > n {
					take = take[:n]
				}
				probe = append(probe, take...)
			}
			// Fresh buffer per chunk: it is handed to the writer goroutine, so
			// it must not alias the reused ddcBuf. siglab.EncodeCapture allocates
			// a new buffer per call, which the cross-goroutine handoff needs.
			buf := siglab.EncodeCapture(samples, format)
			select {
			case writerCh <- buf:
			case werr := <-writeErrCh:
				loopErr = fmt.Errorf("write: %w", werr)
				break loop
			case <-ctx.Done():
				loopErr = ctx.Err()
				break loop
			}
			written += int64(len(samples))
		}
	}

	close(writerCh)
	<-writerDone
	// A write error that landed as the loop was exiting (or during the final
	// drain) still counts.
	if loopErr == nil {
		select {
		case werr := <-writeErrCh:
			loopErr = fmt.Errorf("write: %w", werr)
		default:
		}
	}
	return written, probe, loopErr
}

// captureDropCounter counts SDR IQ-chunk drops for one device during a
// capture, via the process-wide sdr IQ-drop observer. It turns a silently
// corrupt capture (dropped chunks = time gaps) into a count the command can
// warn about.
type captureDropCounter struct {
	serial string
	n      atomic.Uint64
}

// observe is the sdr.SetIQDropObserver callback; it counts drops for the
// capture's device and ignores any other device's.
func (c *captureDropCounter) observe(info sdr.Info) {
	if info.Serial == c.serial {
		c.n.Add(1)
	}
}

// count returns the number of dropped chunks observed so far.
func (c *captureDropCounter) count() uint64 { return c.n.Load() }

// absInt64 is the absolute value of a signed 64-bit integer.
func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// serialsOf renders the serials of the discovered devices for a friendly hint.
// captureSampleRateHint returns a one-line front-end guidance note when the
// requested rate is far higher than a single narrowband trunking channel needs,
// or "" when no hint applies. Very high native rates on a wideband front-end
// (e.g. Airspy R2 at 10 MS/s) can raise the in-channel noise floor via clock
// phase noise / reciprocal mixing and degrade decode even without clipping
// (issue #771). The decoder down-converts one channel to 48 kHz regardless of
// capture rate, so a few MS/s is already generous for the narrowband
// (P25/DMR/NXDN/dPMR/YSF/D-STAR) family.
//
// The hint fires when the rate is high AND the capture is for a narrowband
// protocol (or none was declared — the common case, and the one in the #771
// repro). An explicit non-narrowband protocol is a deliberate wide grab, so it
// is left alone.
func captureSampleRateHint(sampleRateHz uint32, protocol string) string {
	// 4 MS/s sits above the 2.4 MS/s flag default and the recommended
	// ~2.5 MS/s while staying below the common 6/10 MSPS Airspy slots that
	// trip #771 — generous headroom for a single ~12.5 kHz channel.
	const highRateThresholdHz = 4_000_000

	if sampleRateHz <= highRateThresholdHz {
		return ""
	}
	if p := strings.ToLower(strings.TrimSpace(protocol)); p != "" && !isNarrowbandTrunkingProtocol(p) {
		return ""
	}
	return fmt.Sprintf(
		"capture: note — %.1f MS/s is a high native rate for a single narrowband channel "+
			"(the downconverter normalises to 48 kHz, so the extra bandwidth buys nothing). "+
			"On wideband front-ends (e.g. Airspy R2) the top native clock can degrade in-channel "+
			"SNR via phase noise / reciprocal mixing (issue #771); prefer ~2.4-2.5 MS/s for "+
			"control-channel captures. See docs/hardware.md \"Choosing a sample rate\".",
		float64(sampleRateHz)/1e6)
}

// isNarrowbandTrunkingProtocol reports whether p (case-insensitive, already
// trimmed/lowered by the caller) names a trunking protocol whose channels are
// narrow enough that a few hundred kHz of capture bandwidth is ample. Used to
// gate captureSampleRateHint so deliberate wideband IQ grabs aren't nagged.
func isNarrowbandTrunkingProtocol(p string) bool {
	switch p {
	case "p25", "p25p1", "p25-phase1", "p25p2", "p25-phase2",
		"dmr", "dmr-tier1", "dmr-tier2", "dmr-tier3",
		"nxdn", "nxdn48", "nxdn96", "dpmr", "dstar", "d-star", "ysf",
		"tetra", "mpt1327", "mpt-1327", "ltr", "edacs":
		return true
	default:
		return false
	}
}

func serialsOf(infos []sdr.Info) string {
	out := make([]string, 0, len(infos))
	for _, in := range infos {
		out = append(out, in.Serial)
	}
	return strings.Join(out, ", ")
}
